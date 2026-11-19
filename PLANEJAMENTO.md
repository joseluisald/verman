# Planejamento do Verman

## Objetivo e escopo

Criar uma CLI em Go para descobrir, instalar e selecionar versões de Node.js, PHP, Python e outras linguagens por projeto.

Os Steps 1 a 9 estão implementados. Node, PHP e Python podem ser consultados, instalados, selecionados e removidos em Windows/amd64, com ativação PowerShell ou Git Bash/sh, hooks opcionais, seleção global, aliases e execução por projeto. A distribuição local inclui ZIP versionado, checksums e scripts de instalação/atualização/remoção do Verman. As orientações estão em [AGENTS.md](AGENTS.md) e [README.md](README.md).

O primeiro marco, Node.js em Windows/amd64 com PowerShell, foi validado e ampliado para PHP e Python. Linux, macOS e outras arquiteturas serão habilitados conforme cada provedor tiver artefatos e testes correspondentes.

## Experiência desejada

| Comando planejado | Comportamento |
| --- | --- |
| `verman list` | Mostra linguagens suportadas e um resumo das instalações locais. |
| `verman list node` | Lista as versões de Node instaladas localmente. |
| `verman list php` | Lista as versões de PHP instaladas localmente. |
| `verman list node --available` | Consulta versões de Node disponíveis para instalar na plataforma atual. |
| `verman list php --available` | Consulta versões de PHP disponíveis para instalar na plataforma atual. |
| `verman list python --available` | Consulta versões de Python disponíveis para instalar na plataforma atual. |
| `verman list node --available --refresh` | Atualiza a consulta remota, ignorando o cache. |
| `verman install node <versao>` | Instala uma versão exata, sem alterar a seleção do projeto. |
| `verman use node <versao>` | Garante a instalação e grava a seleção em `.verman.toml`. |
| `verman env --shell powershell` | Emite código PowerShell para aplicar as versões do projeto à sessão. |
| `verman uninstall node <versao>` | Remove uma instalação gerenciada pelo Verman. |

`list <linguagem>` deve funcionar offline. A opção `--available` consulta o catálogo remoto. A saída deve ser ordenada por versão, da mais recente para a mais antiga, e indicar qual versão está selecionada pelo projeto. Estar selecionada no arquivo não comprova que a sessão atual está usando aquela versão.

Os comandos acima são o contrato proposto; opções novas ainda precisam ser implementadas. No primeiro marco, versões devem ser exatas. Aliases como `latest`, `lts` e intervalos de versão ficam para uma etapa posterior.

## Como descobrir versões e saber instalá-las

Cada linguagem terá um provedor responsável por traduzir uma versão em um pacote instalável. O catálogo deve vir de índices de releases ou manifestos da distribuição escolhida, em vez de uma lista de versões escrita manualmente no código.

O registro interno de provedores define quais linguagens o Verman suporta. Cada provedor define:

- Fonte do catálogo e sua política de confiança.
- Sistemas e arquiteturas suportados e os nomes usados pelo distribuidor.
- Como identificar versões estáveis, prévias e variantes.
- Como resolver a URL, o formato do arquivo e o checksum esperado.
- Como extrair o pacote e localizar seus executáveis.
- Como validar a instalação, por exemplo executando o runtime com `--version`.

Um registro de artefato deve conter linguagem, versão, sistema, arquitetura, variante quando necessária, URL, formato, checksum e caminhos relativos dos executáveis. A listagem e a instalação devem usar o mesmo resolvedor: uma versão só pode aparecer como instalável se houver um artefato compatível.

Antes de alterar cada provedor, verificar as fontes atuais dos distribuidores. PHP usa pacotes oficiais NTS x64; Python usa CPython do projeto Astral python-build-standalone. Não assumir que todas as versões possuem pacotes portáteis para todas as plataformas.

| Provedor | Investigação necessária |
| --- | --- |
| Node.js | Confirmar índice oficial, pacotes por plataforma, arquivos de integridade e layout de Node/npm/npx. |
| PHP | Distribuição oficial Windows NTS x64, ZIP e SHA-256 do índice atual, com revisões de pacote normalizadas para a versão do runtime. |
| Python | CPython Astral python-build-standalone, variante install_only x86_64-pc-windows-msvc, tar.gz e SHA-256 do asset GitHub. |

No MVP, usar os provedores incluídos no executável. Catálogos personalizados, plugins externos e configuração de mirrors ficam para depois.

## Steps de implementação

### Step 1 — Definir os contratos e corrigir a base

- [x] Validar linguagem e versão antes de formar caminhos ou consultar provedores.
- [x] Rejeitar linguagens desconhecidas com uma mensagem que liste as suportadas.
- [x] Corrigir leitura e gravação TOML para preservar o arquivo anterior em caso de erro.
- [x] Manter a leitura do primeiro `.verman.toml` encontrado nos diretórios ancestrais e documentar que `use` grava no diretório atual.
- [x] Definir metadados de instalação concluída, incluindo versão, plataforma, origem e executáveis relativos.
- [x] Deixar de considerar diretórios vazios como instalações válidas.
- [x] Definir como representar o diretório dos executáveis por provedor; não impor `bin` a todos os pacotes.

Implementação: `internal/tool` valida nomes e versões estáveis `X.Y.Z`; `internal/config` prepara e fecha um arquivo temporário no mesmo diretório antes da substituição; `internal/installer` lê `installation.json` com schema 1, linguagem, versão, sistema, arquitetura, URL HTTPS de origem e mapa de executáveis relativos. Caminhos no mapa usam `/`, inclusive no Windows. Os diretórios de ativação são derivados desse mapa, sem exigir uma pasta `bin`.

Os metadados são um contrato estrutural, não uma prova de integridade ou execução. O instalador do Step 4 agora os publica após essas verificações e registra também o SHA-256 do pacote. Configurações preservam chaves desconhecidas, mas comentários e formatação não são preservados. Escritas concorrentes de configuração ainda não são coordenadas.

**Aceite:** entradas inválidas não escapam do diretório de instalações; TOML inválido não é sobrescrito; uma pasta vazia não é listada como instalação válida.

### Step 2 — Implementar a listagem local

- [x] Criar `verman list` e `verman list <linguagem>`.
- [x] Ler somente instalações válidas e verificar seus metadados e executáveis.
- [x] Ordenar versões numericamente, evitando ordenação textual como `9` antes de `22`.
- [x] Exibir a seleção do projeto separadamente do estado instalado.
- [x] Informar quando não houver instalações, inclusive quando a pasta base ainda não existir.

**Aceite:** `verman list node` e `verman list php` funcionam sem rede. A listagem informa o provedor de cada linguagem. Linguagens desconhecidas retornam erro.

Esclarecimento do contrato: consultar arquivos locais independe do download de runtimes. `--available` foi implementado no Step 3 para Node e ampliado nos Steps 7 e 8 para PHP e Python.

### Step 3 — Implementar o catálogo de versões disponíveis

- [x] Criar uma interface pequena de provedor para listar releases e resolver artefatos.
- [x] Implementar primeiro o catálogo de Node com fontes verificadas.
- [x] Adicionar `list <linguagem> --available` e `--refresh`, válido apenas com `--available`.
- [x] Filtrar os resultados por sistema e arquitetura atuais e excluir prévias no MVP.
- [x] Guardar cache separado das instalações, com fonte, plataforma, formato e data de consulta; usar validade inicial de uma hora.
- [x] Se a consulta falhar, permitir exibir cache antigo com aviso explícito; sem cache, retornar erro claro.
- [x] Tratar timeout, cancelamento, resposta HTTP inválida e alteração inesperada do formato do catálogo.

Implementação: índice oficial `https://nodejs.org/dist/index.json`, com filtro `win-x64-zip` e versões estáveis exatas. O cache schema 1 fica em `~/.verman/cache/node-windows-amd64.json`. Fonte, plataforma, data e artefatos são validados ao ler. URLs são reconstruídas pelo provedor, e o checksum é consultado em `SHASUMS256.txt` ao instalar. A consulta não baixa o checksum de todas as versões. Pré-releases, aliases e outras plataformas permanecem fora do MVP.

**Aceite:** as versões apresentadas possuem artefatos resolvíveis para a plataforma; atualizar o catálogo não modifica instalações nem a configuração do projeto.

### Step 4 — Instalar Node.js de verdade

- [x] Resolver a versão exata no mesmo provedor usado pela listagem.
- [x] Baixar para uma área temporária com timeout e tratamento de erros HTTP.
- [x] Verificar o checksum usando metadados obtidos da fonte confiável definida pelo provedor.
- [x] Extrair impedindo que caminhos ou links escapem do destino.
- [x] Preservar o layout necessário para Node, npm e npx funcionarem.
- [x] Executar a validação da versão antes de publicar a instalação.
- [x] Publicar os metadados de conclusão somente após sucesso e limpar arquivos temporários em falhas.
- [x] Proteger operações concorrentes sobre a mesma instalação e preservar uma instalação válida em caso de falha.
- [x] Fazer `install` repetido reconhecer uma instalação válida sem baixar novamente.

Implementação inicial: SHA-256 contra o manifesto oficial obtido por HTTPS; redirecionamentos ficam restritos à origem oficial. A verificação de assinatura PGP não foi implementada. O ZIP tem limites de tamanho e entradas; caminhos inseguros e links são rejeitados. Node, npm e npx são executados na área temporária com timeout antes da publicação. O lock original era um diretório por versão. A evolução descrita no Step 2 de [ROADMAP_PRODUTO.md](ROADMAP_PRODUTO.md) substituiu-o por locks nativos e journals recuperáveis, acrescentou metadados schema 2 e comandos verify/repair/recover. Os diretórios de lock legados continuam sendo recusados sem remoção automática.

Validação concluída em Windows/amd64: testes locais de cache, filtragem, falhas HTTP, checksum incorreto, download truncado, ZIP malicioso, validação do runtime e concorrência. O teste opt-in `TestOfficialNodeInstallation` baixou e validou Node 22.11.0 e 22.12.0, incluindo npm/npx e instalação repetida. Nenhum runtime do sistema ou configuração persistente de PATH foi alterado.

**Aceite:** instalar duas versões exatas de Node e executar cada uma pelo caminho completo retorna as versões esperadas. Um download interrompido não produz uma instalação listada como válida.

### Step 5 — Selecionar e ativar versões no PowerShell

- [x] Conectar `use` à instalação real e salvar a seleção apenas depois do sucesso.
- [x] Implementar `env --shell powershell`, com escape de caminhos e saída determinística.
- [x] Manter stdout reservado ao código de ativação e enviar diagnósticos para stderr.
- [x] Tratar seleção de versão ausente ou inválida com erro claro.
- [x] Controlar entradas adicionadas ao `PATH` para evitar duplicação e versões antigas acumuladas ao reaplicar o ambiente.
- [x] Documentar o comando que aplica o resultado de `env` à sessão; `use` sozinho não consegue alterar o processo pai.

Implementação: `internal/environment` produz um script PowerShell com caminhos codificados como JSON e passados em uma string literal escapada. O script usa `VERMAN_PATH_ENTRIES` para substituir as entradas gerenciadas da sessão, mantendo as demais entradas do PATH. Sem configuração, remove as entradas anteriormente gerenciadas. Uma seleção com instalação ausente ou inválida retorna erro antes de emitir código. Erros de `env` não imprimem texto de uso em stdout. PowerShell é o padrão no Windows; outros shells recebem erro explícito até o Step 9.

Validação: testes locais executam Windows PowerShell com caminhos contendo espaços, Unicode, aspas, dólar e símbolos. O teste opt-in `TestOfficialPowerShellWorkflow` compilou a CLI temporariamente, instalou Node 22.11.0/22.12.0 e confirmou `node`, `npm` e `npx` em dois projetos, além de reaplicação sem duplicação. A execução ocorreu em um processo filho com política de execução limitada àquele processo e não alterou o perfil nem variáveis persistentes do sistema.

**Aceite:** dois projetos selecionam versões diferentes e, após aplicar o ambiente de cada um, `node --version`, `npm --version` e a execução de `npx` funcionam. Caminhos com espaços são suportados.

### Step 6 — Remover instalações

- [x] Implementar `uninstall <linguagem> <versao>`.
- [x] Validar que o destino resolvido permanece dentro da pasta de instalações gerenciadas.
- [x] Coordenar remoção e instalação concorrentes.
- [x] Detectar seleção no projeto atual e explicar o impacto antes de remover; definir uma opção explícita para permitir esse caso.
- [x] Documentar que outros projetos podem referenciar a versão removida e que não haverá varredura de todo o disco.

Implementação: `uninstall` usa o mesmo lock por versão que a instalação. Valida entradas, caminho absoluto e resolvido, rejeita diretórios gerenciados redirecionados por links/junctions e exige instalação válida antes de remover. A exclusão é ancorada com `os.Root`, sem percorrer links para destinos externos. Se a versão estiver selecionada no projeto atual ou no arquivo herdado do pai, a operação exige `--allow-selected`. A opção permite a remoção e mantém o TOML; a seleção passa a referenciar uma instalação ausente. Pastas incompletas sem metadados válidos são recusadas, sem limpeza automática.

Validação: remoção de uma única versão, preservação da seleção e das demais instalações, bloqueios, cancelamento, recusa de entradas inseguras e links/junctions externos. O fluxo real no PowerShell também verificou recusa da versão selecionada, remoção de outra versão e remoção selecionada com a opção explícita. O comando não altera o PATH da sessão nem procura referências em outros projetos.

**Aceite:** a remoção afeta somente a versão informada, e a listagem local reflete o resultado.

### Step 7 — Adicionar PHP

- [x] Documentar a distribuição, plataformas, variantes e dependências suportadas.
- [x] Implementar catálogo e resolução de artefatos de PHP.
- [x] Reutilizar download, cache, integridade e publicação da infraestrutura comum.
- [x] Definir a configuração inicial de `php.ini` e o comportamento de extensões, sem alterar a configuração global da máquina.
- [x] Habilitar `list php`, `list php --available`, `install php`, `use php` e `uninstall php`.

Implementação: pacotes oficiais NTS x64 de `https://downloads.php.net/~windows/releases/releases.json`, destino atual do endereço antigo windows.php.net. O catálogo inclui as versões publicadas nesse índice, não todo o arquivo histórico. Revisões como `8.4.27-1` são registradas em `build`; o comando usa a versão PHP `8.4.27`. SHA-256 é reconsultado na fonte antes do download; variantes TS, x86 e prévias são excluídas. O runtime Visual C++ x64 compatível é pré-requisito, sem instalação automática global.

Cada instalação recebe um `php.ini` derivado de `php.ini-development` e uma pasta `conf.d`. Extensões opcionais ficam desabilitadas, disponíveis para ativação nesse arquivo. `extension_dir` aponta à própria instalação: no Windows usa o caminho curto quando disponível e codificação ANSI exata para o carregador de DLLs; caminhos sem representação segura são recusados. A ativação define `PHPRC` e `PHP_INI_SCAN_DIR` apenas na sessão e restaura os valores anteriores quando o projeto não seleciona PHP.

Validação real em Windows/amd64: PHP 8.5.12 e 8.4.27 (revisão -1), escolha por projeto, carregamento do php.ini e da extensão cURL por opção temporária, caminhos com espaços/acentos/aspas/símbolos e remoção das duas versões. O teste não habilita extensões na configuração global da máquina.

**Aceite:** duas versões de PHP podem ser instaladas, listadas e selecionadas por projeto; `php --version` confirma a seleção após ativar o ambiente.

### Step 8 — Adicionar Python

- [x] Documentar a distribuição escolhida, licenças, limitações e plataformas suportadas.
- [x] Implementar catálogo, resolução, instalação e validação de Python.
- [x] Verificar `python`, `pip` via `python -m pip` e criação de ambientes com `python -m venv`.
- [x] Documentar a relação entre a versão gerenciada pelo Verman e os ambientes virtuais de cada projeto.
- [x] Habilitar os mesmos comandos de listagem, instalação, seleção e remoção.

Implementação: CPython de Astral python-build-standalone, variante `install_only` para `x86_64-pc-windows-msvc`. O catálogo consulta o release mais recente do projeto, sem varrer todo o histórico. Exclui variantes debug, freethreaded, stripped, outras arquiteturas e prévias. O resolvedor consulta novamente o release pela tag de build para obter URL e SHA-256 confiáveis; o cache não autoriza URLs arbitrárias ou checksums locais. O arquivo tar.gz é extraído com limites e rejeição de links, dispositivos e caminhos inseguros.

Antes da publicação são verificados o runtime, `python -m pip`, criação/execução de venv e o pip do venv. Um wrapper relativo `pip.cmd` aponta para o próprio Python e continua válido após publicar a instalação. O ambiente de validação ignora configurações globais de Python/pip. Na ativação, `PYTHONHOME` e `PYTHONPATH` são limpos somente na sessão; os valores anteriores são restaurados ao sair do projeto gerenciado. O marcador `VERMAN_ENV_BACKUP` guarda os valores originais de variáveis gerenciadas, sem modificar o perfil ou o sistema.

Um venv pertence ao runtime usado ao criá-lo: mudar `.verman.toml` não migra um venv existente. Desative o venv antes de aplicar outra seleção e recrie-o quando quiser trocar sua versão base. Remover a instalação base pode quebrar venvs que dependem dela; o Verman não os procura no disco. Consulte as fontes e notas de licenças em [README.md](README.md); arquivos de licença das distribuições são preservados na extração.

Validação real em Windows/amd64: CPython 3.15.2 e 3.14.10, build 20261110, selecionados em dois projetos; pip por módulo e wrapper, venv criado depois da publicação, pip isolado, manutenção da versão do venv anterior, restauração das variáveis de sessão e remoção das instalações. Os testes locais usam catálogos HTTPS e arquivos ZIP/TAR controlados; o teste com rede é opt-in `TestOfficialPHPPythonWorkflow`.

**Aceite:** duas versões podem ser selecionadas por projeto e criar ambientes virtuais funcionais, sem modificar o Python do sistema.

### Step 9 — Expandir integração e distribuir

- [x] Implementar ativação para shells POSIX e testar nas plataformas anunciadas.
- [x] Definir e implementar integração de troca automática ao entrar e sair de projetos, restaurando o ambiente quando necessário.
- [x] Avaliar `verman exec` para executar um comando no ambiente do projeto sem depender de hooks.
- [x] Adicionar aliases como `latest` e `lts` onde fizerem sentido, gravando a versão exata resolvida no projeto.
- [x] Avaliar seleção global com precedência explícita em relação à configuração do projeto.
- [x] Criar builds e documentação de instalação, atualização e desinstalação do próprio Verman.

Escopo de plataforma: runtimes e pacote de distribuição continuam limitados a Windows/amd64. A ativação POSIX foi implementada para Bash/sh e validada no Git Bash/Git sh do Windows, usando `cygpath`; isso não anuncia instalação nativa de runtimes em Linux/macOS. Hooks são oferecidos para PowerShell e Bash 5+; sh usa ativação manual. Zsh/fish não são anunciados como suportados.

Hooks: `verman init --shell powershell|bash` imprime integração opcional, aplicada explicitamente pelo usuário, sem editar perfis. Antes de cada prompt, relê o projeto mais próximo e o fallback global. Não baixa runtimes nem executa scripts do projeto. Ao sair de um projeto, aplica a configuração efetiva seguinte; sem seleção, restaura o ambiente. Em erro de configuração/instalação ausente, limpa a ativação anterior para não manter outra versão silenciosamente. Preserva o prompt/callbacks existentes e evita instalar o hook duas vezes. `init --disable` remove o hook da sessão e limpa o ambiente. Use um executável permanente, nunca `go run`, para gerar hooks.

Execução: `verman exec -- <comando> [args...]` calcula o ambiente sem alterar o pai, resolve executáveis no PATH efetivo e encaminha stdin/stdout/stderr e código de saída. Não usa um shell implícito. Npm/npx são executados pelo Node gerenciado e pip por `python -m pip`; scripts batch arbitrários e builtins exigem que o usuário invoque seu shell explicitamente. Não instala automaticamente runtimes ausentes.

Aliases: `latest` escolhe a maior versão estável compatível do catálogo da linguagem; `lts` é exclusivo de Node e usa o metadado LTS do índice oficial, forçando sua atualização para compatibilidade com caches antigos. `install`, `use` e `global` aceitam os aliases; configurações continuam armazenando versões exatas. Não foram adicionados intervalos de versão. A disponibilidade continua limitada às fontes de cada provedor; latest não significa estabilidade de uma configuração que nunca foi gravada.

Global: `verman global <linguagem> <versao>` garante a instalação e salva `~/.verman/config.toml`; sem argumentos mostra a seleção. `global --unset <linguagem>` remove somente a seleção. O primeiro arquivo de projeto encontrado prevalece por linguagem; linguagens não declaradas nele usam o global, sem mesclar arquivos de projeto ancestrais. A seleção global também é protegida por `uninstall --allow-selected`. Global inválido é tratado como erro e não é sobrescrito.

Distribuição: `go run ./tools/release -version 0.1.0` gera `dist/verman_0.1.0_windows_amd64.zip` e `SHA256SUMS`, com versão embutida, documentação e scripts PowerShell. O build usa trimpath, timestamps fixos no ZIP e apenas a plataforma validada. A instalação e atualização verificam o checksum e substituem somente o executável no diretório bin informado, preservando o anterior em falhas de substituição. Não editam perfil ou PATH persistente. A desinstalação do aplicativo preserva runtimes/configurações. O pacote foi gerado localmente; não houve publicação externa.

Validação: testes offline com executável real da CLI para precedência global, hooks PowerShell/Bash, ativação em Git sh, caminhos com espaços/aspas/símbolos, erro de configuração com limpeza, preservação do prompt e código de saída/argumentos de exec. Os testes opt-in verificaram novamente Node/npm/npx em PowerShell e Git Bash, aliases no catálogo oficial e exec de PHP/Python/pip, além de instalação/venv/remoção existentes. O ZIP local 0.1.0 teve instalação, atualização, verificação de versão e desinstalação testadas em diretório temporário, preservando um arquivo alheio.

**Aceite:** instruções reproduzíveis permitem instalar o Verman em uma máquina limpa e usar apenas as plataformas e provedores efetivamente verificados.

## Validação por etapa

Após alterações de código, executar `gofmt` nos arquivos alterados, `go test ./...`, `go vet ./...` e a ajuda dos comandos afetados.

Usar diretórios temporários, catálogos de teste e servidores HTTP locais para testar falhas e integridade. Cobrir configuração inválida, versões incompatíveis, cache vencido, download interrompido, checksum incorreto, extração insegura e instalação incompleta. Não usar o diretório pessoal real nos testes automatizados.

As verificações reais de instalação devem ser feitas em ambiente isolado e registradas por sistema, arquitetura, linguagem e shell. Passar nos testes Go, sozinho, não comprova que os runtimes instalados funcionam.

## Primeiro marco entregue

O MVP foi validado com Node, PHP e Python em Windows/amd64. As etapas previstas estão concluídas dentro desse escopo. Expansão nativa para outros sistemas, outras linguagens e histórico completo dos catálogos exigem novos marcos e validação própria.
