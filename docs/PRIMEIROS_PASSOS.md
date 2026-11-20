# Começar a usar

## 1. Instale e configure o PATH

Abra o instalador e mantenha marcada a opção de adicionar a pasta `bin` ao PATH do usuário. Na instalação padrão, ela adiciona:

```text
C:\verman\bin
```

O instalador usa `C:\verman` por padrão e pode gravar onde a conta tiver acesso. Se escolher outra pasta, use a subpasta `bin` dela. Entradas no PATH do sistema podem preceder os shims. Nesse caso, `verman exec -- php -v` usa a versão selecionada.

As ferramentas baixadas ficam em `C:\verman\tools\<ferramenta>\<versão>` na instalação padrão. A pasta `tools` acompanha o destino escolhido no instalador. Versões já instaladas em `%USERPROFILE%\.verman\installs` continuam disponíveis nesse local.

**Feche e abra novamente o terminal e o editor.** Confira:

```powershell
verman --version
```

## 2. Escolha as versões

No desktop: **Selecionar projeto → Ferramentas → Instalar → Usar no projeto**.

Ou, no terminal, dentro da pasta do projeto:

```powershell
verman use php latest
php -v
verman use node lts
node --version
verman use python latest
python --version
```

**Instalar** baixa uma versão. **Usar no projeto** salva a escolha dessa pasta. **Padrão global** vale quando o projeto não define uma versão.

## Se não funcionar

1. Confira se `C:\verman\bin` está no PATH do usuário.
2. Reabra o terminal e o editor. Se ainda não reconhecer os comandos ou mostrar a versão antiga, **reinicie o PC** e teste novamente.
3. Se persistir, execute `verman doctor php`. Para configurar o PATH depois, execute novamente o instalador e marque essa opção.

O desktop precisa de WebView2; o instalador avisa se estiver ausente. O PHP pode precisar do Visual C++ Redistributable x64. Para erros de certificado SSL/TLS, veja a seção de PHP em [Ferramentas e dependências](FERRAMENTAS.md). Para trocar o Python, desative o venv/Conda primeiro.

Não precisa instalar Go ou Node para usar o pacote pronto. Seus runtimes e projetos são preservados ao desinstalar o aplicativo.

Também disponíveis Bun, Go, .NET, Deno, Git, Rust, Composer, Yarn, pnpm, vcpkg e Chocolatey. Veja [Ferramentas e dependências](FERRAMENTAS.md).

Mais detalhes: [Suporte](SUPORTE.md) · [Desktop e build](DESKTOP.md) · [CLI/JSON](CLI_JSON.md).
