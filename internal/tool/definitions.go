package tool

type Definition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	Dependency  string   `json:"dependency,omitempty"`
}

func Definitions() []Definition {
	return []Definition{
		{"node", "Node.js", "runtime", "JavaScript com npm e npx.", []string{"node", "npm", "npx"}, ""},
		{"php", "PHP", "runtime", "Interpretador PHP NTS x64.", []string{"php"}, ""},
		{"python", "Python", "runtime", "CPython com pip e venv.", []string{"python", "pip"}, ""},
		{"bun", "Bun", "runtime", "Runtime JavaScript e gerenciador de pacotes.", []string{"bun", "bunx"}, ""},
		{"go", "Go", "toolchain", "Compilador Go e gofmt.", []string{"go", "gofmt"}, ""},
		{"dotnet", ".NET", "toolchain", "SDK .NET com o runtime correspondente.", []string{"dotnet"}, ""},
		{"deno", "Deno", "runtime", "Runtime JavaScript e TypeScript.", []string{"deno"}, ""},
		{"git", "Git", "tool", "MinGit portátil para controle de versão.", []string{"git"}, ""},
		{"rust", "Rust", "toolchain", "Rust, Cargo e rustup em diretórios isolados.", []string{"rust", "rustc", "rustdoc", "cargo", "rustup"}, ""},
		{"composer", "Composer", "package_manager", "Dependências PHP; usa o PHP selecionado.", []string{"composer"}, "php"},
		{"yarn", "Yarn", "package_manager", "Yarn moderno; usa o Node selecionado.", []string{"yarn", "yarnpkg"}, "node"},
		{"pnpm", "pnpm", "package_manager", "Gerenciador de pacotes JavaScript.", []string{"pnpm"}, ""},
		{"vcpkg", "vcpkg", "package_manager", "Ferramentas e registro de bibliotecas C++.", []string{"vcpkg"}, ""},
		{"chocolatey", "Chocolatey", "system_tool", "CLI isolada; pacotes podem exigir administrador.", []string{"chocolatey", "choco"}, ""},
	}
}
func Lookup(id string) (Definition, bool) {
	for _, d := range Definitions() {
		if d.ID == id {
			return d, true
		}
	}
	return Definition{}, false
}
func Owner(command string) (Definition, bool) {
	for _, d := range Definitions() {
		for _, name := range d.Commands {
			if command == name {
				return d, true
			}
		}
	}
	return Definition{}, false
}
