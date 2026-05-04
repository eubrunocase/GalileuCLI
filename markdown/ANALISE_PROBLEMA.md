# Análise do Problema: Conexão Perdida OpenCode + Galileu Proxy

## Sumário do Problema

Quando o OpenCode (VS Code) tenta se conectar através do proxy Galileu MITM na porta 9000 para acessar `opencode.ai`, o proxy:
- Recebe a conexão corretamente
- Começa a processar a requisição
- Logs mostram: `Cannot read request from mitm'd client opencode.ai:443` com `wsarecv: Foi forçado o cancelamento de uma conexão existente pelo host remoto`
- Retorna erro 500 ao cliente
- Cliente entra em loop de retentativas infinito

---

## Causas Raiz Identificadas

### 1. ❌ **CRÍTICO: Modificação de Body Sem Validação**

**Arquivo**: [internal/guardian/guardian.go](internal/guardian/guardian.go#L150-L185)

**Problema**:
```go
bodyBytes, err := io.ReadAll(r.Body)  // Consome o Body
if err != nil {
    return r, nil
}

// ... análise ...

if result.Modified {
    r.Body = io.NopCloser(bytes.NewReader(result.Result))
    r.ContentLength = int64(len(result.Result))
    r.Header.Set("Content-Length", fmt.Sprintf("%d", len(result.Result)))
    r.Header.Del("Transfer-Encoding")  // ⚠️ Problema!
} else {
    r.Body = io.NopCloser(bytes.NewReader(bodyBytes))  // ⚠️ Duplica reading!
}
```

**Por que falha**:
- O `Content-Length` é atualizado, mas o protocolo HTTP/TLS pode já ter negociado o tamanho original
- Deletar `Transfer-Encoding` sem verificar se existia pode causar problemas de parsing
- Se a requisição for streaming (chunked), forçar um tamanho fixo quebra o protocolo
- O servidor remoto recebe dados de tamanho inconsistente → fecha a conexão

### 2. ❌ **CRÍTICO: Sem Handler de Erro no Proxy**

**Arquivo**: [internal/guardian/guardian.go](internal/guardian/guardian.go#L210-240)

**Problema**:
```go
proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
    if resp != nil && resp.StatusCode >= 500 {
        fmt.Printf("[GALILEU] Erro %d recebido...\n", resp.StatusCode)
    }
    return resp
})

// ⚠️ Não há proxy.OnError() - erros de conexão desaparecem!
```

**Impacto**:
- Erros de conexão não são capturados nem logados
- Impossível diagnosticar o problema
- Cliente vê erro 500 genérico sem contexto

### 3. ⚠️ **Sem Timeouts no Servidor HTTP**

**Arquivo**: [internal/guardian/guardian.go](internal/guardian/guardian.go#L220)

```go
srv := &http.Server{Addr: ":9000", Handler: proxy}
// ⚠️ Sem ReadTimeout, WriteTimeout, IdleTimeout
```

**Impacto**:
- Conexões podem ficar pendentes indefinidamente
- Em caso de erro, cliente não é notificado rápido
- Contribui ao loop de retentativas

### 4. ⚠️ **Tratamento de Streaming Inadequado**

**Arquivo**: [internal/guardian/filter.go](internal/guardian/filter.go#L8-50)

```go
func ShouldAnalyze(host, method, path string) bool {
    // ... verifica se é POST ...
    // ⚠️ Não verifica se stream=true na requisição!
}
```

**Impacto**:
- Requisições com `"stream": true` são modificadas
- Quebra o protocolo de streaming do cliente
- Cliente espera respostas em stream mas recebe erro

### 5. ⚠️ **Body Reading Duplicado**

Quando `result.Modified == false`, o código ainda reconstrói o body:
```go
} else {
    r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
}
```

Isso força uma re-leitura, consumindo banda e potencialmente causando timeout.

### 6. ⚠️ **Sem Validação de Integridade**

Após modificar a requisição, não há verificação se:
- O JSON permaneceu válido
- O Content-Length está correto
- Os headers estão consistentes

---

## Cenário de Falha Específico

```
1. OpenCode envia requisição HTTPS para opencode.ai via proxy
2. Proxy MITM intercepta (CONNECT)
3. Proxy analisa e modifica o Body
4. Proxy tenta resend com novo Body + Content-Length
5. Servidor recebe Content-Length ≠ tamanho real → fecha conexão
6. Proxy recebe "connection reset by peer"
7. Nenhum erro é logado (sem OnError handler)
8. Cliente recebe 500 genérico
9. Cliente retry forever (loop infinito)
```

---

## Solução Proposta

### ✅ Implementar Handler de Erro
```go
proxy.OnError(func(req *http.Request, err error) http.Handler {
    fmt.Printf("[GALILEU] ERRO de proxy: %v para %s%s\n", err, req.Host, req.URL.Path)
    // Log e track do erro
})
```

### ✅ Validar Body Antes de Resend
```go
// Verificar se resultado JSON é válido
if result.Modified {
    var payload map[string]interface{}
    if err := json.Unmarshal(result.Result, &payload); err != nil {
        // JSON inválido após redação! Não enviar.
        logWorkerPool.Submit(LogRequest{...ProxyError: true...})
        return r, nil  // Skip modification
    }
}
```

### ✅ Adicionar Timeouts
```go
srv := &http.Server{
    Addr:         ":9000",
    Handler:      proxy,
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  60 * time.Second,
}
```

### ✅ Respeitar Streaming
```go
// Skip analysis para streaming requests
if payloadInfo.stream {
    return r, nil  // Não modificar streams
}
```

### ✅ Melhorar Body Handling
```go
// Não reconstrói body desnecessariamente
if !result.Modified {
    r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
}
// Apenas para modificado
```

---

## Arquivos a Modificar

1. **[internal/guardian/guardian.go](internal/guardian/guardian.go)**
   - Adicionar `proxy.OnError()` handler
   - Adicionar validação JSON após modificação
   - Implementar respeito a streaming
   - Adicionar timeouts ao servidor
   - Melhorar logging de erros

2. **[internal/guardian/filter.go](internal/guardian/filter.go)** (opcional)
   - Adicionar `stream` check em `ShouldAnalyze()`

---

## Impacto Esperado

Após as correções:
- ✅ Erros de conexão são capturados e logados
- ✅ Requisições inválidas após redação não são enviadas
- ✅ Streaming respeitado (sem modificação)
- ✅ Timeouts previnem conexões suspensas
- ✅ Cliente recebe respostas coerentes
- ✅ Sem loop infinito de retentativas

---

## Testes Recomendados

1. **Teste de Streaming**: Enviar requisição com `"stream": true`, verificar se passa intocada
2. **Teste de Redação**: Enviar requisição com padrão sensível, verificar se redação é válida
3. **Teste de Erro**: Desconectar cliente no meio, verificar logs de erro
4. **Teste de Timeout**: Enviar requisição sem responder, verificar se fecha após 30s
5. **Teste de Integridade**: Verificar se JSON redatado permanece válido

