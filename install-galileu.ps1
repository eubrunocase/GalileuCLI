# ==============================================================================
# install-galileu.ps1
# Script de instalação do Galileu como serviço Windows
#
# Pré-requisitos:
#   - Executar como Administrador
#   - PowerShell 5.1+
#   - Acesso à internet (api.github.com e github.com)
#
# Uso:
#   .\install-galileu.ps1
#   .\install-galileu.ps1 -ConfigRepoUrl "https://raw.githubusercontent.com/sua-org/galileu-configs/main/galileu.yml"
#   .\install-galileu.ps1 -Uninstall
# ==============================================================================

param(
    [string]$InstallPath     = "C:\Program Files\Galileu",
    [string]$ServiceName     = "GalileuProxy",
    [string]$ServiceDisplay  = "Galileu Security Proxy",
    [string]$ServiceDesc     = "Proxy de segurança para ferramentas de AI coding",
    [string]$GithubRepo      = "eubrunocase/GalileuCLI",
    [string]$ConfigRepoUrl   = "",        # URL raw do galileu.yml no repo privado de configs
    [string]$ConfigRepoToken = "",        # Token de acesso se o repo for privado
    [switch]$Uninstall
)

# ------------------------------------------------------------------------------
# Utilitários
# ------------------------------------------------------------------------------

function Write-Step([string]$msg) {
    Write-Host "`n==> $msg" -ForegroundColor Cyan
}

function Write-Success([string]$msg) {
    Write-Host "  [OK] $msg" -ForegroundColor Green
}

function Write-Fail([string]$msg) {
    Write-Host "  [ERRO] $msg" -ForegroundColor Red
}

function Assert-Admin {
    $current = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not $current.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Fail "Execute este script como Administrador."
        exit 1
    }
}

function Get-LatestRelease {
    Write-Step "Consultando última versão no GitHub..."
    try {
        $headers = @{ "User-Agent" = "GalileuInstaller/1.0" }
        $release = Invoke-RestMethod "https://api.github.com/repos/$GithubRepo/releases/latest" -Headers $headers
        Write-Success "Versão mais recente: $($release.tag_name)"
        return $release
    }
    catch {
        Write-Fail "Não foi possível consultar a API do GitHub: $_"
        exit 1
    }
}

function Get-ReleaseChecksum([string]$version) {
    # Baixa o checksums.txt da release e retorna hashtable [filename -> sha256]
    $checksumUrl = "https://github.com/$GithubRepo/releases/download/$version/checksums.txt"
    try {
        $raw = (Invoke-WebRequest $checksumUrl -UseBasicParsing).Content
        $table = @{}
        foreach ($line in ($raw -split "`n")) {
            $line = $line.Trim()
            if ($line -match "^([a-fA-F0-9]{64})\s+(.+)$") {
                $table[$Matches[2].Trim()] = $Matches[1].ToUpper()
            }
        }
        return $table
    }
    catch {
        Write-Fail "Não foi possível baixar checksums.txt: $_"
        exit 1
    }
}

function Confirm-FileChecksum([string]$filePath, [string]$expected) {
    $actual = (Get-FileHash $filePath -Algorithm SHA256).Hash.ToUpper()
    if ($actual -ne $expected.ToUpper()) {
        Write-Fail "Checksum inválido para $filePath"
        Write-Host "    Esperado : $expected"
        Write-Host "    Calculado: $actual"
        return $false
    }
    return $true
}

function Save-CurrentVersion([string]$version) {
    $version | Set-Content "$InstallPath\.version" -Encoding UTF8
}

function Get-CurrentVersion {
    $versionFile = "$InstallPath\.version"
    if (Test-Path $versionFile) {
        return (Get-Content $versionFile -Raw).Trim()
    }
    return $null
}

# ------------------------------------------------------------------------------
# Desinstalação
# ------------------------------------------------------------------------------

function Invoke-Uninstall {
    Write-Step "Desinstalando Galileu..."

    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($svc) {
        if ($svc.Status -eq "Running") {
            Stop-Service -Name $ServiceName -Force
            Write-Success "Serviço parado."
        }
        sc.exe delete $ServiceName | Out-Null
        Write-Success "Serviço removido."
    }
    else {
        Write-Host "  Serviço não encontrado, pulando."
    }

    if (Test-Path $InstallPath) {
        Remove-Item $InstallPath -Recurse -Force
        Write-Success "Diretório $InstallPath removido."
    }

    Write-Host "`nGalileu desinstalado com sucesso." -ForegroundColor Green
}

# ------------------------------------------------------------------------------
# Download do binário
# ------------------------------------------------------------------------------

function Install-Binary([object]$release) {
    $version = $release.tag_name
    $assetName = "galileu.exe"

    $asset = $release.assets | Where-Object { $_.name -eq $assetName }
    if (-not $asset) {
        Write-Fail "Asset '$assetName' não encontrado na release $version."
        exit 1
    }

    Write-Step "Baixando $assetName ($version)..."

    $tmpPath = "$InstallPath\galileu.exe.tmp"
    try {
        Invoke-WebRequest $asset.browser_download_url -OutFile $tmpPath -UseBasicParsing
        Write-Success "Download concluído."
    }
    catch {
        Write-Fail "Falha no download: $_"
        exit 1
    }

    # Validar checksum
    Write-Step "Validando integridade do binário..."
    $checksums = Get-ReleaseChecksum $version
    if ($checksums.ContainsKey($assetName)) {
        if (-not (Confirm-FileChecksum $tmpPath $checksums[$assetName])) {
            Remove-Item $tmpPath -Force
            exit 1
        }
        Write-Success "Checksum válido."
    }
    else {
        Write-Host "  [AVISO] Checksum para $assetName não encontrado. Prosseguindo sem validação." -ForegroundColor Yellow
    }

    # Fazer backup do binário atual se existir
    $binaryPath = "$InstallPath\galileu.exe"
    if (Test-Path $binaryPath) {
        Copy-Item $binaryPath "$InstallPath\galileu.exe.bak" -Force
        Write-Success "Backup do binário anterior criado."
    }

    Move-Item $tmpPath $binaryPath -Force
    Write-Success "Binário instalado em $binaryPath"

    Save-CurrentVersion $version
}

# ------------------------------------------------------------------------------
# Download da config da equipe
# ------------------------------------------------------------------------------

function Install-Config {
    $configPath = "$InstallPath\galileu.yml"

    # Se uma URL de config foi fornecida, baixar do repo da equipe
    if ($ConfigRepoUrl -ne "") {
        Write-Step "Baixando galileu.yml do repositório da equipe..."
        try {
            $headers = @{ "User-Agent" = "GalileuInstaller/1.0" }
            if ($ConfigRepoToken -ne "") {
                $headers["Authorization"] = "Bearer $ConfigRepoToken"
            }
            Invoke-WebRequest $ConfigRepoUrl -Headers $headers -OutFile $configPath -UseBasicParsing
            Write-Success "Configuração da equipe aplicada."
        }
        catch {
            Write-Fail "Não foi possível baixar galileu.yml: $_"
            Write-Host "  Continuando com configuração padrão." -ForegroundColor Yellow
            Write-DefaultConfig $configPath
        }
    }
    else {
        # Sem URL de config: gerar config padrão
        if (-not (Test-Path $configPath)) {
            Write-Step "Gerando galileu.yml padrão..."
            Write-DefaultConfig $configPath
            Write-Success "Configuração padrão criada em $configPath"
        }
        else {
            Write-Host "  galileu.yml já existe. Mantendo configuração atual."
        }
    }
}

function Write-DefaultConfig([string]$path) {
    @"
# galileu.yml — Configuração padrão gerada pelo instalador
# Edite conforme as necessidades da equipe

mode: passive          # passive | whitelist

# Padrões customizados adicionais (opcional)
# custom_patterns:
#   - name: "ID Interno"
#     type: regex
#     pattern: 'CORP-[A-Z0-9]{8}'
"@ | Set-Content $path -Encoding UTF8
}

# ------------------------------------------------------------------------------
# Registro do serviço Windows
# ------------------------------------------------------------------------------

function Install-WindowsService {
    Write-Step "Registrando serviço Windows..."

    $binaryPath = "$InstallPath\galileu.exe"
    $existingSvc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

    if ($existingSvc) {
        Write-Host "  Serviço já existe. Atualizando configuração..." -ForegroundColor Yellow
        if ($existingSvc.Status -eq "Running") {
            Stop-Service -Name $ServiceName -Force
        }
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    }

    New-Service `
        -Name $ServiceName `
        -BinaryPathName "`"$binaryPath`" --service" `
        -DisplayName $ServiceDisplay `
        -StartupType Automatic `
        -Description $ServiceDesc | Out-Null

    # Configurar restart automático em caso de falha
    sc.exe failure $ServiceName reset= 60 actions= restart/5000/restart/10000/restart/30000 | Out-Null

    Write-Success "Serviço registrado com startup automático e restart em falha."
}

function Start-GalileuService {
    Write-Step "Iniciando serviço..."
    try {
        Start-Service -Name $ServiceName
        Start-Sleep -Seconds 2
        $svc = Get-Service -Name $ServiceName
        if ($svc.Status -eq "Running") {
            Write-Success "Serviço iniciado com sucesso."
        }
        else {
            Write-Fail "Serviço não iniciou corretamente. Status: $($svc.Status)"
            Write-Host "  Verifique o Event Viewer (Windows Logs > Application) para detalhes."
        }
    }
    catch {
        Write-Fail "Falha ao iniciar o serviço: $_"
    }
}

# ------------------------------------------------------------------------------
# Instalação principal
# ------------------------------------------------------------------------------

function Invoke-Install {
    Assert-Admin

    Write-Host "`n================================================" -ForegroundColor Cyan
    Write-Host "  Galileu Security Proxy — Instalador"          -ForegroundColor Cyan
    Write-Host "================================================`n" -ForegroundColor Cyan

    # Criar diretório de instalação
    Write-Step "Preparando diretório de instalação..."
    New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null
    Write-Success "Diretório: $InstallPath"

    # Buscar release mais recente
    $release = Get-LatestRelease

    # Verificar se já está na versão mais recente
    $currentVersion = Get-CurrentVersion
    if ($currentVersion -eq $release.tag_name) {
        Write-Host "`n  Galileu já está na versão mais recente ($currentVersion)." -ForegroundColor Green
    }
    else {
        # Baixar e validar binário
        Install-Binary $release
    }

    # Baixar ou gerar config
    Install-Config

    # Registrar e iniciar serviço
    Install-WindowsService
    Start-GalileuService

    # Resumo
    Write-Host "`n================================================" -ForegroundColor Green
    Write-Host "  Instalação concluída!" -ForegroundColor Green
    Write-Host "================================================" -ForegroundColor Green
    Write-Host "  Versão  : $($release.tag_name)"
    Write-Host "  Diretório : $InstallPath"
    Write-Host "  Serviço : $ServiceName (Automático)"
    Write-Host ""
    Write-Host "  Para verificar o status:"
    Write-Host "    Get-Service -Name $ServiceName" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "  Para desinstalar:"
    Write-Host "    .\install-galileu.ps1 -Uninstall" -ForegroundColor DarkGray
    Write-Host ""
}

# ------------------------------------------------------------------------------
# Entry point
# ------------------------------------------------------------------------------

if ($Uninstall) {
    Assert-Admin
    Invoke-Uninstall
}
else {
    Invoke-Install
}