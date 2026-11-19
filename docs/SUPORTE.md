# Uso, suporte e diagnóstico

O alvo validado é Windows/amd64, usando PowerShell e Git Bash/sh. Runtimes nativos de Linux/macOS/ARM64 não estão habilitados. A matriz e as origens estão em [PROVEDORES.md](PROVEDORES.md); saídas e códigos da CLI em [CLI_JSON.md](CLI_JSON.md).

## Primeiro uso

1. Extraia o pacote ou compile o executável com o Go declarado em `go.mod`.
2. Consulte `verman list php --available` e selecione com `verman use php latest` dentro do projeto. `use` instala se necessário e grava a versão exata.
3. Configure os shims uma vez: `scripts/setup-path.ps1 -Scope User`. Se o script identificar runtimes no PATH do sistema, use `-Scope Machine` em PowerShell administrador. Esse escopo instala em Program Files e salva um backup antes de mudar o PATH.
4. Reabra terminais e editor uma vez. Depois, `php -v` consulta a seleção do diretório atual a cada execução. `verman doctor php` ajuda a verificar a precedência.

`verman setup` sozinho instala os shims, sem alterar o PATH. `use` nunca reescreve o PATH persistente. Sem shims, `verman exec -- php -v` executa com o ambiente do projeto; `env` e `init` continuam alternativas para ativar a sessão. Aliases/funções têm precedência própria: no PowerShell confira `Get-Command php -All`. Não remova instalações externas para resolver uma divergência.

Antes de trocar Python, desative venv/Conda. O Verman seleciona a versão base; não migra um venv já criado. Crie outro ambiente com o Python desejado e reinstale as dependências do projeto. `doctor` informa `VIRTUAL_ENV`/`CONDA_PREFIX` e recomenda essa ação; a CLI não desativa ambientes automaticamente.

## Offline e indisponibilidade

`list` local, `doctor`, `verify`, `recover`, `exec` e os shims não consultam rede. `use`/`global` com versão exata já instalada também funcionam offline. `--offline` bloqueia a rede nos serviços de instalação/resolução e permite ler catálogos previamente armazenados, mesmo antigos, com aviso. `--refresh` é incompatível com `--offline`.

O cache remoto vale por uma hora e é separado por linguagem/plataforma e por consulta atual/histórica. Falha de atualização permite usar um cache válido anterior, com data e aviso. Um cache antigo não prova que o arquivo continua disponível. Instalar/reparar exige reconsultar checksums na origem e baixar o pacote; não há cache offline de pacotes nesta entrega. `latest`/`lts` podem ser resolvidos por um catálogo local, mas só selecionam offline uma versão que já esteja instalada.

Falhas HTTP/limites de API têm código e orientação; não há repetição automática que pressione uma API limitada. Aguarde o prazo indicado ou use o cache offline. Não existe configuração de espelho/URL arbitrária para contornar a origem de confiança.

## Falhas e recuperação

- Divergência de versão: `verman doctor php`, depois confira shims/alias/PATH. `verman exec -- php -v` ajuda a isolar o problema.
- Arquivos alterados: `verman verify php VERSAO`; para reinstalar, `verman repair php VERSAO`. O reparo preserva ini/conf.d e pacotes/scripts Python conforme os limites do README.
- Operação interrompida: `verman recover php VERSAO`. Feche processos que usam o runtime antes de reparar/remover. Não apague os arquivos `.lck`.
- Configuração inválida: corrija o TOML indicado; a CLI não ignora silenciosamente esse arquivo.
- Histórico bloqueado: falta de checksum oficial não é resolvida aceitando o hash local do arquivo. Escolha uma entrada verificada.

## Logs e relato de problema

Não há telemetria nem upload automático. O Verman não grava um log persistente por padrão. Para coletar uma operação explicitamente:

```powershell
verman install php VERSAO --json --progress always 1>resultado.json 2>operacao.log
$LASTEXITCODE
verman support --json > suporte.json
```

Revise os arquivos antes de compartilhá-los. Saídas de comandos normais podem conter diretórios pessoais/projetos e URLs; `support` omite esses caminhos, o PATH e os valores de variáveis. Se o TOML estiver quebrado, `support` pode falhar com um erro que inclui o caminho a corrigir; revise também essa saída.

Ao abrir um relato no canal do projeto, inclua versão do Verman, Windows/arquitetura, shell, comando completo (sem segredos), saída/código, resultado esperado, passos mínimos e `suporte.json`. Informe se existe alias, venv/Conda ou outra instalação da linguagem. Não envie tokens, `.env`, PATH completo nem arquivos privados do projeto. Nenhum endereço de suporte público foi presumido nesta entrega.
