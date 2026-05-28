package usecase

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/henrique-avice/goexpert-desafio-conclusao-stress-test/internal/entity"
	"github.com/henrique-avice/goexpert-desafio-conclusao-stress-test/internal/infra/httpclient"
)

// RunStress representa o caso de uso do teste de carga.
type RunStress struct {
	url         string
	requests    int
	concurrency int
	timeout     int
	verbose     bool
	executor    *httpclient.Executor
}

// NewRunStress cria uma nova instância de RunStress.
func NewRunStress(url string, requests, concurrency, timeout int, verbose bool) *RunStress {
	return &RunStress{
		url:         url,
		requests:    requests,
		concurrency: concurrency,
		timeout:     timeout,
		verbose:     verbose,
		executor:    httpclient.NewExecutor(timeout, concurrency),
	}
}

// Execute executa o teste de carga e retorna os resultados.
func (rs *RunStress) Execute(ctx context.Context) ([]entity.RequestResult, error) {
	if rs.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Iniciando teste: %q\n", rs.url)
		fmt.Fprintf(os.Stderr, "[DEBUG] Workers: %d, Total de reqs: %d\n", rs.concurrency, rs.requests)
	}

	resultsChan := make(chan entity.RequestResult, rs.concurrency*2)
	var resultsList []entity.RequestResult

	var workersWg sync.WaitGroup
	var collectorWg sync.WaitGroup

	reqsPerWorker := rs.requests / rs.concurrency
	remainingReqs := rs.requests % rs.concurrency

	workersWg.Add(rs.concurrency)
	for workerID := 0; workerID < rs.concurrency; workerID++ {
		reqs := reqsPerWorker
		if workerID < remainingReqs {
			reqs++
		}
		go rs.workerLoop(ctx, &workersWg, workerID, reqs, resultsChan)
	}

	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for result := range resultsChan {
			resultsList = append(resultsList, result)
		}
	}()

	workersWg.Wait()
	close(resultsChan)
	collectorWg.Wait()

	if rs.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Total de resultados coletados: %d\n", len(resultsList))
	}

	return resultsList, nil
}

func (rs *RunStress) workerLoop(ctx context.Context, wg *sync.WaitGroup, workerID int, numRequests int, resultsChan chan entity.RequestResult) {
	defer wg.Done()

	if rs.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Worker %d iniciado (%d reqs)\n", workerID, numRequests)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[ERRO] Worker %d sofreu panic: %v\n", workerID, r)
		}
	}()

	for i := 0; i < numRequests; i++ {
		select {
		case <-ctx.Done():
			if rs.verbose {
				fmt.Fprintf(os.Stderr, "[DEBUG] Worker %d cancelado\n", workerID)
			}
			return
		default:
		}

		result := rs.executor.ExecuteRequest(ctx, rs.url)

		select {
		case resultsChan <- result:
		case <-ctx.Done():
			return
		}

		if rs.verbose && i%100 == 0 && i > 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] Worker %d: %d/%d reqs feitas\n", workerID, i, numRequests)
		}
	}

	if rs.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Worker %d completou\n", workerID)
	}
}

// Close libera os recursos.
func (rs *RunStress) Close() error {
	return rs.executor.Close()
}
