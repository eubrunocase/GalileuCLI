# Testes do Guardian (Analyzer)

Todos os testes estão em `internal/guardian/`. São testes unitários puros (sem dependências externas) — apenas Go padrão.

---

## Pré-requisitos

- Go 1.21+
- Navegue até a raiz do projeto (`application/`)

---

## Executar todos os testes

```bash
go test -v ./internal/guardian/...
```

---

## Testes de detecção (`analyzer_detection_test.go`)

### `TestAnalyzerTruePositives`
- **O que testa:** Verifica que cada padrão built-in detecta corretamente uma chave/ token real embutido num payload JSON.
- **Padrões testados:** `openai_key`, `openai_project_key`, `anthropic_key`, `google_key`, `github_token`, `slack_token`, `aws_access_key`.
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerTruePositives" ./internal/guardian/...
  ```

### `TestAnalyzerNoFalsePositives`
- **O que testa:** Verifica que padrões built-in **não** disparam em dados benignos: UUIDs, hashes (MD5, SHA1, SHA256), Base64, IDs numéricos, payloads normais de API, nomes de variáveis, URLs, JSON estruturado, tokens de outros serviços, valores nulos/booleanos.
- **Métrica:** Falha se houver **qualquer** falso positivo (tolerância zero).
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerNoFalsePositives" ./internal/guardian/...
  ```

### `TestAnalyzerCustomPatternsRegex`
- **O que testa:** Valida padrões customizados do tipo **regex**. Testa 4 categorias:
  - `DB_PASSWORD` — strings como `DB_PASSWORD=secret123`
  - `Connection String` — URIs de banco (`postgres://...`, `mysql://...`, etc.)
  - `JWT` — tokens JWT no formato `eyJ...eyJ...`
  - `Private Key` — bloco `-----BEGIN ... PRIVATE KEY-----`
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerCustomPatternsRegex" ./internal/guardian/...
  ```

### `TestAnalyzerCustomPatternsLiteral`
- **O que testa:** Valida padrões customizados do tipo **literal** (correspondência exata). Testa:
  - `Tabelas Internas` — palavras exatas como `clientes_vip`, `transacoes_internas`
  - `Projectos Confidenciais` — frases exatas como `Operação Phoenix`
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerCustomPatternsLiteral" ./internal/guardian/...
  ```

### `TestAnalyzerCustomPatternsNoFalsePositives`
- **O que testa:** Verifica que padrões customizados (regex e literal) **não** disparam em entradas benignas similares (ex.: `password=123456` não dispara o padrão `DB_PASSWORD`).
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerCustomPatternsNoFalsePositives" ./internal/guardian/...
  ```

---

## Testes de performance (`analyzer_perf_test.go`)

### `TestAnalyzerLatency`
- **O que testa:** Mede a latência por operação do `Analyzer.Analyze()` em nanossegundos. Executa 1000 iterações sobre 7 payloads diferentes e calcula média, mínimo, máximo, P50, P95 e P99.
- **Critério de falha:** Latência média > 3ms por operação.
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerLatency" ./internal/guardian/...
  ```

### `TestAnalyzerThroughput`
- **O que testa:** Mede o throughput do analisador: total de requisições processadas, tempo total, média por requisição, requisições por segundo e MB/s.
- **Comando:**
  ```bash
  go test -v -run "TestAnalyzerThroughput" ./internal/guardian/...
  ```

---

## Benchmarks (`analyzer_bench_test.go`)

### `BenchmarkAnalyze`
- **O que testa:** Benchmark padrão Go (`go test -bench`) que lê o arquivo `galileu_audit.log` e executa `Analyze()` repetidamente. Útil para comparar performance entre alterações no código.
- **Comando:**
  ```bash
  go test -bench=. -benchmem ./internal/guardian/...
  ```
  - `-benchmem` inclui alocações por operação.

---

## Executar testes de detecção (agrupados)

```bash
# Todos os testes de detecção (true positives + false positives + custom)
go test -v -run "TestAnalyzer" ./internal/guardian/...
```

---

## Resumo visual

| Teste | Tipo | O que valida |
|---|---|---|
| `TestAnalyzerTruePositives` | Detecção | Chaves/tokens reais são detectados |
| `TestAnalyzerNoFalsePositives` | Detecção | Dados benignos não disparam alarme |
| `TestAnalyzerCustomPatternsRegex` | Detecção | Padrões regex customizados funcionam |
| `TestAnalyzerCustomPatternsLiteral` | Detecção | Padrões literais customizados funcionam |
| `TestAnalyzerCustomPatternsNoFalsePositives` | Detecção | Custom patterns não dão falso positivo |
| `TestAnalyzerLatency` | Performance | Latência por operação < 3ms |
| `TestAnalyzerThroughput` | Performance | Throughput em req/s e MB/s |
| `BenchmarkAnalyze` | Benchmark | Benchmark comparativo (`-bench`) |
