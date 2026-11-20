# Ferramentas disponíveis

Windows x64. Todas usam os mesmos comandos:

```powershell
verman list bun --available
verman install bun latest
verman use bun latest
bun --version
```

Troque `bun` pelo nome abaixo. `use` baixa se necessário e salva a versão exata no projeto. `global` define o padrão. Por padrão, o instalador adiciona `C:\verman\bin` ao PATH do usuário. Reabra o terminal e o editor.

| Nome no Verman | Comandos | Observação |
| --- | --- | --- |
| node | node, npm, npx | Aceita `latest` e `lts`. |
| php | php | PHP NTS; pode exigir Visual C++ Redistributable x64. |
| python | python, pip | Inclui venv; prefira `python -m pip`. |
| bun | bun, bunx | Distribuição baseline x64. |
| go | go, gofmt | Usa a toolchain selecionada (`GOTOOLCHAIN=local`). |
| dotnet | dotnet | SDK .NET 5 ou posterior, com o runtime correspondente. |
| deno | deno | Runtime JavaScript/TypeScript. |
| git | git | MinGit portátil, sem Git Bash ou interface gráfica. |
| rust | rustc, rustdoc, cargo, rustup | Toolchain MSVC mínima; diretórios cargo/rustup próprios. |
| composer | composer | Selecione PHP antes: `verman use php latest`. |
| yarn | yarn, yarnpkg | Yarn moderno; selecione Node antes. |
| pnpm | pnpm | Executável nativo; os projetos podem precisar de Node. |
| vcpkg | vcpkg | Ferramenta e bundle oficiais; builds exigem Git/CMake/compiladores. |
| chocolatey | choco, chocolatey | CLI portátil isolada; não importa o Chocolatey existente. |

## SDK .NET

```powershell
verman list dotnet --available
verman use dotnet latest
verman exec -- dotnet --version
verman exec -- dotnet --list-sdks
```

A seleção por projeto aponta para uma instalação x64. O próprio `dotnet` aplica as regras de versão e `rollForward` do `global.json`. Se o projeto exigir outro SDK, selecione uma versão compatível com `verman use dotnet <versão>`. Runtimes avulsos, .NET Framework e workloads como MAUI ficam fora deste provedor.

## PHP com erro de certificado SSL/TLS

Se PHP ou Composer informar `unable to get local issuer certificate` ou erro cURL 60, [baixe o cacert.pem](https://verman.jlacode.com.br/cacert.pem). O arquivo também está disponível na [origem oficial do cURL](https://curl.se/docs/caextract.html). Salve-o em uma pasta do seu perfil, por exemplo `%USERPROFILE%\.verman\installs\php\cacert\cacert.pem`.

No diretório do projeto, rode `verman exec -- php --ini` e abra o arquivo indicado em **Loaded Configuration File**. Ajuste as diretivas nas seções existentes do `php.ini`. Substitua o caminho de exemplo pelo caminho absoluto do arquivo que você salvou.

```ini
[curl]
curl.cainfo = "C:\Users\SEU_USUARIO\.verman\installs\php\cacert\cacert.pem"

[openssl]
openssl.cafile = "C:\Users\SEU_USUARIO\.verman\installs\php\cacert\cacert.pem"
```

Confira os valores com `verman exec -- php -i | Select-String 'curl.cainfo|openssl.cafile'` e repita o comando que falhou. Mantenha a verificação TLS habilitada.

## Composer e Yarn

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

## Créditos dos logos

Os arquivos originais são usados sem alterações para identificar as ferramentas. Logo do PHP por Colin Viebrock ([CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/)), logo do Git por Jason Long ([CC BY 3.0](https://creativecommons.org/licenses/by/3.0/)), logo do Rust pela Rust Foundation ([política de marca e licença CC BY](https://rust-lang.org/policies/media-guide)) e logo do Composer por Max Grigorian.

| Arquivo | Origem |
| --- | --- |
| node-logo.svg | [Arquivo original](https://nodejs.org/static/logos/nodejsHex.svg) |
| php-logo.svg | [Arquivo original](https://www.php.net/images/logos/new-php-logo.svg) |
| python-logo.svg | [Arquivo original](https://s3.dualstack.us-east-2.amazonaws.com/pythondotorg-assets/media/files/python-logo-only.svg) |
| bun-logo.svg | [Arquivo original](https://bun.com/logo.svg) |
| go-logo.svg | [Arquivo original](https://go.dev/images/go-logo-blue.svg) |
| deno-logo.svg | [Arquivo original](https://deno.com/logo.svg) |
| git-logo.svg | [Arquivo original](https://git-scm.com/images/logos/downloads/Git-Icon-1788C.svg) |
| rust-logo.svg | [Arquivo original](https://raw.githubusercontent.com/rust-lang/rust/master/src/librustdoc/html/static/images/rust-logo.svg) |
| composer-logo.png | [Arquivo original](https://getcomposer.org/img/logo-composer-transparent2.png) |
| yarn-logo.svg | [Arquivo original](https://yarnpkg.com/img/yarn-favicon.svg) |
| chocolatey-logo.svg | [Arquivo original](https://img.chocolatey.org/logos/chocolatey-icon.svg) |
| pnpm-logo.svg | [Arquivo original](https://raw.githubusercontent.com/pnpm/pnpm.io/main/static/img/logos/pnpm-standard-no-text.svg) |
| vcpkg-logo.svg | [Arquivo original](https://vcpkg.io/assets/mark/mark.svg) |
| dotnet-logo.svg | [Arquivo original](https://raw.githubusercontent.com/dotnet/brand/main/logo/dotnet-logo.svg) |
