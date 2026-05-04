# Galileu — Proxy de Segurança e Governança para LLMs
> Suporta: macOS (Apple Silicon & Intel) · Windows · Linux

**Galileu** é uma ferramenta de segurança e governança de dados voltada para o monitoramento e sanitização de informações enviadas a provedores de Inteligência Artificial (LLMs). O projeto adota uma arquitetura de **Proxy Reverso MITM (Man-in-the-Middle)**, actuando como camada inteligente entre a sua ferramenta de desenvolvimento e os servidores das LLMs.

---

## Arquitectura do Sistema

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

O Galileu utiliza certificados no formato padronizado: `galileu-ca.pem` e `galileu-ca-key.pem`.

### macOS
O Galileu tentará instalar o certificado automaticamente no Keychain do sistema (será solicitada a senha de administrador na primeira execução). Caso prefira instalar manualmente:

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain galileu-ca.pem
```

### Windows
O Galileu instala automaticamente o certificado CA no repositório de certificados do sistema ao arrancar como **Administrador**. Basta executar `galileu.exe` com privilégios administrativos.

### Linux
No Linux, a instalação é manual. Após compilar, execute:

```bash
sudo cp galileu-ca.pem /usr/local/share/ca-certificates/galileu.crt
sudo update-ca-certificates
```

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
| **Sistema Operativo** | macOS (Apple Silicon & Intel), Windows 10/11, Linux (amd64) |
| **Go** | Versão 1.23 ou superior (necessário apenas para compilação) |
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
| Slack / Discord | `xox[baprs]-...` | `xoxb-123456...` |
| AWS Access Key | `AKIA...` | `AKIAIOSFODNN7...` |
| AWS Secret Key | `wJalr...` | `wJalrXUtnFEM...` |
| Bearer Token | `bearer ...` | `bearer abcdef123456...` |
| Generic API Key | `api_key...` | `api_keyABC123...` |

Todos os dados sensíveis detectados são substituídos por `[REDACTED_BY_GALILEU]`.

---

## Registos de Auditoria Expandidos

O ficheiro `galileu_audit.log` contém um registo JSON detalhado de cada requisição interceptada.

(Consulte a documentação no repositório para o schema completo dos campos de auditoria.)

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
- **Nunca** efectue commit dos ficheiros `.pem` para o repositório — confirme que o `.gitignore` está actualizado.
- O certificado CA é válido por **10 anos** e utiliza chave **RSA 4096-bit**.
- O proxy actua exclusivamente sobre as ferramentas que configurarem explicitamente a porta **9000**.

---

## Licença

Este projecto é para fins educacionais e de segurança interna.
Todos os direitos são reservados ao programador **Bruno Dantas de Oliveira Cazé** — [github.com/eubrunocase/Galileu](https://github.com/eubrunocase/Galileu)
