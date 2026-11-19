import type { Language } from "./types";

export const names = {
  node: "Node.js", php: "PHP", python: "Python", bun: "Bun", go: "Go",
  deno: "Deno", git: "Git", rust: "Rust", composer: "Composer", yarn: "Yarn",
  pnpm: "pnpm", vcpkg: "vcpkg", chocolatey: "Chocolatey",
} satisfies Record<Language, string>;
export const languages = Object.keys(names) as Language[];
export const descriptions: Record<Language, string> = {
  node: "JavaScript com npm e npx.", php: "PHP NTS x64, com configuração local.",
  python: "CPython com pip e venv.", bun: "JavaScript com Bun e bunx.",
  go: "Compilador Go e gofmt. Toolchain local.", deno: "JavaScript e TypeScript com Deno.",
  git: "MinGit portátil. Sem Git Bash ou interface gráfica.",
  rust: "Rust, Cargo e rustup isolados. Linker C++ instalado separadamente.",
  composer: "Gerenciador PHP. Selecione o PHP antes de instalar.",
  yarn: "Yarn moderno. Selecione Node e respeite o packageManager do projeto.",
  pnpm: "Gerenciador JavaScript em distribuição nativa.",
  vcpkg: "Bibliotecas C++. Builds exigem Git e compiladores à parte.",
  chocolatey: "CLI isolada. Instalar pacotes pode exigir administrador.",
};
