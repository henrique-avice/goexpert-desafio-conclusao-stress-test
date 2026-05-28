package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/henrique-avice/goexpert-desafio-conclusao-stress-test/internal/entity"
	"github.com/henrique-avice/goexpert-desafio-conclusao-stress-test/internal/usecase"
)

type Config struct {
	URL         string
	Requests    int
	Concurrency int
	Timeout     int
	Verbose     bool
}

func main() {
	config, err := parseAndValidateFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		fmt.Fprintf(os.Stderr, "Uso: %s --url=<url> [--requests=100] [--concurrency=10] [--timeout=30]\n", os.Args[0])
		os.Exit(1)
	}

	stressTest := usecase.NewRunStress(config.URL, config.Requests, config.Concurrency, config.Timeout, config.Verbose)
	defer stressTest.Close()

	ctx, cancel := context.WithTimeout(context.Background(), calculateTestTimeout(config.Timeout))
	defer cancel()

	startTime := time.Now()
	results, err := stressTest.Execute(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar teste: %v\n", err)
		os.Exit(1)
	}
	duration := time.Since(startTime)

	report := generateReport(config.URL, config.Concurrency, results, duration)
	printReport(os.Stdout, report)
}

func parseAndValidateFlags() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.URL, "url", "", "URL alvo do teste (obrigatório)")
	flag.IntVar(&cfg.Requests, "requests", 100, "Total de requisições (padrão: 100)")
	flag.IntVar(&cfg.Concurrency, "concurrency", 10, "Workers paralelos (padrão: 10)")
	flag.IntVar(&cfg.Timeout, "timeout", 30, "Timeout em segundos (padrão: 30)")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Saída detalhada")

	flag.Parse()

	if cfg.URL == "" {
		return nil, fmt.Errorf("erro: flag --url é obrigatória")
	}

	if err := validateURL(cfg.URL); err != nil {
		return nil, fmt.Errorf("erro: %w", err)
	}

	if cfg.Requests < 1 {
		return nil, fmt.Errorf("erro: --requests deve ser >= 1 (recebido: %d)", cfg.Requests)
	}

	if cfg.Concurrency < 1 {
		return nil, fmt.Errorf("erro: --concurrency deve ser >= 1 (recebido: %d)", cfg.Concurrency)
	}

	if cfg.Concurrency > cfg.Requests {
		cfg.Concurrency = cfg.Requests
	}

	if cfg.Timeout < 1 {
		return nil, fmt.Errorf("erro: --timeout deve ser >= 1 (recebido: %d)", cfg.Timeout)
	}

	return cfg, nil
}

func validateURL(urlStr string) error {
	if len(urlStr) > 2048 {
		return fmt.Errorf("URL excede tamanho máximo de 2048 caracteres")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL inválida: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL deve usar http:// ou https://, recebido: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL sem host: %s", urlStr)
	}

	return nil
}

func calculateTestTimeout(timeoutSecs int) time.Duration {
	const maxTimeout = 5 * time.Minute
	d := time.Duration(timeoutSecs) * time.Second
	if d > maxTimeout {
		return maxTimeout
	}
	return d
}

func generateReport(url string, concurrency int, results []entity.RequestResult, duration time.Duration) *entity.Report {
	report := entity.NewReport()
	report.URL = url
	report.ConcurrentWorkers = concurrency
	report.CalculateFromResults(results, duration)
	return report
}

func printReport(w io.Writer, report *entity.Report) {
	fmt.Fprintf(w, "Teste de Carga Completo\n")
	fmt.Fprintf(w, "=======================\n")
	fmt.Fprintf(w, "Total de Requests: %d\n", report.TotalRequests)
	fmt.Fprintf(w, "Duração Total: %.3fs\n", report.Duration.Seconds())
	fmt.Fprintf(w, "Taxa de Requisições/s: %.2f\n", report.RequestsPerSec)
	fmt.Fprintf(w, "Workers Ativos: %d\n", report.ConcurrentWorkers)
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "Distribuição de Status HTTP:\n")

	var statuses []int
	for status := range report.StatusCounts {
		statuses = append(statuses, status)
	}
	sort.Ints(statuses)

	for _, status := range statuses {
		count := report.StatusCounts[status]
		percentage := float64(count) / float64(report.TotalRequests) * 100
		fmt.Fprintf(w, "- %d %s: %d requisições (%.2f%%)\n",
			status, getStatusText(status), count, percentage)
	}

	if report.ErrorCount > 0 {
		percentage := float64(report.ErrorCount) / float64(report.TotalRequests) * 100
		fmt.Fprintf(w, "- Erros de Rede: %d requisições (%.2f%%)\n",
			report.ErrorCount, percentage)
	}

	fmt.Fprintf(w, "\n")

	if report.LatencyAvg > 0 {
		fmt.Fprintf(w, "Estatísticas de Latência:\n")
		fmt.Fprintf(w, "- Mínimo: %.3fms\n", report.LatencyMin.Seconds()*1000)
		fmt.Fprintf(w, "- Máximo: %.3fms\n", report.LatencyMax.Seconds()*1000)
		fmt.Fprintf(w, "- Média: %.3fms\n", report.LatencyAvg.Seconds()*1000)
		fmt.Fprintf(w, "- Mediana (P50): %.3fms\n", report.LatencyP50.Seconds()*1000)
		fmt.Fprintf(w, "- P75: %.3fms\n", report.LatencyP75.Seconds()*1000)
		fmt.Fprintf(w, "- P90: %.3fms\n", report.LatencyP90.Seconds()*1000)
		fmt.Fprintf(w, "- P95: %.3fms\n", report.LatencyP95.Seconds()*1000)
		fmt.Fprintf(w, "- P99: %.3fms\n", report.LatencyP99.Seconds()*1000)
		fmt.Fprintf(w, "- P999: %.3fms\n", report.LatencyP999.Seconds()*1000)
		fmt.Fprintf(w, "- Desvio Padrão: %.3fms\n", report.LatencyStdDev.Seconds()*1000)
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "Erros:\n")
	fmt.Fprintf(w, "- Timeouts: %d\n", report.TimeoutCount)
	fmt.Fprintf(w, "- Connection Errors: %d\n", report.ErrorCounts["connection_refused"])
	fmt.Fprintf(w, "- DNS Errors: %d\n", report.ErrorCounts["dns_error"])
	fmt.Fprintf(w, "- IO Errors: %d\n", report.ErrorCounts["io_error"])
	fmt.Fprintf(w, "- Outros Erros: %d\n", report.ErrorCounts["other"])
	fmt.Fprintf(w, "- Total de Erros: %d\n", report.ErrorCount)
	fmt.Fprintf(w, "\n")

	if len(report.LatencyHistogram) > 0 {
		fmt.Fprintf(w, "Distribuição de Latência:\n")
		bucketOrder := []string{
			entity.HistogramBucket0_10ms,
			entity.HistogramBucket10_50ms,
			entity.HistogramBucket50_100ms,
			entity.HistogramBucket100_500ms,
			entity.HistogramBucket500_1000ms,
			entity.HistogramBucket1000msPlus,
		}
		for _, bucket := range bucketOrder {
			count := report.LatencyHistogram[bucket]
			percentage := 0.0
			if report.TotalRequests > 0 {
				percentage = float64(count) / float64(report.TotalRequests) * 100
			}
			fmt.Fprintf(w, "- %s: %d (%.2f%%)\n", bucket, count, percentage)
		}
	}
}

func getStatusText(status int) string {
	texts := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		301: "Moved Permanently",
		302: "Found",
		304: "Not Modified",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		409: "Conflict",
		429: "Too Many Requests",
		500: "Internal Server Error",
		502: "Bad Gateway",
		503: "Service Unavailable",
	}

	if text, ok := texts[status]; ok {
		return text
	}
	return "Unknown"
}
