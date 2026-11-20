# Provedores, disponibilidade e plataformas

## Matriz desta entrega

| Distribuição | Sistema/arquitetura habilitados | Formato/variante | Origem de confiança |
| --- | --- | --- | --- |
| Node.js oficial | Windows/amd64 | ZIP win-x64; npm/npx | nodejs.org, SHASUMS256.txt por HTTPS |
| PHP oficial | Windows/amd64 | ZIP NTS x64, vc15/vs16/vs17 conforme catálogo | downloads.php.net, releases.json ou sha256sum.txt oficial |
| CPython via Astral | Windows/amd64 | python-build-standalone install_only, tar.gz | GitHub do Astral, digest SHA-256 do asset por tag |
| Bun | Windows/amd64 | ZIP baseline | GitHub oven-sh/bun, digest SHA-256 |
| Go | Windows/amd64 | ZIP oficial | go.dev/dl, SHA-256 do índice oficial |
| .NET SDK | Windows/amd64 | ZIP win-x64, SDKs estáveis a partir do .NET 5 | builds.dotnet.microsoft.com, SHA-512 dos metadados oficiais |
| Deno | Windows/amd64 | ZIP MSVC | GitHub denoland/deno, digest SHA-256 |
| Git | Windows/amd64 | MinGit ZIP | GitHub git-for-windows/git, digest SHA-256 |
| Rust | Windows/amd64 | rustup-init + toolchain MSVC mínima | static.rust-lang.org, SHA-256 do bootstrap e verificação dos componentes pelo rustup |
| Composer | Windows/amd64 | PHAR executado pelo PHP selecionado | getcomposer.org, sidecar SHA-256 por versão |
| Yarn moderno | Windows/amd64 | @yarnpkg/cli-dist TGZ | registry.npmjs.org, SRI SHA-512 por versão |
| pnpm | Windows/amd64 | ZIP nativo | GitHub pnpm/pnpm, digest SHA-256 |
| vcpkg-tool | Windows/amd64 | EXE + standalone bundle TAR.GZ | GitHub microsoft/vcpkg-tool, SHA-256 dos dois assets |
| Chocolatey | Windows/amd64 | NUPKG portátil, sem executar install.ps1 | GitHub chocolatey/choco, digest SHA-256 |

Os novos provedores preservam URLs canônicas e reconsultam o hash na origem antes da instalação. Yarn autentica o download por SHA-512/SRI; só então registra também o SHA-256 local no manifesto. `publisher_sha512_https` identifica esse caso. Provedores GitHub consultam as 30 releases mais recentes e ignoram assets sem digest; versões exatas fora do catálogo tentam sua tag oficial, sem prometer cobertura histórica completa. Composer lista a versão estável atual e aceita versões exatas com checksum publicado; Yarn lista versões estáveis do pacote moderno. Git usa `X.Y.Z` com revisão Windows em `build`; vcpkg normaliza datas como `2026.10.29`. Go normaliza versões antigas sem patch como `1.20.0` e ignora pacotes antigos sem SHA-256. Instruções e dependências: [Ferramentas](FERRAMENTAS.md).

O catálogo indica compatibilidade do artefato; não significa que cada versão listada tenha sido executada em todos os Windows. Testes oficiais desta entrega executaram Node 22.11.0/22.12.0, PHP 8.5.12/8.4.27 revisão -1, Python 3.15.2/3.14.10 build 20261110 e Python histórico 3.15.1 build 20261021. PHP exige o redistribuível VC++ x64 compatível, sem instalação global automática. PHP TS/mod_php, x86 e Python embeddable não estão habilitados.

O SDK .NET usa as versões estáveis dos catálogos por canal da Microsoft. O Verman revalida o SHA-512 do ZIP na origem antes de instalar e registra também o SHA-256 local.

Linux/amd64 e macOS/arm64 têm distribuições upstream possíveis, mas exigem implementação de formatos/layouts, testes de execução, ativação, permissões e distribuição próprios. Windows/arm64 também requer pacotes e dependências equivalentes para cada provedor. Nenhuma dessas plataformas foi habilitada apenas porque o Go consegue compilar a CLI. A expansão permanece uma decisão futura baseada em testes reais.

## Catálogo atual e histórico

Validação real em 18/11/2026: `TestOfficialAdditionalToolsWorkflow` passou com Bun 1.4.4, Go 1.27.4, Deno 2.9.8, Git 2.56.1/windows.1, Rust 1.100.0, Composer 2.11.2, Yarn 4.18.1, pnpm 12.22.0, vcpkg-tool 2026-10-29 e Chocolatey 2.7.5. O teste baixa, instala, seleciona, verifica hashes e consulta a versão por exec e shims após publicar, com Node 22.12.0 e PHP como dependências em perfil temporário. Não instala pacotes Chocolatey/bibliotecas C++, não testa builds completos e não substitui Windows limpo. O ciclo Go nesta revisão demorou cerca de cinco minutos, incluindo verificação completa dos arquivos.

Para repetir: defina `VERMAN_INTEGRATION=1` e execute `go test ./cmd -run '^TestOfficialAdditionalToolsWorkflow$' -count=1 -v -timeout 30m`. Testes padrão continuam sem downloads públicos.

```powershell
verman list php --available --history
verman list python --available --history
verman list python --available --history --json
```

PHP atual vem de [releases.json](https://downloads.php.net/~windows/releases/releases.json). O histórico lê o [índice oficial de arquivos](https://downloads.php.net/~windows/releases/archives/) e cruza nomes NTS x64 reconhecidos com [sha256sum.txt](https://downloads.php.net/~windows/releases/sha256sum.txt). Não calcula um hash do download e o trata como uma publicação independente. Um arquivo histórico sem hash publicado é mostrado com `installable=false`, `trust=unverified` e motivo; instalação é recusada. Na consulta real desta entrega, foram encontradas 180 versões, das quais 173 entradas de arquivo estavam bloqueadas por ausência de checksum. Esse total é uma evidência datada, não um contrato estável.

O índice atual tem prioridade. Para cada versão histórica, prefere-se uma revisão com checksum e, depois, a revisão numérica mais alta. Revisões `-1`, `-2` e o compilador são registradas em `build`/`variant`; o TOML continua usando `X.Y.Z`. Arquivos com nome fora do contrato, TS, arquitetura incorreta ou caminho externo não são anunciados como pacotes compatíveis.

`installed_build` e `different_build` no JSON permitem comparar a instalação existente ao pacote do catálogo. `install` é idempotente por versão instalada válida; use `repair` para obter uma revisão diferente da mesma versão. O catálogo pode mudar entre consulta e instalação; a reconsulta na origem decide qual revisão pode ser publicada.

Python atual usa o release estável mais recente de [python-build-standalone](https://github.com/astral-sh/python-build-standalone/releases). O histórico consulta até **30 releases**, em três páginas de dez, usando a [API de releases do GitHub](https://docs.github.com/en/rest/releases/releases). Ignora drafts/pré-releases e assets históricos sem digest; mantém o build mais recente por versão. Não é um inventário de todos os patches históricos. A tag/build é fixada no artefato e reconsultada antes de instalar. O cache não pode fornecer sozinho a autorização de checksum.

Ao instalar uma versão exata ausente do catálogo atual, PHP/Python consultam automaticamente o mesmo histórico limitado. Se a versão desaparecer, não houver digest, a API falhar ou o formato mudar, a operação falha com contexto. Uma consulta histórica interrompida não publica um catálogo parcial como completo; pode usar o cache anterior com aviso.

## Suporte, EOL e confiança

`distribution` na saída JSON informa `provider`, `trust`, `installable`, `support`, EOL quando conhecido, `policy_source` e `policy_reviewed`. A política foi revisada em **18/11/2026**, usando o [cronograma Node](https://github.com/nodejs/Release/blob/main/schedule.json), a [tabela PHP](https://www.php.net/supported-versions.php) e o [guia Python](https://devguide.python.org/versions/). É um snapshot mantido no código, não uma consulta de segurança em tempo real. Ramos sem informação ficam `unknown`; datas previstas podem mudar. EOL Python é informado com precisão de mês; a mudança de estado ocorre no mês seguinte ao indicado.

`supported_branch` descreve o ramo, sem afirmar que um patch antigo está atualizado ou livre de vulnerabilidades. EOL é mostrado e gera aviso ao instalar explicitamente; disponibilidade não significa recomendação. Datas de suporte não alteram hashes nem autorizam fontes adicionais.

`publisher_sha256_https` significa conferir o digest publicado pelo distribuidor através de HTTPS. CPython vem do build do Astral; isso não é um instalador oficial publicado pela python.org. Não há verificação de assinatura PGP/Authenticode dos pacotes nesta entrega. Depois da instalação, `verify` compara arquivos imutáveis com o manifesto local, conforme o README.

## Robustez e evidências

Testes locais cobrem catálogos malformados/alterados, URLs adulteradas, variantes/arquiteturas, revisões, builds fixados, digest ausente, checksum inválido, HTTP indisponível, API limitada, cache vencido e offline sem chamadas de rede. Testes oficiais são opt-in e isolam o diretório pessoal:

```powershell
$env:VERMAN_INTEGRATION = '1'
try {
    go test ./cmd -run '^TestOfficial.*Workflow$' -count=1 -v -timeout 15m
    go test ./internal/installer -run '^TestOfficialHistoricalCatalogs$' -count=1 -v -timeout 15m
} finally {
    Remove-Item Env:VERMAN_INTEGRATION -ErrorAction SilentlyContinue
}
```
