package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/OperaAxe/LoadForge/internal/httpapi"
	"github.com/OperaAxe/LoadForge/internal/loadtest"
)

const version="0.1.0"

func main() {
	if len(os.Args)<2 {usage();os.Exit(2)}
	var err error
	switch os.Args[1] {
	case "run": err=runCommand(os.Args[2:])
	case "serve": err=serveCommand(os.Args[2:])
	case "version","--version","-v":fmt.Printf("LoadForge %s\n",version)
	default:usage();os.Exit(2)
	}
	if err!=nil {fmt.Fprintln(os.Stderr,"error:",err);os.Exit(1)}
}
func usage() {
	fmt.Println("LoadForge — bounded HTTP performance and backend signal tester")
	fmt.Println("Usage: loadforge run --url https://your-service.example --authorization-confirmed")
	fmt.Println("       loadforge serve --addr :8080")
	fmt.Println("       loadforge version")
}
func runCommand(args []string) error {
	defaults:=loadtest.DefaultConfig()
	fs:=flag.NewFlagSet("run",flag.ContinueOnError)
	url:=fs.String("url","","HTTP or HTTPS URL to test")
	concurrency:=fs.Int("concurrency",defaults.Concurrency,"Concurrent workers (1-25)")
	duration:=fs.Duration("duration",defaults.Duration,"Test duration (1s-30s)")
	rps:=fs.Int("rps",defaults.RequestsPerSecond,"Maximum scheduled request rate (1-50 per second)")
	maxRequests:=fs.Int("max-requests",defaults.MaxRequests,"Hard request cap (1-5000)")
	timeout:=fs.Duration("timeout",defaults.RequestTimeout,"Per-request timeout (maximum 10s)")
	out:=fs.String("json","","Optional path to write the full JSON report")
	authorized:=fs.Bool("authorization-confirmed",false,"Confirm ownership or explicit permission to test this target")
	allowPrivate:=fs.Bool("allow-private-targets",false,"Allow localhost/private IPs for local development only")
	if err:=fs.Parse(args);err!=nil{return err}
	cfg:=loadtest.Config{URL:*url,Concurrency:*concurrency,Duration:*duration,RequestsPerSecond:*rps,MaxRequests:*maxRequests,RequestTimeout:*timeout,Authorization:*authorized,AllowPrivateTargets:*allowPrivate}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM)
	defer stop()
	result,err:=loadtest.Run(ctx,cfg)
	if err!=nil{return err}
	printSummary(result)
	if *out!="" {
		data,err:=json.MarshalIndent(result,"","  ")
		if err!=nil{return err}
		dir:=filepath.Dir(*out)
		if dir!="." {if err:=os.MkdirAll(dir,0755);err!=nil{return err}}
		if err:=os.WriteFile(*out,append(data,'\n'),0600);err!=nil{return err}
		fmt.Printf("\nJSON report written to %s\n",*out)
	}
	return nil
}
func printSummary(r loadtest.Result) {
	fmt.Printf("\nLoadForge report — %s\n",r.Target)
	fmt.Printf("Elapsed: %.2fs | Scheduled: %d | Completed: %d\n",r.ElapsedSeconds,r.ScheduledRequests,r.CompletedRequests)
	fmt.Printf("RPS: %.2f | Successful: %d | Failed: %d | Network errors: %d\n",r.RequestsPerSecond,r.Successful,r.Failed,r.NetworkErrors)
	fmt.Printf("Latency ms: avg %.2f | p50 %.2f | p90 %.2f | p95 %.2f | p99 %.2f | max %.2f\n",r.Latency.AverageMS,r.Latency.P50MS,r.Latency.P90MS,r.Latency.P95MS,r.Latency.P99MS,r.Latency.MaxMS)
	fmt.Printf("HTTP status counts: %v\n",r.StatusCodes)
	if len(r.BackendSignals)==0 {fmt.Println("Backend distribution: no recognized instance-identifying response headers observed.")} else {fmt.Printf("Backend signals (not proof of load-balancer configuration): %v\n",r.BackendSignals)}
	for _,note:=range r.Notes {fmt.Printf("Note: %s\n",note)}
}
func serveCommand(args []string) error {
	fs:=flag.NewFlagSet("serve",flag.ContinueOnError)
	addr:=fs.String("addr","127.0.0.1:8080","HTTP listen address")
	staticDir:=fs.String("static","web/dist","Built dashboard directory")
	allowPrivate:=fs.Bool("allow-private-targets",strings.EqualFold(os.Getenv("LOADFORGE_ALLOW_PRIVATE_TARGETS"),"true"),"Allow private targets; use only on a trusted local instance")
	if err:=fs.Parse(args);err!=nil{return err}
	if *allowPrivate {fmt.Fprintln(os.Stderr,"WARNING: private-network targets are enabled for this server instance.")}
	server:=&http.Server{Addr:*addr,Handler:httpapi.Server{AllowPrivateTargets:*allowPrivate,StaticDir:*staticDir}.Handler(),ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,WriteTimeout:45*time.Second,IdleTimeout:60*time.Second,MaxHeaderBytes:1<<20}
	fmt.Printf("LoadForge %s listening on %s\n",version,*addr)
	fmt.Println("Do not expose this service publicly without additional authentication and network isolation.")
	err:=server.ListenAndServe()
	if errors.Is(err,http.ErrServerClosed){return nil}
	return err
}
