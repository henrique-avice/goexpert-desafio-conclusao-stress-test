package entity

import (
	"errors"
	"testing"
	"time"
)

func TestRequestResult_Success(t *testing.T) {
	result := RequestResult{
		StatusCode: 200,
		Latency:    100 * time.Millisecond,
		Error:      nil,
		Timestamp:  time.Now(),
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}

	if result.Latency != 100*time.Millisecond {
		t.Errorf("Expected latency 100ms, got %v", result.Latency)
	}
}

func TestRequestResult_WithError(t *testing.T) {
	testErr := errors.New("connection refused")
	result := RequestResult{
		StatusCode: 0,
		Latency:    50 * time.Millisecond,
		Error:      testErr,
		Timestamp:  time.Now(),
	}

	if result.Error == nil {
		t.Errorf("Expected error, got nil")
	}

	if result.Error.Error() != "connection refused" {
		t.Errorf("Expected 'connection refused', got %v", result.Error)
	}
}
