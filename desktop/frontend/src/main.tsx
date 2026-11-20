import { names, descriptions, languages, logos } from "./tools";
import React, { useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, preview } from "./bridge";
import type {
  Catalog,
  Language,
  Operation,
  Request,
  Setup,
  Snapshot,
} from "./types";
import "./style.css";
import "./contrast.css";

const logo = new URL("./assets/verman-logo.png", import.meta.url).href;

const nav = [
  ["overview", "Visão geral", "grid"],
  ["languages", "Ferramentas", "layers"],
  ["projects", "Projetos", "folder"],
  ["operations", "Operações", "activity"],
  ["diagnostics", "Diagnóstico", "shield"],
  ["settings", "Configuração inicial", "settings"],
] as const;
type Page = (typeof nav)[number][0] | "docs";
const paths: Record<string, React.ReactNode> = {
  grid: (
    <>
      <rect x="3" y="3" width="7" height="7" rx="1.5" />
      <rect x="14" y="3" width="7" height="7" rx="1.5" />
      <rect x="3" y="14" width="7" height="7" rx="1.5" />
      <rect x="14" y="14" width="7" height="7" rx="1.5" />
    </>
  ),
  layers: (
    <>
      <path d="m12 3 10 5-10 5L2 8 12 3Z" />
      <path d="m2 12 10 5 10-5M2 16l10 5 10-5" />
    </>
  ),
  folder: (
    <path d="M3 7V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z" />
  ),
  activity: <path d="M2 12h5l3-8 4 16 3-8h5" />,
  shield: (
    <>
      <path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z" />
      <path d="m8 12 3 3 5-6" />
    </>
  ),
  settings: (
    <>
      <circle cx="12" cy="12" r="3" />
      <path d="m9 3 6 0 1 3 3 1 2 5-2 5-3 1-1 3H9l-1-3-3-1-2-5 2-5 3-1 1-3Z" />
    </>
  ),
  arrow: <path d="M5 12h14m-5-5 5 5-5 5" />,
  chevron: <path d="m9 5 7 7-7 7" />,
  plus: <path d="M12 5v14M5 12h14" />,
  terminal: (
    <>
      <rect x="2" y="4" width="20" height="16" rx="3" />
      <path d="m6 9 3 3-3 3m6 0h5" />
    </>
  ),
  book: (
    <>
      <path d="M12 5c-4-3-9-1-9-1v16s5-2 9 1c4-3 9-1 9-1V4s-5-2-9 1Z" />
      <path d="M12 5v16" />
    </>
  ),
  check: <path d="m5 12 4 4L19 6" />,
  refresh: (
    <>
      <path d="M20 7v5h-5M4 17v-5h5" />
      <path d="M6 6a8 8 0 0 1 14 6M18 18a8 8 0 0 1-14-6" />
    </>
  ),
  close: <path d="m6 6 12 12M6 18 18 6" />,
  search: (
    <>
      <circle cx="10" cy="10" r="7" />
      <path d="m15 15 6 6" />
    </>
  ),
  download: (
    <>
      <path d="M12 3v12m-5-5 5 5 5-5M4 16v5h16v-5" />
    </>
  ),
  alert: (
    <>
      <path d="m12 3 10 18H2L12 3Z" />
      <path d="M12 9v5m0 3v1" />
    </>
  ),
};
function Icon({ name, size = 19 }: { name: string; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.65"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {paths[name] || paths.grid}
    </svg>
  );
}
function Mark({ lang }: { lang: Language }) {
  return (
    <span className="langmark" aria-hidden="true">
      <img src={logos[lang]} alt="" />
    </span>
  );
}
const phaseName: Record<string, string> = {
  preparing: "Preparando",
  downloading: "Baixando",
  download_progress: "Baixando",
  extracting: "Extraindo pacote",
  validating: "Validando runtime",
  installed: "Instalado",
  already_installed: "Já instalado",
};
const actionName: Record<string, string> = {
  install: "Instalar",
  select: "Selecionar",
  repair: "Reparar",
  verify: "Verificar integridade",
  recover: "Recuperar",
  uninstall: "Remover",
};
const message = (error: unknown) =>
  error instanceof Error ? error.message : String(error);
const folderName = (path: string) =>
  path.split(/[\\/]/).filter(Boolean).at(-1) || path;
function App() {
  const [page, setPage] = useState<Page>("overview"),
    [project, setProject] = useState(
      () => localStorage.getItem("verman.project") || "",
    ),
    [saved, setSaved] = useState<string[]>(() => {
      try {
        return JSON.parse(localStorage.getItem("verman.projects") || "[]");
      } catch {
        return [];
      }
    });
  const [snapshot, setSnapshot] = useState<Snapshot>(),
    [ops, setOps] = useState<Operation[]>([]),
    [setup, setSetup] = useState<Setup>(),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [loading, setLoading] = useState(true);
  const [lang, setLang] = useState<Language>("node"),
    [catalog, setCatalog] = useState<Catalog>(),
    [catalogLoading, setCatalogLoading] = useState(false),
    [history, setHistory] = useState(false),
    [offline, setOffline] = useState(false),
    [filter, setFilter] = useState(""),
    [scope, setScope] = useState<"project" | "global">(
      project ? "project" : "global",
    );
  const [confirm, setConfirm] = useState<Request>(),
    [allowSelected, setAllowSelected] = useState(false),
    [docTopic, setDocTopic] = useState("PRIMEIROS_PASSOS"),
    [doc, setDoc] = useState(""),
    [docLoading, setDocLoading] = useState(false);
  const requestCounter = useRef(0),
    snapshotCounter = useRef(0),
    previousOps = useRef(""),
    dialogRef = useRef<HTMLDialogElement>(null),
    headingRef = useRef<HTMLHeadingElement>(null);
  const running = ops.find((o) => o.status === "running");
  async function refresh() {
    const counter = ++snapshotCounter.current;
    setLoading(true);
    try {
      const [data, info] = await Promise.all([
        api.Snapshot(project),
        api.SetupInfo(),
      ]);
      if (counter === snapshotCounter.current) {
        setSnapshot(data);
        setSetup(info);
      }
    } catch (e) {
      if (counter === snapshotCounter.current) {
        setSnapshot(undefined);
        setError(message(e));
      }
    } finally {
      if (counter === snapshotCounter.current) setLoading(false);
    }
  }
  useEffect(() => {
    void refresh();
    localStorage.setItem("verman.project", project);
    setScope(project ? "project" : "global");
  }, [project]);
  useEffect(() => {
    const timer = setInterval(() => {
      api
        .Operations()
        .then((value) => {
          setOps(value);
          const signature = value.map((o) => o.id + o.status).join();
          if (
            previousOps.current &&
            signature !== previousOps.current &&
            !value.some((o) => o.status === "running")
          ) {
            void refresh();
            if (page === "languages") void loadCatalog();
          }
          previousOps.current = signature;
        })
        .catch((e) => setError(message(e)));
    }, 600);
    return () => clearInterval(timer);
  }, [project, page, lang, history, offline]);
  useEffect(() => {
    headingRef.current?.focus();
  }, [page]);
  async function chooseProject() {
    try {
      const dir = await api.PickProject();
      if (!dir) return;
      setError("");
      setProject(dir);
      const next = [dir, ...saved.filter((p) => p !== dir)].slice(0, 15);
      setSaved(next);
      localStorage.setItem("verman.projects", JSON.stringify(next));
    } catch (e) {
      setError(message(e));
    }
  }
  async function loadCatalog(refreshRemote = false) {
    const request = ++requestCounter.current;
    setError("");
    setCatalogLoading(true);
    setCatalog(undefined);
    try {
      const result = await api.Catalog(lang, history, refreshRemote, offline);
      if (request === requestCounter.current) setCatalog(result);
    } catch (e) {
      if (request === requestCounter.current) setError(message(e));
    } finally {
      if (request === requestCounter.current) setCatalogLoading(false);
    }
  }
  useEffect(() => {
    if (page === "languages") void loadCatalog();
  }, [lang, history, offline, page]);
  useEffect(() => {
    if (page !== "docs") return;
    let active = true;
    setDocLoading(true);
    api
      .Documentation(docTopic)
      .then((content) => {
        if (active) setDoc(content);
      })
      .catch((e) => setError(message(e)))
      .finally(() => {
        if (active) setDocLoading(false);
      });
    return () => {
      active = false;
    };
  }, [page, docTopic]);
  useEffect(() => {
    const dialog = dialogRef.current;
    if (confirm) dialog?.showModal();
    else dialog?.close();
    setAllowSelected(false);
  }, [confirm]);
  async function perform(r: Request) {
    try {
      setError("");
      await api.Start(r);
      setOps(await api.Operations());
      setPage("operations");
      setConfirm(undefined);
      setNotice("Operação iniciada. Você pode acompanhar o progresso aqui.");
    } catch (e) {
      setError(message(e));
      setConfirm(undefined);
    }
  }
  function request(action: string, version: string) {
    const r: Request = {
      action,
      language: lang,
      version,
      directory: project || snapshot?.directory || "",
      scope,
      allow_selected: false,
    };
    if (action === "uninstall" || action === "repair" || action === "select")
      setConfirm(r);
    else void perform(r);
  }
  async function terminal() {
    try {
      await api.OpenTerminal(project);
      setNotice("Terminal aberto com o ambiente deste projeto.");
    } catch (e) {
      setError(message(e));
    }
  }
  const showDocs = (topic = "PRIMEIROS_PASSOS") => {
    setDocTopic(topic);
    setPage("docs");
  };
  const installedCount =
    snapshot?.tools.reduce((sum, t) => sum + t.installed.length, 0) || 0;
  const issues =
    snapshot?.diagnosis.tools.filter(
      (t) =>
        t.status === "activation_required" ||
        t.status === "installation_invalid",
    ).length || 0;
  const current = snapshot?.tools.find((t) => t.language === lang);
  const versions =
    catalog?.versions.filter((v) => v.version.includes(filter)) || [];
  return (
    <div className="app">
      <a className="skip" href="#main">
        Ir para o conteúdo
      </a>
      <aside className="sidebar">
        <a
          className="brand"
          aria-label="Verman — visão geral"
          href="#overview"
          onClick={(e) => {
            e.preventDefault();
            setPage("overview");
          }}
        >
          <img className="brand-symbol" src={logo} alt="" width="42" height="42" />
          <span>
            verman<span className="brand-desktop">DESKTOP</span>
          </span>
        </a>
        <div className="workspace-label">SEU WORKSPACE</div>
        <nav aria-label="Navegação principal">
          {nav.map(([id, label, icon]) => (
            <button
              key={id}
              aria-current={page === id ? "page" : undefined}
              className={page === id ? "navitem active" : "navitem"}
              onClick={() => {
                setError("");
                setPage(id);
              }}
            >
              <Icon name={icon} />
              {label}
              {id === "operations" && running && <span className="live-dot" />}
              {id === "diagnostics" && issues > 0 && (
                <span className="nav-count">{issues}</span>
              )}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="sidebar-guide">
            <span className="guide-icon">
              <Icon name="terminal" size={23} />
            </span>
            <h3>
              CLI + desktop.
              <br />
              Um só ambiente.
            </h3>
            <p>
              Escolha pela interface.
              <br />
              Continue no seu terminal.
            </p>
            <button onClick={() => showDocs()}>
              Ver primeiros passos <Icon name="arrow" size={15} />
            </button>
          </div>
          <button
            className={"navitem " + (page === "docs" ? "active" : "")}
            onClick={() => showDocs()}
          >
            <Icon name="book" />
            Documentação
          </button>
          <div className="sidebar-footer">
            <span className="status-dot" /> Windows / amd64{" "}
            <span className="version">{setup?.version || "0.4.0"}</span>
          </div>
        </div>
      </aside>
      <div className="shell">
        <header className="topbar">
          <div className="breadcrumb">
            Workspace <Icon name="chevron" size={13} />{" "}
            <span>{nav.find((n) => n[0] === page)?.[1] || "Documentação"}</span>
          </div>
          <div className="topbar-actions">
            <span className="local-label">
              <span className="status-dot" />{" "}
              {preview ? "Prévia interativa" : "Ambiente local"}
            </span>
            <button className="project-picker" onClick={chooseProject}>
              <Icon name="folder" size={17} />
              <span>
                {project ? folderName(project) : "Selecionar projeto"}
              </span>
              <Icon name="chevron" size={13} />
            </button>
          </div>
        </header>
        <main id="main" className="main" tabIndex={-1}>
          {preview && (
            <div className="preview-note">
              Prévia com dados de demonstração. Instalações e alterações são
              simuladas; use o aplicativo Windows para gerenciar sua máquina.
            </div>
          )}
          {error && (
            <div className="notice error" role="alert">
              <Icon name="alert" />
              <span>{error}</span>
              <button aria-label="Fechar erro" onClick={() => setError("")}>
                <Icon name="close" />
              </button>
            </div>
          )}
          {notice && (
            <div className="notice success" role="status">
              <Icon name="check" />
              <span>{notice}</span>
              <button
                aria-label="Fechar mensagem"
                onClick={() => setNotice("")}
              >
                <Icon name="close" />
              </button>
            </div>
          )}
          <div className="page-heading">
            <div>
              <div className="eyebrow">
                {page === "overview"
                  ? "MENOS CONFIGURAÇÃO. MAIS CÓDIGO."
                  : "VERMAN WORKSPACE"}
              </div>
              <h1 ref={headingRef} tabIndex={-1}>
                {
                  {
                    overview: "Seu ambiente, sob controle.",
                    languages: "A versão certa para cada projeto.",
                    projects: "Projetos com a sua configuração.",
                    operations: "Tudo o que está acontecendo.",
                    diagnostics: "Encontre o caminho certo.",
                    settings: "Pronto para começar?",
                    docs: "Documentação, sempre por perto.",
                  }[page]
                }
              </h1>
              <p>
                {
                  {
                    overview:
                      "Todas as suas ferramentas. Um único lugar para gerenciar.",
                    languages: "Instale uma vez. Alterne quando precisar.",
                    projects:
                      "Cada pasta pode ter suas próprias versões, sem interferir nas outras.",
                    operations:
                      "Acompanhe downloads, validação e alterações no seu ambiente.",
                    diagnostics:
                      "Compare a versão selecionada com o que o terminal encontra.",
                    settings:
                      "Conecte a CLI, escolha um projeto e comece a desenvolver.",
                    docs: "Guias locais para usar o Verman, mesmo sem conexão.",
                  }[page]
                }
              </p>
            </div>
            <div className="heading-actions">
              {page === "overview" || page === "projects" ? (
                <button className="primary" onClick={chooseProject}>
                  <Icon name="plus" />
                  Adicionar projeto
                </button>
              ) : page === "diagnostics" ? (
                <button
                  className="button"
                  disabled={loading}
                  onClick={() => void refresh()}
                >
                  <Icon name="refresh" />
                  Verificar novamente
                </button>
              ) : null}
            </div>
          </div>
          {page === "overview" && (
            <>
              <section className="stats" aria-label="Resumo do ambiente">
                <div>
                  <span className="stat-icon green">
                    <Icon name="layers" />
                  </span>
                  <div>
                    <span className="stat-value">
                      {languages.length} <small>ferramentas</small>
                    </span>
                    <p>Runtimes, compiladores e gerenciadores</p>
                  </div>
                </div>
                <div>
                  <span className="stat-icon blue">
                    <Icon name="download" />
                  </span>
                  <div>
                    <span className="stat-value">
                      {loading ? "—" : installedCount}{" "}
                      <small>versões instaladas</small>
                    </span>
                    <p>Disponíveis neste computador</p>
                  </div>
                </div>
                <div>
                  <span className={"stat-icon " + (issues ? "amber" : "green")}>
                    <Icon name="shield" />
                  </span>
                  <div>
                    <span className="stat-value">
                      {issues || "Tudo certo"}{" "}
                      <small>{issues ? "ajuste no ambiente" : ""}</small>
                    </span>
                    <p>
                      {issues
                        ? "Confira o diagnóstico do PATH"
                        : "Seu ambiente está pronto"}
                    </p>
                  </div>
                  <button
                    className="icon-btn"
                    aria-label="Abrir diagnóstico"
                    onClick={() => setPage("diagnostics")}
                  >
                    <Icon name="arrow" />
                  </button>
                </div>
              </section>
              <div className="section-heading">
                <div>
                  <h2>
                    Suas ferramentas <span className="count">{languages.length}</span>
                  </h2>
                  <p>Versões independentes, projetos em harmonia.</p>
                </div>
                <button
                  className="text-btn"
                  onClick={() => setPage("languages")}
                >
                  Explorar versões <Icon name="arrow" size={16} />
                </button>
              </div>
              <section
                className="runtime-grid"
                aria-label="Ferramentas instaladas"
              >
                {languages.map((language) => {
                  const t = snapshot?.tools.find(
                    (t) => t.language === language,
                  );
                  return (
                    <article className="runtime-card" key={language}>
                      <div className="runtime-top">
                        <Mark lang={language} />
                        <span className="runtime-status">
                          <span className="status-dot" />
                          {t?.installed.length ? "Instalado" : "Disponível"}
                        </span>
                      </div>
                      <h3>{names[language]}</h3>
                      <p>{descriptions[language]}</p>
                      <div className="selected-runtime">
                        <span>
                          {project && t?.project_version
                            ? "VERSÃO DO PROJETO"
                            : "PADRÃO GLOBAL"}
                        </span>
                        <strong>
                          {loading
                            ? "Carregando…"
                            : (project && t?.project_version) ||
                              t?.global_version ||
                              "Não selecionada"}
                        </strong>
                        <span
                          className={
                            "badge " +
                            (project && t?.project_version
                              ? "project"
                              : "neutral")
                          }
                        >
                          {project && t?.project_version ? "Projeto" : "Global"}
                        </span>
                      </div>
                      <div className="card-bottom">
                        <span>
                          {t?.installed.length || 0} versão(ões) instalada(s)
                        </span>
                        <button
                          aria-label={"Gerenciar " + names[language]}
                          onClick={() => {
                            setLang(language);
                            setPage("languages");
                          }}
                        >
                          Gerenciar <Icon name="arrow" size={16} />
                        </button>
                      </div>
                    </article>
                  );
                })}
              </section>
              <section className="project-banner">
                <div className="project-banner-icon">
                  <Icon name="folder" size={27} />
                </div>
                <div>
                  <span className="eyebrow">
                    {project ? "PROJETO ATUAL" : "CADA PROJETO, SEU AMBIENTE"}
                  </span>
                  <h2>
                    {project
                      ? folderName(project)
                      : "Seu próximo projeto começa aqui."}
                  </h2>
                  <p className={project ? "mono" : ""}>
                    {project ||
                      "Selecione uma pasta para definir versões só para ela."}
                  </p>
                </div>
                <button
                  className="button"
                  onClick={project ? () => setPage("projects") : chooseProject}
                >
                  {project ? "Ver configuração" : "Selecionar uma pasta"}
                  <Icon name="arrow" size={16} />
                </button>
              </section>
              <div className="overview-bottom">
                <section className="panel">
                  <div className="panel-title">
                    <h2>Atividade recente</h2>
                    <button
                      className="text-btn"
                      onClick={() => setPage("operations")}
                    >
                      Ver tudo <Icon name="arrow" size={15} />
                    </button>
                  </div>
                  {ops.length ? (
                    ops
                      .slice(0, 2)
                      .map((op) => <OperationRow key={op.id} op={op} />)
                  ) : (
                    <div className="empty-inline">
                      <span className="empty-icon">
                        <Icon name="activity" />
                      </span>
                      <div>
                        <strong>Um novo começo.</strong>
                        <p>Suas instalações e alterações vão aparecer aqui.</p>
                      </div>
                    </div>
                  )}
                </section>
                <section className="panel quickstart">
                  <div className="panel-title">
                    <h2>Dois comandos. Tudo pronto.</h2>
                    <span className="badge neutral">GUIA RÁPIDO</span>
                  </div>
                  <code>
                    <span>$</span> verman use node lts
                    <br />
                    <span>$</span> node --version
                  </code>
                  <button className="text-btn" onClick={() => showDocs()}>
                    Conheça o fluxo completo <Icon name="arrow" size={15} />
                  </button>
                </section>
              </div>
            </>
          )}
          {page === "languages" && (
            <>
              <p className="tool-description">{descriptions[lang]}</p>
              <div
                className="language-tabs"
                role="tablist"
                aria-label="Linguagem"
                onKeyDown={(event) => {
                  if (
                    !["ArrowLeft", "ArrowRight", "Home", "End"].includes(
                      event.key,
                    )
                  )
                    return;
                  event.preventDefault();
                  
                  const index =
                    event.key === "Home"
                      ? 0
                      : event.key === "End"
                        ? languages.length - 1
                        : (languages.indexOf(lang) +
                            (event.key === "ArrowRight" ? 1 : languages.length - 1)) %
                          languages.length;
                  setLang(languages[index]);
                  setFilter("");
                  document
                    .getElementById("language-tab-" + languages[index])
                    ?.focus();
                }}
              >
                {languages.map((value) => (
                  <button
                    key={value}
                    role="tab"
                    id={"language-tab-" + value}
                    aria-controls="language-panel"
                    tabIndex={lang === value ? 0 : -1}
                    aria-selected={lang === value}
                    className={lang === value ? "active" : ""}
                    onClick={() => {
                      setLang(value);
                      setFilter("");
                    }}
                  >
                    <Mark lang={value} />
                    {names[value]}
                    <span className="count">
                      {snapshot?.tools.find((t) => t.language === value)
                        ?.installed.length || 0}
                    </span>
                  </button>
                ))}
              </div>
              <section
                className="panel catalog"
                id="language-panel"
                role="tabpanel"
                aria-labelledby={"language-tab-" + lang}
              >
                <div className="catalog-toolbar">
                  <label className="search">
                    <Icon name="search" />
                    <input
                      aria-label="Filtrar versão"
                      placeholder="Buscar uma versão…"
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                    />
                  </label>
                  <label className="check-label">
                    <input
                      type="checkbox"
                      checked={history}
                      onChange={(e) => setHistory(e.target.checked)}
                    />
                    Incluir histórico
                  </label>
                  <label className="check-label">
                    <input
                      type="checkbox"
                      checked={offline}
                      onChange={(e) => setOffline(e.target.checked)}
                    />
                    Offline
                  </label>
                  <button
                    className="button small"
                    disabled={catalogLoading || offline}
                    onClick={() => void loadCatalog(true)}
                  >
                    <Icon name="refresh" size={16} />
                    Atualizar
                  </button>
                </div>
                <div className="catalog-scope">
                  <span>Aplicar seleção em:</span>
                  <div className="segmented">
                    <button
                      aria-pressed={scope === "project"}
                      disabled={!project}
                      className={scope === "project" ? "active" : ""}
                      onClick={() => setScope("project")}
                    >
                      Projeto atual
                    </button>
                    <button
                      aria-pressed={scope === "global"}
                      className={scope === "global" ? "active" : ""}
                      onClick={() => setScope("global")}
                    >
                      Padrão global
                    </button>
                  </div>
                  <span className="scope-description">
                    {scope === "project"
                      ? folderName(project)
                      : "Usado quando o projeto não define a linguagem."}
                  </span>
                </div>
                {catalog?.warning && (
                  <div className="catalog-warning" role="status">
                    {catalog.warning}
                  </div>
                )}
                {catalogLoading ? (
                  <div className="empty-state" role="status">
                    <span className="spinner" />
                    Consultando versões de {names[lang]}…
                  </div>
                ) : catalog ? (
                  <>
                    <div className="table-wrap">
                      <table>
                        <thead>
                          <tr>
                            <th>Versão</th>
                            <th>Distribuição / build</th>
                            <th>Estado</th>
                            <th className="align-right">Ações</th>
                          </tr>
                        </thead>
                        <tbody>
                          {versions.map((v) => (
                            <tr key={v.version}>
                              <td>
                                <strong className="mono">{v.version}</strong>
                                {(scope === "project"
                                  ? current?.project_version
                                  : current?.global_version) === v.version && (
                                  <span className="badge project">
                                    Selecionada
                                  </span>
                                )}
                              </td>
                              <td>
                                <span>{v.variant || v.policy.provider}</span>
                                <small className="mono">
                                  {v.build || "Build oficial"}
                                </small>
                              </td>
                              <td>
                                {!v.policy.installable ? (
                                  <span
                                    className="badge warning"
                                    title={v.policy.unavailable_reason}
                                  >
                                    Sem checksum oficial
                                  </span>
                                ) : v.installed ? (
                                  <span className="badge ready">Instalada</span>
                                ) : (
                                  <span className="badge neutral">
                                    Disponível
                                  </span>
                                )}
                                {v.policy.support === "eol" && (
                                  <span className="badge warning">EOL</span>
                                )}
                              </td>
                              <td>
                                <div className="table-actions">
                                  {v.installed ? (
                                    <>
                                      <button
                                        className="button small"
                                        disabled={!!running}
                                        onClick={() =>
                                          request("select", v.version)
                                        }
                                      >
                                        Usar{" "}
                                        {scope === "project"
                                          ? "no projeto"
                                          : "global"}
                                      </button>
                                      <details>
                                        <summary
                                          aria-label={
                                            "Mais ações para " + v.version
                                          }
                                        >
                                          •••
                                        </summary>
                                        <div className="menu">
                                          {[
                                            "verify",
                                            "repair",
                                            "recover",
                                            "uninstall",
                                          ].map((action) => (
                                            <button
                                              key={action}
                                              onClick={() =>
                                                request(action, v.version)
                                              }
                                              disabled={!!running}
                                            >
                                              {actionName[action]}
                                            </button>
                                          ))}
                                        </div>
                                      </details>
                                    </>
                                  ) : (
                                    <button
                                      className="button small"
                                      disabled={
                                        !!running || !v.policy.installable
                                      }
                                      title={v.policy.unavailable_reason}
                                      onClick={() =>
                                        request("install", v.version)
                                      }
                                    >
                                      <Icon name="download" size={15} />
                                      Instalar
                                    </button>
                                  )}
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    {versions.length === 0 && (
                      <div className="empty-state">
                        Nenhuma versão encontrada. Ajuste a busca ou atualize o
                        catálogo.
                      </div>
                    )}
                    <div className="catalog-footer">
                      {versions.length} versões ·{" "}
                      {catalog.from_cache ? "Cache local" : "Catálogo remoto"} ·{" "}
                      {new Date(catalog.fetched_at).toLocaleString("pt-BR")}
                      <span>A disponibilidade pode mudar na origem.</span>
                    </div>
                  </>
                ) : (
                  <div className="empty-state">
                    Não foi possível carregar o catálogo. Verifique o erro acima
                    e tente novamente.
                    <button
                      className="button"
                      onClick={() => void loadCatalog()}
                    >
                      Tentar novamente
                    </button>
                  </div>
                )}
              </section>
              <p className="footnote">
                PHP sem SHA-256 oficial não é instalável. O histórico de Python
                consulta até 30 releases. Disponibilidade não significa
                recomendação de segurança.
              </p>
            </>
          )}
          {page === "projects" && (
            <>
              <section className="panel">
                <div className="panel-title">
                  <h2>
                    {project
                      ? "Projeto selecionado"
                      : "Escolha seu primeiro projeto"}
                  </h2>
                  {project && (
                    <button className="button" onClick={terminal}>
                      <Icon name="terminal" />
                      Abrir terminal
                    </button>
                  )}
                </div>
                {project ? (
                  <>
                    <div className="project-detail">
                      <Icon name="folder" size={32} />
                      <div>
                        <h3>{folderName(project)}</h3>
                        <p className="mono">{project}</p>
                        <small>
                          Configuração:{" "}
                          {snapshot?.config_path ||
                            "Ainda não existe .verman.toml nesta pasta."}
                        </small>
                      </div>
                    </div>
                    <div className="project-tools">
                      {snapshot?.tools.map((t) => (
                        <div key={t.language}>
                          <Mark lang={t.language} />
                          <div>
                            <strong>{names[t.language]}</strong>
                            <p className="mono">
                              {t.project_version ||
                                t.global_version ||
                                "Não selecionada"}
                            </p>
                          </div>
                          <span
                            className={
                              "badge " +
                              (t.project_version ? "project" : "neutral")
                            }
                          >
                            {t.project_version
                              ? "Projeto"
                              : t.global_version
                                ? "Fallback global"
                                : "Sem seleção"}
                          </span>
                          <button
                            className="text-btn"
                            onClick={() => {
                              setLang(t.language);
                              setPage("languages");
                            }}
                          >
                            Alterar <Icon name="arrow" size={15} />
                          </button>
                        </div>
                      ))}
                    </div>
                  </>
                ) : (
                  <div className="empty-state">
                    <Icon name="folder" size={38} />
                    <h3>Uma pasta. Um ambiente independente.</h3>
                    <p>
                      Defina Runtimes, compiladores e gerenciadores sem alterar outros projetos.
                    </p>
                    <button className="primary" onClick={chooseProject}>
                      Selecionar pasta
                    </button>
                  </div>
                )}
              </section>
              {saved.length > 0 && (
                <section className="panel recent-projects">
                  <div className="panel-title">
                    <h2>Pastas recentes</h2>
                  </div>
                  {saved.map((dir) => (
                    <button
                      key={dir}
                      className="recent-project"
                      onClick={() => setProject(dir)}
                    >
                      <Icon name="folder" />
                      <span>
                        <strong>{folderName(dir)}</strong>
                        <small className="mono">{dir}</small>
                      </span>
                      {dir === project ? (
                        <span className="badge project">Atual</span>
                      ) : (
                        <Icon name="arrow" />
                      )}
                    </button>
                  ))}
                </section>
              )}
              <div className="info-note">
                <Icon name="book" />
                <p>
                  O Verman lê o primeiro <code>.verman.toml</code> até a raiz e
                  completa linguagens ausentes com o padrão global. Salvar em
                  uma subpasta pode criar uma seleção local.
                </p>
              </div>
            </>
          )}
          {page === "operations" && (
            <section className="panel">
              <div className="panel-title">
                <h2>Histórico desta sessão</h2>
                <span className="badge neutral">{ops.length} operações</span>
              </div>
              {ops.length ? (
                ops.map((op) => (
                  <div className="operation-detail" key={op.id}>
                    <OperationRow op={op} />
                    {op.status === "running" && (
                      <>
                        <progress
                          max={op.total || 100}
                          value={op.total ? op.bytes : undefined}
                          aria-label={
                            "Progresso de " + names[op.request.language]
                          }
                        />
                        <div className="progress-meta">
                          <span>
                            {phaseName[op.phase] || "Processando"}
                            {op.total > 0
                              ? " · " +
                                (op.bytes / 1048576).toFixed(1) +
                                " / " +
                                (op.total / 1048576).toFixed(1) +
                                " MiB"
                              : ""}
                          </span>
                          <button
                            className="text-btn danger"
                            onClick={() =>
                              api
                                .Cancel(op.id)
                                .catch((e) => setError(message(e)))
                            }
                          >
                            Cancelar operação
                          </button>
                        </div>
                      </>
                    )}
                    {op.message && (
                      <p
                        className={
                          op.status === "failed" ? "operation-error" : ""
                        }
                      >
                        {op.message}
                      </p>
                    )}
                    {op.hint && <p className="footnote">{op.hint}</p>}
                  </div>
                ))
              ) : (
                <div className="empty-state">
                  <Icon name="activity" size={38} />
                  <h3>Nenhuma operação por enquanto.</h3>
                  <p>
                    Instale uma linguagem ou selecione uma versão para começar.
                  </p>
                  <button
                    className="primary"
                    onClick={() => setPage("languages")}
                  >
                    Explorar ferramentas
                  </button>
                </div>
              )}
              <div className="catalog-footer">
                Até 50 operações por sessão. A CLI e o desktop compartilham os
                mesmos locks de instalação.
              </div>
            </section>
          )}
          {page === "diagnostics" && (
            <>
              {snapshot?.diagnosis.warnings?.map((warning, i) => (
                <div className="notice warning" key={i}>
                  <Icon name="alert" />
                  {warning}
                </div>
              ))}
              {snapshot?.diagnosis.tools.map((d) => (
                <section className="panel diagnosis" key={d.language}>
                  <div className="diagnosis-heading">
                    <Mark lang={d.language} />
                    <div>
                      <h2>{names[d.language]}</h2>
                      <p>
                        {d.version
                          ? `${d.version} · ${d.source === "project" ? "seleção do projeto" : "padrão global"}`
                          : "Nenhuma versão selecionada"}
                      </p>
                    </div>
                    <span
                      className={
                        "badge " +
                        (d.path_matches || d.uses_shim
                          ? "ready"
                          : d.version
                            ? "warning"
                            : "neutral")
                      }
                    >
                      {d.uses_shim
                        ? "Shim pronto"
                        : d.path_matches
                          ? "Caminho corresponde"
                          : d.status === "not_selected"
                            ? "Sem seleção"
                            : "Ajuste necessário"}
                    </span>
                  </div>
                  <dl>
                    <div>
                      <dt>Executável esperado</dt>
                      <dd>
                        {d.expected_executable ||
                          "Selecione uma versão para comparar."}
                      </dd>
                    </div>
                    <div>
                      <dt>Encontrado no PATH</dt>
                      <dd>{d.path_executable || "Não encontrado"}</dd>
                    </div>
                  </dl>
                  {d.detail && (
                    <div className="diagnosis-action">
                      <Icon name="alert" />
                      <p>{d.detail}</p>
                      <button
                        className="button small"
                        onClick={() => setPage("settings")}
                      >
                        Como configurar
                      </button>
                    </div>
                  )}
                </section>
              ))}
              <div className="info-note">
                <Icon name="shield" />
                <p>
                  Este diagnóstico é offline e não executa runtimes. Aliases e
                  funções do terminal não são inspecionados; no PowerShell,
                  confira <code>Get-Command php -All</code>.
                </p>
              </div>
            </>
          )}
          {page === "settings" && (
            <>
              <section className="panel onboarding">
                <div className="panel-title">
                  <h2>Seu primeiro ambiente em 3 passos</h2>
                  <span className="badge ready">SEM CONTA</span>
                </div>
                <div className="setup-step">
                  <span
                    className={
                      "step-number " + (setup?.cli_exists ? "done" : "")
                    }
                  >
                    {setup?.cli_exists ? <Icon name="check" /> : "1"}
                  </span>
                  <div>
                    <h3>Instale a CLI e conecte o PATH</h3>
                    <p>
                      O instalador pode colocar os shims antes das instalações
                      externas. Isso é feito uma vez, com a opção de
                      configuração do PATH.
                    </p>
                    <div className="setup-checks">
                      {languages.map(
                        (value) => (
                          <span
                            key={value}
                            className={
                              "badge " +
                              (setup?.shims[value] ? "ready" : "warning")
                            }
                          >
                            {names[value]}:{" "}
                            {setup?.shims[value]
                              ? "shim encontrado"
                              : "verificar PATH"}
                          </span>
                        ),
                      )}
                    </div>
                    <p className="mono tiny">
                      CLI: {setup?.cli_path || "Carregando…"}
                    </p>
                    <button className="text-btn" onClick={() => showDocs()}>
                      Abrir guia de configuração <Icon name="arrow" size={15} />
                    </button>
                  </div>
                </div>
                <div className="setup-step">
                  <span className={"step-number " + (project ? "done" : "")}>
                    {project ? <Icon name="check" /> : "2"}
                  </span>
                  <div>
                    <h3>Escolha uma pasta e suas versões</h3>
                    <p>
                      Uma seleção local vale só para o projeto. O padrão global
                      completa as linguagens que ele não definiu.
                    </p>
                    <button className="button" onClick={chooseProject}>
                      <Icon name="folder" />
                      Selecionar projeto
                    </button>
                  </div>
                </div>
                <div className="setup-step">
                  <span className="step-number">3</span>
                  <div>
                    <h3>Abra um terminal e comece a criar</h3>
                    <p>
                      Após a configuração do PATH, reabra o editor e os
                      terminais uma vez. As próximas trocas pelo Verman já valem
                      na próxima execução dos shims.
                    </p>
                    <div className="code-block">
                      <code>
                        verman use php latest
                        <br />
                        php -v
                        <br />
                        verman doctor php
                      </code>
                    </div>
                    <button
                      className="primary"
                      disabled={!project}
                      onClick={terminal}
                    >
                      <Icon name="terminal" />
                      Abrir terminal do projeto
                    </button>
                  </div>
                </div>
              </section>
              <div className="info-note">
                <Icon name="alert" />
                <p>
                  Um venv Python existente continua ligado à versão base com que
                  foi criado. Desative o venv antes de trocar de interpretador e
                  recrie-o quando necessário.
                </p>
              </div>
            </>
          )}
          {page === "docs" && (
            <>
              <div className="doc-tabs">
                {[
                  ["PRIMEIROS_PASSOS", "Primeiros passos"],
                  ["SUPORTE", "Suporte"],
                  ["CLI_JSON", "CLI e JSON"],
                  ["PROVEDORES", "Provedores"],
                  ["FERRAMENTAS", "Ferramentas"],
                  ["DESKTOP", "Desktop e instalação"],
                ].map(([topic, label]) => (
                  <button
                    className={
                      "button " + (docTopic === topic ? "selected" : "")
                    }
                    key={topic}
                    onClick={() => setDocTopic(topic)}
                  >
                    {label}
                  </button>
                ))}
              </div>
              <article className="panel markdown">
                {docLoading ? (
                  <div className="empty-state" role="status">
                    Carregando documentação…
                  </div>
                ) : (
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={{
                      a: ({ href, children }) => (
                        <a
                          href={href}
                          onClick={(e) => {
                            e.preventDefault();
                            const topic = href
                              ?.split("/")
                              .pop()
                              ?.replace(".md", "");
                            if (
                              topic &&
                              [
                                "PRIMEIROS_PASSOS",
                                "SUPORTE",
                                "CLI_JSON",
                                "PROVEDORES",
                                "FERRAMENTAS",
                                "DESKTOP",
                              ].includes(topic)
                            )
                              setDocTopic(topic);
                            else setNotice("Referência externa: " + href);
                          }}
                        >
                          {children}
                        </a>
                      ),
                    }}
                  >
                    {doc}
                  </ReactMarkdown>
                )}
              </article>
            </>
          )}
          <footer className="main-footer">
            <span>Feito para quem quer desenvolver.</span>
            <span>Seus projetos. Suas versões. Sua máquina.</span>
          </footer>
        </main>
      </div>
      <dialog
        ref={dialogRef}
        onCancel={() => setConfirm(undefined)}
        aria-labelledby="confirm-title"
      >
        <div className="dialog-heading">
          <span className="eyebrow">CONFIRMAR ALTERAÇÃO</span>
          <button
            className="icon-btn"
            aria-label="Fechar confirmação"
            onClick={() => setConfirm(undefined)}
          >
            <Icon name="close" />
          </button>
        </div>
        <h2 id="confirm-title">
          {confirm && actionName[confirm.action]}{" "}
          {confirm && names[confirm.language]} {confirm?.version}?
        </h2>
        <p>
          {confirm?.action === "uninstall"
            ? "A instalação será removida deste computador. Outros projetos e venvs podem depender dela. Suas seleções não serão apagadas."
            : confirm?.action === "repair"
              ? "O pacote será baixado e validado novamente. A instalação anterior será preservada até a publicação da substituta."
              : "A seleção será salva " +
                (confirm?.scope === "project"
                  ? "na pasta do projeto atual."
                  : "como padrão global. Projetos com seleção própria continuam com prioridade.")}
        </p>
        {confirm?.action === "uninstall" && (
          <label className="check-label">
            <input
              type="checkbox"
              checked={allowSelected}
              onChange={(e) => setAllowSelected(e.target.checked)}
            />
            Permitir remoção mesmo se estiver selecionada.
          </label>
        )}
        <div className="dialog-actions">
          <button className="button" onClick={() => setConfirm(undefined)}>
            Voltar
          </button>
          <button
            className={
              confirm?.action === "uninstall"
                ? "primary destructive"
                : "primary"
            }
            disabled={!!running}
            onClick={() =>
              confirm && perform({ ...confirm, allow_selected: allowSelected })
            }
          >
            Confirmar {confirm && actionName[confirm.action].toLowerCase()}
          </button>
        </div>
      </dialog>
    </div>
  );
}
function OperationRow({ op }: { op: Operation }) {
  return (
    <div className="operation-row">
      <span className={"operation-icon " + op.status}>
        <Icon
          name={
            op.status === "succeeded"
              ? "check"
              : op.status === "failed"
                ? "alert"
                : "download"
          }
        />
      </span>
      <div>
        <strong>
          {actionName[op.request.action]} {names[op.request.language]}{" "}
          <span className="mono">{op.request.version}</span>
        </strong>
        <p>
          {op.status === "running"
            ? phaseName[op.phase] || "Processando"
            : op.status === "succeeded"
              ? "Concluído"
              : op.status === "canceled"
                ? "Cancelado"
                : "Não foi possível concluir"}{" "}
          ·{" "}
          {new Date(op.started).toLocaleTimeString("pt-BR", {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      </div>
      <span
        className={
          "badge " +
          (op.status === "succeeded"
            ? "ready"
            : op.status === "failed"
              ? "warning"
              : "neutral")
        }
      >
        {op.status === "running"
          ? "Em andamento"
          : op.status === "succeeded"
            ? "Concluído"
            : op.status === "canceled"
              ? "Cancelado"
              : "Falhou"}
      </span>
    </div>
  );
}
createRoot(document.getElementById("root")!).render(<App />);
