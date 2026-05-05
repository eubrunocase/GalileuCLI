# Galileu — Proxy de Segurança e Governança para LLMs
> Suporta: macOS (Apple Silicon & Intel) · Windows · Linux

**Galileu** é uma ferramenta de segurança e governança de dados voltada para o monitoramento e sanitização de informações enviadas a provedores de Inteligência Artificial (LLMs). O projeto adota uma arquitetura de **Proxy Reverso MITM (Man-in-the-Middle)**, atuando como camada inteligente entre a sua ferramenta de desenvolvimento e os servidores das LLMs.

---

## Demonstração

### Funcionamento em Tempo Real

![OpenCode com Galileu](media/opencode-galileu.gif)

*O GIF acima mostra o OpenCode tentando ler dados sensíveis de ficheiros `.env` — o Galileu intercepta e sanitiza automaticamente as informações antes de chegarem à LLM.*

### Terminal em Execução

![Terminal Galileu](media/terminal-galileu.png)

*Print do terminal durante a execução do Galileu, mostrando o proxy ativo e os registos de auditoria.*

---

## Arquitetura do Sistema

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Cliente   │───▶  │  Galileu    │───▶  │   LLM       │
│  (OpenCode) │◀───  │  Proxy MITM │◀───  │  Provider   │
└─────────────┘      └─────────────┘      └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Analyzer   │
                    │ (Sanitização)│
                    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Audit     │
                    │    Log      │
                    └─────────────┘
```

---

## Compilação

### macOS — Apple Silicon (M1/M2/M3)
```bash
make build-mac-arm
```

### macOS — Intel
```bash
make build-mac-intel
```

### Windows
```bash
make build-windows
```

### Linux
```bash
make build-linux
```

Ou compilar para todas as plataformas de uma vez:
```bash
make build-all
```

---

## Configuração do Certificado CA

> **⚠️ PONTO CRÍTICO DE SEGURANÇA**
>
> O Galileu gera um Certificado de Autoridade (CA) **localmente na sua máquina**. Este certificado é exclusivo para o seu ambiente e **nunca deve sair do seu computador**.

### Como Funciona

```
┌─────────────────────────────────────────────────────────────┐
│                    SUA MÁQUINA LOCAL                        │
│                                                             │
│  ┌──────────┐    ┌──────────────────┐    ┌──────────┐    │
│  │ Cliente  │───▶│  Galileu Proxy  │───▶│   LLM    │    │
│  │ (OpenCode)│◀───│  (localhost:9000)│◀───│ Provider │    │
│  └──────────┘    └──────────────────┘    └──────────┘    │
│                        │                                    │
│                        ▼                                    │
│              ┌──────────────────┐                          │
│              │   Certificado CA  │                          │
│              │  (Local apenas)   │                          │
│              │                   │                          │
│              │ galileu-ca.pem    │  ⚠️ NUNCA               │
│              │ galileu-ca-key.pem│  ⚠️ COMPARTILHAR       │
│              └──────────────────┘                          │
│                        │                                    │
│                        ▼                                    │
│              ┌──────────────────┐                          │
│              │  Keychain / Cert │                          │
│              │  Store do SO     │                          │
│              └──────────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

### O que acontece tecnicamente

1. O Galileu gera um par de chaves RSA 4096-bit **localmente** (`galileu-ca.pem` + `galileu-ca-key.pem`)
2. O certificado é instalado **apenas no seu sistema operacional** (Keychain no macOS, Cert Store no Windows, `/usr/local/share/ca-certificates/` no Linux)
3. Quando o proxy intercepta uma requisição HTTPS, ele apresenta um certificado assinado por esta CA
4. O seu SO confia no certificado porque a CA está instalada localmente
5. A chave privada (`galileu-ca-key.pem`) **nunca sai da sua máquina**

### Instalação por Sistema Operativo

#### macOS
O Galileu tentará instalar o certificado automaticamente no Keychain do sistema (será solicitada a senha de administrador na primeira execução). Caso prefira instalar manualmente:

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain galileu-ca.pem
```

#### Windows
O Galileu instala automaticamente o certificado CA no repositório de certificados do sistema ao arrancar como **Administrador**. Basta executar `galileu.exe` com privilégios administrativos.

#### Linux
No Linux, a instalação é manual. Após compilar, execute:

```bash
sudo cp galileu-ca.pem /usr/local/share/ca-certificates/galileu.crt
sudo update-ca-certificates
```

### ⚠️ Proteção dos Ficheiros `.pem`

O seu `.gitignore` **já está configurado** para impedir o commit acidental:

```gitignore
# Certificados — nunca versionar
*.pem
galileu-ca-key.pem
galileu-ca.pem
```

**NUNCA** remova estas linhas do `.gitignore`. A chave privada (`galileu-ca-key.pem`) é o que permite ao Galileu fazer o MITM — se ela for exposta, um atacante pode criar certificados falsificados em seu nome.

---

## Execução

### macOS / Linux
```bash
./galileu
./scripts/start.sh
```

### Windows
```bash
galileu.exe
scripts\start.bat
```

> **Nota:** Certifique-se de que o OpenCode (ou outra ferramenta) está configurado para usar o proxy na porta **9000**.

---

## Pré-requisitos

| Requisito | Detalhe |
|---|---|
| **Sistema Operacional** | macOS (Apple Silicon & Intel), Windows 10/11, Linux (amd64) |
| **Go** | Versão 1.25.0 ou superior (necessário apenas para compilação) |
| **Privilégios** | macOS: `sudo` na primeira execução; Windows: Administrador |

---

## Estrutura de Ficheiros

```
Galileu/
├── galileu                  # Executável (macOS/Linux)
├── galileu.exe              # Executável (Windows)
├── galileu-ca.pem           # Certificado CA gerado automaticamente
├── galileu-ca-key.pem       # Chave privada do CA (⚠️ NÃO submeter para o repositório)
├── galileu.yml              # Configuração do analyzer (não versionado)
├── galileu.yml.example      # Exemplo de configuração (versionado)
├── Makefile                 # Compilação multiplataforma
├── scripts/
│   ├── start.sh             # Script shell para iniciar o OpenCode com proxy (macOS/Linux)
│   └── start.bat            # Script batch para iniciar o OpenCode com proxy (Windows)
├── cmd/
│   └── sentinel/
│       └── main.go          # Ponto de entrada
├── internal/
│   ├── ca/                  # Geração e gestão do certificado CA
│   └── guardian/           # Proxy MITM, Analyzer, Audit, instalação de certificado por plataforma
└── galileu_audit.log        # Registo de auditoria (gerado automaticamente)
```

---

## Hosts Monitorizados

O Galileu intercepta requisições para os seguintes provedores:

| Provedor | Host |
|---|---|
| OpenCode | `opencode.ai` |
| OpenAI | `api.openai.com` |
| Anthropic | `api.anthropic.com` |
| Google AI | `generativelanguage.googleapis.com` |
| Cohere | `api.cohere.ai` |
| Mistral | `api.mistral.ai` |

---

## Detecção de Dados Sensíveis

O **Analyzer** detecta e sanitiza automaticamente os seguintes padrões:

| Tipo | Padrão | Exemplo |
|---|---|---|
| OpenAI API Key | `sk-...` | `sk-1234567890abcdef...` |
| OpenAI Project Key | `sk-proj-...` | `sk-proj-abc123...` |
| Anthropic API Key | `sk-ant-...` | `sk-ant-abc123...` |
| Google API Key | `AIzaSy...` | `AIzaSyABC123...` |
| GitHub Token | `ghp_...` | `ghp_abcdef123456...` |
| Slack Token | `xox[baprs]-...` | `xoxb-123456...` |
| Discord Token | `xox[baprs]-...` | `xoxb-123456...` |
| AWS Access Key | `AKIA...` | `AKIAIOSFODNN7...` |
| AWS Secret Key | `wJalr...` | `wJalrXUtnFEM...` |

Todos os dados sensíveis detectados são substituídos por `[REDACTED_BY_GALILEU]`.

---

## Registos de Auditoria Expandidos

O ficheiro `galileu_audit.log` contém um registo JSON detalhado de cada requisição interceptada, incluindo:

- **Identificação**: Timestamp, Request ID, Session ID, Machine ID
- **Requisição**: Host, Provider, Path, Method, Modelo de LLM
- **Detecção**: Padrões detectados, contagem, posições de redacção
- **Payload**: Contagem de mensagens, presença de system prompt, streaming
- **Performance**: Latência do proxy, duração da análise
- **Resposta**: Status code, tamanhos de request/response

(Consulte a documentação no repositório para o schema completo dos campos de auditoria.)

---

## Configuração (galileu.yml)

O Galileu suporta configuração via ficheiro `galileu.yml` para personalizar os padrões de detecção sem recompilar o código.

### Padrões Built-in

Todos os padrões embutidos podem ser ativados ou desativados individualmente:

```yaml
analyzer:
  built_in:
    openai_key:         true
    openai_project_key: true
    anthropic_key:      true
    google_key:         true
    github_token:       true
    slack_token:        true
    discord_token:      true
    aws_key:            true
```

### Padrões Customizados

Adicione os seus próprios padrões de dois tipos:

**Regex** — para padrões complexos:
```yaml
custom_patterns:
  - name: "JWT Token"
    type: regex
    pattern: 'eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+'
    label: "[JWT_REDACTED]"
    enabled: true
```

**Literal** — para strings exatas:
```yaml
custom_patterns:
  - name: "Projectos Confidenciais"
    type: literal
    values:
      - "Operação Phoenix"
      - "Projecto Stargate"
    label: "[CONFIDENTIAL_PROJECT_REDACTED]"
    enabled: true
```

> **Nota:** Se o ficheiro `galileu.yml` não existir, o Galileu usa todos os padrões built-in activados por omissão.

---

## Resolução de Problemas

### "Falha ao ler certificado CA"
Remova os ficheiros `galileu-ca.pem` e `galileu-ca-key.pem` e execute novamente. O certificado será regenerado automaticamente.

### Windows: "Privilégios de administrador necessários"
Execute o `galileu.exe` como Administrador (clique direito → "Executar como administrador").

### Linux: Erro de certificado SSL/TLS
Certifique-se de que instalou o certificado conforme as instruções na secção "Configuração do Certificado CA".

---

## Segurança

- A chave privada (`galileu-ca-key.pem`) é gerada localmente e **nunca** sai da sua máquina.
- **Nunca** efetue commit dos ficheiros `.pem` para o repositório — confirme que o `.gitignore` está atualizado.
- O certificado CA é válido por **10 anos** e utiliza chave **RSA 4096-bit**.
- O proxy atua exclusivamente sobre as ferramentas que configurarem explicitamente a porta **9000**.

---

## Licença

Este projeto está licenciado sob a **Apache License, Version 2.0**.

Copyright © 2026 **Bruno Dantas de Oliveira Casé**

Ver ficheiro [LICENSE](LICENSE) para mais detalhes.
