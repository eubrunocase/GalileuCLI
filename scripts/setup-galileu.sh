#!/bin/bash
# Galileu - Script de Setup Inicial para macOS
# Este script automatiza a instalação do certificado CA e a configuração do proxy.

set -e

echo ""
echo "================================================"
echo "  Galileu - Setup Inicial (macOS)"
echo "================================================"
echo ""

# Verificar se o Go está instalado
if ! command -v go &> /dev/null; then
    echo "[ERRO] Go não foi encontrado. Instale o Go 1.23+ antes de continuar."
    echo "       Download: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "[OK] Go encontrado: $GO_VERSION"

# Verificar se está na raiz do projeto
if [ ! -f "go.mod" ]; then
    echo "[ERRO] Este script deve ser executado na raiz do projeto Galileu."
    exit 1
fi

echo "[OK] Diretório do projeto confirmado."
echo ""

# Build do Galileu
echo "[1/3] Compilando o Galileu..."
GOOS=darwin GOARCH=arm64 go build -o galileu ./cmd/sentinel/main.go 2>/dev/null || \
GOOS=darwin GOARCH=amd64 go build -o galileu ./cmd/sentinel/main.go

if [ -f "galileu" ]; then
    echo "[OK] Build concluído com sucesso."
else
    echo "[ERRO] Falha na compilação."
    exit 1
fi
echo ""

# Verificar permissão de administrador
echo "[2/3] Verificando permissões de administrador..."
if sudo -n true 2>/dev/null; then
    echo "[OK] Permissões de administrador confirmadas."
else
    echo "[INFO] Será solicitada sua senha para instalar o certificado CA no Keychain do sistema."
    echo "       Esta é uma etapa necessária para que o proxy MITM funcione com HTTPS."
    echo ""
    sudo -v
    if [ $? -ne 0 ]; then
        echo "[ERRO] Permissões de administrador são necessárias."
        exit 1
    fi
    echo "[OK] Permissões de administrador obtidas."
fi
echo ""

# Executar o Galileu pela primeira vez para gerar e instalar o certificado
echo "[3/3] Gerando e instalando o certificado CA..."
echo ""

# Verificar se o certificado já existe
if [ -f "galileu-ca.pem" ]; then
    echo "[INFO] Certificado CA existente encontrado."
    echo "       Deseja regenerar o certificado? (s/N)"
    read -r REGEN
    if [[ "$REGEN" =~ ^[Ss]$ ]]; then
        echo "[INFO] Removendo certificados antigos..."
        rm -f galileu-ca.pem galileu-ca-key.pem
    else
        echo "[INFO] Usando certificado existente."
    fi
fi

# Instalar o certificado CA no Keychain
CERT_PATH="$(pwd)/galileu-ca.pem"

# Tentar instalar o certificado
if sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$CERT_PATH" 2>/dev/null; then
    echo "[OK] Certificado CA instalado no Keychain do sistema."
else
    echo "[INFO] Certificado pode já estar instalado ou houve um erro."
    echo "       Você pode instalar manualmente com:"
    echo "       sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CERT_PATH"
fi

echo ""
echo "================================================"
echo "  Setup concluído!"
echo "================================================"
echo ""
echo "  Para iniciar o Galileu, execute:"
echo "    ./galileu"
echo ""
echo "  Em outro terminal, configure o proxy:"
echo "    export HTTP_PROXY=http://127.0.0.1:9000"
echo "    export HTTPS_PROXY=http://127.0.0.1:9000"
echo "    opencode"
echo ""
echo "  Ou use o script pronto:"
echo "    ./start-opencode.sh"
echo ""
