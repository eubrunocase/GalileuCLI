#!/bin/bash
# Script para iniciar o OpenCode com proxy configurado no macOS

echo "[GALILEU] Configurando proxy..."
export HTTP_PROXY="http://127.0.0.1:9000"
export HTTPS_PROXY="http://127.0.0.1:9000"

# (Opcional) Esta linha garante que a extensão do Copilot/OpenCode aceite o 
# certificado MITM gerado pelo Galileu sem dar erro de "Self-Signed Certificate"
export NODE_TLS_REJECT_UNAUTHORIZED=0

echo "[GALILEU] Abrindo OpenCode..."
# Chama o executável do OpenCode (partindo do princípio que ele está no seu PATH, tal como no Windows)
opencode