import type { Language } from "./types";

export const names = {
  node: "Node.js", php: "PHP", python: "Python", bun: "Bun", go: "Go", dotnet: ".NET",
  deno: "Deno", git: "Git", rust: "Rust", composer: "Composer", yarn: "Yarn",
  pnpm: "pnpm", vcpkg: "vcpkg", chocolatey: "Chocolatey",
} satisfies Record<Language, string>;
export const languages = Object.keys(names) as Language[];

export const logos = {
  bun: new URL("./assets/tools/bun-logo.svg", import.meta.url).href,
  chocolatey: new URL("./assets/tools/chocolatey-logo.svg", import.meta.url).href,
  composer: new URL("./assets/tools/composer-logo.png", import.meta.url).href,
  deno: new URL("./assets/tools/deno-logo.svg", import.meta.url).href,
  dotnet: new URL("./assets/tools/dotnet-logo.svg", import.meta.url).href,
  git: new URL("./assets/tools/git-logo.svg", import.meta.url).href,
  go: new URL("./assets/tools/go-logo.svg", import.meta.url).href,
  node: new URL("./assets/tools/node-logo.svg", import.meta.url).href,
  php: new URL("./assets/tools/php-logo.svg", import.meta.url).href,
  pnpm: new URL("./assets/tools/pnpm-logo.svg", import.meta.url).href,
  python: new URL("./assets/tools/python-logo.svg", import.meta.url).href,
  rust: new URL("./assets/tools/rust-logo.svg", import.meta.url).href,
  vcpkg: new URL("./assets/tools/vcpkg-logo.svg", import.meta.url).href,
  yarn: new URL("./assets/tools/yarn-logo.svg", import.meta.url).href,
} satisfies Record<Language, string>;

export const descriptions: Record<Language, string> = {
  node: "JavaScript com npm e npx.", php: "PHP NTS x64, com configuração local.",
  python: "CPython com pip e venv.", bun: "JavaScript com Bun e bunx.",
  go: "Compilador Go e gofmt. Toolchain local.", deno: "JavaScript e TypeScript com Deno.",
  dotnet: "SDK .NET com o runtime correspondente. Respeita o global.json do projeto.",
  git: "MinGit portátil. Sem Git Bash ou interface gráfica.",
  rust: "Rust, Cargo e rustup isolados. Linker C++ instalado separadamente.",
  composer: "Gerenciador PHP. Selecione o PHP antes de instalar.",
  yarn: "Yarn moderno. Selecione Node e respeite o packageManager do projeto.",
  pnpm: "Gerenciador JavaScript em distribuição nativa.",
  vcpkg: "Bibliotecas C++. Builds exigem Git e compiladores à parte.",
  chocolatey: "CLI isolada. Instalar pacotes pode exigir administrador.",
};
