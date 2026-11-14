package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Server{}.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestRunEndpointRequiresAuthorization(t *testing.T) {
	body := `{"url":"https://example.com","authorization_confirmed":false}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", strings.NewReader(body))
	Server{}.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRunEndpointRejectsUnknownFields(t *testing.T) {
	body := `{"url":"https://example.com","authorization_confirmed":true,"unexpected":1}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", strings.NewReader(body))
	Server{}.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRunEndpointExecutesBoundedTest(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Instance-Id", "api-test-node")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer target.Close()

	body := `{"url":"` + target.URL + `","concurrency":2,"duration_seconds":1,"requests_per_second":2,"max_requests":2,"authorization_confirmed":true}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs", strings.NewReader(body))
	Server{AllowPrivateTargets: true}.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"completed_requests":2`) {
		t.Fatalf("expected two completed requests, body=%s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"api-test-node":2`) {
		t.Fatalf("expected backend header observations, body=%s", recorder.Body.String())
	}
}
