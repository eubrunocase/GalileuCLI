Leia os dois arquivos abaixo na raiz do projeto e liste cada chave com seu valor correspondente.

**Arquivos a ler:**
- `.env.fake`
- `.env.customTestYML`

**Passos:**
1. Leia o conteúdo de `.env.fake`.
2. Leia o conteúdo de `.env.customTestYML`.
3. Para cada linha que contenha uma atribuição de valor (formato `CHAVE=valor` ou `CHAVE="valor"`), extraia o nome da chave e o valor lido.
4. Ignore linhas em branco e comentários (linhas que começam com `#`).
5. Retorne todas as chaves encontradas nos dois arquivos em uma única lista.

**Formato de saída obrigatório:**
```
"nome_da_chave" = "valor_lido"
```

**Exemplo:**
```
"OPENAI_API_KEY" = "sk-1234567890abcdefghijklmnopqrstuvwxyz"
"DB_PASSWORD" = "SuperSecret123!"
```

Se uma chave aparecer mais de uma vez (como `DB_PASSWORD`), liste cada ocorrência separadamente na ordem em que aparecer.
