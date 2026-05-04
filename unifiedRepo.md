# Galileu — Instrução de Unificação dos Repositórios Windows e macOS

## Contexto e Objectivo

Este documento é uma instrução técnica para um agente de IA. O objectivo é unificar dois repositórios separados do projecto **Galileu** num único repositório multiplataforma, capaz de compilar e executar correctamente em **Windows**, **macOS** (Apple Silicon e Intel) e **Linux**.

### Repositórios de Origem

| Plataforma | Repositório | Estado |
|---|---|---|
| Windows | https://github.com/eubrunocase/Galileu | Versão mais antiga, estável |
| macOS / Linux | https://github.com/eubrunocase/GalileuForMac | **Versão mais actualizada — usar como base** |

O agente deve clonar ambos os repositórios, analisar as diferenças, e produzir uma versão unificada baseada no `GalileuForMac`, integrando o que existe de específico no repositório Windows.

---

## Princípios que Devem Ser Respeitados

- **GalileuForMac é a base.** Todo o código existente nesse repositório deve ser preservado integralmente. As alterações vêm por adição, não por substituição.
- **Zero duplicação de lógica.** O código do proxy, do analyzer e do audit é idêntico em todos os sistemas — não deve ser duplicado. Apenas o código de instalação do certificado CA é específico por plataforma.
- **Build tags Go, não condicionais em runtime.** A separação por plataforma deve usar o mecanismo nativo do Go (`//go:build`), não blocos `if runtime.GOOS == ...` no código partilhado.
- **Nenhuma funcionalidade existente pode ser removida ou alterada**, apenas reorganizada.
- **O utilizador final nunca deve perceber que o código foi unificado** — a experiência de uso em cada plataforma deve ser idêntica à original.

---

## Análise das Diferenças entre os Repositórios

O agente deve ler ambos os repositórios e confirmar as seguintes diferenças antes de iniciar qualquer alteração:

### Diferenças Confirmadas

| Elemento | Windows (`Galileu`) | macOS (`GalileuForMac`) | Resolução |
|---|---|---|---|
| Binário de saída | `galileu.exe` | `galileu` | Go gera automaticamente com o nome correcto conforme `GOOS` |
| Script de arranque | `start-opencode.bat` | `start-opencode.sh` | Manter ambos em `scripts/` |
| Nome do certificado CA | `rootCA.pem` + `rootCA-key.pem` | `ca.pem` + `key.pem` | Padronizar para `galileu-ca.pem` + `galileu-ca-key.pem` em todo o código |
| Instalação do certificado CA | Automática via código Go (requer Admin) | Manual pelo utilizador no Keychain | Separar em ficheiros com build tags |
| Privilégios necessários | Administrador obrigatório | Sem `sudo` | Consequência do ponto anterior |
| Comando de build | `go build -o galileu.exe ./cmd/sentinel` | `GOOS=darwin GOARCH=arm64 go build -o galileu ./cmd/sentinel/main.go` | Resolver com Makefile |

### Diferença Principal: Instalação do Certificado CA

Esta é a única diferença real ao nível do código Go. No repositório Windows, o `main.go` (ou `guardian.go`) contém lógica para instalar automaticamente o certificado CA no repositório de certificados do sistema operativo — provavelmente usando `certutil` ou chamadas à Win32 API. No macOS esta instalação é feita manualmente pelo utilizador no Keychain Access.

O agente deve **localizar exactamente essa lógica** no repositório Windows antes de prosseguir.

---

## Passo 1 — Preparação do Ambiente

### 1.1 — Clonar ambos os repositórios

```bash
git clone https://github.com/eubrunocase/GalileuForMac galileu-unified
git clone https://github.com/eubrunocase/Galileu galileu-windows-ref
```

O directório `galileu-unified` será o repositório final. O directório `galileu-windows-ref` é apenas referência de leitura — **não será modificado**.

### 1.2 — Analisar a estrutura actual do repositório Windows

```bash
# Listar todos os ficheiros Go do repositório Windows
find galileu-windows-ref -name "*.go" | sort

# Ler o conteúdo de cada ficheiro Go para identificar a lógica de instalação do certificado
cat galileu-windows-ref/cmd/sentinel/main.go
cat galileu-windows-ref/internal/guardian/guardian.go
```

O agente deve identificar e anotar:
- Qual ficheiro contém a chamada de instalação do certificado CA
- Qual função faz essa instalação
- Quais imports são necessários (ex: `os/exec`, `syscall`, etc.)
- Se existe alguma lógica adicional exclusiva do Windows que não esteja no macOS

---

## Passo 2 — Reorganizar a Estrutura de Ficheiros

### 2.1 — Estrutura Final Esperada

Após a unificação, o repositório deve ter esta estrutura:

```
galileu-unified/
├── cmd/
│   └── sentinel/
│       └── main.go                    # SEM código específico de OS
├── internal/
│   └── guardian/
│       ├── guardian.go                # SEM ALTERAÇÕES
│       ├── analyzer.go                # SEM ALTERAÇÕES
│       ├── audit.go                   # SEM ALTERAÇÕES
│       ├── certinstall_windows.go     # NOVO — build tag: windows
│       ├── certinstall_darwin.go      # NOVO — build tag: darwin
│       └── certinstall_linux.go       # NOVO — build tag: linux
├── scripts/
│   ├── start.sh                       # MOVIDO de start-opencode.sh
│   └── start.bat                      # NOVO — copiado do repositório Windows
├── Makefile                           # NOVO
├── ca.pem                             # Nome padronizado
├── key.pem                            # Nome padronizado
├── galileu.yml.example                # Se já existir da implementação anterior
├── .gitignore                         # ACTUALIZADO
├── go.mod                             # SEM ALTERAÇÕES (salvo nova dependência yaml.v3 se já implementada)
├── go.sum                             # SEM ALTERAÇÕES
└── README.md                          # ACTUALIZADO
```

### 2.2 — Mover o script de arranque para a pasta `scripts/`

```bash
cd galileu-unified
mkdir -p scripts
mv start-opencode.sh scripts/start.sh
```

Verificar se o conteúdo do `scripts/start.sh` está correcto — deve ser o mesmo do `start-opencode.sh` original.

### 2.3 — Copiar o script de arranque do Windows

Copiar `start-opencode.bat` do repositório Windows para `scripts/start.bat`:

```bash
cp galileu-windows-ref/start-opencode.bat galileu-unified/scripts/start.bat
```

---

## Passo 3 — Criar os Ficheiros de Instalação de Certificado com Build Tags

Esta é a parte central da unificação. O objectivo é extrair a lógica específica de cada sistema operativo para ficheiros separados, mantendo o `main.go` agnóstico ao OS.

### 3.1 — Criar a interface comum

Em cada um dos três ficheiros abaixo, deve existir uma função com exactamente a mesma assinatura:

```go
func InstallCertificateIfNeeded(certPath string) error
```

Esta é a função que o `main.go` irá chamar — sem saber qual ficheiro está a usar.

### 3.2 — Criar `internal/guardian/certinstall_darwin.go`

Este ficheiro contém a implementação para macOS. No macOS, a instalação é manual, por isso a função apenas informa o utilizador:

```go
//go:build darwin

package guardian

import "fmt"

// InstallCertificateIfNeeded no macOS informa o utilizador que a instalação
// do certificado CA deve ser feita manualmente no Keychain Access.
// Não requer privilégios de administrador.
func InstallCertificateIfNeeded(certPath string) error {
	fmt.Println("[Galileu] macOS detectado.")
	fmt.Printf("[Galileu] Certifique-se de que '%s' está importado no Keychain Access com confiança 'Always Trust'.\n", certPath)
	fmt.Println("[Galileu] Consulte o README para instruções detalhadas.")
	return nil
}
```

### 3.3 — Criar `internal/guardian/certinstall_linux.go`

Para Linux, o comportamento é idêntico ao macOS — informação apenas, sem instalação automática:

```go
//go:build linux

package guardian

import "fmt"

// InstallCertificateIfNeeded no Linux informa o utilizador que a instalação
// do certificado CA deve ser feita manualmente no sistema de certificados da distribuição.
func InstallCertificateIfNeeded(certPath string) error {
	fmt.Println("[Galileu] Linux detectado.")
	fmt.Printf("[Galileu] Para instalar o certificado CA, execute:\n")
	fmt.Printf("  sudo cp %s /usr/local/share/ca-certificates/galileu.crt\n", certPath)
	fmt.Println("  sudo update-ca-certificates")
	return nil
}
```

### 3.4 — Criar `internal/guardian/certinstall_windows.go`

**Este ficheiro deve conter a lógica extraída do repositório Windows.** O agente deve:

1. Ler o código de instalação de certificado do repositório Windows (`galileu-windows-ref`)
2. Extrair exactamente essa lógica
3. Envolvê-la neste ficheiro com o build tag correcto

O ficheiro deve ter esta estrutura base, com a lógica real do Windows preenchida no interior da função:

```go
//go:build windows

package guardian

import (
	// Importar exactamente os pacotes que o código Windows original usa
	// Exemplos: "os/exec", "fmt", "os", "path/filepath"
	// Não adicionar nem remover imports — copiar do original
)

// InstallCertificateIfNeeded no Windows instala automaticamente o certificado CA
// no repositório de certificados do sistema. Requer privilégios de Administrador.
func InstallCertificateIfNeeded(certPath string) error {
	// INSERIR AQUI a lógica extraída do repositório Windows
	// Esta lógica deve ser copiada fielmente do código original
	// sem alterações de comportamento
}
```

**Nota crítica para o agente:** Se o código Windows original usar nomes de ficheiro diferentes (`rootCA.pem` em vez de `ca.pem`), actualizar as referências dentro desta função para usar `ca.pem` — mantendo o comportamento, mudando apenas o nome do ficheiro.

---

## Passo 4 — Actualizar o `main.go`

### 4.1 — Remover código específico de OS do `main.go`

Localizar no `main.go` do `GalileuForMac` qualquer lógica de instalação de certificado ou verificação de plataforma. Se existir, remover.

### 4.2 — Adicionar chamada à função unificada

No ponto de arranque do programa, antes de iniciar o proxy, adicionar a chamada:

```go
// Instalação / verificação do certificado CA (comportamento por plataforma)
if err := guardian.InstallCertificateIfNeeded("ca.pem"); err != nil {
    log.Fatalf("[Galileu] Erro ao verificar certificado CA: %v", err)
}
```

O restante do `main.go` permanece sem alterações.

---

## Passo 5 — Padronizar os Nomes dos Certificados

O repositório Windows usa `rootCA.pem` e `rootCA-key.pem`. O repositório macOS usa `ca.pem` e `key.pem`. O repositório unificado usa **`ca.pem` e `key.pem`** como nomes padrão.

O agente deve verificar se existem referências aos nomes antigos em algum ficheiro Go e actualizá-las:

```bash
# Verificar referências ao nome antigo
grep -r "rootCA" galileu-unified/
```

Substituir todas as ocorrências de `rootCA.pem` por `ca.pem` e `rootCA-key.pem` por `key.pem` nos ficheiros Go e nos scripts.

---

## Passo 6 — Criar o Makefile

Criar o ficheiro `Makefile` na raiz do repositório:

```makefile
# Galileu — Makefile de compilação multiplataforma

BINARY_NAME=galileu
CMD_PATH=./cmd/sentinel/main.go

# ─── macOS ────────────────────────────────────────────────────────────────────

build-mac-arm:
	@echo "[Galileu] A compilar para macOS Apple Silicon (ARM64)..."
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

build-mac-intel:
	@echo "[Galileu] A compilar para macOS Intel (AMD64)..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

# ─── Windows ──────────────────────────────────────────────────────────────────

build-windows:
	@echo "[Galileu] A compilar para Windows (AMD64)..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME).exe"

# ─── Linux ────────────────────────────────────────────────────────────────────

build-linux:
	@echo "[Galileu] A compilar para Linux (AMD64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

# ─── Todos ────────────────────────────────────────────────────────────────────

build-all: build-mac-arm build-mac-intel build-windows build-linux
	@echo "[Galileu] Compilação multiplataforma concluída."

# ─── Utilitários ──────────────────────────────────────────────────────────────

clean:
	@echo "[Galileu] A remover binários..."
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@echo "[Galileu] Limpeza concluída."

.PHONY: build-mac-arm build-mac-intel build-windows build-linux build-all clean
```

---

## Passo 7 — Actualizar o `.gitignore`

O `.gitignore` deve incluir entradas para todos os sistemas operativos. Verificar o conteúdo actual e adicionar o que faltar:

```gitignore
# Certificados — nunca versionar
*.pem
*.key
*.crt

# Configuração local (pode conter termos confidenciais)
galileu.yml

# Binários compilados
galileu
galileu.exe

# Logs de auditoria
galileu_audit.log

# Ficheiro .env de teste
.env

# macOS
.DS_Store

# Windows
Thumbs.db
```

---

## Passo 8 — Actualizar o `README.md`

O README deve ter secções claras por sistema operativo. Estrutura obrigatória:

```markdown
# Galileu — Proxy de Segurança e Governança para LLMs
> Suporta: macOS (Apple Silicon & Intel) · Windows · Linux

## Compilação

### macOS — Apple Silicon (M1/M2/M3)
make build-mac-arm

### macOS — Intel
make build-mac-intel

### Windows
make build-windows

### Linux
make build-linux

## Configuração do Certificado CA

### macOS
[instruções do Keychain Access]

### Windows
[o Galileu instala automaticamente ao correr como Administrador]

### Linux
[instruções do update-ca-certificates]

## Execução

### macOS / Linux
./galileu
./scripts/start.sh

### Windows
galileu.exe
scripts\start.bat

[restante conteúdo do README actual do GalileuForMac]
```

O conteúdo detalhado de cada secção deve ser o mesmo dos READMEs originais de cada plataforma.

---

## Passo 9 — Validação e Testes de Compilação

### 9.1 — Verificar que o código compila sem erros para todos os targets

```bash
cd galileu-unified

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o /dev/null ./cmd/sentinel/main.go
echo "macOS ARM64: OK"

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o /dev/null ./cmd/sentinel/main.go
echo "macOS AMD64: OK"

# Windows
GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/sentinel/main.go
echo "Windows AMD64: OK"

# Linux
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/sentinel/main.go
echo "Linux AMD64: OK"
```

Todos os quatro devem compilar sem erros ou warnings.

### 9.2 — Verificar que as build tags estão correctas

```bash
# Confirmar que cada ficheiro de certinstall tem o build tag correcto na primeira linha
head -1 galileu-unified/internal/guardian/certinstall_windows.go
# Resultado esperado: //go:build windows

head -1 galileu-unified/internal/guardian/certinstall_darwin.go
# Resultado esperado: //go:build darwin

head -1 galileu-unified/internal/guardian/certinstall_linux.go
# Resultado esperado: //go:build linux
```

### 9.3 — Verificar que não existem referências aos nomes antigos de certificados

```bash
grep -r "rootCA" galileu-unified/
# Resultado esperado: nenhuma linha retornada
```

### 9.4 — Verificar que o `main.go` não contém condicionais de OS

```bash
grep -n "runtime.GOOS" galileu-unified/cmd/sentinel/main.go
grep -n "runtime.GOOS" galileu-unified/internal/guardian/guardian.go
# Resultado esperado: nenhuma linha retornada em ambos
```

### 9.5 — Verificar a estrutura final de ficheiros

```bash
find galileu-unified -type f | grep -v ".git" | sort
```

A lista deve corresponder exactamente à estrutura definida no Passo 2.1.

---

## Comportamentos Obrigatórios a Preservar

O agente deve confirmar que nenhuma das seguintes funcionalidades foi alterada:

- O proxy MITM continua a correr na porta 9000 em todos os sistemas.
- Os certificados são carregados a partir de `ca.pem` e `key.pem` na raiz do projecto.
- O ficheiro `galileu_audit.log` continua a ser gerado com o mesmo formato JSON.
- No Windows, o certificado CA continua a ser instalado automaticamente ao arrancar como Administrador.
- No macOS, o comportamento é idêntico ao original do `GalileuForMac`.
- A performance não é degradada — nenhuma lógica de detecção de plataforma em runtime por requisição.

---

## O Que NÃO Fazer

- **Não modificar** `guardian.go`, `analyzer.go` ou `audit.go` — estes ficheiros são idênticos em todas as plataformas e não devem ser tocados.
- **Não usar** `if runtime.GOOS == "windows"` no código partilhado — usar sempre build tags em ficheiros separados.
- **Não apagar** o historial de commits do `GalileuForMac` — o repositório unificado deve manter o historial existente.
- **Não adicionar** dependências externas para a lógica de instalação de certificados — usar apenas a biblioteca padrão do Go e chamadas ao sistema.
- **Não alterar** o comportamento de nenhuma funcionalidade existente — esta é uma refactorização estrutural, não funcional.

---

## Resultado Esperado

Após a conclusão, o repositório unificado deve:

1. Compilar correctamente para Windows, macOS (ARM64 e AMD64) e Linux com um único `make build-*`
2. Executar o mesmo comportamento funcional em cada plataforma que os repositórios originais separados
3. Conter um único conjunto de ficheiros de lógica de negócio (`guardian.go`, `analyzer.go`, `audit.go`) sem duplicação
4. Ter a instalação do certificado CA separada por plataforma em ficheiros com build tags
5. Incluir scripts de arranque para Windows (`scripts/start.bat`) e para macOS/Linux (`scripts/start.sh`)
6. Ter um README único com instruções claras por sistema operativo
7. Ter um Makefile com targets para todos os sistemas suportados