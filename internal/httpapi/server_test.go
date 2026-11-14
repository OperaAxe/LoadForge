package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	recorder:=httptest.NewRecorder()
	request:=httptest.NewRequest(http.MethodGet,"/healthz",nil)
	Server{}.Handler().ServeHTTP(recorder,request)
	if recorder.Code!=http.StatusOK {t.Fatalf("status=%d body=%s",recorder.Code,recorder.Body.String())}
	if !strings.Contains(recorder.Body.String(),`"status":"ok"`) {t.Fatalf("unexpected body: %s",recorder.Body.String())}
}
func TestAuthorizationRequired(t *testing.T) {
	body:=`{"url":"https://example.com","authorization_confirmed":false}`
	recorder:=httptest.NewRecorder()
	request:=httptest.NewRequest(http.MethodPost,"/api/v1/runs",strings.NewReader(body))
	Server{}.Handler().ServeHTTP(recorder,request)
	if recorder.Code!=http.StatusBadRequest {t.Fatalf("status=%d body=%s",recorder.Code,recorder.Body.String())}
}
func TestUnknownFieldsRejected(t *testing.T) {
	body:=`{"url":"https://example.com","authorization_confirmed":true,"unexpected":1}`
	recorder:=httptest.NewRecorder()
	request:=httptest.NewRequest(http.MethodPost,"/api/v1/runs",strings.NewReader(body))
	Server{}.Handler().ServeHTTP(recorder,request)
	if recorder.Code!=http.StatusBadRequest {t.Fatalf("status=%d body=%s",recorder.Code,recorder.Body.String())}
}
