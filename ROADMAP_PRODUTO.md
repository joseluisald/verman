# Verman — revisão do núcleo, desktop e lançamento

## Expansão de ferramentas — 0.5.0

Implementados Bun, Go, Deno, Git (MinGit), Rust/Cargo/rustup, Composer, Yarn moderno, pnpm nativo, vcpkg-tool e Chocolatey portátil na CLI e no desktop: catálogo, instalação verificada, seleção, shims e diagnóstico. [Ferramentas](docs/FERRAMENTAS.md) registra dependências e limites. O teste oficial isolado das dez ferramentas passou com instalação, seleção, verify, exec e shims; versões em [Provedores](docs/PROVEDORES.md). Não instala pacotes Chocolatey, compiladores ou bibliotecas C++ automaticamente. Aceites humanos/Windows limpo dos Steps 5 e 6 continuam pendentes.

Interface ampliada conferida no Chrome: 13 ferramentas, abas com quebra de linha, seleção por teclado e documentação. CLI, desktop, shims e instalador 0.5.0 compilados. A inspeção automática do novo desktop em janela oculta não expôs o conteúdo WebView2 à árvore de acessibilidade; não conta como validação interativa nativa desta versão. O teste de navegação/instalação em Windows limpo continua pendente.

Este plano continua o MVP descrito em [PLANEJAMENTO.md](PLANEJAMENTO.md). O objetivo é preparar uma CLI confiável e um desktop que compartilhem serviços Go. Implementar funcionalidades não equivale a validar um lançamento público.

## Diagnóstico inicial e evidências

Na revisão inicial, o projeto declarava PHP `8.5.12`, enquanto o PowerShell resolvia `php` como `D:\PHP\php-8.3.7\php.exe`. Não havia ativação do Verman aplicada à sessão. O binário gerenciado estava instalado corretamente.

Antes de modificar o código, foram executados `go test ./...` e `go vet ./...`, sem falhas. O cenário real foi reproduzido: `php -v` retornou 8.3.7; `go run . exec -- php -v` retornou 8.5.12; após aplicar `env`, `php -v` retornou 8.5.12; limpar a ativação restaurou o caminho original. As verificações ocorreram em um processo filho, sem editar perfil, PATH persistente, seleção do usuário ou sua instalação existente.

Conclusão: é necessário distinguir claramente versão instalada, seleção salva e executável resolvido pelo terminal. Um executável filho não altera o ambiente do terminal pai. O usuário precisa aplicar o ambiente ou habilitar a integração de shell; o produto deve explicar e diagnosticar isso.

## Step 1 — Revisar seleção/ativação e criar serviços compartilhados

- [x] Criar serviços Go de seleção e planejamento de execução, com resultados estruturados e sem apresentação de CLI.
- [x] Separar eventos de instalação/resolução da formatação de mensagens, preservando wrappers de compatibilidade.
- [x] Fazer `use`, `global` e `exec` consumirem esses serviços, mantendo as interfaces e configurações existentes.
- [x] Validar configurações antes de instalar e verificar cancelamento/instalação completa antes de salvar uma seleção.
- [x] Acrescentar `doctor [linguagem]`, incluindo JSON, para comparar seleção, instalação e resolução por PATH sem executar binários ou consultar rede.
- [x] Informar a origem da seleção, o caminho esperado e o caminho encontrado; não alegar detectar aliases/funções do shell pai a partir do processo Go.
- [x] Tornar a saída de `use` explícita sobre ativação e fornecer instruções executáveis e alternativa com `exec`.
- [x] Aplicar o ambiente ao habilitar o hook e recusar hooks vinculados a binários temporários de `go run`.
- [x] Adicionar testes de regressão do PHP externo, conflitos por alias, configuração inválida, cancelamento e preservação do ambiente original.

**Implementado:** `internal/application` fornece seleção, planejamento de execução e diagnóstico estruturados. Eventos tipados de instalação/resolução são formatados pela CLI; os wrappers antigos com writer permanecem para compatibilidade. `doctor` retorna JSON schema 1 e código de saída 1 para instalação inválida ou ativação pendente no PATH. Não executa o binário para afirmar sua versão e não consegue observar aliases/funções do shell pai. `use` informa que salvou uma seleção e fornece os comandos necessários para ativar ou executar sem ativação.

**Validação concluída:** suíte padrão e `go vet`; cenários reais opt-in de Node/npm/npx e PHP/Python/pip/venv em Windows/amd64. O teste PHP foi ampliado para começar com outra versão no PATH, provar a ausência de ativação após `use`, verificar `doctor`/`exec`, habilitar o hook com ativação imediata e reproduzir um alias sobrepondo a resolução. Os testes de serviços cobrem seleção exata a partir de alias, preflight de TOML inválido, instalador que não produz metadados válidos, cancelamento após instalar e falha tardia de gravação global. Hooks rejeitam tanto o diretório temporário tradicional quanto o executável em cache do Go 1.27.

**Limite desta entrega:** a instalação do PHP do usuário foi apenas lida/executada para diagnóstico; seu arquivo de projeto, o PHP externo, o perfil e o PATH persistente não foram alterados. O terminal aberto do usuário precisa aplicar o código de ativação. O executável atualizado é gerado na raiz para uso local; não houve publicação externa. Esta etapa não substitui as revisões de concorrência/recuperação do Step 2 nem autoriza anunciar o produto como pronto para produção.

**Aceite:** o caso PHP 8.3.7/8.5.12 é explicado pelo diagnóstico; execução gerenciada e ativação usam a versão selecionada; a CLI não diz que alterou o terminal ao apenas salvar TOML. Seleção e execução podem ser chamadas por um futuro desktop sem interpretar texto da CLI. Registrar testes e limites ao concluir.

## Step 2 — Durabilidade, concorrência e recuperação

- [x] Revisar gravações simultâneas de configurações locais/globais e impedir perda de atualizações.
- [x] Testar processo interrompido nas fases de preparação/publicação e recuperação de temporários/locks residuais.
- [x] Revisar exclusão com arquivos ocupados no Windows, remoção parcial e restauração de instalações.
- [x] Endurecer validação de metadados, dependências de executáveis e alterações posteriores à instalação.
- [x] Automatizar regressões de links/junctions, falhas de disco, permissão e checksum.

**Implementado:** locks nativos persistentes por identidade, compartilhados na execução e exclusivos em instalação/reparo/remoção; TOML local/global com leitura-modificação-escrita sob lock e substituição após sincronizar temporário. Journals identificam operações e permitem restaurar a instalação anterior ou concluir publicação/remoção. Comandos `verify`, `repair` e `recover` oferecem verificação de hashes, reinstalação e recuperação explícita. Metadados schema 2 registram hashes dos arquivos imutáveis e dependências necessárias; schema 1 permanece compatível, mas exige reparo para verificação de conteúdo. Reparo preserva configurações PHP e pacotes/scripts Python do usuário, mantendo as ferramentas bootstrap da distribuição nova.

**Validação concluída:** `go test ./...` e `go vet ./...`; subprocessos concorrentes gravam linguagens distintas em configurações locais e globais sem perder seleções; processos são mortos nas fases building/prepared/old_moved/published e após remoção total/parcial em quarentena. A recuperação foi verificada e os locks foram readquiridos após a morte do processo. Testes cobrem substituição bloqueada, escrita parcial com ENOSPC injetado, executável nativo ocupado sem exclusão parcial, corrupção de conteúdo, links/junctions, arquivos somente leitura, checksum e truncamento de download. Os fluxos oficiais Node/npm/npx e PHP/Python/pip passaram, incluindo reparo PHP que preserva ini e verificação schema 2.

**Entrega adicional de ativação:** shims nativos consultam a seleção a cada chamada e permitem projetos simultâneos sem reaplicar `env`. `setup` instala os executáveis; `scripts/setup-path.ps1` configura explicitamente PATH de usuário ou sistema com backup, prévia e detecção de conflitos. O escopo de sistema usa Program Files e requer administrador. Testes oficiais verificaram troca por diretório e dois processos PHP simultâneos com versões diferentes.

**Limites:** testes de interrupção de processo não simulam queda física de energia. Remoção separada em quarentena é um commit: a recuperação conclui sua exclusão, sem restaurar uma árvore parcialmente apagada como se fosse válida. Leases protegem execuções pelo Verman; programas que abrem caminhos diretamente continuam sujeitos ao Windows. Hashes locais não são assinaturas e excluem configurações/pacotes mutáveis; integridade completa é explícita em `verify`, não recalculada em cada chamada. Journals inválidos, áreas sem journal e locks legados são preservados. Atualização dos próprios shims tem rollback de falhas normais, mas ainda requer recuperação manual se interrompida à força. Não houve alteração do PATH persistente/perfil nem reparo da instalação real do usuário para testar; o script administrativo foi validado em modo prévia.

**Aceite:** falhas e concorrência preservam seleções e instalações anteriores quando recuperáveis; operações parciais são diagnosticáveis e possuem procedimento de recuperação testado. Nenhum bloqueio residual é removido por suposição.

## Step 3 — Experiência e contratos da CLI

- [x] Definir contrato JSON versionado para listagem, seleção, instalação e diagnóstico.
- [x] Padronizar erros, códigos de saída e mensagens de recuperação.
- [x] Exibir progresso e cancelamento de downloads adequados a terminal interativo e pipelines.
- [x] Melhorar onboarding de shell, conflitos com instalações externas e interação com venvs.
- [x] Documentar suporte, comportamento offline, logs e procedimento para reportar um problema.

**Implementado:** envelope JSON schema 2 compartilhado por list/install/use/global/doctor e comandos de manutenção, com success/data/error, um documento por execução e stderr separado. Códigos de argumentos/plataforma, rede/limite/offline, integridade, lock ocupado, TOML inválido e cancelamento são estáveis; falhas não especializadas usam código 1. `doctor` conserva resultados em data mesmo em diagnóstico com problema. Progresso em bytes/fases é controlado por `--progress auto|always|never`; cancelamento propaga o contexto e mantém a recuperação transacional. `--offline` bloqueia rede no núcleo e admite catálogo local antigo com aviso. Onboarding documenta shims, precedência, sessão e venv/Conda; doctor detecta esses ambientes. `support --json` gera resumo offline sem caminhos, sem enviar dados.

**Validação:** testes da CLI real verificam seleção/global/instalação já existente/listagem, falhas de argumentos/flags/TOML, offline, diagnóstico com dados em erro e um único documento JSON em stdout. Testes locais de download verificam contagem monotônica e cancelamento reconhecido; renderização em pipes separa stdout/stderr e evita carriage return. Os fluxos oficiais PowerShell Node/npm/npx e PHP/Python/pip/venv passaram com o novo JSON.

**Contrato e limites:** veja [docs/CLI_JSON.md](docs/CLI_JSON.md) e [docs/SUPORTE.md](docs/SUPORTE.md). O antigo JSON de doctor foi migrado explicitamente: relatório em data sob envelope schema 2. Não há arquivo de log automático/telemetria; coleta é explícita por redirecionamento. Não há cache offline de pacotes: instalar/reparar exige origem online. Arquivos ocupados/permissões e falhas ainda não especializadas usam operation_failed/1. Venv é identificado e explicado, sem migração automática. Não houve alteração do PATH/perfil do usuário para testar.

**Aceite:** um usuário novo consegue instalar e selecionar uma versão em um projeto sem precisar conhecer detalhes internos; scripts podem consumir saídas estáveis.

## Step 4 — Catálogos e provedores

- [x] Ampliar histórico de PHP/Python sem prometer versões sem artefatos compatíveis.
- [x] Definir disponibilidade, EOL, revisões, origem e confiança por distribuição.
- [x] Testar catálogo alterado, indisponibilidade, limites de API e cache antigo.
- [x] Avaliar plataformas adicionais individualmente, implementando apenas as que tiverem artefatos e testes reais.

**Implementado:** `list php/python --available --history` com cache próprio e metadados de provedor/confiança/instalabilidade/EOL/revisão. PHP reconhece NTS x64 no índice oficial de arquivos e exige checksum publicado no sha256sum oficial; sem ele, mostra o motivo de bloqueio e recusa instalar. Python consulta até 30 releases em páginas limitadas, fixa o build mais recente de cada versão e reconsulta a tag/digest antes de instalar. Versões exatas ausentes do catálogo atual consultam automaticamente esse histórico. Datas de EOL são um snapshot revisado em 18/11/2026 com fontes explícitas; ramos desconhecidos não ganham datas presumidas.

**Validação:** testes locais de alteração/adulteração de catálogo e URLs, variantes/revisões, ausência de digest, build fixado, limite HTTP/API, fallback com cache vencido e offline sem rede. Cache tem leitura limitada, rejeita duplicatas e é publicado com lock/sincronização. Na consulta oficial isolada, PHP retornou 180 versões, com 173 entradas antigas sem checksum (bloqueadas). Python histórico 3.15.1 build 20261021 foi baixado, validado funcionalmente (pip/venv), publicado e conferido com verify. Também passaram os fluxos oficiais de versões atuais.

**Matriz e limites:** [docs/PROVEDORES.md](docs/PROVEDORES.md) registra as plataformas avaliadas e versões executadas. Windows/amd64 permanece o único alvo habilitado; Linux/macOS/ARM64 exigem provedores/layouts e validação próprios. Catálogo não é certificação de execução de cada patch. Histórico PHP fica limitado pela disponibilidade de checksums; Python não varre todos os releases nem inclui assets sem digest. `supported_branch` não afirma que um patch antigo está atualizado. Política EOL não é feed de segurança em tempo real e hashes por HTTPS não equivalem a assinaturas dos distribuidores.

**Aceite:** a matriz de versões/plataformas anunciada corresponde a instalações verificadas. O Windows/amd64 continua sendo o alvo inicial; expansão nativa de sistema exige validação própria.

## Step 5 — UI/UX e protótipo navegável

- [x] Desenhar fluxos de instalar, selecionar por projeto, diagnosticar e remover.
- [x] Projetar telas de linguagens, projetos, operações, diagnóstico e configurações.
- [x] Distinguir visualmente seleção local/global, instalação e ambiente do terminal.
- [x] Projetar estados vazios, carregamento, erro, progresso, cancelamento e teclado/acessibilidade.
- [ ] Validar o protótipo com usuários antes de consolidar a interface.

**Aceite:** usuários conseguem identificar a versão do projeto e resolver um conflito de PATH no protótipo; a interface não promete alterar terminais que já estão abertos.

**Entrega implementada:** frontend React/TypeScript funcional com visão geral, linguagens/catálogo/histórico, projetos, operações, diagnóstico, configuração inicial e documentos offline. Confirmações de seleção/reparo/remoção, checkbox explícito para remover seleção, status, erros, foco/modal, atalhos de abas e layout responsivo. A prévia no navegador identifica dados simulados; o desktop usa a ponte Go real. [docs/UI_UX.md](docs/UI_UX.md) registra fluxos e roteiro de pesquisa.

**Validação e pendência:** compilação TypeScript/Vite, navegação/snapshots no Chrome, auditoria automatizada de acessibilidade dos estados revisados e inspeção nativa WebView2. Contraste foi corrigido após auditoria. Não houve sessão de usabilidade com participantes nem validação abrangente com leitor de tela; o último checkbox permanece aberto e o aceite com usuários ainda não foi comprovado.

## Step 6 — Aplicativo desktop

- [x] Avaliar e fixar uma versão compatível de Wails com frontend React/TypeScript.
- [x] Expor os mesmos serviços Go usados pela CLI, sem interpretar stdout como protocolo.
- [x] Integrar progresso/cancelamento e coordenar operações simultâneas entre CLI e GUI.
- [x] Implementar abrir terminal do projeto com ambiente correto.
- [ ] Testar acessibilidade, WebView2, instalação e atualização em máquinas Windows limpas.

**Aceite:** CLI e desktop produzem a mesma seleção e os mesmos efeitos, com testes compartilhados. O desktop não exige conta para gerenciar ferramentas locais.

**Entrega implementada:** Wails v2.17.1 fixado, React/TypeScript com lockfile, assets incorporados, diálogos nativos de pasta, documentação incorporada e PowerShell novo com ambiente do projeto. `internal/desktop` chama os serviços compartilhados; operações assíncronas têm progresso, cancelamento, exclusividade por janela e os mesmos locks da CLI. Verificação de arquivos aceita contexto. O histórico de até 50 operações fica na sessão; pastas recentes ficam no WebView local.

**Instalador adicional solicitado:** Inno Setup instala CLI, desktop, shims, guias HTML offline, atalhos e desinstalador. PATH do sistema é opcional, com backup e pasta protegida em Program Files. A desinstalação retira somente a entrada exata dos shims quando gerenciada pelo instalador, preservando runtimes e projetos. O guia abre por botão antes da instalação e por opções ao concluir. WebView2 ausente é informado com acesso à origem oficial; não há instalação silenciosa de dependência. `scripts/build-windows.ps1` gerou `dist/Verman-Setup-0.5.0-windows-amd64.exe` e checksum. A LP foi adiada nessa etapa e posteriormente autorizada; consulte a entrega local registrada abaixo.

**Validação e pendência:** `go test ./...`, `go vet ./...`, build TS/Vite e compilação do instalador. Testes verificam que a CLI lê a seleção do desktop, preservação de outro projeto, cancelamento/progresso e remoção bloqueada por lease. A janela nativa abriu com WebView2 e foi inspecionada por acessibilidade em perfil isolado. O instalador não foi aplicado no PATH/perfil real para testar. Instalação/atualização/desinstalação, UAC, WebView2 ausente e leitor de tela em Windows limpo continuam pendentes; por isso o último checkbox permanece aberto. Assinatura e licença comercial do compilador Inno são critérios de release, não conclusões desta revisão local.

## Step 7 — Engenharia de release

**Retorno do usuário sobre a instalação:** relatou funcionamento normal após reiniciar o PC. A orientação foi incluída no instalador e nos guias como próximo passo se reabrir o terminal/editor não resolver. Esse teste no ambiente do usuário não substitui os cenários pendentes em Windows limpo.

- [ ] Adicionar CI com testes, análise estática, integração e artefatos rastreáveis.
- [ ] Definir licença do próprio Verman e preservar os avisos das distribuições utilizadas.
- [ ] Assinar executáveis/instaladores e definir origem confiável de atualizações.
- [ ] Validar instalação/atualização/remoção em usuário comum e Windows limpo.
- [ ] Preparar changelog, documentação pública, suporte e distribuição por gerenciadores de pacotes.

**Aceite:** instalar uma versão publicada não depende de Go; atualização e falhas possuem recuperação testada. Publicação externa e assinatura usam credenciais/origens explicitamente autorizadas.

## Step 8 — Beta e lançamento

- [ ] Definir público inicial e recrutar um grupo pequeno de desenvolvedores.
- [ ] Observar instalação, seleção por projeto e resolução de problemas reais.
- [ ] Priorizar bloqueadores e validar o diferencial Windows + diagnóstico + interface visual.
- [ ] Definir política de suporte e, se necessária, coleta opcional de diagnósticos com consentimento.
- [ ] Decidir lançamento público e modelo comercial com base no uso e feedback.

**Aceite:** a beta resolve os fluxos principais com confiabilidade e sem incidentes de perda de configuração; critérios de lançamento são revisados com evidências, sem datas ou preços presumidos.

## Regras de validação

### Landing page e documentação — entrega local

A LP anteriormente adiada foi autorizada em 18/11/2026 e implementada em `verman-frontend-lp` com Astro, Tailwind CSS e Alpine.js. Usa a identidade do aplicativo, apresenta as 13 ferramentas, CLI/desktop, instalação, PATH, variáveis e dez guias. Três capturas foram feitas diretamente no EXE Wails 0.5.2 em perfil privado, com proveniência e sem mocks. O download local copia o instalador existente somente após conferir SHA-256. O build gera páginas estáticas e mantém a prévia/limitações de release explícitas. Não foi criada hospedagem, definida licença comercial, validado Windows limpo nem concluída beta com usuários; esses aceites permanecem abertos.

Testes padrão usam diretórios temporários e fontes HTTP locais. Testes com downloads oficiais são opt-in. Qualquer alegação de suporte deve identificar sistema, arquitetura e shell verificados. Não modificar o terminal/perfil real nem remover instalações do usuário para testar. Registrar limitações e itens pendentes em vez de marcar etapas amplas como prontas para produção.
