# Galileu — Makefile de compilação multiplataforma
#
# Uso: make <target>
#
# Comandos disponíveis:
#   make build        - Compilar para a plataforma atual
#   make build-all    - Compilar para todas as plataformas
#   make checksums    - Gerar checksums SHA256 de todos os binários
#   make run          - Executar o proxy
#   make doctor       - Executar diagnóstico do sistema
#   make version      - Mostrar versão
#   make clean        - Remover binários

BINARY_NAME=galileu
CMD_PATH=./cmd/sentinel/main.go

# Binários para cada plataforma
BINARY_DARWIN_ARM64=galileu-darwin-arm64
BINARY_DARWIN_AMD64=galileu-darwin-amd64
BINARY_WINDOWS=galileu-windows-amd64.exe
BINARY_LINUX=galileu-linux-amd64

# ─── Compilação ─────────────────────────────────────────────────────────────────

build:
	@echo "[Galileu] A compilar para a plataforma atual..."
	go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

# ─── macOS ───────────────────────────────────────────────────────────────────────

build-mac-arm:
	@echo "[Galileu] A compilar para macOS Apple Silicon (ARM64)..."
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_DARWIN_ARM64) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_DARWIN_ARM64)"

build-mac-intel:
	@echo "[Galileu] A compilar para macOS Intel (AMD64)..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DARWIN_AMD64) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_DARWIN_AMD64)"

# ─── Windows ──────────────────────────────────────────────────────────────────────

build-windows:
	@echo "[Galileu] A compilar para Windows (AMD64)..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_WINDOWS) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_WINDOWS)"

# ─── Linux ───────────────────────────────────────────────────────────────────────

build-linux:
	@echo "[Galileu] A compilar para Linux (AMD64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_LINUX) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_LINUX)"

# ─── Todos ───────────────────────────────────────────────────────────────────────

build-all: build-mac-arm build-mac-intel build-windows build-linux checksums
	@echo "[Galileu] Compilação multiplataforma concluída."

# ─── Checksums SHA256 ────────────────────────────────────────────────────────────

checksums:
	@echo "[Galileu] A calcular checksums SHA256..."
	@echo "galileu-darwin-arm64  $$(sha256sum $(BINARY_DARWIN_ARM64) | cut -d' ' -f1)" > checksums.txt
	@echo "galileu-darwin-amd64  $$(sha256sum $(BINARY_DARWIN_AMD64) | cut -d' ' -f1)" >> checksums.txt
	@echo "galileu-windows-amd64.exe $$(sha256sum $(BINARY_WINDOWS) | cut -d' ' -f1)" >> checksums.txt
	@echo "galileu-linux-amd64  $$(sha256sum $(BINARY_LINUX) | cut -d' ' -f1)" >> checksums.txt
	@echo "[Galileu] Checksums salvos em checksums.txt"

checksums-darwin-arm64:
	@echo "galileu-darwin-arm64  $$(sha256sum $(BINARY_DARWIN_ARM64) | cut -d' ' -f1)"

checksums-darwin-amd64:
	@echo "galileu-darwin-amd64  $$(sha256sum $(BINARY_DARWIN_AMD64) | cut -d' ' -f1)"

checksums-windows:
	@echo "galileu-windows-amd64.exe $$(sha256sum $(BINARY_WINDOWS) | cut -d' ' -f1)"

checksums-linux:
	@echo "galileu-linux-amd64  $$(sha256sum $(BINARY_LINUX) | cut -d' ' -f1)"

# ─── Execução ────────────────────────────────────────────────────────────────────

run:
	@echo "[Galileu] A iniciar o proxy..."
	go run $(CMD_PATH)

doctor:
	@echo "[Galileu] A executar diagnóstico..."
	go run $(CMD_PATH) doctor

version:
	@echo "[Galileu] A obter versão..."
	go run $(CMD_PATH) version

# ─── Utilitários ────────────────────────────────────────────────────────────────

clean:
	@echo "[Galileu] A remover binários..."
	rm -f $(BINARY_NAME) $(BINARY_DARWIN_ARM64) $(BINARY_DARWIN_AMD64) $(BINARY_WINDOWS) $(BINARY_LINUX) checksums.txt
	@echo "[Galileu] Limpeza concluída."

help:
	@echo "Galileu - Makefile de compilação"
	@echo ""
	@echo "Compilação:"
	@echo "  make build           - Compilar para a plataforma atual"
	@echo "  make build-all       - Compilar para todas as plataformas + checksums"
	@echo "  make build-mac-arm   - Compilar para macOS Apple Silicon"
	@echo "  make build-mac-intel - Compilar para macOS Intel"
	@echo "  make build-windows   - Compilar para Windows"
	@echo "  make build-linux     - Compilar para Linux"
	@echo ""
	@echo "Checksums:"
	@echo "  make checksums        - Gerar checksums SHA256 de todos os binários"
	@echo ""
	@echo "Execução:"
	@echo "  make run             - Iniciar o proxy"
	@echo "  make doctor          - Executar diagnóstico"
	@echo "  make version         - Mostrar versão"
	@echo ""
	@echo "Outros:"
	@echo "  make clean           - Remover binários"
	@echo "  make help            - Mostrar esta ajuda"

.PHONY: build build-mac-arm build-mac-intel build-windows build-linux build-all checksums run doctor version clean help