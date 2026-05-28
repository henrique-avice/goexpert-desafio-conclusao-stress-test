package entity

import (
	"fmt"
	"math"
	"net"
	"sort"
	"strings"
	"time"
)

const (
	HistogramBucket0_10ms     = "0-10ms"
	HistogramBucket10_50ms    = "10-50ms"
	HistogramBucket50_100ms   = "50-100ms"
	HistogramBucket100_500ms  = "100-500ms"
	HistogramBucket500_1000ms = "500-1000ms"
	HistogramBucket1000msPlus = "1000ms+"
)

// Report represents the aggregated results of a load test
type Report struct {
	URL               string
	TotalRequests     int
	ConcurrentWorkers int
	Duration          time.Duration
	RequestsPerSec    float64

	SuccessCount int
	ErrorCount   int
	SuccessRate  float64

	StatusCounts map[int]int

	LatencyMin    time.Duration
	LatencyMax    time.Duration
	LatencyAvg    time.Duration
	LatencyP50    time.Duration
	LatencyP75    time.Duration
	LatencyP90    time.Duration
	LatencyP95    time.Duration
	LatencyP99    time.Duration
	LatencyP999   time.Duration
	LatencyStdDev time.Duration

	ErrorCounts  map[string]int
	TimeoutCount int

	LatencyHistogram map[string]int
}

// NewReport creates a new Report instance
func NewReport() *Report {
	return &Report{
		StatusCounts:     make(map[int]int),
		ErrorCounts:      make(map[string]int),
		LatencyHistogram: make(map[string]int),
	}
}

// CalculateFromResults aggregates results into report metrics
func (r *Report) CalculateFromResults(results []RequestResult, duration time.Duration) {
	if len(results) == 0 {
		return
	}

	r.TotalRequests = len(results)
	r.Duration = duration
	if duration.Seconds() > 0 {
		r.RequestsPerSec = float64(len(results)) / duration.Seconds()
	}

	var latencies []time.Duration
	var successCount, errorCount, timeoutCount int

	for _, result := range results {
		if result.Error != nil {
			errorCount++
			errType := classifyError(result.Error)
			r.ErrorCounts[errType]++
			if errType == "timeout" {
				timeoutCount++
			}
		} else {
			successCount++
			r.StatusCounts[result.StatusCode]++
			latencies = append(latencies, result.Latency)
		}
	}

	r.SuccessCount = successCount
	r.ErrorCount = errorCount
	r.TimeoutCount = timeoutCount
	r.SuccessRate = float64(successCount) / float64(r.TotalRequests) * 100

	if len(latencies) > 0 {
		r.calculateLatencyMetrics(latencies)
		r.generateLatencyHistogram(latencies)
	}
}

func (r *Report) calculateLatencyMetrics(latencies []time.Duration) {
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	sort.Slice(sortedLatencies, func(i, j int) bool {
		return sortedLatencies[i] < sortedLatencies[j]
	})

	r.LatencyMin = sortedLatencies[0]
	r.LatencyMax = sortedLatencies[len(sortedLatencies)-1]

	var sum time.Duration
	for _, lat := range sortedLatencies {
		sum += lat
	}
	r.LatencyAvg = time.Duration(int64(sum) / int64(len(latencies)))

	r.LatencyP50 = r.percentile(sortedLatencies, 50)
	r.LatencyP75 = r.percentile(sortedLatencies, 75)
	r.LatencyP90 = r.percentile(sortedLatencies, 90)
	r.LatencyP95 = r.percentile(sortedLatencies, 95)
	r.LatencyP99 = r.percentile(sortedLatencies, 99)
	r.LatencyP999 = r.percentile(sortedLatencies, 99.9)

	r.LatencyStdDev = r.calculateStdDev(r.LatencyAvg, latencies)
}

func (r *Report) percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	fraction := index - float64(lower)
	lowerVal := float64(sorted[lower].Nanoseconds())
	upperVal := float64(sorted[upper].Nanoseconds())

	return time.Duration(int64(lowerVal + (upperVal-lowerVal)*fraction))
}

func (r *Report) calculateStdDev(avgLatency time.Duration, latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	avgNs := float64(avgLatency.Nanoseconds())
	var sumSquares float64
	for _, lat := range latencies {
		diff := float64(lat.Nanoseconds()) - avgNs
		sumSquares += diff * diff
	}

	return time.Duration(int64(math.Sqrt(sumSquares / float64(len(latencies)))))
}

func (r *Report) generateLatencyHistogram(latencies []time.Duration) {
	buckets := map[string]int{
		HistogramBucket0_10ms:     0,
		HistogramBucket10_50ms:    0,
		HistogramBucket50_100ms:   0,
		HistogramBucket100_500ms:  0,
		HistogramBucket500_1000ms: 0,
		HistogramBucket1000msPlus: 0,
	}

	for _, lat := range latencies {
		ms := lat.Milliseconds()
		if ms < 0 {
			continue
		}
		switch {
		case ms < 10:
			buckets[HistogramBucket0_10ms]++
		case ms < 50:
			buckets[HistogramBucket10_50ms]++
		case ms < 100:
			buckets[HistogramBucket50_100ms]++
		case ms < 500:
			buckets[HistogramBucket100_500ms]++
		case ms < 1000:
			buckets[HistogramBucket500_1000ms]++
		default:
			buckets[HistogramBucket1000msPlus]++
		}
	}

	r.LatencyHistogram = buckets
}

func (r *Report) String() string {
	return fmt.Sprintf(
		"Report{Requests:%d Duration:%v SuccessRate:%.1f%% AvgLatency:%v P99:%v Errors:%d}",
		r.TotalRequests, r.Duration, r.SuccessRate,
		r.LatencyAvg, r.LatencyP99, r.ErrorCount,
	)
}

func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return "timeout"
	}

	if _, ok := err.(*net.DNSError); ok {
		return "dns_error"
	}

	if opErr, ok := err.(*net.OpError); ok {
		if (opErr.Op == "dial" || opErr.Op == "read") && strings.Contains(opErr.Error(), "connection refused") {
			return "connection_refused"
		}
	}

	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "i/o"):
		return "io_error"
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "operation timed out"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "no such host"):
		return "dns_error"
	default:
		return "other"
	}
}
