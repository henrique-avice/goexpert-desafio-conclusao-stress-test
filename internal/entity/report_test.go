package entity

import (
	"errors"
	"testing"
	"time"
)

func TestNewReport(t *testing.T) {
	report := NewReport()

	if report == nil {
		t.Fatalf("Expected report, got nil")
	}

	if report.StatusCounts == nil {
		t.Fatalf("Expected StatusCounts initialized")
	}

	if report.ErrorCounts == nil {
		t.Fatalf("Expected ErrorCounts initialized")
	}

	if report.LatencyHistogram == nil {
		t.Fatalf("Expected LatencyHistogram initialized")
	}
}

func TestCalculateFromResults_Empty(t *testing.T) {
	report := NewReport()
	results := []RequestResult{}

	report.CalculateFromResults(results, 1*time.Second)

	if report.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", report.TotalRequests)
	}
}

func TestCalculateFromResults_SuccessfulRequests(t *testing.T) {
	results := []RequestResult{
		{StatusCode: 200, Latency: 10 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 20 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 30 * time.Millisecond, Error: nil, Timestamp: time.Now()},
	}

	report := NewReport()
	report.CalculateFromResults(results, 100*time.Millisecond)

	if report.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", report.TotalRequests)
	}

	if report.SuccessCount != 3 {
		t.Errorf("Expected 3 successful requests, got %d", report.SuccessCount)
	}

	if report.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", report.ErrorCount)
	}

	if report.StatusCounts[200] != 3 {
		t.Errorf("Expected 3 status 200 responses, got %d", report.StatusCounts[200])
	}
}

func TestCalculateFromResults_WithErrors(t *testing.T) {
	results := []RequestResult{
		{StatusCode: 200, Latency: 10 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 0, Latency: 0, Error: errors.New("connection refused"), Timestamp: time.Now()},
		{StatusCode: 200, Latency: 20 * time.Millisecond, Error: nil, Timestamp: time.Now()},
	}

	report := NewReport()
	report.CalculateFromResults(results, 100*time.Millisecond)

	if report.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", report.TotalRequests)
	}

	if report.SuccessCount != 2 {
		t.Errorf("Expected 2 successful requests, got %d", report.SuccessCount)
	}

	if report.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", report.ErrorCount)
	}

	if report.ErrorCounts["connection_refused"] != 1 {
		t.Errorf("Expected 1 connection_refused error, got %d", report.ErrorCounts["connection_refused"])
	}
}

func TestCalculateFromResults_Percentiles(t *testing.T) {
	results := []RequestResult{
		{StatusCode: 200, Latency: 5 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 10 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 15 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 20 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 100 * time.Millisecond, Error: nil, Timestamp: time.Now()},
	}

	report := NewReport()
	report.CalculateFromResults(results, 200*time.Millisecond)

	if report.LatencyMin != 5*time.Millisecond {
		t.Errorf("Expected min 5ms, got %v", report.LatencyMin)
	}

	if report.LatencyMax != 100*time.Millisecond {
		t.Errorf("Expected max 100ms, got %v", report.LatencyMax)
	}

	if report.LatencyP50 == 0 {
		t.Errorf("Expected P50 to be calculated, got 0")
	}

	if report.LatencyP99 == 0 {
		t.Errorf("Expected P99 to be calculated, got 0")
	}
}

func TestCalculateFromResults_LatencyHistogram(t *testing.T) {
	results := []RequestResult{
		{StatusCode: 200, Latency: 5 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 25 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 75 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 200 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 600 * time.Millisecond, Error: nil, Timestamp: time.Now()},
		{StatusCode: 200, Latency: 1500 * time.Millisecond, Error: nil, Timestamp: time.Now()},
	}

	report := NewReport()
	report.CalculateFromResults(results, 3*time.Second)

	if report.LatencyHistogram[HistogramBucket0_10ms] != 1 {
		t.Errorf("Expected 1 request in 0-10ms bucket, got %d", report.LatencyHistogram[HistogramBucket0_10ms])
	}

	if report.LatencyHistogram[HistogramBucket10_50ms] != 1 {
		t.Errorf("Expected 1 request in 10-50ms bucket, got %d", report.LatencyHistogram[HistogramBucket10_50ms])
	}

	if report.LatencyHistogram[HistogramBucket1000msPlus] != 1 {
		t.Errorf("Expected 1 request in 1000ms+ bucket, got %d", report.LatencyHistogram[HistogramBucket1000msPlus])
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "none"},
		{"timeout error", errors.New("context deadline exceeded: operation timed out"), "timeout"},
		{"connection refused", errors.New("connection refused"), "connection_refused"},
		{"dns error", errors.New("no such host"), "dns_error"},
		{"io error", errors.New("i/o timeout"), "io_error"},
		{"other error", errors.New("random error"), "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
