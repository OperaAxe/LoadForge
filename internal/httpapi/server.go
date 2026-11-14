package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/OperaAxe/LoadForge/internal/loadtest"
)

type Server struct {
	AllowPrivateTargets bool
	StaticDir string
}
type runRequest struct {
	URL string `json:"url"`
	Concurrency int `json:"concurrency"`
	DurationSeconds int `json:"duration_seconds"`
	RequestsPerSecond int `json:"requests_per_second"`
	MaxRequests int `json:"max_requests"`
	AuthorizationConfirmed bool `json:"authorization_confirmed"`
}
type errorResponse struct { Error string `json:"error"` }

func (s Server) Handler() http.Handler {
	mux:=http.NewServeMux()
	mux.HandleFunc("GET /healthz",func(w http.ResponseWriter,_ *http.Request){writeJSON(w,http.StatusOK,map[string]string{"status":"ok","service":"loadforge"})})
	mux.HandleFunc("POST /api/v1/runs",s.run)
	mux.HandleFunc("GET /",s.index)
	return securityHeaders(mux)
}
func (s Server) run(w http.ResponseWriter,r *http.Request) {
	defer r.Body.Close()
	decoder:=json.NewDecoder(http.MaxBytesReader(w,r.Body,64*1024))
	decoder.DisallowUnknownFields()
	var input runRequest
	if err:=decoder.Decode(&input); err!=nil {writeError(w,http.StatusBadRequest,"invalid JSON request: "+err.Error());return}
	if err:=ensureEOF(decoder); err!=nil {writeError(w,http.StatusBadRequest,"request body must contain one JSON object");return}
	cfg:=loadtest.DefaultConfig()
	cfg.URL=input.URL
	cfg.Authorization=input.AuthorizationConfirmed
	cfg.AllowPrivateTargets=s.AllowPrivateTargets
	if input.Concurrency!=0 {cfg.Concurrency=input.Concurrency}
	if input.DurationSeconds!=0 {cfg.Duration=time.Duration(input.DurationSeconds)*time.Second}
	if input.RequestsPerSecond!=0 {cfg.RequestsPerSecond=input.RequestsPerSecond}
	if input.MaxRequests!=0 {cfg.MaxRequests=input.MaxRequests}
	result,err:=loadtest.Run(r.Context(),cfg)
	if err!=nil {
		status:=http.StatusBadRequest
		if errors.Is(err,context.Canceled)||errors.Is(err,context.DeadlineExceeded) {status=http.StatusRequestTimeout}
		writeError(w,status,err.Error())
		return
	}
	writeJSON(w,http.StatusOK,result)
}
func (s Server) index(w http.ResponseWriter,r *http.Request) {
	if s.StaticDir=="" {s.StaticDir="web/dist"}
	if _,err:=os.Stat(filepath.Join(s.StaticDir,"index.html"));err!=nil {
		w.Header().Set("Content-Type","text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_,_=io.WriteString(w,"LoadForge API is running. Build the dashboard with npm install && npm run build in web/.\\n")
		return
	}
	http.FileServer(http.Dir(s.StaticDir)).ServeHTTP(w,r)
}
func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err:=decoder.Decode(&extra);err!=io.EOF {if err==nil{return fmt.Errorf("multiple JSON values")};return err}
	return nil
}
func writeJSON(w http.ResponseWriter,status int,value any) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.WriteHeader(status)
	_=json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter,status int,message string) {writeJSON(w,status,errorResponse{Error:message})}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("X-Content-Type-Options","nosniff")
		w.Header().Set("Referrer-Policy","no-referrer")
		w.Header().Set("X-Frame-Options","DENY")
		next.ServeHTTP(w,r)
	})
}
