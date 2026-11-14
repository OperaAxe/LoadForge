package loadtest

import (
	"math"
	"sort"
	"time"
)

type LatencyStats struct {
	AverageMS float64 `json:"average_ms"`
	MinMS float64 `json:"min_ms"`
	MaxMS float64 `json:"max_ms"`
	P50MS float64 `json:"p50_ms"`
	P90MS float64 `json:"p90_ms"`
	P95MS float64 `json:"p95_ms"`
	P99MS float64 `json:"p99_ms"`
}

type Result struct {
	Target string `json:"target"`
	StartedAt time.Time `json:"started_at"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	ScheduledRequests int `json:"scheduled_requests"`
	CompletedRequests int `json:"completed_requests"`
	Successful int `json:"successful_requests"`
	Failed int `json:"failed_requests"`
	NetworkErrors int `json:"network_errors"`
	HTTPErrorResponses int `json:"http_error_responses"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	BytesRead int64 `json:"bytes_read"`
	Latency LatencyStats `json:"latency"`
	StatusCodes map[string]int `json:"status_codes"`
	BackendSignals map[string]map[string]int `json:"backend_signals"`
	Notes []string `json:"notes"`
}

type sample struct {
	latency time.Duration
	status int
	bytes int64
	err bool
	signals map[string]string
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 { return 0 }
	index := int(math.Ceil(p*float64(len(values)))) - 1
	if index < 0 { index = 0 }
	if index >= len(values) { index = len(values)-1 }
	return values[index]
}

func summarizeLatencies(samples []sample) LatencyStats {
	if len(samples) == 0 { return LatencyStats{} }
	values := make([]float64, 0, len(samples))
	var sum float64
	for _, item := range samples {
		ms := float64(item.latency)/float64(time.Millisecond)
		values = append(values, ms)
		sum += ms
	}
	sort.Float64s(values)
	return LatencyStats{
		AverageMS: sum/float64(len(values)), MinMS: values[0], MaxMS: values[len(values)-1],
		P50MS: percentile(values,.50), P90MS: percentile(values,.90), P95MS: percentile(values,.95), P99MS: percentile(values,.99),
	}
}
