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
