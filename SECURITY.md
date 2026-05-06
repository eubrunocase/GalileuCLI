# Política de Segurança

## Reporte de Vulnerabilidades

Levamos a segurança do Galileu a sério. Se você descobrir uma vulnerabilidade de segurança no Galileu, por favor reporte de forma responsável.

**⚠️ NÃO reporte vulnerabilidades de segurança através de issues públicas do GitHub, discussões ou outros canais públicos.**

### Método Preferencial de Reporte
Utilize a funcionalidade [Reporte Privado de Vulnerabilidades](https://github.com/eubrunocase/GalileuCLI/security/advisories/new) do GitHub. Isso garante que seu reporte seja visível apenas para os mantenedores.

### O que Incluir
Por favor, forneça o máximo de detalhes possível:
- Descrição clara da vulnerabilidade
- Instruções passo a passo para reproduzir o problema
- Impacto e severidade estimados
- Quaisquer correções ou mitigações potenciais (opcional)

### Cronograma de Resposta
- **Confirmação**: Confirmaremos o recebimento do seu reporte em até 48 horas.
- **Avaliação Inicial**: Forneceremos uma avaliação inicial em até 7 dias úteis.
- **Resolução**: Trabalharemos com você para validar, corrigir e testar a vulnerabilidade. Notificaremos você quando uma correção estiver pronta.
- **Divulgação**: Coordenaremos a divulgação pública da vulnerabilidade após o lançamento de uma correção. Você receberá crédito pela descoberta, a menos que deseje permanecer anônimo.

## Escopo

### Dentro do Escopo
- Implementação do proxy MITM do Galileu
- Lógica de análise e sanitização de dados
- Geração de CA e gerenciamento de certificados
- Sistema de registro de auditoria
- Manipulação de configuração (`galileu.yml`)
- Todo o código nos diretórios `internal/` e `cmd/`

### Fora do Escopo
- Dependências de terceiros (reporte vulnerabilidades aos respectivos mantenedores)
- Problemas em builds não oficiais ou modificadas do Galileu
- Ataques de engenharia social contra mantenedores ou usuários
- Segurança física das máquinas dos usuários

## Reconhecimento
Com o seu consentimento, creditaremos a sua descoberta em:
- Notas de lançamento da versão corrigida
- Uma seção dedicada a agradecimentos de segurança na documentação
