package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"github.com/henrique-avice/goexpert-desafio-conclusao-stress-test/internal/entity"
	"time"
)

// Executor executa requisições HTTP.
type Executor struct {
	client *http.Client
}

// NewExecutor cria um novo executor HTTP com o timeout e concorrência fornecidos.
func NewExecutor(timeoutSeconds int, concurrency int) *Executor {
	maxPerHost := concurrency / 2
	if maxPerHost < 1 {
		maxPerHost = 1
	}
	return &Executor{
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        concurrency,
				MaxIdleConnsPerHost: maxPerHost,
				MaxConnsPerHost:     concurrency,
			},
		},
	}
}

// ExecuteRequest executa uma única requisição GET e retorna o resultado.
func (e *Executor) ExecuteRequest(ctx context.Context, url string) entity.RequestResult {
	result := entity.RequestResult{
		Timestamp: time.Now(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Error = fmt.Errorf("erro ao criar requisição: %w", err)
		return result
	}

	req.Header.Set("User-Agent", "GoStressTest/1.0")

	startTime := time.Now()
	resp, err := e.client.Do(req)

	if err != nil {
		result.Latency = time.Since(startTime)
		result.Error = err
		return result
	}

	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	result.Latency = time.Since(startTime)

	result.StatusCode = resp.StatusCode

	return result
}

// Close encerra o cliente HTTP.
func (e *Executor) Close() error {
	if e.client != nil && e.client.Transport != nil {
		if transport, ok := e.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}
