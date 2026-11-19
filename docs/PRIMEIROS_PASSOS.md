# Começar a usar

## 1. Instale e configure o PATH

Abra o instalador e mantenha marcada a opção de adicionar a pasta `bin` ao PATH do sistema. Na instalação padrão, ela adiciona:

```text
C:\Program Files\Verman\bin
```

Essa pasta deve ficar **antes dos caminhos de outros PHP, Node ou Python**. Se instalar em outra pasta, use a subpasta `bin` dela.

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

1. Confira se `C:\Program Files\Verman\bin` está no início do PATH do sistema.
2. Reabra o terminal e o editor. Se ainda não reconhecer os comandos ou mostrar a versão antiga, **reinicie o PC** e teste novamente.
3. Se persistir, execute `verman doctor php`. Para configurar o PATH depois, execute novamente o instalador e marque essa opção.

O desktop precisa de WebView2; o instalador avisa se estiver ausente. O PHP pode precisar do Visual C++ Redistributable x64. Para trocar o Python, desative o venv/Conda primeiro.

Não precisa instalar Go ou Node para usar o pacote pronto. Seus runtimes e projetos são preservados ao desinstalar o aplicativo.

Também disponíveis: Bun, Go, Deno, Git, Rust, Composer, Yarn, pnpm, vcpkg e Chocolatey. Veja [Ferramentas e dependências](FERRAMENTAS.md).

Mais detalhes: [Suporte](SUPORTE.md) · [Desktop e build](DESKTOP.md) · [CLI/JSON](CLI_JSON.md).
