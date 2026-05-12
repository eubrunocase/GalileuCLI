# 🧪 Guia de Testes - Validação das Correções Galileu

## Preparação do Ambiente

### Pré-requisitos
- ✅ Go 1.21+ instalado
- ✅ Windows 10+ (para certificados CA)
- ✅ Privilégios de administrador (necessário)
- ✅ VS Code instalado (para teste com OpenCode)

### Compilar Versão Corrigida
```powershell
cd "c:\Desenvolvimento\galileulinux\src\galileu-unified"
go build -o galileu.exe ./cmd/galileu
```

**Validar**: Arquivo `galileu.exe` deve ser criado (~15-20 MB)

---

## Teste 1: Startup Básico (5 min)

### Executar Galileu
```powershell
# Com privilégios de admin
cd "c:\Desenvolvimento\galileulinux\src\galileu-unified"
.\galileu.exe
```

### Observar Inicialização
```
✅ ESPERADO:
[GALILEU] Certificado CA encontrado e carregado.
[GALILEU] Certificado Root CA instalado com sucesso.
[Galileu] XX padrão(ões) de detecção carregado(s) a partir de 'galileu.yml'.
[GALILEU] Proxy ativo na porta 9000.
[GALILEU] Logging de auditoria ativo: galileu_audit.log
[GALILEU] Proxy MITM Ativo na porta 9000...
```

### Validar Ausência de Erros
```
❌ NÃO DEVE APARECER:
- Qualquer "error" message
- Stack trace
- "listener error" na porta
```

### Resultado
- ✅ **PASSOU**: Todas as mensagens aparecem, sem erros
- ❌ **FALHOU**: Erros na inicialização

---

## Teste 2: Teste com CURL (10 min)

### Preparar Payload de Teste
Criar arquivo `test_payload.json`:
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there!"}
  ]
}
```

### Teste 2A: Requisição Normal
```powershell
# Em outro PowerShell (deixar Galileu rodando)
$Headers = @{"Content-Type" = "application/json"}
$Body = Get-Content test_payload.json

curl -X POST http://localhost:9000 `
  -H "Content-Type: application/json" `
  -d $Body `
  -v
```

### Observar Logs do Galileu
```
✅ ESPERADO:
Sem "ERRO DE CONEXÃO"
Requisição passa pelo proxy normalmente
```

### Resultado
- ✅ **PASSOU**: Sem erros no Galileu
- ❌ **FALHOU**: Erro de conexão logado

### Teste 2B: Requisição com Streaming
Criar `test_streaming.json`:
```json
{
  "model": "gpt-4",
  "stream": true,
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

```powershell
curl -X POST http://localhost:9000 `
  -H "Content-Type: application/json" `
  -d (Get-Content test_streaming.json) `
  -v
```

### Observar Logs
```
✅ ESPERADO:
Nem "[GALILEU] Interceptado" nem modificações para streaming
Apenas passa requisição adiante
```

### Resultado
- ✅ **PASSOU**: Streaming não é modificado
- ❌ **FALHOU**: Log mostra "Interceptado" para stream

---

## Teste 3: Teste com OpenCode (15 min)

### Preparar VS Code para Proxy
1. Abrir VS Code `settings.json`:
   - `Ctrl+Shift+P` → "Preferences: Open Settings (JSON)"
   
2. Adicionar configuração de proxy:
```json
{
  "http.proxy": "http://localhost:9000",
  "http.proxyStrictSSL": false,
  "https.proxy": "http://localhost:9000"
}
```

3. Salvar e recarregar VS Code

### Teste 3A: Teste de Comando AI
1. Abrir terminal integrado em VS Code
   - `Ctrl+` ` (backtick)
   
2. Enviar comando de IA:
   - `/explain` (ou outro comando disponível)
   
3. Observar resposta

### Observar Logs do Galileu
```
✅ ESPERADO:
[GALILEU] Interceptado em opencode.ai: Dados sensiveis removidos.
[GALILEU] Multiple requests going through successfully
Sem "❌ ERRO DE CONEXÃO"
```

### Resultado
- ✅ **PASSOU**: VS Code responde, logs mostram sucesso
- ⚠️ **PARCIAL**: Resposta lenta (> 5s para primeira tentativa OK)
- ❌ **FALHOU**: VS Code mostra erro, logs mostram "ERRO DE CONEXÃO"

### Teste 3B: Teste de Timeout (verificar nova funcionalidade)
1. Deixar requisição aberta por > 30 segundos sem responder
2. Aguardar resposta do proxy

### Observar
```
✅ ESPERADO:
Depois de ~30s, conexão fecha gracefully
Sem "hang" indefinido
```

### Resultado
- ✅ **PASSOU**: Conexão fecha após timeout
- ❌ **FALHOU**: Conexão permanece aberta indefinidamente

---

## Teste 4: Teste de Erro Simulado (5 min)

### Simular Erro de Conexão
1. Pausar OpenCode (não responder ao proxy)
2. Enviar requisição via curl

### Observar Logs
```
✅ ESPERADO:
[GALILEU] ❌ ERRO DE CONEXÃO: <error details>
Erro é capturado e logado
Cliente recebe 502 Bad Gateway
```

### Resultado
- ✅ **PASSOU**: Erro capturado e logado
- ❌ **FALHOU**: Nenhum log de erro apareceu

---

## Teste 5: Teste de Integridade JSON (5 min)

### Criar Payload com Dados Sensíveis
Arquivo `test_sensitive.json`:
```json
{
  "model": "gpt-4",
  "api_key": "sk-proj-abcdefghijklmnopqrst1234567890ab",
  "messages": [
    {"role": "user", "content": "Help me with this code"}
  ]
}
```

### Enviar via Proxy
```powershell
curl -X POST http://localhost:9000 `
  -H "Content-Type: application/json" `
  -d (Get-Content test_sensitive.json) `
  -v
```

### Observar Logs
```
✅ ESPERADO:
[GALILEU] Interceptado em opencode.ai: Dados sensiveis removidos.
Sem "[GALILEU] ERRO: JSON inválido após redação"
```

### Validar JSON Redatado
1. Verificar em `galileu_audit.log` se redação foi aplicada
2. Confirmar que JSON redatado é válido

### Resultado
- ✅ **PASSOU**: Redação aplicada, JSON mantém validade
- ⚠️ **PARCIAL**: Redação aplicada mas JSON ficou inválido (bug!)
- ❌ **FALHOU**: Redação não foi aplicada

---

## Teste 6: Teste de Carga (20 min) - Opcional

### Enviar Múltiplas Requisições
```powershell
# Teste de 10 requisições simultâneas
$tasks = @()
for ($i = 1; $i -le 10; $i++) {
    $tasks += {
        curl -X POST http://localhost:9000 `
          -H "Content-Type: application/json" `
          -d (Get-Content test_payload.json) `
          -s
    }
}

$tasks | ForEach-Object { Invoke-Command $_ }
```

### Observar
```
✅ ESPERADO:
Todas as 10 requisições passam
Logs mostram processamento normal
Sem "fila de logging cheia"
```

### Resultado
- ✅ **PASSOU**: Carga suportada
- ⚠️ **PARCIAL**: Alguns timeouts (aumentar para 45s)
- ❌ **FALHOU**: Muitos erros de conexão

---

## Teste 7: Verificação de Auditoria (5 min)

### Inspecionar Logs de Auditoria
```powershell
# Ver últimas 20 linhas
Get-Content .\galileu_audit.log -Tail 20

# Procurar por erros
Select-String -Path .\galileu_audit.log -Pattern '"proxy_error":true'

# Procurar por sucessos
Select-String -Path .\galileu_audit.log -Pattern '"redacted":true'
```

### Validar
```
✅ ESPERADO:
Muitas entradas com "redacted": true ou false
Algumas "proxy_error": true para testes de erro
Timestamps bem distribuídos
```

### Resultado
- ✅ **PASSOU**: Logs completos e consistentes
- ⚠️ **PARCIAL**: Logs incompletos (buffer pequeno)
- ❌ **FALHOU**: Nenhum log gerado

---

## Checklist Final de Teste

Após completar todos os testes:

- [ ] Teste 1: Startup sem erros ✅
- [ ] Teste 2A: CURL básico funciona ✅
- [ ] Teste 2B: Streaming não é modificado ✅
- [ ] Teste 3A: OpenCode responde ✅
- [ ] Teste 3B: Timeout funciona após 30s ✅
- [ ] Teste 4: Erros são capturados e logados ✅
- [ ] Teste 5: JSON redatado permanece válido ✅
- [ ] Teste 6: Carga suportada ✅ (opcional)
- [ ] Teste 7: Auditoria completa ✅

---

## Se Algum Teste Falhar

### Passo 1: Coletar Evidência
```powershell
# Salvar logs
Copy-Item .\galileu_audit.log .\audit_backup_YYYY-MM-DD.log
```

### Passo 2: Aumentar Verbosidade
Editar `guardian.go` para adicionar mais `fmt.Printf()` no ponto de falha

### Passo 3: Verificar Compilação
```powershell
go version
go env
```

### Passo 4: Testar Isoladamente
- Testar sem OpenCode (só curl)
- Testar com payload mínimo
- Testar sem proxy (direto ao servidor)

### Passo 5: Recompilar e Testar
```powershell
go clean -cache
go build -o galileu.exe ./cmd/galileu
```

---

## Documentação de Resultados

Criar arquivo `TEST_RESULTS.md`:
```markdown
# Resultados de Teste - Galileu v1.x [DATA]

## Testes Executados

### Teste 1: Startup
- Status: ✅ PASSOU / ❌ FALHOU
- Observações: [descrever]

### Teste 2A: CURL Básico
- Status: ✅ PASSOU / ❌ FALHOU
- Observações: [descrever]

...

## Resumo
- Total de testes: 7
- Testes passaram: X
- Testes falharam: Y

## Recomendações
[descrever próximos passos]
```

---

## Próximas Fases

### Fase 1: Validação (Esta semana)
- Completar todos os 7 testes
- Documentar resultados
- Corrigir qualquer problema identificado

### Fase 2: Stress Test (Semana 2)
- Teste de carga (100+ requisições)
- Teste de longa duração (1+ hora)
- Monitoramento de recursos (CPU, Memória)

### Fase 3: Production (Semana 3)
- Deploy em ambiente de staging
- Monitoramento 24h
- Fine-tuning de configurações

---

## Contato & Suporte

Se tiver dúvidas durante os testes:
1. Verificar `TROUBLESHOOTING.md`
2. Verificar `CORRECOES_IMPLEMENTADAS.md`
3. Aumentar verbosidade no código
4. Coletar logs e analisar

