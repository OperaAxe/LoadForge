package loadtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var signalHeaders = []string{
	"Server", "Via", "X-Served-By", "X-Backend", "X-Backend-Server",
	"X-Instance-Id", "X-Pod-Name", "X-Upstream-Addr", "X-Upstream-Host",
}

func Run(ctx context.Context, cfg Config) (Result, error) {
	if err := cfg.Validate(); err != nil { return Result{}, err }
	u, err := validateTarget(ctx, cfg.URL, cfg.AllowPrivateTargets)
	if err != nil { return Result{}, err }

	transport := &http.Transport{
		Proxy: nil, DialContext: safeDialContext(cfg.AllowPrivateTargets),
		MaxIdleConns: cfg.Concurrency*2, MaxIdleConnsPerHost: cfg.Concurrency, MaxConnsPerHost: cfg.Concurrency,
		IdleConnTimeout: 30*time.Second, TLSHandshakeTimeout: 5*time.Second, ResponseHeaderTimeout: cfg.RequestTimeout,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: cfg.RequestTimeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}

	started, startClock := time.Now().UTC(), time.Now()
	jobs := make(chan struct{}, cfg.Concurrency*2)
	samples := make([]sample, 0, cfg.MaxRequests)
	var sampleMu sync.Mutex
	var workers sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range jobs {
				item := performRequest(ctx, client, u.String(), cfg.RequestTimeout)
				sampleMu.Lock()
				samples = append(samples, item)
				sampleMu.Unlock()
			}
		}()
	}

	ticker := time.NewTicker(time.Second/time.Duration(cfg.RequestsPerSecond))
	defer ticker.Stop()
	deadline := time.NewTimer(cfg.Duration)
	defer deadline.Stop()
	scheduled := 0
scheduleLoop:
	for scheduled < cfg.MaxRequests {
		select {
		case <-ctx.Done(): break scheduleLoop
		case <-deadline.C: break scheduleLoop
		case <-ticker.C:
			select {
			case jobs <- struct{}{}: scheduled++
			default: // Skip a tick rather than building an unbounded backlog.
			}
		}
	}
	close(jobs)
	workers.Wait()
	elapsed := time.Since(startClock).Seconds()
	if elapsed <= 0 { elapsed = 0.001 }
	result := buildResult(u.String(), started, elapsed, scheduled, samples)
	if ctx.Err() != nil { result.Notes = append(result.Notes, "The run stopped because the client request was cancelled.") }
	if len(result.BackendSignals) == 0 {
		result.Notes = append(result.Notes, "No recognized backend-instance response headers were observed. This does not prove that a load balancer is absent.")
	} else {
		result.Notes = append(result.Notes, "Backend signals are based only on response headers exposed by the target; they do not prove internal load-balancer configuration.")
	}
	if result.CompletedRequests == 0 { return result, fmt.Errorf("the test completed no requests; check the target and network access") }
	return result, nil
}

func performRequest(parent context.Context, client *http.Client, target string, timeout time.Duration) sample {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil { return sample{err:true} }
	request.Header.Set("User-Agent", "LoadForge/0.1 (+authorized-performance-test)")
	request.Header.Set("Accept", "*/*")
	start := time.Now()
	response, err := client.Do(request)
	latency := time.Since(start)
	if err != nil { return sample{latency:latency, err:true} }
	defer response.Body.Close()
	bytesRead, _ := io.CopyN(io.Discard, response.Body, 64*1024)
	item := sample{latency:latency, status:response.StatusCode, bytes:bytesRead}
	if response.StatusCode >= 400 { item.err = true }
	item.signals = make(map[string]string)
	for _, name := range signalHeaders {
		if value := response.Header.Get(name); value != "" {
			if len(value)>120 { value = value[:120] }
			item.signals[name] = value
		}
	}
	return item
}

func buildResult(target string, started time.Time, elapsed float64, scheduled int, samples []sample) Result {
	result := Result{Target:target, StartedAt:started, ElapsedSeconds:elapsed, ScheduledRequests:scheduled,
		StatusCodes:make(map[string]int), BackendSignals:make(map[string]map[string]int)}
	var latencies []sample
	for _, item := range samples {
		result.CompletedRequests++
		result.BytesRead += item.bytes
		if item.status>0 {
			result.StatusCodes[fmt.Sprintf("%d",item.status)]++
			if item.status>=400 { result.HTTPErrorResponses++ }
		}
		if item.err {
			result.Failed++
			if item.status==0 { result.NetworkErrors++ }
		} else { result.Successful++ }
		if item.latency>0 { latencies=append(latencies,item) }
		for header,value := range item.signals {
			if _,ok := result.BackendSignals[header]; !ok { result.BackendSignals[header]=make(map[string]int) }
			result.BackendSignals[header][value]++
		}
	}
	result.RequestsPerSecond=float64(result.CompletedRequests)/elapsed
	result.Latency=summarizeLatencies(latencies)
	return result
}
