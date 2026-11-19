# Verman — orientações para agentes

## Objetivo

Construir um gerenciador de versões de runtimes e ferramentas, inspirado no nvm, para Node.js, PHP, Python, Bun, Go, Deno, Git, Rust, Composer, Yarn, pnpm, vcpkg e Chocolatey no Windows/amd64. Cada projeto declara suas versões e executa as ferramentas correspondentes sem instalações manuais repetidas.

O núcleo e a CLI usam Go. O desktop Windows usa Wails v2.17.1 e React/TypeScript, chamando os mesmos serviços. Preserve essa separação; não reescreva o núcleo nem use stdout da CLI como API do desktop. A landing page em `verman-frontend-lp` foi autorizada em 18/11/2026: Astro, Tailwind CSS e Alpine.js, mesmas cores/logo e capturas reais do desktop. O trabalho é local; não publicar nem registrar hospedagem sem solicitação.

## Estado atual confirmado

Esta descrição corresponde ao código de novembro de 2026. Atualize-a quando o comportamento mudar. Os Steps 1 a 9 estão implementados dentro do escopo Windows/amd64, incluindo PowerShell, Git Bash/sh, hooks opcionais, exec, seleção global, aliases e pacote local versionado.

O plano de amadurecimento está em [ROADMAP_PRODUTO.md](ROADMAP_PRODUTO.md). Steps 1 a 4 foram implementados; Steps 5 e 6 têm interface/desktop funcionais e instalador local, com validações de usuários e Windows limpo ainda pendentes. Não marque esses critérios como atendidos por testes automatizados ou compilação. Leia os limites registrados no roadmap.

- Módulo: `github.com/joseluisald/verman`.
- `go.mod` declara Go `1.27.4`; confira o arquivo antes de alterar a versão mínima.
- Cobra implementa a CLI; BurntSushi/toml lê e escreve a configuração.
- Existem testes em `cmd`, `internal/config` e `internal/installer`, usando diretórios pessoais temporários.
- Binários de revisão e instaladores ficam em `dist`; não gere nem mantenha executáveis na raiz. A limpeza preserva o instalador mais recente, seu checksum, o payload atual e o compilador Inno usado no build.

## Estrutura e fluxo

- `verman-frontend-lp/`: site estático Astro, Tailwind v4 via Vite, Alpine CSP e fontes locais. `npm ci --ignore-scripts`, `npm run dev` e `npm run build` dentro da pasta. Dez guias em `src/data/docs.ts` são renderizados no build; não há backend ou conta. `scripts/prepare-assets.mjs` copia o logo e verifica o SHA-256 do instalador antes de incluí-lo. Builds isolados usam a exportação versionada do logo em `assets/` e o instalador em `assets/installers/`. O pacote é prévia; não anunciar assinatura/comercialização pronta. Capturas nativas estão em `public/screenshots`, com proveniência; não substituir por mocks nem expor dados pessoais. Atualize `src/data/product.json` ao preparar outra versão. Não misture dependências da LP com as do desktop.

### Expansão de ferramentas (0.5.0)

- `internal/tool/definitions.go` centraliza IDs, comandos e dependências das 13 ferramentas. `Languages()` preserva o nome legado; `tool.Owner` liga cargo/choco/bunx à instalação correta. O frontend mantém sua lista em `tools.ts`, com abas responsivas e teclado para todos os itens.
- `github_tools.go` atende Bun baseline, Deno, MinGit, pnpm nativo, vcpkg-tool e Chocolatey portátil por releases oficiais com digest SHA-256. Lista até 30 releases e tenta a tag oficial para instalação exata fora da lista; não prometa histórico completo. Git preserva revisão Windows em build; vcpkg normaliza datas para X.Y.Z.
- `go_composer.go`, `yarn.go`, `rust.go`: Go ZIP/SHA-256, Composer PHAR/sidecar SHA-256, Yarn moderno @yarnpkg/cli-dist/SRI SHA-512 e rustup-init/SHA-256. Yarn recebe SHA-256 local somente depois de autenticar SHA-512; metadados preservam ambos. Nunca substitua SRI por hash de um download não autenticado.
- `tool_runtime.go` valida ferramentas e prepara vcpkg (segundo bundle também verificado) e Rust (`--no-modify-path`, homes privados, servidores oficiais explícitos). ToolVariables é compartilhada pela ativação e validação; backup/restauração inclui Go, Rust, vcpkg e Chocolatey. Não reutilize homes externos nem instale compiladores/VC++ globalmente.
- Composer exige PHP selecionado e Yarn exige Node selecionado. `WithWorkingDirectory` leva o projeto à validação sem os.Chdir global. Exec/shims despacham scripts pelo runtime selecionado e verificam o packageManager Yarn sem editar package.json. Rust força a toolchain exata; Go usa GOTOOLCHAIN=local.
- Repair preserva raízes mutáveis explícitas e bins Cargo do usuário, mantendo proxies novos. Chocolatey com pacotes em lib/lib-bad/lib-bkp recusa remoção da versão; não desinstala as aplicações desses pacotes. Choco usa tools/chocolateyInstall dentro da instalação, nunca o Chocolatey externo.
- `docs/FERRAMENTAS.md` é incorporado ao desktop e HTML offline; `docs/PROVEDORES.md` registra versões executadas. `TestOfficialAdditionalToolsWorkflow` é opt-in, usa perfil temporário, valida download/seleção/verify/exec/shims das dez ferramentas; timeout 30m. Não equivale a teste de builds completos, pacotes Chocolatey ou bibliotecas C++.

As referências a três linguagens abaixo descrevem os provedores originais; para IDs/comandos e fontes atuais, prevalecem o registro acima e os arquivos citados.

- `desktop/`: entrypoint Wails Windows, ponte tipada com diálogos de pasta/terminal/documentação e frontend React/TypeScript. `desktop/frontend/src` contém o código; `npm ci --ignore-scripts` e `npm run build` geram os assets incorporados. Nunca distribua um build Go sem antes compilar a UI. A prévia de navegador usa dados simulados claramente identificados; o executável chama `window.go.main.App` e serviços reais.
- `internal/desktop`: snapshot/catálogo/seleção/manutenção usando os serviços existentes, sem `os.Chdir` global. Operações assíncronas têm progresso, cancelamento, limite de histórico e exclusividade por janela; locks nativos também coordenam a CLI. Confirmar exclusão é responsabilidade da UI; o núcleo mantém as proteções de seleção/arquivos ocupados.
- `internal/application/terminal.go`: ambiente do novo terminal, com prioridade preservada de shims verificados. Não altere o ambiente do processo pai nem execute scripts/expressões construídos a partir da pasta escolhida.
- `docs/content.go`, `tools/windowsbundle`: documentação Markdown incorporada no app e HTML offline autossuficiente no instalador. Não introduza dependências de rede na documentação local. `scripts/build-windows.ps1` compila CLI/desktop/shims/docs e chama Inno Setup; não muda o PATH para testar.
- `scripts/build-tools.ps1` resolve npm.exe (shim Verman) com fallback npm.cmd (Node tradicional), sem usar npm.ps1 nem alterar PATH/seleções. O build imprime o caminho usado. Preserve suporte ao cenário sem npm.cmd; testes PowerShell em tools/windowsbundle cobrem resolução, argumentos e erros do processo filho.
- `tools/windowsicon` exporta o PNG original para ICO multirresolução e imagens do instalador, sem dependências novas. O build incorpora RT_ICON/RT_GROUP_ICON ao desktop por UpdateResource antes de empacotar (ID 32512 usado pelo Wails); preserve os demais recursos. Os atalhos apontam explicitamente para `{app}\assets\verman.ico`. Não altere executáveis instalados/assinados para testar; use somente o payload recém-compilado e arquivos temporários. Assinatura futura deve ocorrer após incorporar os recursos.
- `installer/windows/verman.iss`: assistente Windows x64 por usuário, com instalação padrão em `C:\verman` sem solicitar elevação, opção de PATH do usuário, detecção de WebView2, guias e atalhos. `scripts/remove-path.ps1` retira somente a entrada exata dos shims no escopo configurado. Não apague runtimes/perfis/projetos ao desinstalar e não restaure o PATH inteiro de um backup sobre alterações posteriores.
- O compilador Inno usado na revisão identificou licença não comercial; conferir licença aplicável e assinatura é parte do release público. Não anuncie o artefato de revisão como comercialmente pronto.

- `main.go`: chama `cmd.Execute()`. O dispatch identifica o nome do executável; cópias chamadas php.exe/node.exe/python.exe/npm.exe/npx.exe/pip.exe executam o shim.
- `internal/shims` e `internal/shimmeta`: setup de cópias nativas com manifesto de propriedade, execução pelo serviço compartilhado e reconhecimento do shim em doctor/use. O PATH de fallback exclui pastas de shims para evitar recursão. `scripts/setup-path.ps1` é a alteração persistente explícita: prévia, backup, escopo User/Machine, detecção de conflito no sistema. Machine requer administrador e destino em Program Files; não o execute para validar uma mudança no computador real.
- `internal/fslock`, `internal/durable` e `internal/installer/transaction.go`: locks nativos, substituição após Sync e journal de instalação/reparo/remoção ancorado em os.Root. Nunca use caminhos arbitrários de journals para apagar dados. Áreas sem journal/estados ambíguos são preservados. `cmd/recovery.go` expõe verify/repair/recover.
- `cmd/root.go`, `cmd/output.go` e `internal/fault`: contrato JSON schema 2 para consulta/gerenciamento, códigos de saída por categoria, erros contextualizados e cancelamento 130. `exec`/shims preservam o código do filho; env/init/exec recusam --json. Não emita dois documentos JSON nem texto de progresso em stdout. Veja [docs/CLI_JSON.md](docs/CLI_JSON.md); o relatório antes emitido diretamente por doctor agora fica em data. O schema do relatório interno permanece 1.
- `--offline` passa pelo contexto e bloqueia rede no núcleo. `--progress auto|always|never` renderiza progresso em stderr; pipes usam linhas sem carriage return quando solicitado. `cmd/support.go` gera estado offline sem caminhos/valores de ambiente, sem upload nem log persistente automático.
- `cmd/install.go`: recebe exatamente uma linguagem e uma versão e chama `installer.Install`.
- `cmd/list.go`: implementa listagem local e `list <linguagem> --available [--refresh]` para node/php/python, com seleção do projeto separada do estado instalado. Alertas de cache remoto vão para stderr.
- `cmd/use.go`: resolve aliases, garante a instalação e grava a versão exata no diretório atual. `cmd/global.go` faz a seleção global; `global --unset` não remove instalações.
- `internal/application`: serviços compartilhados para seleção, planejamento de execução e diagnóstico. Resultados são estruturados; não imprimem mensagens da CLI nem executam um processo filho por conta própria. `use`, `global` e `exec` chamam esses serviços. A seleção verifica configuração antes de instalar e revalida instalação/cancelamento antes de gravar.
- `internal/installer/events.go`: eventos tipados para resolução/instalação. O fluxo principal usa sinks; adapters com writer preservam os consumidores antigos. `cmd/events.go` cuida da apresentação. O futuro desktop deve consumir dados/eventos, não interpretar texto.
- `cmd/doctor.go`: `doctor [linguagem] [--json]` compara seleção, instalação e resolução por PATH. É offline, não executa runtimes e não inspeciona aliases/funções do terminal pai. JSON schema 2 com diagnóstico em data; código 1 para instalação inválida/ativação pendente. Informa venv/Conda ativo sem migrá-lo ou desativá-lo. Não chame um caminho coincidente de versão efetivamente executada.
- `cmd/env.go`, `internal/environment/powershell.go` e `state.go`: emitem PowerShell ou POSIX e calculam o ambiente efetivo. `env --clear` limpa a ativação ignorando seleções. O renderer POSIX no Windows usa cygpath para converter o PATH.
- `cmd/init.go` e `internal/environment/hooks.go`: imprimem hooks opcionais de prompt para PowerShell e Bash 5+, sem editar perfis. `init --disable` remove o hook da sessão. O hook deve referenciar um executável permanente e não o binário temporário de go run.
- Ao aplicar `init`, a seleção atual também é ativada imediatamente, antes do próximo prompt. Rejeite caminhos temporários/caches de `go run`, inclusive o layout do Go 1.27. Use um binário compilado estável nos testes de hooks.
- `cmd/exec.go`: executa um filho com ambiente calculado, lookup pelo PATH efetivo e argumentos separados. Preserva código de saída. Não introduza shell implícito: npm/npx/pip gerenciados têm despacho explícito para o runtime correspondente.
- `cmd/uninstall.go` e `internal/installer/uninstall.go`: removem uma instalação válida sem alterar o TOML ou a sessão; `--allow-selected` é obrigatório se o projeto atual selecionar a versão.
- `internal/config/config.go`: define caminhos, procura e serializa `.verman.toml`.
- `internal/config/global.go`: lê/grava `~/.verman/config.toml`; a seleção do projeto prevalece por linguagem e as demais usam o global. Não combina arquivos de projeto ancestrais e não ignora TOML inválido.
- `internal/tool/tool.go`: valida as ferramentas do registro `definitions.go` e versões exatas estáveis `X.Y.Z` antes de construir caminhos.
- `internal/installer/manager.go`: define metadados de instalação, verifica arquivos e deriva diretórios de executáveis. `Install` é um wrapper de compatibilidade; a CLI usa `InstallContext` com cancelamento.
- `internal/installer/list.go`: descobre somente instalações locais válidas, ignora diretórios incompletos e ordena versões numericamente, sem conversão para inteiros de tamanho limitado.
- `internal/installer/provider.go` e `node.go`: contrato de provedor e distribuição oficial Node, limitada a Windows/amd64. Consulta `https://nodejs.org/dist/index.json` e resolve ZIPs e `SHASUMS256.txt` na mesma origem.
- `internal/installer/php.go`: catálogo oficial `https://downloads.php.net/~windows/releases/releases.json`, variante NTS x64. Normaliza revisões de pacote, preservando build/variante nos metadados, e reconsulta o checksum remoto antes de instalar.
- `internal/installer/python.go`: CPython Astral python-build-standalone, variante install_only Windows/amd64. Consulta o release GitHub mais recente; resolve pela tag de build com SHA-256 do asset remoto. Não usa o pacote embeddable da python.org nem anuncia todas as versões históricas.
- `internal/installer/runtimes.go`: prepara php.ini/conf.d locais e valida PHP/Python, incluindo pip e venv. `php_path_windows.go` trata os caminhos de DLL do PHP (LoadLibraryA), preferindo nomes curtos e conversão ANSI sem substituição de caracteres. Não remova esse tratamento: o teste com acentos e cURL detecta regressões reais.
- `internal/installer/catalog.go`: cache separado por linguagem/plataforma e histórico, schema 1, validade de uma hora, atualização forçada e fallback com aviso/data; offline admite cache antigo sem rede. Valida tamanho, duplicatas e artefatos e usa lock/substituição durável ao salvar. O provedor reconstrói URLs e reconsulta checksums antes de instalar.
- `internal/installer/history.go` e `distribution.go`: PHP cruza índice de archives com sha256sum oficial; entradas sem hash são visíveis mas bloqueadas. Python consulta até 30 releases paginadas e fixa o build mais recente por versão, ignorando assets históricos sem digest. Instalação exata ausente no índice atual tenta esse histórico limitado. Política EOL é snapshot revisado em 18/11/2026 com fonte/data expostas, sem inferir segurança de patches antigos. Consulte [docs/PROVEDORES.md](docs/PROVEDORES.md).
- `internal/installer/aliases.go`: latest para as três linguagens e lts somente para Node. O índice Node fornece o campo LTS. Aliases não são aceitos como componentes de caminho nem persistidos em TOML.
- `internal/installer/install.go` e `extract.go`: download com limites e SHA-256, extração ZIP ou tar.gz sem links ou escapes de caminho, validação por linguagem, publicação por rename e lock por versão. Metadados só são publicados após validar o runtime; incluem o hash do pacote, build e variante quando aplicáveis.
- `internal/installer/lock.go`: coordena instalação e remoção com o mesmo bloqueio exclusivo por versão. A remoção verifica contenção lexical e resolvida e usa `os.Root` para ancorar a exclusão.
- `tools/release/main.go`: gera o ZIP Windows/amd64 e SHA256SUMS em dist, com versão injetada em cmd.Version. `scripts/install.ps1` verifica o pacote e instala/atualiza somente o executável; uninstall.ps1 preserva ~/.verman. Não publique pacotes externamente sem solicitação.

Exemplo de configuração:

```toml
[tools]
node = "22.11.0"
php = "8.4.1"
python = "3.13.0"
```

Esses valores ilustram o formato, não uma lista de versões suportadas ou recomendadas.

`LoadProjectConfig` procura o primeiro `.verman.toml` do diretório informado até a raiz. Não combina configurações ancestrais. Sem arquivo, retorna uma configuração vazia. `SaveProjectTool` escreve apenas no diretório recebido; portanto, executar `use` em uma subpasta pode criar um arquivo local mesmo quando existe configuração no pai. `LoadEffectiveConfig` acrescenta somente as linguagens globais ausentes desse primeiro arquivo; env e exec usam essa configuração efetiva. List exibe seleções locais e globais separadamente.

O diretório de instalação é `~/.verman/installs/<linguagem>/<versao>`, calculado com `os.UserHomeDir` e `filepath.Join`. O arquivo `installation.json` novo usa schema 2 e registra identidade, plataforma, origem HTTPS, executáveis relativos com separador `/`, transação, arquivos imutáveis com SHA-256 e dependências necessárias. Schema 1 permanece legível, mas verify exige repair para obter hashes. `BinDirs` deriva os diretórios do mapa, sem impor `bin`, e mantém o runtime principal primeiro. Hashes são recalculados em verify e na recuperação de uma substituição publicada; leituras comuns não fazem verificação completa de conteúdo. Configurações PHP e pacotes/scripts Python são mutáveis e excluídos dos hashes; repair os preserva com limites de cópia e sem links.

## Limitações que não devem ser confundidas com funcionalidades

- Os três provedores suportam Windows/amd64. Linux, macOS e outras arquiteturas não são anunciados como suportados. PHP exige Visual C++ Redistributable x64 compatível. Histórico é limitado: PHP sem checksum oficial não é instalável e Python consulta até 30 releases com digest. Não prometa cobertura histórica completa nem instale dependências globais silenciosamente.
- `IsInstalled` exige metadados compatíveis com a plataforma e arquivos regulares não vazios dentro da instalação (executáveis em sistemas POSIX). Não reexecuta código nem recalcula hashes; a instalação valida o pacote e o funcionamento antes de publicar os metadados.
- `env --shell powershell` é o padrão no Windows; bash/sh funcionam no Git Bash com cygpath. Não anuncie zsh/fish/cmd.exe nem suporte nativo a runtimes Linux/macOS. Hooks são opcionais, aplicam antes do próximo prompt e não interceptam mudanças de diretório dentro de uma mesma linha de comandos.
- O script de ativação substitui as entradas gerenciadas usando `VERMAN_PATH_ENTRIES`; não apague esse marcador manualmente durante uma sessão ativada. Sem configuração, aplicar `env` remove essas entradas. Instalações ausentes/ inválidas retornam erro sem emitir código; caminhos contendo separadores PATH não representáveis são recusados.
- A ativação gerencia PHPRC/PHP_INI_SCAN_DIR para usar a configuração local do PHP e limpa PYTHONHOME/PYTHONPATH para Python. `VERMAN_ENV_BACKUP` guarda valores anteriores, restaurados quando a linguagem deixa de estar selecionada. Restrinja os nomes gerenciados à lista permitida e valide o marcador antes de alterar o ambiente.
- Extensões PHP opcionais ficam desabilitadas por padrão. O usuário pode editar somente o php.ini da instalação ou seus arquivos conf.d. Python inclui um wrapper pip.cmd relativo; prefira documentar `python -m pip`. Venvs existentes não são migrados ao trocar a seleção e podem depender da versão base removida.
- `SaveProjectTool` rejeita configuração inválida e prepara/sincroniza um temporário no mesmo diretório antes de substituir o arquivo. Preserva chaves desconhecidas, mas não comentários e formatação. Gravações locais/globais coordenam leitura-modificação-escrita com lock nativo e timeout/cancelamento; não retire esse lock.
- O SHA-256 é conferido contra o manifesto oficial via HTTPS; assinaturas PGP não são verificadas. Não aceite URLs arbitrárias de cache/configuração como origem de executáveis.
- Locks atuais são arquivos `.lock-<versao>.lck` com lock nativo. Permanecem por identidade; não os apague ao liberar. O sistema libera o bloqueio ao encerrar o processo. Diretórios legados `.lock-<versao>` continuam sendo recusados e preservados. Shims/exec mantêm leases compartilhados e mutações exigem exclusividade.
- A remoção exige instalação válida e recusa diretórios gerenciados que sejam links/junctions. Não varre outros projetos e não edita suas seleções. Uma versão em uso por outro processo pode não ser removível no Windows; trate o erro de arquivo ocupado sem alegar sucesso.
- Hooks relêem a configuração sem instalar runtimes ou executar scripts do projeto; em erro, limpam o ambiente anterior. Preserve o prompt e callbacks do usuário, evite duplicação e mantenha o código de saída anterior. A retirada do hook não edita o perfil; sua linha de inicialização deve ser retirada manualmente se persistida pelo usuário.
- `use` salva a seleção e não altera o ambiente do processo pai. Sua mensagem deve distinguir esses estados e oferecer ativação/exec. A regressão original foi PHP 8.5.12 selecionado com PHP 8.3.7 externo no PATH; não corrija isso mudando o PATH persistente ou removendo o PHP externo silenciosamente.
- Exec não baixa runtimes, não interpreta scripts batch implicitamente e não pesquisa entradas relativas/vazias do PATH; o usuário deve fornecer um caminho explícito para executar arquivos no diretório atual. Venv ativo deve ser desativado antes de trocar de seleção, como documentado.

## Direção de implementação

Implemente incrementos verificáveis, mantendo a CLI utilizável:

1. Tornar real a instalação de Node.js para uma combinação explícita de sistema e arquitetura. Validar entradas, resolver o artefato oficial, baixar com tratamento de falhas, verificar integridade, extrair e confirmar o executável.
2. Separar o ciclo comum de instalação das particularidades de cada linguagem. Definir uma abstração pequena de provedor conforme as necessidades concretas, sem assumir URLs, pacotes ou layouts iguais para Node, PHP e Python.
3. Corrigir a detecção de instalação completa e a persistência de configuração. Manter arquivos anteriores intactos em caso de falha.
4. Implementar a ativação por shell, incluindo PowerShell no Windows e shells POSIX nos sistemas correspondentes. Definir como identificar ou selecionar o shell e testar o escape dos caminhos.
5. Acrescentar consulta de versões disponíveis/instaladas e remoção, e depois novos provedores. Avaliar shims ou um comando de execução com ambiente próprio conforme a experiência desejada.

Essa lista é direção de produto, não autorização para implementar todos os itens em uma tarefa restrita.

## Convenções de alteração

- Mantenha `cmd` responsável por argumentos e apresentação; concentre configuração e instalação em `internal`.
- Preserve mensagens e ajuda em português e arquivos em UTF-8.
- Use `filepath` para caminhos locais e construa URLs separadamente. Considere `runtime.GOOS` e `runtime.GOARCH` sem assumir equivalência com os nomes usados pelos distribuidores.
- Valide linguagem e versão antes de formar caminhos; rejeite separadores, caminhos absolutos e tentativas de escapar da pasta de instalações.
- Não execute comandos montados por concatenação com entradas do usuário. Prefira bibliotecas Go e argumentos separados quando processos externos forem necessários.
- Nunca considere uma instalação concluída apenas por existir uma pasta. Use uma área temporária, valide o resultado e só então publique a instalação; limpe resíduos de falhas.
- Downloads devem usar fontes confiáveis, tratar status HTTP e ter limites e cancelamento adequados. Na extração, impeça entradas e links de escrever fora do destino.
- Não altere o `PATH` persistente do sistema nem instale globalmente para validar uma mudança. A ativação deve ter escopo e comportamento explícitos.
- Para `env`, mantenha stdout reservado ao código de shell; diagnósticos devem ir para stderr. Garanta escape correto e ordem determinística.
- Propague erros com contexto e não descarte erros de leitura, gravação ou fechamento relevantes. Evite truncar configurações antes de validar e preparar seu novo conteúdo.
- Não adicione dependências ou reorganizações extensas sem necessidade concreta. Não edite artefatos gerados ou configurações da IDE para modificar comportamento da aplicação.

## Validação

Execute na raiz após mudanças de código:

```text
go test ./...
go vet ./...
go run . --help
```

Aplique `gofmt` aos arquivos Go alterados. Para verificar compilação explicitamente no Windows, use `go build -o <diretorio-temporario>/verman.exe .`, sem gerar binários na raiz.

Os testes padrão usam diretórios temporários e servidores HTTPS locais. Cobrem configuração, listagem, catálogo/cache, integridade, falhas de download, extração insegura, validação e concorrência. O teste real de duas distribuições oficiais é opt-in: defina `VERMAN_INTEGRATION=1` e execute `go test ./internal/installer -run '^TestOfficialNodeInstallation$' -count=1 -v -timeout 10m`. Ele foi validado em Windows/amd64 com Node 22.11.0 e 22.12.0, incluindo npm/npx. Não habilite rede nos testes padrão.

Testes locais de PowerShell verificam escape, reaplicação, troca e limpeza do PATH em processos filhos. Testes de remoção verificam seleção, bloqueios, escopo e links/junctions. Para validar o ciclo real de seleção/ativação/remoção, use `VERMAN_INTEGRATION=1` e `go test ./cmd -run '^TestOfficialPowerShellWorkflow$' -count=1 -v -timeout 10m`. Esse teste também foi validado em Windows/amd64 com Node 22.11.0/22.12.0, usando caminhos com espaços e símbolos.

Para PHP/Python: `VERMAN_INTEGRATION=1` e `go test ./cmd -run '^TestOfficialPHPPythonWorkflow$' -count=1 -v -timeout 15m`. Seleciona duas versões de cada catálogo e verifica a CLI real, cURL do PHP, pip/venv após publicar, seleção em dois projetos, restauração de variáveis e remoção. Validado com PHP 8.5.12/8.4.27 revisão -1 e Python 3.15.2/3.14.10 build 20261110. Catálogos, revisões, hashes, filtros e arquivos TAR/ZIP maliciosos são testados localmente, sem rede nos testes padrão.

Os testes offline de Step 9 compilam a CLI em diretórios temporários, verificam hooks reais de PowerShell/Git Bash, sh, precedência global, exec e argumentos/código de saída. Não execute fixtures de runtime sintéticas; só os binários oficiais são executados nos testes opt-in. Use PSModuleAnalysisCachePath temporário ao iniciar PowerShell nos testes para não criar caches no repositório ou no perfil real.

Os testes de application usam backends e metadados locais para erros de configuração, seleção incompleta, cancelamento e falhas de gravação. O teste opt-in PHP/Python também reproduz PHP externo antes da ativação, doctor JSON, alias PowerShell sobrepondo o PATH e ativação imediata por init. Ao diagnosticar uma máquina real, use leitura e execução controlada; não altere seleções/instalações/perfil do usuário para testar.

Para gerar um pacote revisável: `go run ./tools/release -version 0.1.0 -out dist`. Mantenha apenas plataformas verificadas. Verifique instalação/atualização/remoção em um BinDir temporário e confira --version; não instale no perfil real para testar. Dist e verman.exe da raiz são artefatos ignorados, não fontes.

Ao implementar comportamento, priorize testes com diretórios temporários e servidores HTTP locais para configuração herdada, TOML inválido, validação de caminhos, falhas de download, extração maliciosa e detecção de instalação incompleta. Não dependa de rede pública ou do diretório pessoal real nos testes. Valide a sintaxe de ativação nos shells suportados e deixe claro quais plataformas foram efetivamente verificadas.
