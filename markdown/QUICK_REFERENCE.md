# 🚀 Quick Reference - Comandos Úteis

## Compilação

### Compilar versão corrigida
```powershell
cd "c:\Desenvolvimento\galileulinux\src\galileu-unified"
go build -o galileu.exe ./cmd/sentinel
```

### Verificar Go version
```powershell
go version
```

### Limpar cache e recompilar
```powershell
go clean -cache
go build -o galileu.exe ./cmd/sentinel
```

---

## Execução

### Iniciar Galileu (modo normal)
```powershell
cd "c:\Desenvolvimento\galileulinux\src\galileu-unified"
.\galileu.exe
```

### Iniciar Galileu (modo debug)
```powershell
$env:DEBUG=1
.\galileu.exe
```

---

## Testes

### Teste CURL básico
```powershell
curl -X POST http://localhost:9000 `
  -H "Content-Type: application/json" `
  -d '{"model":"gpt-4","messages":[]}' `
  -v
```

### Teste com arquivo
```powershell
curl -X POST http://localhost:9000 `
  -H "Content-Type: application/json" `
  -d (Get-Content payload.json) `
  -v
```

### Teste com streaming
```powershell
curl -X POST http://localhost:9000 `
  -H "Content-Type: application/json" `
  -d '{"model":"gpt-4","stream":true,"messages":[]}' `
  -v
```

---

## Monitoramento

### Ver logs em tempo real
```powershell
Get-Content .\galileu_audit.log -Tail 20 -Wait
```

### Ver últimas 50 linhas
```powershell
Get-Content .\galileu_audit.log -Tail 50
```

### Procurar por erros
```powershell
Select-String -Path .\galileu_audit.log -Pattern '"proxy_error":true'
```

### Procurar por redações
```powershell
Select-String -Path .\galileu_audit.log -Pattern '"redacted":true'
```

### Contar requisições processadas
```powershell
(Get-Content .\galileu_audit.log | Measure-Object -Line).Lines
```

---

## Diagnóstico

### Verificar se porta 9000 está em uso
```powershell
netstat -ano | findstr :9000
```

### Matar processo na porta 9000
```powershell
$pid = (Get-NetTCPConnection -LocalPort 9000).OwningProcess
Stop-Process -Id $pid -Force
```

### Verificar conexões ativas
```powershell
Get-NetTCPConnection | Where-Object LocalPort -eq 9000
```

### Ver quantidade de conexões
```powershell
(Get-NetTCPConnection | Where-Object LocalPort -eq 9000 | Measure-Object).Count
```

---

## Documentação

### Abrir SUMARIO_EXECUTIVO
```powershell
.\SUMARIO_EXECUTIVO.md
```

### Listar todos os arquivos de documentação
```powershell
dir *.md | Format-Table Name, @{Name="KB";Expression={[math]::Round($_.Length/1KB,2)}}
```

---

## VS Code Setup

### Configurar proxy em VS Code
```json
{
    "http.proxy": "http://localhost:9000",
    "http.proxyStrictSSL": false,
    "https.proxy": "http://localhost:9000"
}
```

### Limpar cache de proxy do VS Code
```powershell
# Fechar VS Code
# Deletar pasta de cache
Remove-Item "$env:APPDATA\Code\CachedData" -Recurse -Force
```

---

## Troubleshooting

### Se compilação falhar
```powershell
# Limpar módulos
go mod tidy

# Reintentar
go build -o galileu.exe ./cmd/sentinel

# Se ainda falhar, verificar versão Go
go version  # Deve ser 1.21+
```

### Se porta 9000 estiver em uso
```powershell
# Opção 1: Mudar porta em código
# Editar guardian.go linha ~310
# srv := &http.Server{Addr: ":9001", ...}

# Opção 2: Liberar porta
netstat -ano | findstr :9000  # Find PID
taskkill /PID <PID> /F         # Kill process
```

### Se OpenCode não conectar
```powershell
# 1. Verificar se Galileu está rodando
Get-Process | Where-Object Name -like "*galileu*"

# 2. Verificar se porta está aberta
netstat -ano | findstr :9000

# 3. Testar com curl primeiro
curl -X POST http://localhost:9000 -H "Content-Type: application/json" -d "{}"

# 4. Verificar logs de erro
Select-String -Path .\galileu_audit.log -Pattern '"proxy_error":true'
```

### Se ver muitos timeouts
```powershell
# Aumentar timeouts em guardian.go:
# ReadTimeout: 60 * time.Second  (de 30)
# WriteTimeout: 60 * time.Second (de 30)
# IdleTimeout: 120 * time.Second (de 60)

# Recompilar
go build -o galileu.exe ./cmd/sentinel
```

---

## Backup & Cleanup

### Fazer backup de logs
```powershell
$date = Get-Date -Format "yyyy-MM-dd_HHmmss"
Copy-Item .\galileu_audit.log .\galileu_audit_$date.log
```

### Limpar logs antigos
```powershell
# Remover logs com mais de 7 dias
Get-ChildItem .\galileu_audit_*.log | Where-Object LastWriteTime -lt (Get-Date).AddDays(-7) | Remove-Item
```

### Fazer backup completo
```powershell
$date = Get-Date -Format "yyyy-MM-dd"
Compress-Archive -Path . -DestinationPath "../galileu_backup_$date.zip"
```

---

## Performance Tuning

### Aumentar workers de logging
```go
// Em guardian.go linha ~129
// Antes:
logWorkerPool = NewLogWorkerPool(4, 100)

// Depois:
logWorkerPool = NewLogWorkerPool(8, 200)
```

### Aumentar buffer de canal
```go
// Em guardian.go linha ~129
// Antes:
make(chan LogRequest, 100)

// Depois:
make(chan LogRequest, 500)
```

### Reduzir verbosidade de logs
```go
// Comentar linhas fmt.Printf que não forem críticas
// Deixar apenas:
// - [GALILEU] Erro
// - [GALILEU] Interceptado
// - [GALILEU] ERRO DE CONEXÃO
```

---

## Variáveis de Ambiente (Opcional)

### Definir proxy globalmente
```powershell
$env:HTTP_PROXY = "http://localhost:9000"
$env:HTTPS_PROXY = "http://localhost:9000"
$env:NO_PROXY = "localhost,127.0.0.1"
```

### Verificar proxy definido
```powershell
$env:HTTP_PROXY
$env:HTTPS_PROXY
```

---

## Desenvolvimento Rápido

### Ciclo de desenvolvimento
```powershell
# 1. Editar código
code internal/guardian/guardian.go

# 2. Compilar
go build -o galileu.exe ./cmd/sentinel

# 3. Testar
.\galileu.exe

# 4. Monitorar logs (em outro terminal)
Get-Content .\galileu_audit.log -Tail 10 -Wait

# 5. Testar com curl
curl -X POST http://localhost:9000 ...
```

---

## Dicas & Tricks

### Ver status do proxy em tempo real
```powershell
# Script PowerShell que monitora
while ($true) {
    Clear-Host
    Write-Host "Galileu Status:"
    Write-Host "Conexões: $((Get-NetTCPConnection | Where-Object LocalPort -eq 9000 | Measure-Object).Count)"
    Write-Host "Últimos logs:"
    Get-Content .\galileu_audit.log -Tail 5
    Start-Sleep -Seconds 2
}
```

### Teste de carga simples
```powershell
# Enviar 10 requisições
for ($i=1; $i -le 10; $i++) {
    curl -X POST http://localhost:9000 `
      -H "Content-Type: application/json" `
      -d "{}" -s | Out-Null
    Write-Host "Requisição $i enviada"
}
```

### Verificar integridade JSON
```powershell
# Validar JSON antes de enviar
$json = Get-Content payload.json
$parsed = $json | ConvertFrom-Json  # Se falhar, JSON inválido
Write-Host "JSON válido!"
```

---

## Links Rápidos

- Documentação: [README_CORRECOES.md](README_CORRECOES.md)
- Testes: [GUIA_TESTES.md](GUIA_TESTES.md)
- Troubleshooting: [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

---

## Atalhos Windows

### PowerShell
| Comando | Atalho |
|---------|--------|
| Limpar tela | `Clear-Host` ou `cls` |
| Histórico | ↑ ou ↓ |
| Ctrl+C | Interromper |
| Ctrl+L | Limpar tela |

### VS Code
| Comando | Atalho |
|---------|--------|
| Terminal | Ctrl+` |
| Comando | Ctrl+Shift+P |
| Buscar arquivo | Ctrl+P |
| Buscar texto | Ctrl+F |

---

**Atualizado em**: 2026-05-04  
**Versão**: 1.x-fixed

