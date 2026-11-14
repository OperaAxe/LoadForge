package loadtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunCollectsMetricsAndSignals(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("X-Instance-Id","node-a")
		w.WriteHeader(http.StatusOK)
		_,_=w.Write([]byte("ok"))
	}))
	defer server.Close()
	cfg:=DefaultConfig()
	cfg.URL=server.URL
	cfg.Authorization=true
	cfg.AllowPrivateTargets=true
	cfg.Concurrency=2
	cfg.Duration=time.Second
	cfg.RequestsPerSecond=5
	cfg.MaxRequests=5
	result,err:=Run(context.Background(),cfg)
	if err!=nil { t.Fatalf("Run() error=%v",err) }
	if result.CompletedRequests==0 { t.Fatal("expected requests") }
	if result.Successful!=result.CompletedRequests { t.Fatalf("successful=%d completed=%d",result.Successful,result.CompletedRequests) }
	if result.BackendSignals["X-Instance-Id"]["node-a"]==0 { t.Fatalf("missing signal: %#v",result.BackendSignals) }
}

func TestAuthorizationRequired(t *testing.T) {
	cfg:=DefaultConfig()
	cfg.URL="https://example.com"
	if _,err:=Run(context.Background(),cfg); err==nil { t.Fatal("expected authorization error") }
}

func TestPrivateIPBlockedByDefault(t *testing.T) {
	ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
	defer cancel()
	if _,err:=validateTarget(ctx,"http://127.0.0.1:8080",false); err==nil { t.Fatal("expected private IP blocked") }
}

func TestConcurrencyHardLimit(t *testing.T) {
	cfg:=DefaultConfig()
	cfg.URL="https://example.com"
	cfg.Authorization=true
	cfg.Concurrency=MaxConcurrency+1
	if err:=cfg.Validate(); err==nil { t.Fatal("expected limit error") }
}
