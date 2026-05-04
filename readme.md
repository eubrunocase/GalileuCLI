# Galileu — Proxy de Segurança e Governança para LLMs
### Versão macOS (Apple Silicon & Intel)
 
> **Galileu** é uma ferramenta de segurança e governança de dados voltada para o monitoramento e sanitização de informações enviadas a provedores de Inteligência Artificial (LLMs). O projeto adota uma arquitetura de **Proxy Reverso MITM (Man-in-the-Middle)**, actuando como camada inteligente entre a sua ferramenta de desenvolvimento e os servidores das LLMs.
 
---
 
## Arquitectura do Sistema
 
```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Cliente   │───▶  │  Galileu    │───▶  │   LLM       │
│  (OpenCode) │◀───  │  Proxy MITM │◀───  │  Provider   │
└─────────────┘      └─────────────┘      └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Analyzer   │
                    │ (Sanitização)│
                    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Audit     │
                    │    Log      │
                    └─────────────┘
```
 
---
 
## Instalação Rápida (Recomendado)

O Galileu agora gera e instala o certificado CA **automaticamente** na primeira execução. Não é mais necessário criar ou importar certificados manualmente.

### Opção 1 — Setup Automático

Execute o Galileu diretamente. Ele irá:
1. Verificar se o Go está instalado
2. Compilar o Galileu para sua arquitetura
3. Gerar o certificado CA automaticamente
4. Instalar o certificado no Keychain do sistema (será solicitada a senha de administrador)

### Opção 2 — Execução Direta

Basta compilar e executar. O certificado será gerado e instalado automaticamente:

```bash
go build -o galileu ./cmd/sentinel/main.go
./galileu
```

> **Nota:** Na primeira execução, será solicitada a senha de administrador para instalar o certificado CA no Keychain do sistema. Esta etapa é necessária para que o proxy MITM funcione com HTTPS.

---
 
## Pré-requisitos
 
| Requisito | Detalhe |
|---|---|
| **Sistema Operativo** | macOS — compatível com Apple Silicon (M1/M2/M3) e arquitetura Intel |
| **Go** | Versão 1.23 ou superior (necessário apenas para compilação) |
| **Privilégios** | `sudo` necessário apenas na **primeira execução** para instalar o certificado CA |
 
---
 
## Compilação
 
Abra o Terminal na raiz do projeto e execute o comando adequado à sua arquitectura:
 
**Apple Silicon (M1/M2/M3 — ARM64):**
```bash
GOOS=darwin GOARCH=arm64 go build -o galileu ./cmd/sentinel/main.go
```
 
**Intel (AMD64):**
```bash
GOOS=darwin GOARCH=amd64 go build -o galileu ./cmd/sentinel/main.go
```
 
---
 
## Estrutura de Ficheiros
 
```
Galileu/
├── galileu                  # Executável principal (macOS)
├── galileu-ca.pem           # Certificado CA gerado automaticamente
├── galileu-ca-key.pem       # Chave privada do CA (⚠️ NÃO submeter para o repositório)
├── galileu.yml              # Configuração do analyzer (não versionado)
├── galileu.yml.example      # Exemplo de configuração (versionado)
├── start-opencode.sh        # Script shell para iniciar o OpenCode com proxy
└── galileu_audit.log        # Registo de auditoria (gerado automaticamente)
```
 
---
 
## Como Utilizar
 
### Passo 1 — Executar o Galileu

Abra o Terminal na raiz do projecto e execute:
 
```bash
./galileu
```
 
Na **primeira execução**, o Galileu irá:

- Gerar automaticamente um certificado CA (`galileu-ca.pem` e `galileu-ca-key.pem`).
- Instalar o certificado no Keychain do sistema (será solicitada a senha via `sudo`).
- Iniciar o proxy na porta **9000**.
- Activar o registo (logging) de auditoria expandido.
- Inicializar o worker pool de logs (4 workers assíncronos).

Nas **execuções seguintes**, o Galileu reutilizará o certificado existente:

- Carregar os certificados locais (`galileu-ca.pem` e `galileu-ca-key.pem`).
- Verificar se o certificado está confiado no Keychain.
- Iniciar o proxy normalmente.

> Não são necessários privilégios `sudo` para a porta 9000.
> 
> Ao encerrar com `Ctrl+C`, o Galileu executa um **graceful shutdown**, garantindo que todos os logs sejam persistidos antes do encerramento.
 
### Passo 2 — Configurar o OpenCode
 
Num **novo Terminal**, dê permissão de execução ao script e execute-o:
 
```bash
chmod +x start-opencode.sh
./start-opencode.sh
```
 
Ou configure manualmente as variáveis de ambiente na sua sessão:
 
```bash
export HTTP_PROXY="http://127.0.0.1:9000"
export HTTPS_PROXY="http://127.0.0.1:9000"
export NODE_TLS_REJECT_UNAUTHORIZED=0
opencode
```
 
> **Nota:** Certifique-se de que o OpenCode está instalado e acessível no PATH.
 
### Passo 3 — Utilizar o OpenCode normalmente
 
A partir deste momento, **todas as requisições do OpenCode** para os provedores de IA passarão pelo proxy Galileu, que irá:
 
- Detectar e remover dados sensíveis automaticamente.
- Registar cada requisição com métricas detalhadas para auditoria.

---
 
## Hosts Monitorizados
 
O Galileu intercepta requisições para os seguintes provedores:
 
| Provedor | Host |
|---|---|
| OpenCode | `opencode.ai` |
| OpenAI | `api.openai.com` |
| Anthropic | `api.anthropic.com` |
| Google AI | `generativelanguage.googleapis.com` |
| Cohere | `api.cohere.ai` |
| Mistral | `api.mistral.ai` |
 
---
 
## Detecção de Dados Sensíveis
 
O **Analyzer** detecta e sanitiza automaticamente os seguintes padrões, retornando o **tipo de padrão** e a **posição de redação** em cada requisição:
 
| Tipo | Padrão | Exemplo |
|---|---|---|
| OpenAI API Key | `sk-...` | `sk-1234567890abcdef...` |
| OpenAI Project Key | `sk-proj-...` | `sk-proj-abc123...` |
| Anthropic API Key | `sk-ant-...` | `sk-ant-abc123...` |
| Google API Key | `AIzaSy...` | `AIzaSyABC123...` |
| GitHub Token | `ghp_...` | `ghp_abcdef123456...` |
| Slack / Discord | `xox[baprs]-...` | `xoxb-123456...` |
| AWS Access Key | `AKIA...` | `AKIAIOSFODNN7...` |
| AWS Secret Key | `wJalr...` | `wJalrXUtnFEM...` |
| Bearer Token | `bearer ...` | `bearer abcdef123456...` |
| Generic API Key | `api_key...` | `api_keyABC123...` |
 
Todos os dados sensíveis detectados são substituídos por `[REDACTED_BY_GALILEU]`.
 
---
 
## Registos de Auditoria Expandidos
 
O ficheiro `galileu_audit.log` contém um registo JSON detalhado de cada requisição interceptada, com métricas expandidas para análise de segurança e governança:
 
### Campos de Identidade
| Campo | Descrição |
|---|---|
| `request_id` | UUID v4 gerado por requisição — permite correlacionar entradas de uma mesma transação |
| `session_id` | Identificador gerado no boot do Galileu — agrupa logs por sessão de uso |
| `machine_id` | Hash SHA-256 (12 chars) do hostname — segmenta logs por utilizador sem dados pessoais |
 
### Detalhamento da Detecção
| Campo | Descrição |
|---|---|
| `detected_patterns` | Array com os tipos de padrão detectados (ex: `"openai_key"`, `"github_token"`) |
| `pattern_count` | Total de ocorrências redatadas naquela requisição |
| `redaction_positions` | Campos do JSON onde a detecção ocorreu (ex: `"body"`) |
 
### Volume e Payload
| Campo | Descrição |
|---|---|
| `request_body_size_bytes` | Tamanho em bytes do payload antes da sanitização |
| `response_body_size_bytes` | Tamanho em bytes da resposta do provider |
| `model` | Modelo LLM extraído da requisição (ex: `"gpt-4o"`, `"claude-3-5-sonnet"`) |
| `provider` | Provider normalizado (ex: `"openai"`, `"anthropic"`, `"google"`) |
 
### Performance do Proxy
| Campo | Descrição |
|---|---|
| `proxy_latency_ms` | Tempo total em ms desde o recebimento até o repasse ao provider |
| `analysis_duration_ms` | Tempo exclusivo em ms da etapa de análise e sanitização |
 
### Resultado
| Campo | Descrição |
|---|---|
| `response_status_code` | HTTP status code retornado pelo provider |
| `proxy_error` | Indica se houve erro interno no processamento |
| `error_message` | Mensagem de erro quando `proxy_error` é verdadeiro |
| `was_blocked` | Indica se a requisição foi bloqueada (reservado para implementação futura) |
 
### Contexto da Conversa LLM
| Campo | Descrição |
|---|---|
| `message_count` | Número de mensagens no array `"messages"` da requisição |
| `has_system_prompt` | `true` se houver campo `"system"` ou role `"system"` |
| `stream` | Valor do campo `"stream"` na requisição |
 
### Exemplo de Entrada no Log
 
```json
{
  "timestamp": "2026-04-30T10:00:00Z",
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
  "redaction_positions": ["body"],
  "has_system_prompt": true,
  "message_count": 5,
  "stream": true,
  "request_body_size_bytes": 4200,
  "response_body_size_bytes": 0,
  "response_status_code": 200,
  "proxy_latency_ms": 143,
  "analysis_duration_ms": 4,
  "proxy_error": false,
  "error_message": "",
  "was_blocked": false
}
```
 
---
 
## Comprovação de Testes
 
Os testes foram realizados com sucesso. As imagens de comprovação encontram-se na pasta `img/`:
 
| # | Teste | Descrição |
|---|---|---|
| 1 | **Ficheiro `.env` com Dados Sensíveis** | Ambiente simulado com múltiplas chaves de API |
| 2 | **Terminal do Galileu** | Registo de execução do proxy com interceptação das requisições |
| 3 | **Resposta do OpenCode** | O assistente recebe o payload com as chaves substituídas pela etiqueta de segurança |
 
### Resultado dos Testes
 
| Dados Enviados | Dados Redatados | Estado |
|---|---|---|
| `sk-...` (OpenAI) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `sk-proj-...` (OpenAI Project) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `sk-ant-...` (Anthropic) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `AIzaSy...` (Google) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `ghp_...` (GitHub) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `xoxb-...` (Slack/Discord) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `AKIA...` (AWS Key) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `wJalr...` (AWS Secret) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
| `api_key...` (Generic) | ✅ `[REDACTED_BY_GALILEU]` | Detectado |
 
---
 
## Performance
 
O Galileu foi optimizado para ambientes de desenvolvimento:
 
- **Log Worker Pool** — 4 workers assíncronos dedicados ao processamento de logs, com canal buffered (100).
- **Buffer Pooling** — Reutilização de memória com `sync.Pool` (32KB por buffer).
- **Regex Pré-compilado** — Padrões de detecção compilados na inicialização, com tipagem nomeada.
- **Graceful Shutdown** — Encerramento controlado que persiste todos os logs pendentes.
- **UUID Generation** — `request_id` único por requisição via `crypto/rand`.
- **Session/Machine ID** — Identificadores persistentes por sessão e máquina.
- **CA Auto-Generation** — Certificado RSA 4096-bit gerado automaticamente, válido por 10 anos.
 
---
 
## Resolução de Problemas
 
### "Falha ao garantir o certificado CA"

O Galileu não conseguiu gerar ou ler o certificado. Tente remover os ficheiros `galileu-ca.pem` e `galileu-ca-key.pem` e executar novamente. O certificado será regenerado automaticamente.
 
### "Não foi possível instalar o certificado automaticamente"

Verifique se inseriu correctamente a senha de administrador. Pode instalar manualmente com:

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain galileu-ca.pem
```
 
### OpenCode não conecta ao proxy
 
Confirme que as variáveis de ambiente foram exportadas correctamente na sessão actual:
 
```bash
echo $HTTP_PROXY
```
 
O resultado deve ser: `http://127.0.0.1:9000`
 
### Erros de certificado SSL/TLS no cliente
 
O certificado CA (`galileu-ca.pem`) deve constar no **Acesso às Chaves (Keychain Access)** do macOS com a confiança definida como **Confiar Sempre** (Always Trust).

Verifique no Keychain Access se o certificado **Galileu Local CA** está presente e confiado.
 
---
 
## Arquitectura do Código
 
```
cmd/sentinel/main.go      # Ponto de entrada, carregamento de config, auto-geração e instalação do CA
internal/
  ├── ca/
  │   ├── ca.go           # Geração programática do certificado CA (RSA 4096)
  │   └── install_darwin.go # Instalação automática no Keychain do macOS
  └── guardian/
      ├── guardian.go     # Proxy MITM, LogWorkerPool, extractPayloadInfo, inferProvider
      ├── analyzer.go     # Detecção tipada (AnalysisResult), CompiledPattern, sanitização
      ├── config.go       # Carregamento do galileu.yml, CompiledPattern, padrões built-in
      ├── filter.go       # Filtro de hosts e paths para análise
      └── audit.go        # AuditEntry expandido, session_id, machine_id, LogAudit
```
 
---
 
## Segurança
 
- A chave privada (`galileu-ca-key.pem`) é gerada localmente e **nunca** sai da sua máquina.
- **Nunca** efectue commit dos ficheiros `.pem` para o repositório — confirme que o `.gitignore` está actualizado.
- O certificado CA é válido por **10 anos** e utiliza chave **RSA 4096-bit**.
- O proxy actua exclusivamente sobre as ferramentas que configurarem explicitamente a porta **9000**.
- Os `detected_patterns` identificam exactamente quais tipos de segredos foram encontrados em cada requisição.
 
---
 
## Licença
 
Este projecto é para fins educacionais e de segurança interna.  
Todos os direitos são reservados ao programador **Bruno Dantas de Oliveira Cazé** — [github.com/eubrunocase/Galileu](https://github.com/eubrunocase/Galileu)
