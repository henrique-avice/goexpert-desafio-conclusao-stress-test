# goexpert-desafio-conclusao-stress-test

> Ferramenta CLI de teste de carga que executa requisições HTTP concorrentes e gera relatório detalhado de desempenho.

## Índice

- [Visão Geral](#visão-geral)
- [Funcionalidades](#funcionalidades)
- [Requisitos](#requisitos)
- [Execução](#execução)
- [Arquitetura](#arquitetura)
- [Testes](#testes)
- [Como Utilizar](#como-utilizar)

## Visão Geral

Binário de linha de comando que dispara requisições HTTP em paralelo contra uma URL alvo usando um pool de workers e exibe um relatório com distribuição de status, estatísticas de latência e throughput. Implementado sem dependências externas, usando apenas a biblioteca padrão do Go.

## Funcionalidades

### Requisitos do Desafio

- [x] Flag `--url` (URL alvo, obrigatório)
- [x] Flag `--requests` (total de requisições, padrão: 100)
- [x] Flag `--concurrency` (workers simultâneos, padrão: 10)
- [x] Relatório com tempo total de execução
- [x] Relatório com total de requisições realizadas
- [x] Relatório com contagem de respostas HTTP 200
- [x] Relatório com distribuição de outros códigos de status HTTP
- [x] Execução via Docker

### Extras Implementados

- Flag `--timeout` para timeout por requisição (padrão: 30s)
- Flag `--verbose` para exibir log de cada requisição individualmente
- Estatísticas de latência: P50, P75, P90, P95, P99, P99.9 e desvio padrão
- Histograma de latências com faixas: 0–10ms, 10–50ms, 50–100ms, 100–500ms, 500ms–1s, 1s+
- Throughput em requisições por segundo
- Classificação de erros por tipo

## Requisitos

- Go 1.26.2+
- Docker e Docker Compose

## Execução

### Docker Compose (Recomendado)

```bash
docker-compose up --build
```

Sobe um servidor `mccutchen/go-httpbin` como alvo e executa o stress test contra ele.

### Docker

```bash
docker build -t stress-test .
docker run stress-test --url=https://httpbin.org/get --requests=100 --concurrency=10
```

### Local

```bash
go build -o stress ./cmd/stress
./stress --url=https://httpbin.org/get --requests=100 --concurrency=10
```

## Arquitetura

```
[CLI flags] ──► [Worker Pool (--concurrency workers)]
                         │
                  [HTTP Target URL]
                         │
                 [Results Channel] ──► [Reporter]
```

| Componente | Responsabilidade |
|---|---|
| `cmd/stress` | Parsing de flags e ponto de entrada |
| `internal/usecase` | Coordenação do pool de workers e coleta de resultados |
| `internal/entity` | Estrutura de relatório, cálculo de percentis e histograma |

## Testes

```bash
go test -v -race ./...
```

---

## Como Utilizar

### 1. Iniciando o Sistema

```bash
docker-compose up --build
```

Sobe um servidor `go-httpbin` como alvo em `:8081` e executa o stress test imediatamente com os parâmetros padrão: `100 requisições`, `10 workers`, `timeout 30s`. O resultado aparece no terminal e o contêiner encerra sozinho.

### 2. Testando com Parâmetros Personalizados

Via Docker (após o build inicial):

```bash

docker build -t goexpert-desafio-conclusao-stress-test .

docker run --rm goexpert-desafio-conclusao-stress-test \
  --url=https://httpbin.org/get \
  --requests=100 \
  --concurrency=10 \
  --timeout=30
```

Localmente:

```bash
go build -o stress ./cmd/stress

./stress --url=https://httpbin.org/get --requests=100 --concurrency=10
./stress --url=https://httpbin.org/get --requests=500 --concurrency=50 --verbose
```

### 3. Resultado Esperado

```
Teste de Carga Completo
=======================
Total de Requests: 100
Duração Total: 2.341s
Taxa de Requisições/s: 42.72
Workers Ativos: 10

Distribuição de Status HTTP:
- 200 OK: 100 requisições (100.00%)

Estatísticas de Latência:
- Mínimo: 45.123ms
- Máximo: 312.456ms
- Média: 120.789ms
- Mediana (P50): 110.234ms
- P75: 145.678ms
- P90: 201.345ms
- P95: 245.890ms
- P99: 298.123ms
- P999: 311.234ms
- Desvio Padrão: 48.567ms

Erros:
- Timeouts: 0
- Connection Errors: 0
- DNS Errors: 0
- IO Errors: 0
- Outros Erros: 0
- Total de Erros: 0

Distribuição de Latência:
- 0–10ms: 0 (0.00%)
- 10–50ms: 3 (3.00%)
- 50–100ms: 32 (32.00%)
- 100–500ms: 65 (65.00%)
- 500ms–1s: 0 (0.00%)
- 1s+: 0 (0.00%)
```

Os valores numéricos variam conforme a latência real do alvo. O relatório exibe a distribuição de todos os status HTTP encontrados, permitindo identificar falhas por código.
