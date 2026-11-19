# Contrato da CLI — schema 2

`--json` é suportado por `list`, `install`, `use`, `global`, `doctor`, `setup`, `uninstall`, `verify`, `repair`, `recover` e `support`. Cada execução emite **um documento JSON** em stdout, inclusive em erro. Progresso e avisos ficam em stderr. `env`, `init` e `exec` recusam essa opção para preservar seus protocolos de shell/processo. Help, completion e `--version` mantêm a apresentação própria do Cobra.

```json
{
  "schema": 2,
  "command": "use",
  "success": true,
  "data": {
    "language": "php",
    "version": "8.5.12",
    "scope": "project",
    "config_path": "D:\\app\\.verman.toml",
    "executable": "C:\\Users\\example\\.verman\\installs\\php\\8.5.12\\php.exe",
    "session_changed": false
  },
  "error": null
}
```

Em falha, `success=false` e `error` contém `code`, `message` e `hint`. `data` é null, exceto quando há resultado útil: `doctor` conserva o diagnóstico em `data` mesmo ao retornar código 1. Mensagens em português podem evoluir; scripts devem usar `schema`, `success` e `error.code`.

| Comando | Conteúdo de `data` |
| --- | --- |
| `list [linguagem]` | Array de `language`, `installed` (array), `project_version`, `global_version`; seleções ausentes são strings vazias |
| `list --available` | `language`, `catalog_result` (catálogo, data/origem, `from_cache`, `stale`, aviso opcional), `versions` |
| `versions[]` | Artefato, `installed`, `installed_build` opcional, `different_build`, `distribution` com provedor/confiança, `installable`, motivo de bloqueio, estado de suporte, EOL/origem/data da política |
| `install`, `repair` | Metadados de instalação; `schema_version` é independente do schema da CLI |
| `use`, `global linguagem versão` | Seleção exata, escopo, arquivo e executável; `session_changed=false` |
| `global` | Objeto `tools` com seleções globais |
| `global --unset` | `language`, `scope=global`, `unset=true` |
| `doctor` | Diagnóstico com seleção, origem, caminhos, status, `issues`, `warnings` e ambiente virtual opcional |
| `setup` | `directory`, `path_changed=false`; não modifica o PATH persistente |
| `verify`, `recover`, `uninstall` | Linguagem, versão e booleano `verified`, `recovered` ou `removed` |
| `support` | Versão do Verman, plataforma, presença de ambiente virtual e ferramentas/status; omite caminhos |

Campos novos podem ser acrescentados no mesmo schema; consumidores devem ignorar campos desconhecidos. Mudanças incompatíveis exigem outro número. Ordem de propriedades não faz parte do contrato. O catálogo interno usa schema 1 e os metadados de instalação usam schema próprio.

**Migração do doctor:** antes o relatório schema 1 ocupava a raiz. Agora o envelope é schema 2 e o relatório está em `data`. No PowerShell:

```powershell
$response = (verman doctor php --json | Out-String | ConvertFrom-Json)
$response.data.tools
```

| Saída | Categoria |
| --- | --- |
| 0 | Sucesso |
| 1 | Falha geral de operação/IO ou `diagnostic_issue` |
| 2 | `invalid_argument`, `unsupported_platform` |
| 3 | `network_error`, `http_error`, `rate_limited`, `offline` |
| 4 | `integrity_error`, `integrity_unavailable`, `artifact_unverified` |
| 5 | `operation_busy` (lock nativo ocupado) |
| 6 | `configuration_invalid` (TOML inválido) |
| 130 | `canceled` (cancelamento reconhecido) |

`exec` e os shims preservam o código do programa filho. Erros ainda não especializados, permissões, arquivo ocupado detectado pelo Windows e estados ambíguos usam `operation_failed`/1, com contexto. Não interprete qualquer código 4 de um programa filho como erro de integridade do Verman.

`--progress auto` apresenta bytes/MiB e fases em stderr quando ele é um terminal. Em pipes, `auto` omite atualizações. `always` escreve linhas de progresso também em pipes; `never` as desabilita. As mensagens de sucesso e avisos continuam disponíveis. Downloads emitem eventos limitados a aproximadamente quatro atualizações por segundo, além do resultado final. Ctrl+C solicita cancelamento e o journal preserva ou recupera a instalação anterior conforme a fase; encerramento forçado usa `recover` na próxima operação.
