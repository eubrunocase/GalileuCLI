#!/bin/bash

# Configura o roteamento de rede para passar pelo proxy do GalileuCLI
export HTTP_PROXY="http://127.0.0.1:9000"
export HTTPS_PROXY="http://127.0.0.1:9000"

# (Opcional) Desabilita a verificação de certificados TLS se o proxy interceptar tráfego HTTPS
# Isso evita o erro "SELF_SIGNED_CERT_IN_CHAIN" do Node.js
export NODE_TLS_REJECT_UNAUTHORIZED=0

echo "Iniciando Gemini CLI através do proxy em http://127.0.0.1:9000..."

# Inicia o Gemini CLI
gemini
