import Alpine from "@alpinejs/csp";
import { animateContent, initMotion } from "./motion.js";

Alpine.data("siteHeader", () => ({
  open: false,
  toggleMenu() {
    this.open = !this.open;
  },
  closeMenu() {
    this.open = false;
  },
}));

Alpine.data("toolTerminal", () => ({
  active: "node",
  init() {
    this.select("node");
  },
  select(id) {
    const item = this.$root.querySelector(`[data-tool="${id}"]`);
    if (!item) return;
    const changed = this.active !== id;
    this.active = id;
    const dependency = item.dataset.dependency;
    const commands = [
      ...(dependency
        ? [
            `verman use ${dependency} ${dependency === "node" ? "lts" : "latest"}`,
          ]
        : []),
      `verman list ${id} --available`,
      `verman use ${id} ${id === "node" ? "lts" : "latest"}`,
      item.dataset.command,
    ];
    this.$root.querySelector("[data-terminal-code]").textContent =
      commands.join("\n");
    this.$root.querySelector("[data-tool-description]").textContent =
      item.dataset.description;
    this.$root.querySelectorAll("[data-tool]").forEach((button) => {
      const selected = button.dataset.tool === id;
      button.setAttribute("aria-selected", String(selected));
      button.tabIndex = selected ? 0 : -1;
    });
    if (changed)
      animateContent(this.$root.querySelector(".terminal-body"), {
        distance: 8,
        duration: 280,
      });
  },
  chooseTool(event) {
    this.select(event.currentTarget.dataset.tool);
  },
  keyboard(event) {
    const buttons = [...this.$root.querySelectorAll("[data-tool]")];
    const current = buttons.indexOf(document.activeElement);
    if (
      current === -1 ||
      !["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)
    )
      return;
    event.preventDefault();
    const next =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? buttons.length - 1
          : (current + (event.key === "ArrowRight" ? 1 : -1) + buttons.length) %
            buttons.length;
    this.select(buttons[next].dataset.tool);
    buttons[next].focus();
  },
}));

Alpine.data("desktopGallery", () => ({
  current: 0,
  screens: [
    {
      src: "/screenshots/visao-geral.png",
      alt: "Visão geral real do Verman no Windows, mostrando as ferramentas e seleções do ambiente.",
      caption: "Seu ambiente, visto de perto.",
      label: "Visão geral",
    },
    {
      src: "/screenshots/ferramentas.png",
      alt: "Tela real de ferramentas do Verman com catálogo de versões do Node.js.",
      caption: "Encontre a versão que seu projeto precisa.",
      label: "Ferramentas",
    },
    {
      src: "/screenshots/documentacao.png",
      alt: "Documentação de primeiros passos aberta no aplicativo Windows do Verman.",
      caption: "Um guia ao seu lado, mesmo offline.",
      label: "Documentação",
    },
  ],
  get screen() {
    return this.screens[this.current];
  },
  chooseScreen(event) {
    const next = Number(event.currentTarget.dataset.screen);
    if (next === this.current) return;
    this.current = next;
    this.$nextTick(() =>
      animateContent(this.$root.querySelector(".desktop-frame img"), {
        distance: 10,
        duration: 380,
      }),
    );
  },
  openScreenshot() {
    const dialog = document.getElementById("screenshot-dialog");
    const image = dialog.querySelector("img");
    image.src = this.screen.src;
    image.alt = this.screen.alt;
    dialog.showModal();
  },
  closeScreenshot() {
    document.getElementById("screenshot-dialog").close();
  },
  backdrop(event) {
    if (event.target === event.currentTarget) event.currentTarget.close();
  },
}));

Alpine.data("docsNav", () => ({
  open: false,
  query: "",
  get sidebarClass() {
    return this.open ? "is-open" : "";
  },
  toggle() {
    this.open = !this.open;
  },
  close() {
    this.open = false;
  },
  filter() {
    const query = this.query
      .normalize("NFD")
      .replace(/[\u0300-\u036f]/g, "")
      .toLowerCase();
    let count = 0;
    this.$root.querySelectorAll("[data-doc-link]").forEach((link) => {
      const text = link.textContent
        .normalize("NFD")
        .replace(/[\u0300-\u036f]/g, "")
        .toLowerCase();
      link.hidden = !text.includes(query);
      if (!link.hidden) count++;
    });
    this.$root.querySelectorAll(".docs-nav-group").forEach((group) => {
      group.hidden = ![...group.querySelectorAll("[data-doc-link]")].some(
        (link) => !link.hidden,
      );
    });
    this.$root.querySelector("[data-no-results]").hidden = count > 0;
  },
}));

let toastTimeout;
async function copyText(text) {
  const status = document.getElementById("copy-status");
  try {
    await navigator.clipboard.writeText(text.trim());
    status.textContent = "Copiado. Agora é só colar no terminal.";
  } catch {
    status.textContent =
      "Não foi possível copiar. Selecione o comando e use Ctrl+C.";
  }
  status.classList.add("visible");
  clearTimeout(toastTimeout);
  toastTimeout = setTimeout(() => status.classList.remove("visible"), 3000);
}
document.addEventListener("click", (event) => {
  const button = event.target.closest("[data-copy]");
  if (!button) return;
  const code = button.closest("[data-code-block]").querySelector("code");
  if (code) copyText(code.innerText);
});
document.querySelectorAll(".doc-prose pre").forEach((pre) => {
  pre.setAttribute("data-code-block", "");
  const button = document.createElement("button");
  button.type = "button";
  button.className = "code-copy";
  button.dataset.copy = "";
  button.textContent = "Copiar";
  button.setAttribute("aria-label", "Copiar comando");
  pre.append(button);
});
Alpine.start();
initMotion();
