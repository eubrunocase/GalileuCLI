# Contribuindo com o Galileu

Obrigado por considerar contribuir com o Galileu! Este guia irá ajudá-lo a começar.

## Pré-requisitos
- Go 1.25.0 ou superior
- Make (para compilação)
- Git

## Configuração de Desenvolvimento
1. Faça um fork do [repositório GalileuCLI](https://github.com/eubrunocase/GalileuCLI)
2. Clone o seu fork:
   ```bash
   git clone https://github.com/SEU_USUARIO/GalileuCLI.git
   cd GalileuCLI/application
   ```
3. Instale as dependências:
   ```bash
   go mod download
   ```
4. Compile o projeto para verificar a configuração:
   ```bash
   make build-mac-arm  # Ou a plataforma alvo desejada
   ```

## Estratégia de Branches
- `main`: Branch de produção estável. Apenas hotfixes críticos e commits de lançamento são mergeados aqui.
- `dev`: Branch de desenvolvimento ativo. Todos os PRs de funcionalidades devem ter como alvo `dev`.

## Diretrizes de Código
- Siga o estilo de código existente no projeto.
- **Não adicione comentários de código a menos que seja explicitamente solicitado.**
- Escreva código Go idiomático, reutilize utilitários e padrões existentes do projeto.
- Minimize alterações de código desnecessárias; mantenha commits focados em uma única tarefa.

## Testes
- Todas as novas funcionalidades devem incluir testes unitários e/ou de integração.
- Certifique-se de que todos os testes existentes passem antes de submeter um PR:
  ```bash
  go test ./...
  ```
- Para alterações nos pacotes `analyzer` ou `guardian`, execute benchmarks de performance:
  ```bash
  go test -bench=. -benchmem ./internal/guardian/...
  ```

## Processo de Pull Request
1. Crie uma nova branch a partir de `dev` para a sua funcionalidade ou correção.
2. Faça as suas alterações, seguindo as diretrizes de código.
3. Faça commit das suas alterações com mensagens claras e concisas.
4. Envie a sua branch para o seu fork.
5. Abra um Pull Request (PR) para a branch `dev` do repositório principal.
6. Preencha a descrição do PR com:
   - Objetivo da alteração
   - Issues relacionadas (se houver)
   - Testes realizados
7. Aguarde a revisão de um mantenedor. Responda prontamente a qualquer feedback.

## Contribuições de Segurança
Se você estiver submetendo uma correção de segurança, por favor:
1. Siga a [Política de Segurança](SECURITY.md) e reporte a vulnerabilidade privadamente primeiro.
2. Coordene com os mantenedores antes de submeter um PR público.
3. Inclua detalhes da vulnerabilidade e da correção na descrição do PR.

## Licença
Ao contribuir com o Galileu, você concorda que as suas contribuições serão licenciadas sob a Apache License 2.0, a mesma licença do projeto.
