# Guia de Troubleshooting - Galileu Proxy OpenCode

## Problema Original

Ao usar OpenCode via proxy Galileu:
- Conexão cai com: `wsarecv: Foi forçado o cancelamento de uma conexão existente pelo host remoto`
- OpenCode recebe: `Internal server error [retrying in 15s attempt #4`
- Resulta em: Loop infinito de retentativas

---

## Causas Diagnosticadas e Corrigidas

| Causa | Severidade | Status | Descrição |
|-------|-----------|--------|-----------|
| Sem handler de erro | 🔴 CRÍTICA | ✅ CORRIGIDA | Erros de conexão não eram capturados |
| JSON inválido após redação | 🔴 CRÍTICA | ✅ CORRIGIDA | Redação quebrava estrutura JSON |
| Streaming modificado | 🔴 CRÍTICA | ✅ CORRIGIDA | Requisições stream=true eram modificadas |
| Sem timeouts | 🟠 IMPORTANTE | ✅ CORRIGIDA | Conexões pendentes indefinidamente |
| Headers inconsistentes | 🟠 IMPORTANTE | ✅ CORRIGIDA | Transfer-Encoding não respeitado |

---

## Como Verificar se as Correções Funcionaram

### 1. Check de Compilação
```bash
cd c:\Desenvolvimento\galileulinux\src\galileu-unified
go build -o galileu.exe ./cmd/sentinel
# Esperado: Sem erros
```

### 2. Iniciar com Debug
```bash
# Execute com privilégios de admin
galileu.exe

# Procure por estas linhas:
# [GALILEU] Certificado CA encontrado e carregado.
# [GALILEU] Certificado Root CA instalado com sucesso.
# [Galileu] XX padrão(ões) de detecção carregado(s) a partir de 'galileu.yml'.
# [GALILEU] Proxy ativo na porta 9000.
# [GALILEU] Logging de auditoria ativo: galileu_audit.log
# [GALILEU] Proxy MITM Ativo na porta 9000...
```

### 3. Teste com OpenCode
1. Configure OpenCode para usar proxy: `localhost:9000`
2. Abra terminal integrado
3. Execute comando que aciona LLM (ex: `/explain`)
4. Observe logs

**Esperado**:
```
[GALILEU] Interceptado em opencode.ai: Dados sensiveis removidos.
# ou
[GALILEU] Streaming não será modificado
# Sem "ERRO DE PROXY" messages
```

**Problema** (se ainda ocorrer):
```
[GALILEU] ❌ ERRO DE PROXY: connection refused
```

---

## Checklist de Verificação

- [ ] Galileu compilado sem erros
- [ ] Proxy iniciando corretamente (porta 9000)
- [ ] OpenCode consegue estabelecer conexão
- [ ] Primeiro request vai através (verificar logs)
- [ ] Não há mensagens "❌ ERRO DE PROXY"
- [ ] Logs mostram interceptações normais
- [ ] OpenCode responde com sucesso
- [ ] Não há loop de retentativas

---

## Possíveis Problemas Remanescentes

### Problema: "ERRO DE PROXY: connection refused"
**Causa**: OpenCode não consegue conectar no proxy
**Solução**:
```bash
# Verificar se porta 9000 está em uso
netstat -ano | findstr :9000

# Se estiver em uso, mudar porta em guardian.go:
# Linha: srv := &http.Server{Addr: ":9001", ...}
# E em OpenCode settings: "127.0.0.1:9001"
```

### Problema: "Invalid JSON after redaction"
**Causa**: Pattern de redação quebrou o JSON
**Solução**:
```bash
# Verificar quais padrões estão ativos em galileu.yml
# Pode haver padrão regex muito agressivo
# Editar para ser mais específico
```

### Problema: Muitos "Timeout" errors
**Causa**: 30s é muito pouco para requisições longas
**Solução**:
```bash
# Em guardian.go, aumentar timeout:
srv := &http.Server{
    ReadTimeout:  60 * time.Second,  # Aumentado
    WriteTimeout: 60 * time.Second,  # Aumentado
    IdleTimeout:  120 * time.Second, # Aumentado
}
# Recompilar e testar
```

### Problema: "Streaming não será modificado" mas OpenCode ainda lento
**Causa**: Proxy ainda analisa (apenas não modifica)
**Solução**: 
```bash
# Se análise é lenta, otimizar padrões em galileu.yml
# Ou aumentar LogWorkerPool size em guardian.go linha 129:
# logWorkerPool = NewLogWorkerPool(8, 200)  # Aumentado workers
```

---

## Monitoramento Contínuo

### Logs a Observar

**Bom** ✅:
```
[GALILEU] Interceptado em opencode.ai: Dados sensiveis removidos.
[GALILEU] Streaming não será modificado
# Requisições fluindo normalmente
```

**Preocupante** ⚠️:
```
[GALILEU] Aviso: fila de logging cheia, descartando log
# Aumentar buffer: NewLogWorkerPool(4, 200)
```

**Crítico** 🔴:
```
[GALILEU] ❌ ERRO DE PROXY: read tcp: connection reset by peer
# Investigar imediatamente
```

### Verificar Arquivo de Auditoria

```bash
# Logs detalhados são salvos em:
# c:\Desenvolvimento\galileulinux\src\galileu-unified\galileu_audit.log

# Ver últimas linhas:
Get-Content .\galileu_audit.log -Tail 20

# Procurar por erros:
Select-String -Path .\galileu_audit.log -Pattern '"proxy_error":true'
```

---

## Configuração Recomendada para OpenCode

**VS Code Settings** (`settings.json`):
```json
{
    "[redacted]": {
        "proxy": "http://localhost:9000",
        "proxyStrictSSL": false,
        "proxyAuthorization": ""
    }
}
```

**Variáveis de Ambiente** (opcional):
```powershell
$env:HTTP_PROXY = "http://localhost:9000"
$env:HTTPS_PROXY = "http://localhost:9000"
$env:NO_PROXY = "localhost,127.0.0.1"
```

---

## Performance Tuning

Se Galileu está lento:

### 1. Aumentar Workers (mais paralelismo)
```go
// Em guardian.go linha 129
logWorkerPool = NewLogWorkerPool(8, 200)  // De 4 para 8
```

### 2. Otimizar Padrões
```yaml
# Em galileu.yml
analyzer:
  built_in:
    openai_key: true        # Manter ativado
    github_token: false     # Desativar se não necessário
    discord_token: false    # Desativar se não necessário
```

### 3. Aumentar Buffer de Logging
```go
// Em guardian.go linha 129
logWorkerPool = NewLogWorkerPool(4, 500)  // De 100 para 500
```

---

## Debugging Avançado

### 1. Habilitar Verbose Logging

Editar `guardian.go` para adicionar mais logs:

```go
fmt.Printf("[GALILEU] DEBUG: Analisando %d bytes de %s\n", 
    len(bodyBytes), r.Host)
fmt.Printf("[GALILEU] DEBUG: Streaming=%v, Model=%s\n", 
    payloadInfo.stream, payloadInfo.model)
```

### 2. Usar Wireshark

Para ver tráfego de rede:
```bash
# No Wireshark, filtrar por:
tcp.port == 9000
# Verificar se dados estão sendo transmitidos corretamente
```

### 3. CURL para Teste Manual

```bash
# Teste direto sem OpenCode
curl -X POST http://localhost:9000 \
  -H "Content-Type: application/json" \
  --data-binary @payload.json \
  -v

# -v mostra headers e detalhes de conexão
```

---

## Checklist Final de Deployment

- [ ] Compilar versão com correções
- [ ] Testar com curl primeiro
- [ ] Testar com OpenCode em terminal simples
- [ ] Testar com OpenCode em Integrated Terminal
- [ ] Monitorar logs por 30 minutos
- [ ] Testar com streaming (ex: /explain)
- [ ] Testar com dados sensíveis (verificar redação)
- [ ] Testar timeout (aguardar 40s)
- [ ] Verificar arquivo de auditoria
- [ ] Confirmar sem "ERRO DE PROXY" messages

---

## Suporte Adicional

Se problemas persistirem:

1. Coletar logs completos: `copy .\galileu_audit.log audit_backup.log`
2. Executar com debug aumentado
3. Testar com configuração mínima (padrões built-in apenas)
4. Isolar problema: testar com/sem OpenCode
5. Documentar erro exato com timestamp

