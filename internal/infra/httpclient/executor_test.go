package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExecutor_ExecuteRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	executor := NewExecutor(30, 1)
	defer executor.Close()

	result := executor.ExecuteRequest(context.Background(), server.URL)

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}

	if result.Latency == 0 {
		t.Errorf("Expected non-zero latency, got 0")
	}
}

func TestExecutor_ExecuteRequest_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	executor := NewExecutor(30, 1)
	defer executor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := executor.ExecuteRequest(ctx, server.URL)

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
}

func TestExecutor_ExecuteRequest_InvalidURL(t *testing.T) {
	executor := NewExecutor(30, 1)
	defer executor.Close()

	result := executor.ExecuteRequest(context.Background(), "invalid://")

	if result.Error == nil {
		t.Errorf("Expected error for invalid URL, got nil")
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected status code 0, got %d", result.StatusCode)
	}
}

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor(30, 10)

	if executor == nil {
		t.Fatalf("Expected executor, got nil")
	}

	if executor.client == nil {
		t.Fatalf("Expected client to be initialized")
	}

	if executor.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", executor.client.Timeout)
	}
}
