# Galileu — Makefile de compilação multiplataforma
#
# Uso: make <target>
#
# Comandos disponíveis:
#   make build        - Compilar para a plataforma atual
#   make build-all    - Compilar para todas as plataformas
#   make run          - Executar o proxy
#   make doctor       - Executar diagnóstico do sistema
#   make version      - Mostrar versão
#   make clean        - Remover binários

BINARY_NAME=galileu
CMD_PATH=./cmd/sentinel/main.go

# ─── Compilação ─────────────────────────────────────────────────────────────────

build:
	@echo "[Galileu] A compilar para a plataforma atual..."
	go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

# ─── macOS ───────────────────────────────────────────────────────────────────────

build-mac-arm:
	@echo "[Galileu] A compilar para macOS Apple Silicon (ARM64)..."
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

build-mac-intel:
	@echo "[Galileu] A compilar para macOS Intel (AMD64)..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

# ─── Windows ──────────────────────────────────────────────────────────────────────

build-windows:
	@echo "[Galileu] A compilar para Windows (AMD64)..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME).exe"

# ─── Linux ───────────────────────────────────────────────────────────────────────

build-linux:
	@echo "[Galileu] A compilar para Linux (AMD64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "[Galileu] Binário gerado: ./$(BINARY_NAME)"

# ─── Todos ───────────────────────────────────────────────────────────────────────

build-all: build-mac-arm build-mac-intel build-windows build-linux
	@echo "[Galileu] Compilação multiplataforma concluída."

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
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@echo "[Galileu] Limpeza concluída."

help:
	@echo "Galileu - Makefile de compilação"
	@echo ""
	@echo "Compilação:"
	@echo "  make build        - Compilar para a plataforma atual"
	@echo "  make build-all    - Compilar para macOS, Windows e Linux"
	@echo "  make build-mac-arm   - Compilar para macOS Apple Silicon"
	@echo "  make build-mac-intel - Compilar para macOS Intel"
	@echo "  make build-windows   - Compilar para Windows"
	@echo "  make build-linux    - Compilar para Linux"
	@echo ""
	@echo "Execução:"
	@echo "  make run          - Iniciar o proxy"
	@echo "  make doctor       - Executar diagnóstico"
	@echo "  make version      - Mostrar versão"
	@echo ""
	@echo "Outros:"
	@echo "  make clean         - Remover binários"
	@echo "  make help          - Mostrar esta ajuda"

.PHONY: build build-mac-arm build-mac-intel build-windows build-linux build-all run doctor version clean help