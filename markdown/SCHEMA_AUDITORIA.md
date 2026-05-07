# Schema do Log de Auditoria — `AuditEntry`

Arquivo: `internal/guardian/audit.go`

---

## Campos

### Metadados de tempo

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `Timestamp` | `string` | `timestamp` | Momento exato do evento no formato RFC 3339 (ISO 8601). Ex.: `2025-01-15T14:30:00-03:00`. |

### Identificação

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `RequestID` | `string` | `request_id` | Identificador único da requisição HTTP recebida pelo proxy. Usado para correlacionar o log com os dados da request. |
| `SessionID` | `string` | `session_id` | ID da sessão de execução do proxy. Gerado uma vez por inicialização (hash do timestamp Unix em hex, 8 chars). Permite agrupar múltiplas requisições de uma mesma execução. |
| `MachineID` | `string` | `machine_id` | Identificador da máquina que executou o proxy. Gerado a partir do hostname da máquina (SHA‑256 truncado para 12 chars hex). Útil para ambientes com múltiplos agentes/containers. |

### Destino da requisição

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `Host` | `string` | `host` | Hostname (e porta, se presente) para onde a requisição foi encaminhada. Ex.: `api.openai.com`. |
| `Provider` | `string` | `provider` | Nome do provedor inferido a partir do host. Ex.: `openai`, `anthropic`. |
| `Path` | `string` | `path` | Caminho da URL da requisição. Ex.: `/v1/chat/completions`. |
| `Method` | `string` | `method` | Método HTTP utilizado. Ex.: `POST`, `GET`. |
| `Model` | `string` | `model` | Nome do modelo de IA identificado no corpo da requisição. Opcional (`omitempty`). Ex.: `gpt-4`, `claude-3-opus-20240229`. |

### Redação de dados sensíveis

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `Redacted` | `bool` | `redacted` | Indica se houve redação de dados sensíveis no payload da requisição. |
| `PatternCount` | `int` | `pattern_count` | Quantidade total de ocorrências de padrões sensíveis encontradas. |
| `DetectedPatterns` | `[]string` | `detected_patterns` | Lista dos tipos de padrões detectados. Ex.: `["api_key", "credit_card", "aws_secret_key"]`. |
| `RedactionPositions` | `[]string` | `redaction_positions` | Posições (caminhos JSON) onde as redações foram aplicadas. Ex.: `["$.messages[0].content"]`. |

### Conteúdo da mensagem

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `MessageCount` | `int` | `message_count` | Número de mensagens no array `messages` da requisição. |
| `HasSystemPrompt` | `bool` | `has_system_prompt` | Indica se existe ao menos uma mensagem com papel `system` no array `messages`. |
| `Stream` | `bool` | `stream` | Indica se a requisição utilizou o parâmetro `stream: true` para resposta em streaming. |

### Tamanho e status HTTP

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `RequestBodySizeBytes` | `int` | `request_body_size_bytes` | Tamanho do corpo da requisição em bytes. |
| `ResponseBodySizeBytes` | `int` | `response_body_size_bytes` | Tamanho do corpo da resposta em bytes. |
| `ResponseStatusCode` | `int` | `response_status_code` | Código de status HTTP retornado pelo provedor. Ex.: `200`, `400`, `429`. |

### Latência

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `ProxyLatencyMs` | `int` | `proxy_latency_ms` | Tempo total do proxy entre receber a requisição e receber a resposta completa do provedor, em milissegundos. |
| `AnalysisDurationMs` | `int` | `analysis_duration_ms` | Tempo gasto exclusivamente na análise/inspeção do payload (detecção de padrões, redação), em milissegundos. |

### Erro e bloqueio

| Campo | Tipo | JSON | Descrição |
|---|---|---|---|
| `ProxyError` | `bool` | `proxy_error` | Indica se ocorreu um erro durante o proxy da requisição (ex.: falha de conexão, timeout). |
| `ErrorMessage` | `string` | `error_message` | Mensagem de erro detalhada. Opcional (`omitempty`), presente apenas quando `proxy_error = true`. |
| `WasBlocked` | `bool` | `was_blocked` | Indica se a requisição foi bloqueada pelo Guardian (ex.: padrão proibido detectado, regra de configuração). |

---

## Exemplo de saída JSON

```json
{
  "timestamp": "2025-06-01T10:00:00-03:00",
  "request_id": "abc123",
  "session_id": "1f4a3b2c",
  "machine_id": "a1b2c3d4e5f6",
  "host": "api.openai.com",
  "provider": "openai",
  "path": "/v1/chat/completions",
  "method": "POST",
  "model": "gpt-4",
  "redacted": true,
  "pattern_count": 2,
  "detected_patterns": ["api_key", "email"],
  "redaction_positions": ["$.messages[0].content"],
  "message_count": 3,
  "has_system_prompt": true,
  "stream": false,
  "request_body_size_bytes": 2048,
  "response_body_size_bytes": 4096,
  "response_status_code": 200,
  "proxy_latency_ms": 1234,
  "analysis_duration_ms": 42,
  "proxy_error": false,
  "error_message": "",
  "was_blocked": false
}
```
