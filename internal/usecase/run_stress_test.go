package usecase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewRunStress(t *testing.T) {
	stress := NewRunStress("http://example.com", 100, 10, 30, false)

	if stress == nil {
		t.Fatalf("Expected RunStress instance, got nil")
	}

	if stress.url != "http://example.com" {
		t.Errorf("Expected URL http://example.com, got %s", stress.url)
	}

	if stress.requests != 100 {
		t.Errorf("Expected 100 requests, got %d", stress.requests)
	}

	if stress.concurrency != 10 {
		t.Errorf("Expected 10 concurrency, got %d", stress.concurrency)
	}

	if stress.timeout != 30 {
		t.Errorf("Expected 30 timeout, got %d", stress.timeout)
	}
}

func TestRunStress_Execute_Simple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	stress := NewRunStress(server.URL, 10, 2, 30, false)
	defer stress.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := stress.Execute(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}

	successCount := 0
	for _, result := range results {
		if result.Error == nil && result.StatusCode == 200 {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful requests, got %d", successCount)
	}
}

func TestRunStress_Execute_WithErrors(t *testing.T) {
	stress := NewRunStress("http://invalid-host-that-does-not-exist.com", 5, 1, 1, false)
	defer stress.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := stress.Execute(ctx)

	if err != nil {
		t.Fatalf("Expected no error from Execute, got %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	errorCount := 0
	for _, result := range results {
		if result.Error != nil {
			errorCount++
		}
	}

	if errorCount == 0 {
		t.Errorf("Expected some errors, got 0")
	}
}

func TestRunStress_Execute_Concurrency(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0
	maxConcurrent := 0
	currentConcurrent := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		currentConcurrent++
		requestCount++

		if currentConcurrent > maxConcurrent {
			maxConcurrent = currentConcurrent
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		currentConcurrent--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	stress := NewRunStress(server.URL, 20, 5, 30, false)
	defer stress.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := stress.Execute(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 20 {
		t.Errorf("Expected 20 results, got %d", len(results))
	}
}

func TestRunStress_Close(t *testing.T) {
	stress := NewRunStress("http://example.com", 100, 10, 30, false)

	err := stress.Close()

	if err != nil {
		t.Errorf("Expected no error from Close, got %v", err)
	}
}
