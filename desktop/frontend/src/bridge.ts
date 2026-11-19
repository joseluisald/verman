import { languages, names, descriptions } from "./tools";
import type { Bridge, Snapshot, Operation, Language, Request } from "./types";
export const preview = !window.go?.main?.App;
const installed: Record<Language, string[]> = {
  node: ["22.12.0", "22.11.0"],
  php: ["8.5.12"],
  python: ["3.14.10"], bun: [], go: [], deno: [], git: [], rust: [], composer: [], yarn: [], pnpm: [], vcpkg: [], chocolatey: [],
};
const local: Partial<Record<Language, string>> = {
  node: "22.12.0",
  php: "8.5.12",
};
const global: Partial<Record<Language, string>> = {
  node: "22.11.0",
  python: "3.14.10",
};
const operations: Operation[] = [];
const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
const demo: Bridge = {
  async Snapshot(dir) {
    return {
      directory: dir || "C:\\Projetos\\meu-app",
      config_path: dir ? dir + "\\.verman.toml" : "",
      tools: (Object.keys(installed) as Language[]).map((language) => ({
        language,
        installed: [...installed[language]],
        project_version: dir ? local[language] || "" : "",
        global_version: global[language] || "",
      })),
      diagnosis: {
        issues: true,
        warnings: [],
        tools: (Object.keys(installed) as Language[]).map((language) => ({
          language,
          version: local[language] || global[language],
          source: local[language] ? "project" : "global",
          installed: true,
          path_matches: language !== "php",
          uses_shim: language !== "php",
          status: language === "php" ? "activation_required" : "shim_ready",
          path_executable:
            language === "php"
              ? "D:\\PHP\\php-8.3.7\\php.exe"
              : "C:\\verman\\bin\\" + language + ".exe",
          expected_executable:
            "C:\\Users\\voce\\.verman\\installs\\" +
            language +
            "\\" +
            (local[language] || global[language]) +
            "\\" +
            language +
            ".exe",
          detail:
            language === "php"
              ? "Há um PHP externo antes dos shims. Configure o PATH e reabra o terminal uma vez."
              : "",
        })),
      },
    } as Snapshot;
  },
  async Catalog(language, history) {
    await wait(220);
    const samples: Record<Language, string[]> = {
      node: ["24.0.0", "22.12.0", "22.11.0"],
      php: ["8.5.12", "8.4.27", "8.3.1"],
      python: ["3.15.2", "3.14.10", "3.15.1"], bun: ["1.4.4"], go: ["1.27.4"], deno: ["2.9.8"], git: ["2.56.1"], rust: ["1.92.0"], composer: ["2.11.2"], yarn: ["4.18.1"], pnpm: ["12.21.0"], vcpkg: ["2026.10.29"], chocolatey: ["2.7.5"],
    };
    return {
      language,
      versions: samples[language]
        .filter((_, i) => history || i < 2)
        .map((version) => ({
          version,
          build: version,
          variant: language === "php" ? "nts-vs17-x64" : "",
          installed: installed[language].includes(version),
          policy: {
            provider:
              language === "python"
                ? "Astral python-build-standalone"
                : "Distribuição oficial",
            trust: "publisher_sha256_https",
            installable: version !== "8.3.1",
            unavailable_reason:
              version === "8.3.1" ? "SHA-256 oficial ausente" : undefined,
            support: "supported_branch",
            policy_source: "",
            policy_reviewed: "2026-11-18",
          },
        })),
      fetched_at: new Date().toISOString(),
      warning: "",
      from_cache: false,
      stale: false,
    };
  },
  async Start(request: Request) {
    if (operations.some((o) => o.status === "running"))
      throw Error("Aguarde ou cancele a operação atual.");
    const op: Operation = {
      id: crypto.randomUUID(),
      request,
      status: "running",
      phase: "downloading",
      bytes: 0,
      total: 30 * 1024 * 1024,
      message: "",
      started: new Date().toISOString(),
    };
    operations.unshift(op);
    const timer = setInterval(() => {
      if (op.status !== "running") {
        clearInterval(timer);
        return;
      }
      op.bytes = Math.min(op.bytes + 5 * 1024 * 1024, op.total);
      if (op.bytes === op.total) {
        op.status = "succeeded";
        op.finished = new Date().toISOString();
        if (request.action === "uninstall")
          installed[request.language] = installed[request.language].filter(
            (v) => v !== request.version,
          );
        else if (request.action === "install" || request.action === "select") {
          if (!installed[request.language].includes(request.version))
            installed[request.language].push(request.version);
          if (request.action === "select")
            (request.scope === "project" ? local : global)[request.language] =
              request.version;
        }
        clearInterval(timer);
      }
    }, 300);
    return op.id;
  },
  async Operations() {
    return structuredClone(operations);
  },
  async Cancel(id) {
    const op = operations.find((o) => o.id === id);
    if (op) {
      op.status = "canceled";
      op.finished = new Date().toISOString();
    }
  },
  async PickProject() {
    return "C:\\Projetos\\meu-app";
  },
  async SetupInfo() {
    return {
      version: "0.5.4 · prévia",
      cli_path: "C:\\verman\\verman.exe",
      cli_exists: true,
      shims: Object.fromEntries(languages.map(language => [language, language !== "php"])) as Record<Language, boolean>,
      message: "Reabra terminais e editor depois de configurar o PATH.",
    };
  },
  async Documentation(topic) {
    if (topic === "FERRAMENTAS") return "# Ferramentas disponíveis\n\nPrévia com dados simulados. O aplicativo Windows usa os serviços reais.\n\n" + languages.map(language => `- **${names[language]}**: ${descriptions[language]}`).join("\n") + "\n\n```powershell\nverman list bun --available\nverman use bun latest\nbun --version\n```";
    return "# Primeiros passos\n\nEsta é uma prévia interativa, com dados de demonstração. O aplicativo Windows usa o estado real da sua máquina.\n\n## 1. Configure o PATH\n\nNo instalador, marque a configuração do PATH. Reabra os terminais e o editor uma vez.\n\n## 2. Escolha um projeto\n\nSelecione a pasta do seu projeto e escolha as versões de cada linguagem.\n\n```powershell\nverman use php latest\nphp -v\nverman doctor php\n```\n\n## 3. Abra um terminal\n\nO botão Abrir terminal prepara o ambiente do processo novo. Outros terminais já abertos não são alterados.";
  },
  async OpenTerminal() {
    throw Error(
      "A abertura de terminal está disponível no aplicativo Windows. A prévia não executa comandos na sua máquina.",
    );
  },
};
export const api: Bridge = window.go?.main?.App || demo;
