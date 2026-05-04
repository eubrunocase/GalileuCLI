@echo off
REM Script para iniciar o OpenCode com proxy configurado
REM Execute este arquivo em um CMD sem privilegios de administrador

echo [GALILEU] Configurando proxy...
set HTTP_PROXY=http://127.0.0.1:9000
set HTTPS_PROXY=http://127.0.0.1:9000

echo [GALILEU] Abrindo OpenCode...
opencode

pause