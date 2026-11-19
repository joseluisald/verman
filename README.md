# Verman

Gerencie Node.js, PHP, Python, Bun, Go, Deno, Git, Rust, Composer, Yarn, pnpm, vcpkg e Chocolatey no Windows x64 pela CLI ou pelo desktop.

## Instalar

1. Abra `dist/Verman-Setup-0.5.4-windows-amd64.exe`.
2. Deixe marcada a opção de adicionar a pasta `bin` ao PATH do usuário.
3. Feche e abra novamente o terminal e o editor.

A CLI e o desktop são instalados por padrão em `C:\verman`, sem solicitação de elevação. A entrada adicionada ao PATH do usuário é:

```text
C:\verman\bin
```

Se escolher outra pasta de instalação, use a subpasta `bin` dela. O instalador faz essa configuração quando a opção está marcada.

Novas versões ficam em `C:\verman\tools\<ferramenta>\<versão>` por padrão. Se escolher outro destino, a subpasta `tools` acompanha a instalação. As versões que já estão em `%USERPROFILE%\.verman\installs` continuam disponíveis no mesmo local.

Confira em um terminal novo:

```powershell
verman --version
```

## Usar

Dentro da pasta do projeto:

```powershell
verman use php latest
php -v
verman use node lts
node --version
verman use python latest
python --version
```

Ou abra **Verman** no menu Iniciar: **Selecionar projeto → Ferramentas → Instalar → Usar no projeto**.

Para consultar versões: `verman list bun --available`. Troque `bun` pelo nome da ferramenta. Composer exige PHP selecionado; Yarn exige Node. Se aparecer erro de certificado no PHP, veja [como configurar o cacert.pem](docs/FERRAMENTAS.md#php-com-erro-de-certificado-ssltls). [Ferramentas e dependências](docs/FERRAMENTAS.md).

`install` baixa uma versão. `use` salva a escolha do projeto. `global` define o padrão para projetos sem seleção própria. Os shims consultam a seleção a cada execução.

## Se aparecer outra versão

Confira se `C:\verman\bin` está no PATH do usuário e reabra o terminal e o editor. Entradas do PATH do sistema podem ter prioridade. Use `verman exec -- php -v` para rodar o PHP selecionado. Se ainda não funcionar, **reinicie o PC** e teste novamente. Se persistir, execute:

```powershell
verman doctor php
Get-Command php -All
```

Para configurar o PATH depois, execute novamente o instalador e marque essa opção. Instalações externas são preservadas.

## Gerar o instalador

Tenha Go (versão do `go.mod`), Node/npm e Inno Setup. Na raiz:

```powershell
.\scripts\build-windows.ps1 -Version 0.5.4
```

O script gera interface, CLI, desktop, documentação e instalador em `dist`. O build sozinho não altera o PATH. Passo a passo: [COMO_GERAR_INSTALADOR.txt](COMO_GERAR_INSTALADOR.txt).

## Documentação

- [Primeiros passos](docs/PRIMEIROS_PASSOS.md)
- [Suporte e offline](docs/SUPORTE.md)
- [Desktop e build](docs/DESKTOP.md)
- [Comandos JSON e códigos de saída](docs/CLI_JSON.md)
- [Provedores, histórico e plataformas](docs/PROVEDORES.md)
- [Roadmap e validações pendentes](ROADMAP_PRODUTO.md)

O desktop precisa de WebView2. PHP pode precisar de Visual C++ Redistributable x64. Desative venv/Conda antes de trocar Python. Desinstalar o aplicativo preserva runtimes e projetos.

O pacote local é para revisão. Assinatura, validação em Windows limpo e licença comercial do compilador Inno continuam pendentes.

## Landing page

O site e os guias estão em [verman-frontend-lp](verman-frontend-lp/README.md), com Astro, Tailwind CSS, Alpine.js e capturas reais do desktop. Na pasta, execute `npm ci --ignore-scripts` e `npm run dev`. Para gerar o site estático, use `npm run build`. O instalador da LP é versionado em `verman-frontend-lp/assets/installers/`. Nenhuma hospedagem é publicada automaticamente.
