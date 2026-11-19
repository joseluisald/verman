# Desktop e instalador Windows

O desktop usa Wails v2.17.1, React e TypeScript, com WebView2. O backend chama `internal/application`, `internal/installer` e `internal/config` diretamente; não interpreta stdout da CLI como protocolo. Não há conta, serviço remoto ou telemetria.

## Telas e uso

- **Visão geral:** runtimes instalados, seleções e resumo de diagnóstico.
- **Linguagens:** catálogo, histórico, busca, instalar, selecionar no projeto/global, verificar, reparar, recuperar e remover com confirmação.
- **Projetos:** seleção da pasta pelo diálogo nativo, versões locais/fallback e abertura de terminal com ambiente preparado.
- **Operações:** progresso, cancelamento e histórico da sessão. Uma operação por janela; locks nativos coordenam a CLI e outras janelas.
- **Diagnóstico:** esperado/encontrado, shims, instalação ausente e venv/Conda, sem executar runtimes.
- **Configuração inicial e documentação:** guia offline e instruções executáveis para o PATH, CLI e primeiro projeto.

Pastas recentes ficam no armazenamento local do WebView. Seleções reais continuam nos mesmos TOMLs da CLI. O desktop não altera o ambiente de terminais já abertos. O botão Abrir terminal prepara o ambiente somente do processo novo. Remover um runtime não varre referências de outros projetos ou venvs.

A prévia no navegador usa dados de demonstração identificados. Não instala runtimes nem executa comandos da máquina. O executável Windows conecta a interface aos serviços reais.

## Compilar

Use o Go de `go.mod`, Node compatível com Vite e o compilador Inno Setup. As versões npm estão fixadas no package-lock. Não é necessário instalar a CLI do Wails globalmente.

O build reconhece `npm.exe` do Verman e `npm.cmd` do Node tradicional. Se usar Verman, selecione Node na raiz com `verman use node lts` e confira `npm --version`. Não precisa instalar outro Node para obter `npm.cmd`.

O logo é exportado automaticamente para ICO em vários tamanhos e incorporado ao executável do desktop antes de empacotar. O instalador, suas telas, o desinstalador e os atalhos do aplicativo usam o mesmo logo. Para criar o atalho, marque **Criar atalho na área de trabalho** no instalador. O ícone e as imagens são instalados em `{app}\assets`.

```powershell
& ./scripts/build-windows.ps1 -Version 0.5.4 -InnoCompiler 'CAMINHO\ISCC.exe'
```

O script executa npm ci/build, compila CLI e desktop com tags desktop/production, prepara shims em uma pasta do payload (sem mudar o PATH da máquina), renderiza a documentação HTML e compila o instalador. `-SkipInstaller` permite gerar somente o payload. `-SkipNpmInstall` reutiliza dependências já instaladas quando o lockfile não mudou.

Os bundles de frontend são gerados e ignorados no Git; compile a interface antes de compilar/distribuir o desktop. O `.gitkeep` permite compilar os testes Go em um checkout novo, mas não substitui o build da UI. O compilador Inno Setup 7.1.1 usado na revisão local identificou a edição como “Non-commercial use only”; verifique/adquira a licença aplicável antes de produzir uma distribuição comercial. Essa licença do compilador é independente da licença do aplicativo e de suas bibliotecas.

O instalador gerado fica em `dist/Verman-Setup-0.5.4-windows-amd64.exe` e o checksum fica ao lado. O payload em `dist/windows/payload` também pode ser usado para revisão. O build não gera executáveis na raiz.

Para desenvolver a interface:

```powershell
cd desktop/frontend
npm ci --ignore-scripts
npm run dev
```

## Instalar, atualizar e remover

O instalador usa `C:\verman` por padrão e roda na conta atual, sem solicitar elevação. Se a política do Windows bloquear a gravação nessa pasta, escolha um diretório em que sua conta possa gravar.

Deixe marcada a opção de adicionar a pasta `bin` ao PATH do usuário. Na instalação padrão, o caminho é **`C:\verman\bin`**. Ferramentas no PATH do sistema podem ter prioridade. Use `verman exec -- php -v` para executar a versão selecionada. Se mudar a pasta de instalação, use a subpasta `bin` dela. Reabra o terminal e o editor e confira `verman --version`.

A CLI, o desktop e os shims do pacote armazenam novas versões em `<pasta do Verman>\tools\<ferramenta>\<versão>`. As versões já instaladas em `%USERPROFILE%\.verman\installs` continuam lá e aparecem na listagem. Se a mesma versão estiver nas duas pastas, a pasta `tools` tem prioridade. A CLI avulsa continua usando `%USERPROFILE%\.verman\installs`. O perfil também guarda a configuração global e o cache de catálogos.

A detecção de WebView2 é local. Se estiver ausente, o assistente informa a dependência e oferece a página oficial; nenhum componente externo é baixado/executado silenciosamente como administrador. A CLI e os guias continuam disponíveis.

Se os comandos ainda não forem reconhecidos ou continuarem na versão antiga após reabrir o terminal/editor, reinicie o PC e teste novamente. Esse procedimento funcionou no teste relatado pelo usuário. Se persistir, confira o PATH e execute `verman doctor`.

Para migrar da edição instalada para todos os usuários, desinstale a edição antiga em **Aplicativos instalados** antes de instalar a nova. Se sobrar uma entrada no PATH do sistema, remova apenas a pasta `bin` da instalação antiga. Essa alteração pode exigir administrador. Os runtimes e os projetos são preservados. Para atualizar edições por usuário, execute o instalador novo com o desktop fechado. A configuração do PATH preserva entradas anteriores e faz backup. A desinstalação retira a entrada exata dos shims quando ela foi configurada pelo instalador, remove arquivos/atalhos registrados e preserva as ferramentas em `tools`, os dados em `~/.verman` e os projetos. Para voltar a usar as ferramentas de `tools` após reinstalar, escolha o mesmo destino. Backups/dados adicionais não registrados podem permanecer na pasta do aplicativo. O PATH dos processos já abertos não pode ser atualizado pelo desinstalador.

## Validação e limites

Testes locais devem usar diretórios e perfis temporários. Não altere o PATH/perfil real nem desinstale runtimes do usuário para verificar uma mudança. Os critérios pendentes de pesquisa com usuários e de Windows limpo estão no ROADMAP_PRODUTO.md; compilar um instalador não substitui esses testes.

O executável/instalador local ainda não tem assinatura de publicação. A disponibilidade do WebView2 sem ele instalado, os fluxos de UAC, instalação/atualização/remoção e acessibilidade com leitor de tela precisam ser avaliados em VMs Windows limpas antes de um lançamento público.

A landing page foi autorizada e implementada em `verman-frontend-lp`, com Astro, Tailwind CSS e Alpine.js, guias simples e capturas reais do desktop. Consulte o README dessa pasta para rodar localmente e preparar os downloads. Os critérios de release público continuam pendentes.
