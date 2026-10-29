package test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"payment-processing-simulator/internal/api"
	"payment-processing-simulator/internal/domain"
	"payment-processing-simulator/internal/service"
	"payment-processing-simulator/internal/simulator"
	"payment-processing-simulator/internal/store"
)

func newTestServer() *httptest.Server {
	repository := store.NewMemoryTransactions()
	paymentService := service.New(repository, simulator.SimulatedIssuer{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := api.NewHandler(paymentService, store.NewMemoryIdempotency(0), []string{"merchant_demo"}, logger)
	return httptest.NewServer(handler.Routes())
}

func TestAuthorizeCaptureRefundAndVoid(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	authBody := `{"amount":150000,"currency":"IDR","card":{"number":"4111111111111111","exp_month":12,"exp_year":2030,"cvv":"123"}}`
	response := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/authorize", authBody, "auth-1")
	if response.Code != http.StatusCreated {
		t.Fatalf("authorize status=%d body=%s", response.Code, response.Body)
	}
	if strings.Contains(response.Body, "4111111111111111") || strings.Contains(response.Body, `"cvv"`) {
		t.Fatalf("response leaked sensitive card data: %s", response.Body)
	}
	var tx domain.Transaction
	if err := json.Unmarshal([]byte(response.Body), &tx); err != nil {
		t.Fatal(err)
	}
	if tx.Status != domain.StatusAuthorized || tx.Card.Last4 != "1111" {
		t.Fatalf("unexpected authorization result: %+v", tx)
	}
	capture := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/"+tx.ID+"/capture", `{"amount":100000,"currency":"IDR"}`, "capture-1")
	if capture.Code != http.StatusOK || !strings.Contains(capture.Body, `"status":"CAPTURED"`) {
		t.Fatalf("capture status=%d body=%s", capture.Code, capture.Body)
	}
	refund := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/"+tx.ID+"/refund", `{"amount":100000,"currency":"IDR"}`, "refund-1")
	if refund.Code != http.StatusOK || !strings.Contains(refund.Body, `"status":"REFUNDED"`) {
		t.Fatalf("refund status=%d body=%s", refund.Code, refund.Body)
	}

	response = doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/authorize", authBody, "auth-2")
	if response.Code != http.StatusCreated {
		t.Fatalf("second authorize status=%d body=%s", response.Code, response.Body)
	}
	if err := json.Unmarshal([]byte(response.Body), &tx); err != nil {
		t.Fatal(err)
	}
	void := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/"+tx.ID+"/void", "", "void-1")
	if void.Code != http.StatusOK || !strings.Contains(void.Body, `"status":"VOIDED"`) {
		t.Fatalf("void status=%d body=%s", void.Code, void.Body)
	}
}

func TestIdempotencyReplayAndBodyMismatch(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	body := `{"amount":150000,"currency":"IDR","card":{"number":"4111111111111111","exp_month":12,"exp_year":2030,"cvv":"123"}}`
	first := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/authorize", body, "same-key")
	replay := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/authorize", body, "same-key")
	if first.Code != replay.Code || first.Body != replay.Body {
		t.Fatalf("replayed response differs: first=%+v replay=%+v", first, replay)
	}
	mismatch := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/authorize", strings.Replace(body, "150000", "140000", 1), "same-key")
	if mismatch.Code != http.StatusUnprocessableEntity || !strings.Contains(mismatch.Body, "IDEMPOTENCY_KEY_REUSED") {
		t.Fatalf("mismatch status=%d body=%s", mismatch.Code, mismatch.Body)
	}
}

func TestRequiredHeadersAndHealth(t *testing.T) {
	server := newTestServer()
	defer server.Close()
	response := doRequest(t, server.Client(), http.MethodGet, server.URL+"/healthz", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body)
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/v1/transactions/authorize", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Merchant-ID", "merchant_demo")
	resp, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(data), "MISSING_IDEMPOTENCY_KEY") {
		t.Fatalf("missing key status=%d body=%s", resp.StatusCode, data)
	}
	request, err = http.NewRequest(http.MethodGet, server.URL+"/v1/transactions/missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Merchant-ID", "unconfigured")
	resp, err = server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown merchant status=%d", resp.StatusCode)
	}
}

func TestRequestLogsNeverContainCardData(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	paymentService := service.New(store.NewMemoryTransactions(), simulator.SimulatedIssuer{})
	handler := api.NewHandler(paymentService, store.NewMemoryIdempotency(0), []string{"merchant_demo"}, logger)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()
	body := `{"amount":150000,"currency":"IDR","card":{"number":"4111111111111111","exp_month":12,"exp_year":2030,"cvv":"9876"}}`
	response := doRequest(t, server.Client(), http.MethodPost, server.URL+"/v1/transactions/authorize", body, "safe-log")
	if response.Code != http.StatusCreated {
		t.Fatalf("authorize status=%d body=%s", response.Code, response.Body)
	}
	if strings.Contains(logOutput.String(), "4111111111111111") || strings.Contains(logOutput.String(), "9876") {
		t.Fatalf("request logs contain sensitive card data: %s", logOutput.String())
	}
}

type result struct {
	Code int
	Body string
}

func doRequest(t *testing.T, client *http.Client, method, url, body, key string) result {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Merchant-ID", "merchant_demo")
	request.Header.Set("Idempotency-Key", key)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return result{Code: response.StatusCode, Body: string(data)}
}
