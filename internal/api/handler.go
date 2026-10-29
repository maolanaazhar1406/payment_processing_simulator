package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"payment-processing-simulator/internal/domain"
	"payment-processing-simulator/internal/service"
	"payment-processing-simulator/internal/store"
)

const maxRequestBody = 1 << 20

type transactionService interface {
	Authorize(context.Context, service.AuthorizeInput) (domain.Transaction, error)
	Capture(context.Context, string, string, domain.Money) (domain.Transaction, error)
	Void(context.Context, string, string) (domain.Transaction, error)
	Refund(context.Context, string, string, domain.Money) (domain.Transaction, error)
	GetTransaction(context.Context, string, string) (domain.Transaction, error)
}

// Handler serves the simulator HTTP API.
type Handler struct {
	service     transactionService
	idempotency idempotencyStore
	merchants   map[string]struct{}
	logger      *slog.Logger
}

type idempotencyStore interface {
	Start(context.Context, string, string, string) (store.IdempotencyRecord, bool, error)
	Complete(context.Context, string, string, string, int, []byte) error
	Abort(context.Context, string, string, string)
}

// NewHandler creates an API handler with merchant authentication and idempotency.
func NewHandler(paymentService transactionService, idempotencyStore idempotencyStore, merchants []string, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	merchantSet := make(map[string]struct{}, len(merchants))
	for _, merchant := range merchants {
		merchant = strings.TrimSpace(merchant)
		if merchant != "" {
			merchantSet[merchant] = struct{}{}
		}
	}
	return &Handler{service: paymentService, idempotency: idempotencyStore, merchants: merchantSet, logger: logger}
}

// Routes returns the method-aware API router.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("POST /v1/transactions/authorize", h.authorized(h.idempotent(http.HandlerFunc(h.authorize))))
	mux.Handle("POST /v1/transactions/{id}/capture", h.authorized(h.idempotent(http.HandlerFunc(h.capture))))
	mux.Handle("POST /v1/transactions/{id}/void", h.authorized(h.idempotent(http.HandlerFunc(h.void))))
	mux.Handle("POST /v1/transactions/{id}/refund", h.authorized(h.idempotent(http.HandlerFunc(h.refund))))
	mux.Handle("GET /v1/transactions/{id}", h.authorized(http.HandlerFunc(h.getTransaction)))
	return h.logging(mux)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
		Card     struct {
			Number   string `json:"number"`
			ExpMonth int    `json:"exp_month"`
			ExpYear  int    `json:"exp_year"`
			CVV      string `json:"cvv"`
		} `json:"card"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	amount, err := domain.NewMoney(request.Amount, request.Currency)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_AMOUNT", "amount or currency is invalid")
		return
	}
	tx, err := h.service.Authorize(r.Context(), service.AuthorizeInput{
		MerchantID: r.Header.Get("X-Merchant-ID"), Amount: amount, PAN: request.Card.Number,
		ExpMonth: request.Card.ExpMonth, ExpYear: request.Card.ExpYear, CVV: request.Card.CVV,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}

func (h *Handler) capture(w http.ResponseWriter, r *http.Request) {
	amount, err := readAmount(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid amount")
		return
	}
	tx, err := h.service.Capture(r.Context(), r.Header.Get("X-Merchant-ID"), r.PathValue("id"), amount)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) void(w http.ResponseWriter, r *http.Request) {
	tx, err := h.service.Void(r.Context(), r.Header.Get("X-Merchant-ID"), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) refund(w http.ResponseWriter, r *http.Request) {
	amount, err := readAmount(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid amount")
		return
	}
	tx, err := h.service.Refund(r.Context(), r.Header.Get("X-Merchant-ID"), r.PathValue("id"), amount)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	tx, err := h.service.GetTransaction(r.Context(), r.Header.Get("X-Merchant-ID"), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) authorized(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		merchantID := strings.TrimSpace(r.Header.Get("X-Merchant-ID"))
		if merchantID == "" {
			writeError(w, http.StatusBadRequest, "MISSING_MERCHANT", "X-Merchant-ID is required")
			return
		}
		if _, ok := h.merchants[merchantID]; !ok {
			writeError(w, http.StatusUnauthorized, "INVALID_MERCHANT", "merchant is not configured")
			return
		}
		r.Header.Set("X-Merchant-ID", merchantID)
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) idempotent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" {
			writeError(w, http.StatusBadRequest, "MISSING_IDEMPOTENCY_KEY", "Idempotency-Key is required")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body is too large or unreadable")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		hashBytes := sha256.Sum256(body)
		hash := hex.EncodeToString(hashBytes[:])
		merchantID := r.Header.Get("X-Merchant-ID")
		record, replay, err := h.idempotency.Start(r.Context(), merchantID, key, hash)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrIdempotencyKeyReused):
				writeError(w, http.StatusUnprocessableEntity, "IDEMPOTENCY_KEY_REUSED", "key was already used with a different request")
			case errors.Is(err, store.ErrRequestInProgress):
				writeError(w, http.StatusConflict, "REQUEST_IN_PROGRESS", "request with this key is still in progress")
			default:
				h.logger.Error("idempotency reservation failed", "error", err)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
			}
			return
		}
		if replay {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.ResponseStatus)
			_, _ = w.Write(record.ResponseBody)
			return
		}
		defer h.idempotency.Abort(context.WithoutCancel(r.Context()), merchantID, key, hash)
		buffer := newResponseBuffer()
		next.ServeHTTP(buffer, r)
		if err := h.idempotency.Complete(r.Context(), merchantID, key, hash, buffer.status, buffer.body.Bytes()); err != nil {
			h.logger.Error("idempotency response could not be saved", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
			return
		}
		buffer.copyTo(w)
	})
}

type responseBuffer struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{header: make(http.Header), status: http.StatusOK}
}

func (w *responseBuffer) Header() http.Header { return w.header }
func (w *responseBuffer) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}
}
func (w *responseBuffer) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(body)
}
func (buffer *responseBuffer) copyTo(destination http.ResponseWriter) {
	for key, values := range buffer.header {
		for _, value := range values {
			destination.Header().Add(key, value)
		}
	}
	if !buffer.wroteHeader {
		buffer.status = http.StatusOK
	}
	destination.WriteHeader(buffer.status)
	_, _ = destination.Write(buffer.body.Bytes())
}

func (h *Handler) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		h.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration", time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidMoney), errors.Is(err, domain.ErrCurrencyMismatch):
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "request values are invalid")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "transaction was not found")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "merchant does not own this transaction")
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrVersionConflict):
		writeError(w, http.StatusConflict, "INVALID_TRANSITION", "transaction operation is not valid in its current state")
	default:
		h.logger.Error("payment operation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
	}
}

func readAmount(r *http.Request) (domain.Money, error) {
	var request struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := decodeJSON(r, &request); err != nil {
		return domain.Money{}, err
	}
	return domain.NewMoney(request.Amount, request.Currency)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxRequestBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode JSON: expected exactly one object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
