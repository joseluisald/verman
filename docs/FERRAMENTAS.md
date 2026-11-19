# Ferramentas disponíveis

Windows x64. Todas usam os mesmos comandos:

```powershell
verman list bun --available
verman install bun latest
verman use bun latest
bun --version
```

Troque `bun` pelo nome abaixo. `use` baixa se necessário e salva a versão exata no projeto; `global` define o padrão. O instalador deve colocar `C:\Program Files\Verman\bin` no início do PATH. Reabra terminal/editor; se necessário, reinicie o PC.

| Nome no Verman | Comandos | Observação |
| --- | --- | --- |
| node | node, npm, npx | Aceita `latest` e `lts`. |
| php | php | PHP NTS; pode exigir Visual C++ Redistributable x64. |
| python | python, pip | Inclui venv; prefira `python -m pip`. |
| bun | bun, bunx | Distribuição baseline x64. |
| go | go, gofmt | Usa a toolchain selecionada (`GOTOOLCHAIN=local`). |
| deno | deno | Runtime JavaScript/TypeScript. |
| git | git | MinGit portátil, sem Git Bash ou interface gráfica. |
| rust | rustc, rustdoc, cargo, rustup | Toolchain MSVC mínima; diretórios cargo/rustup próprios. |
| composer | composer | Selecione PHP antes: `verman use php latest`. |
| yarn | yarn, yarnpkg | Yarn moderno; selecione Node antes. |
| pnpm | pnpm | Executável nativo; os projetos podem precisar de Node. |
| vcpkg | vcpkg | Ferramenta e bundle oficiais; builds exigem Git/CMake/compiladores. |
| chocolatey | choco, chocolatey | CLI portátil isolada; não importa o Chocolatey existente. |

Para instalar Yarn/Composer:

```powershell
verman use node lts
verman use yarn latest
yarn --version
verman use php latest
verman use composer latest
composer --version
```

Yarn respeita o `packageManager` do `package.json`: se a versão selecionada divergir, o Verman pede a versão correta. Não altera o arquivo do projeto. Yarn Classic 1 não faz parte deste provedor.

Rust instala pelo rustup oficial com `--no-modify-path`. Não altera o rustup/Cargo já existente. Compilar executáveis MSVC pode exigir os **Visual Studio Build Tools com C++**, instalados separadamente.

vcpkg usa versões da ferramenta, por exemplo `2026.10.29` corresponde à tag `2026-10-29`. O bundle consulta o registro Git de portas quando necessário; instalar bibliotecas exige rede e ferramentas de build. Não instala compiladores automaticamente.

Chocolatey não executa seu instalador global. Pacotes solicitados posteriormente com `choco install` podem exigir administrador, instalar aplicações no sistema e ter requisitos de licença próprios. O Verman não eleva privilégios automaticamente. Remova/migre esses pacotes antes de remover a versão do Chocolatey; remover o Verman preserva suas instalações.

`list <ferramenta> --available` é a referência de versões instaláveis. Catálogos GitHub consultam até 30 releases com digest publicado; não cobrem todo o histórico. `latest` depende da origem. Falhas de rede podem usar cache com aviso; instalar/reparar exige rede. Detalhes: [Provedores](PROVEDORES.md).
