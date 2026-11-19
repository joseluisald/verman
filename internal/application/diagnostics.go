package application

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/shimmeta"
	"github.com/joseluisald/verman/internal/tool"
)

type ToolDiagnostic struct {
	Language           string `json:"language"`
	Version            string `json:"version,omitempty"`
	Source             string `json:"source"`
	ConfigPath         string `json:"config_path,omitempty"`
	Installed          bool   `json:"installed"`
	ExpectedExecutable string `json:"expected_executable,omitempty"`
	PATHExecutable     string `json:"path_executable,omitempty"`
	PATHMatches        bool   `json:"path_matches"`
	UsesShim           bool   `json:"uses_shim,omitempty"`
	Status             string `json:"status"`
	Detail             string `json:"detail,omitempty"`
}
type DiagnosticReport struct {
	Schema                int              `json:"schema"`
	Directory             string           `json:"directory"`
	Platform              string           `json:"platform"`
	ShellAliasesInspected bool             `json:"shell_aliases_inspected"`
	Issues                bool             `json:"issues"`
	Tools                 []ToolDiagnostic `json:"tools"`
	VirtualEnvironment    string           `json:"virtual_environment,omitempty"`
	Warnings              []string         `json:"warnings"`
}

func (s *Service) Diagnose(dir string, languages, environ []string) (DiagnosticReport, error) {
	if len(languages) == 0 {
		languages = tool.Languages()
	}
	for _, language := range languages {
		if err := tool.ValidateLanguage(language); err != nil {
			return DiagnosticReport{}, err
		}
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return DiagnosticReport{}, err
	}
	project, projectFile, err := config.LoadProjectConfigWithPath(dir)
	if err != nil {
		return DiagnosticReport{}, err
	}
	global, err := config.LoadGlobalConfig()
	if err != nil {
		return DiagnosticReport{}, err
	}
	base, err := config.BaseDir()
	if err != nil {
		return DiagnosticReport{}, err
	}
	report := DiagnosticReport{Schema: 1, Directory: dir, Platform: runtime.GOOS + "/" + runtime.GOARCH, Tools: []ToolDiagnostic{}, Warnings: []string{}}
	for _, entry := range environ {
		pair := strings.SplitN(entry, "=", 2)
		if len(pair) == 2 && (strings.EqualFold(pair[0], "VIRTUAL_ENV") || strings.EqualFold(pair[0], "CONDA_PREFIX")) && pair[1] != "" {
			report.VirtualEnvironment = pair[1]
			report.Warnings = append(report.Warnings, "Ambiente virtual ativo: desative-o antes de trocar a versão base; um venv existente não é migrado por use.")
		}
	}
	for _, language := range languages {
		item := ToolDiagnostic{Language: language, Source: "none", Status: "not_selected"}
		item.PATHExecutable, _ = FindExecutable(language, environ)
		if version := global.Tools[language]; version != "" {
			item.Version = version
			item.Source = "global"
			item.ConfigPath = filepath.Join(base, config.GlobalFileName)
		}
		if version := project.Tools[language]; version != "" {
			item.Version = version
			item.Source = "project"
			item.ConfigPath = projectFile
		}
		if item.Version != "" {
			meta, err := installer.ReadMetadata(language, item.Version)
			if err != nil {
				item.Status = "installation_invalid"
				item.Detail = fmt.Sprintf("instalação ausente ou inválida; execute verman install %s %s: %v", language, item.Version, err)
				report.Issues = true
			} else {
				root, err := config.InstallDir(language, item.Version)
				if err != nil {
					return DiagnosticReport{}, err
				}
				item.Installed = true
				item.ExpectedExecutable = filepath.Join(root, meta.Executables[language])
				item.PATHMatches = sameExecutable(item.ExpectedExecutable, item.PATHExecutable)
				if shimmeta.Matches(item.PATHExecutable) {
					item.UsesShim = true
					item.Status = "shim_ready"
				} else if item.PATHMatches {
					item.Status = "path_matches"
				} else {
					item.Status = "activation_required"
					report.Issues = true
					item.Detail = fmt.Sprintf("a seleção %s foi salva, mas o PATH não aponta para seu executável; aplique env, habilite init ou use exec", item.Version)
				}
				if definition, ok := tool.Lookup(language); ok && definition.Dependency != "" {
					dependencyVersion := global.Tools[definition.Dependency]
					if project.Tools[definition.Dependency] != "" {
						dependencyVersion = project.Tools[definition.Dependency]
					}
					if dependencyVersion == "" || !installer.IsInstalled(definition.Dependency, dependencyVersion) {
						item.Status = "dependency_missing"
						item.Detail = fmt.Sprintf("%s exige %s selecionado e instalado; execute verman use %s latest (ou global)", language, definition.Dependency, definition.Dependency)
						report.Issues = true
					}
				}
			}
		}
		report.Tools = append(report.Tools, item)
	}
	return report, nil
}

func sameExecutable(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
