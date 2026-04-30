# Contexto do Projeto — Galileu

Galileu é uma ferramenta de segurança e governança de dados desenvolvida em Go que atua como um proxy reverso MITM (Man-in-the-Middle) entre aplicações cliente de LLM e os provedores de IA (OpenAI, Anthropic, Google, Cohere, Mistral, etc.). Atualmente compatível apenas com Windows e integrado ao OpenCode. O proxy intercepta requisições HTTPS, analisa os payloads em busca de dados sensíveis (API keys, tokens) usando regex pré-compilados, sanitiza o conteúdo substituindo ocorrências por [REDACTED_BY_GALILEU] e repassa a requisição ao provider. Cada evento é registrado em um arquivo galileu_audit.log em formato JSON (uma entrada por linha).

---

# Tarefa — Expandir o sistema de logging de auditoria

A estrutura atual do log registra apenas os seguintes campos:
{ "timestamp", "host", "path", "method", "redacted", "pattern_type" }

Essa estrutura precisa ser expandida para suportar análise de métricas ao final de um período piloto de 1-2 semanas com a equipe. Abaixo estão todos os novos campos que devem ser adicionados ao struct de log e persistidos no galileu_audit.log.

---

# Campos a adicionar na entrada de log (formato JSON)

## Identidade da requisição
- request_id (string): UUID v4 gerado por requisição. Permite correlacionar entradas de uma mesma transação.
- session_id (string): identificador gerado no boot do Galileu, fixo durante toda a execução do processo. Permite agrupar logs por sessão de uso.
- machine_id (string): hash SHA-256 (primeiros 12 chars) do hostname da máquina. Segmenta logs por usuário sem armazenar dados pessoais.

## Detalhamento da detecção
- detected_patterns ([]string): array com os tipos de padrão detectados na requisição. Valores possíveis: "openai_key", "openai_project_key", "anthropic_key", "google_key", "github_token", "slack_token", "aws_access_key", "aws_secret_key", "generic_api_key". Substituir o campo pattern_type atual.
- pattern_count (int): total de ocorrências redactadas naquela requisição (pode haver múltiplas chaves no mesmo payload).
- redaction_positions ([]string): campos do JSON onde a detecção ocorreu. Ex: "messages[2].content", "system". Permite identificar de onde no fluxo o dado está escapando.

## Volume e payload
- request_body_size_bytes (int): tamanho em bytes do payload da requisição antes da sanitização.
- response_body_size_bytes (int): tamanho em bytes da resposta recebida do provider.
- model (string): valor do campo "model" extraído do corpo da requisição. Ex: "gpt-4o", "claude-3-5-sonnet".
- provider (string): nome normalizado do provider inferido pelo host. Ex: "openai", "anthropic", "google", "mistral", "cohere".

## Performance do proxy
- proxy_latency_ms (int): tempo total em ms desde o recebimento da requisição até o repasse ao provider (inclui análise + overhead de rede).
- analysis_duration_ms (int): tempo exclusivo em ms da etapa de análise e sanitização do payload, isolado do tempo de rede.

## Resultado
- response_status_code (int): HTTP status code retornado pelo provider. Essencial para detectar se a sanitização quebrou o payload (ex: 400 Bad Request).
- proxy_error (bool): indica se o Galileu encontrou um erro interno durante o processamento (TLS, timeout, parsing, etc.).
- error_message (string): mensagem de erro quando proxy_error for true. Vazio caso contrário.
- was_blocked (bool): indica se a requisição foi bloqueada em vez de sanitizada. Preparar o campo mesmo que o bloqueio ainda não esteja implementado.

## Contexto da conversa LLM
- message_count (int): número de mensagens no array "messages" da requisição. Indica profundidade do contexto enviado.
- has_system_prompt (bool): true se a requisição contiver um campo "system" ou uma mensagem com role "system".
- stream (bool): valor do campo "stream" na requisição. Requisições com streaming têm comportamento diferente no proxy (chunked response).

---

# Estrutura JSON de saída esperada por entrada de log

{
  "timestamp": "2026-04-29T10:00:00Z",
  "request_id": "f3a1b2c4-7e6d-4a1b-9c3f-1234567890ab",
  "session_id": "a1b2c3d4",
  "machine_id": "3f9a12bc4d1e",

  "host": "api.openai.com",
  "provider": "openai",
  "path": "/v1/chat/completions",
  "method": "POST",
  "model": "gpt-4o",

  "redacted": true,
  "pattern_count": 2,
  "detected_patterns": ["openai_key", "github_token"],
  "redaction_positions": ["messages[1].content", "system"],
  "has_system_prompt": true,
  "message_count": 5,
  "stream": true,

  "request_body_size_bytes": 4200,
  "response_body_size_bytes": 1800,
  "response_status_code": 200,

  "proxy_latency_ms": 143,
  "analysis_duration_ms": 4,

  "proxy_error": false,
  "error_message": "",
  "was_blocked": false
}

---

# Arquitetura relevante
- O logging está em internal/guardian/audit.go
- A detecção e sanitização estão em internal/guardian/analyzer.go
- O ponto de entrada do proxy está em internal/guardian/guardian.go
- O entrypoint da aplicação está em cmd/sentinel/main.go
- O logging é assíncrono (não bloqueante) — manter essa característica ao expandir os campos.
- Os regex de detecção já são pré-compilados na inicialização — ao adicionar o campo detected_patterns, retornar o tipo do padrão junto com a ocorrência no analyzer, em vez de apenas um booleano.