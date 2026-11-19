package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/environment"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/tool"
)

type ExecutionPlan struct {
	Executable             string
	Arguments, Environment []string
	Directory              string
}

func ExecutionLease(ctx context.Context, dir string) (func(), error) {
	cfg, err := config.LoadEffectiveConfig(dir)
	if err != nil {
		return nil, err
	}
	languages := []string{}
	for language := range cfg.Tools {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	leases := []func(){}
	release := func() {
		for i := len(leases) - 1; i >= 0; i-- {
			leases[i]()
		}
	}
	for _, language := range languages {
		lease, err := installer.Lease(ctx, language, cfg.Tools[language])
		if err != nil {
			release()
			return nil, err
		}
		leases = append(leases, lease)
	}
	return release, nil
}

func (s *Service) PlanExecution(dir string, args, environ []string) (ExecutionPlan, error) {
	if len(args) == 0 || args[0] == "" {
		return ExecutionPlan{}, fmt.Errorf("informe um comando para executar")
	}
	childEnv, err := environment.Compute(dir, environ)
	if err != nil {
		return ExecutionPlan{}, err
	}
	binary, arguments, err := s.ResolveCommand(dir, args, childEnv)
	if err != nil {
		return ExecutionPlan{}, err
	}
	return ExecutionPlan{Executable: binary, Arguments: arguments, Environment: childEnv, Directory: dir}, nil
}

func (s *Service) ResolveCommand(dir string, args, environ []string) (string, []string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", nil, fmt.Errorf("informe um comando para executar")
	}
	name := args[0]
	if runtime.GOOS == "windows" && !strings.ContainsAny(name, `/\`) {
		name = strings.ToLower(name)
	}
	cfg, err := config.LoadEffectiveConfig(dir)
	if err != nil {
		return "", nil, err
	}
	if name == "npm" || name == "npx" || name == "pip" {
		language := "node"
		if name == "pip" {
			language = "python"
		}
		if version := cfg.Tools[language]; version != "" {
			root, err := config.InstallDir(language, version)
			if err != nil {
				return "", nil, err
			}
			if name == "pip" {
				return filepath.Join(root, "python.exe"), append([]string{"-m", "pip"}, args[1:]...), nil
			}
			return filepath.Join(root, "node.exe"), append([]string{filepath.Join(root, "node_modules", "npm", "bin", name+"-cli.js")}, args[1:]...), nil
		}
		binary, err := FindExecutable(language, environ)
		if err != nil {
			return "", nil, err
		}
		if name == "pip" {
			return binary, append([]string{"-m", "pip"}, args[1:]...), nil
		}
		cli := filepath.Join(filepath.Dir(binary), "node_modules", "npm", "bin", name+"-cli.js")
		if info, err := os.Stat(cli); err != nil || !info.Mode().IsRegular() {
			return "", nil, fmt.Errorf("%s não encontrado ao lado do Node do sistema; selecione uma versão com global/use", name)
		}
		return binary, append([]string{cli}, args[1:]...), nil
	}
	if definition, ok := tool.Owner(name); ok {
		if version := cfg.Tools[definition.ID]; version != "" {
			meta, err := installer.ReadMetadata(definition.ID, version)
			if err != nil {
				return "", nil, err
			}
			root, err := config.InstallDir(definition.ID, version)
			if err != nil {
				return "", nil, err
			}
			relative := meta.Executables[name]
			if relative == "" {
				return "", nil, fmt.Errorf("comando %s ausente na instalação", name)
			}
			binary := filepath.Join(root, filepath.FromSlash(relative))
			arguments := args[1:]
			if name == "bunx" {
				arguments = append([]string{"x"}, arguments...)
			}
			if definition.Dependency != "" {
				dependencyVersion := cfg.Tools[definition.Dependency]
				if dependencyVersion == "" {
					return "", nil, fmt.Errorf("%s exige %s selecionado; execute verman use %s latest", name, definition.Dependency, definition.Dependency)
				}
				dependency, err := installer.ReadMetadata(definition.Dependency, dependencyVersion)
				if err != nil {
					return "", nil, err
				}
				dependencyRoot, err := config.InstallDir(definition.Dependency, dependencyVersion)
				if err != nil {
					return "", nil, err
				}
				if definition.ID == "yarn" {
					if err := checkYarnPin(dir, version); err != nil {
						return "", nil, err
					}
				}
				return filepath.Join(dependencyRoot, filepath.FromSlash(dependency.Executables[definition.Dependency])), append([]string{binary}, arguments...), nil
			}
			return binary, arguments, nil
		}
	}
	if strings.ContainsAny(name, `/\`) || filepath.IsAbs(name) {
		return name, args[1:], nil
	}
	binary, err := FindExecutable(name, environ)
	return binary, args[1:], err
}

func checkYarnPin(dir, version string) error {
	for {
		path := filepath.Join(dir, "package.json")
		file, err := os.Open(path)
		if err == nil {
			var pkg struct {
				PackageManager string `json:"packageManager"`
			}
			data, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
			closeErr := file.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			if len(data) > 1<<20 {
				return fmt.Errorf("package.json excede 1 MiB")
			}
			if err := json.Unmarshal(data, &pkg); err != nil {
				return fmt.Errorf("ler package.json: %w", err)
			}
			if strings.HasPrefix(pkg.PackageManager, "yarn@") {
				pinned := strings.SplitN(strings.TrimPrefix(pkg.PackageManager, "yarn@"), "+", 2)[0]
				if pinned != version {
					return fmt.Errorf("packageManager exige Yarn %s; selecione essa versão com verman use yarn %s", pinned, pinned)
				}
			}
			if pkg.PackageManager != "" {
				return nil
			}
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// FindExecutable consulta só o PATH fornecido. Não executa binários nem tenta
// inspecionar aliases/funções do terminal pai. Entradas relativas são ignoradas.
func FindExecutable(name string, environ []string) (string, error) {
	path := ""
	for _, value := range environ {
		pair := strings.SplitN(value, "=", 2)
		if len(pair) == 2 && strings.EqualFold(pair[0], "PATH") {
			path = pair[1]
		}
	}
	extensions := []string{""}
	if runtime.GOOS == "windows" {
		if strings.EqualFold(filepath.Ext(name), ".cmd") || strings.EqualFold(filepath.Ext(name), ".bat") {
			return "", fmt.Errorf("exec requer executável nativo; use npm/npx/pip sem sufixo para ferramentas gerenciadas")
		}
		extensions = []string{".exe", ".com"}
		if filepath.Ext(name) != "" {
			extensions = []string{""}
		}
	}
	for _, entry := range filepath.SplitList(path) {
		if runtime.GOOS == "windows" {
			entry = strings.Trim(entry, `"`)
		}
		if entry == "" || !filepath.IsAbs(entry) {
			continue
		}
		for _, extension := range extensions {
			candidate := filepath.Join(entry, name+extension)
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && (runtime.GOOS == "windows" || info.Mode().Perm()&0111 != 0) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("comando %q não encontrado no PATH", name)
}
