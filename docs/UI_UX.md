# UI/UX do Verman Desktop

## Estrutura e linguagem visual

Sidebar fixa para orientação, superfícies claras com separação por bordas, verde como ação principal e cores de cada linguagem. Tipografia de sistema e fontes monoespaçadas para versões/caminhos, sem fontes remotas. A visão geral concentra linguagens, instalações e diagnóstico; ações de manutenção ficam no catálogo e na sessão de operações.

Rótulos distintos evitam confundir **instalada**, **seleção do projeto**, **padrão global** e **encontrada no PATH**. Alterar uma seleção não é apresentado como alteração do terminal pai. A interface usa estados vazios, carregamento, erro recuperável, item bloqueado sem checksum, progresso, cancelamento solicitado e resultado final.

## Fluxos implementados

| Intenção | Caminho na interface | Efeito verificável |
| --- | --- | --- |
| Instalar | Linguagens → linguagem → Instalar | Download, checksum, validação, publicação; sem mudar seleção |
| Selecionar local | Selecionar projeto → linguagem → Projeto atual → Usar → Confirmar | Versão exata em `.verman.toml` da pasta |
| Selecionar global | Linguagens → Padrão global → Usar → Confirmar | Fallback em `~/.verman/config.toml` |
| Diagnosticar | Diagnóstico → Verificar novamente | Comparação offline de seleção, instalação, PATH e shims |
| Remover | Versão instalada → Mais ações → Remover → Confirmar | Recusa seleção/arquivo ocupado por padrão; exceção de seleção exige checkbox explícito |
| Cancelar | Operações → Cancelar operação | Solicita cancelamento do contexto; mantém estado em andamento até o núcleo terminar |
| Trabalhar | Projetos → Abrir terminal | Novo PowerShell com ambiente preparado; outros terminais não são alterados |
| Começar | Configuração inicial → guia offline | CLI/PATH/projeto/terminal com comandos concretos e dependências |

O catálogo permite filtrar, incluir histórico, atualizar e usar cache offline. A seleção por projeto exige uma pasta escolhida. Diálogos nativos de pasta evitam pedir digitação de caminhos; pastas recentes ficam somente no armazenamento local do WebView.

## Acessibilidade e revisão local

Navegação semântica com landmark/nav/main, link para pular ao conteúdo, botões com rótulos, mensagens status/alert, `<dialog>` modal com foco, tabela com cabeçalhos, progresso nativo e foco visível. As abas de linguagem têm navegação por setas/Home/End e associação com o painel. O layout responde a telas menores e respeita redução de movimento.

Foi feita revisão no Chrome da prévia e inspeção da árvore de acessibilidade do WebView2 nativo, que confirmou a ponte real e a documentação offline. O contraste encontrado na primeira versão foi corrigido; auditorias automatizadas WCAG A/AA nos estados revisados não substituem teste manual com leitor de tela. Os testes Go cobrem seleção compartilhada com CLI, preservação de outro projeto, progresso/cancelamento, exclusividade por janela e lease da CLI impedindo remoção pelo desktop.

## Pesquisa com usuários pendente

O protótipo funcional já permite revisar os fluxos. Ainda não houve uma sessão de usabilidade com usuários. Antes de consolidar a interface, peça a participantes para:

1. Instalar PHP, selecionar uma versão em um projeto e explicar a diferença entre instalação e seleção.
2. Criar outro projeto com versão diferente e identificar qual delas o terminal usará.
3. Diagnosticar um PHP externo que tem prioridade no PATH e encontrar a instrução correta.
4. Cancelar uma instalação e interpretar o estado final.
5. Tentar remover uma versão selecionada e explicar o impacto.

Registrar conclusão sem ajuda, erros, dúvidas e comportamento com teclado/leitor de tela. Não marcar o critério de pesquisa como concluído apenas por navegação automatizada. As verificações de UAC/WebView2 ausente/instalação/atualização em Windows limpo também permanecem pendentes.
