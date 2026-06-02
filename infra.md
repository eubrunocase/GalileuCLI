# Planejamento de infraestrutura do Galileu no NDSI

## Objetivo

Transformar a aplicação do **Galileu CLI** em um serviço do SO que deverá ser executado junto do sistema. E automatizar o controle de versões utilizadas pela equipe.

## O que é necessário

### 1. Pacote Updater

Criar um pacote separado chamado "updater" que será chamado antes do proxy iniciar para executar a lógica de verificação. O updater deverá ter a lógica de verificar através da API do GitHub as últimas versões disponibilizadas, e caso tenham novas versões, se auto-atualizar para última versão estável.

### 2. Lógica de autoupdate no Galileu

O Galileu precisa de um mecanismo interno de atualização. Seguindo o seguinte fluxo:

- Criar um pacote separado chamado "updater" que será chamado antes do proxy iniciar para executar a lógica de verificação.
- A substituição do binário em Windows exige renomear o executável antes de escrever o novo (Windows não sobrescreve executável em execução). O padrão é reconhecido e bem documentado pelo GO.

**Ponto Crítico**: Validar o SHA256 antes de substituir.

### 3. Windows Service

Para conseguir rodar como serviço no Windows, deve usar a biblioteca **golang.org/x/sys/windows/svc**, que é a biblioteca oficial do Go para essa finalidade, sem dependências externas.

A instalação nas máquinas de cada dev pode ser feita via script powershell que o time de infra executa um vez por máquina:

```powershell
$release = Invoke-RestMethod "https://api.github.com/repos/eubrunocase/Galileu/releases/latest"
$asset = $release.assets | Where-Object { $_.name -eq "galileu.exe" }
Invoke-WebRequest $asset.browser_download_url -OutFile "$installPath\galileu.exe"
```

Após isso, o serviço irá se auto-gerenciar.

### Configuração centralizada da equipe

Para garantir que todos estejam na mesma versão do **galileu.yml**, foi criado o arquivo **install-galileu.ps1** na raiz com as seguintes opções de instalação:

1. **.\install-galileu.ps1** — Instalação simples sem config centralizada do yml

2. **.\install-galileu.ps1 -ConfigRepoUrl "https://raw.githubusercontent.com/sua-org/galileu-configs/main/galileu.yml"** — Instalação com config centralizada da equipe (repositório público)

3. **.\install-galileu.ps1 -ConfigRepoUrl "https://raw.githubusercontent.com/sua-org/galileu-configs/main/galileu.yml" -ConfigRepoToken "ghp_seu_token_aqui"** — Instalação com config centralizada da equipe (repositório privado)

4 - **.\install-galileu.ps1 -Uninstall**: Desinstalação completa

### O que o script cobre

- **Segurança** — baixa o checksums.txt da release, valida o SHA256 do binário antes de instalar. Se o checksum não bater, aborta e descarta o arquivo.
- **Resiliência** — faz backup do binário anterior como galileu.exe.bak antes de substituir. Se a instalação falhar no meio, o serviço pode reverter.
- **Restart automático** — configura o Windows Service Manager para reiniciar o serviço automaticamente em caso de falha, com backoff progressivo (5s → 10s → 30s).
- **Idempotência** — pode ser executado múltiplas vezes sem problema. Se o serviço já existir, reconfigura. Se o binário já estiver na versão mais recente, pula o download.
- **Config flexível** — se não passar -ConfigRepoUrl, gera um galileu.yml padrão local. Se a URL falhar, cai no padrão sem travar a instalação.