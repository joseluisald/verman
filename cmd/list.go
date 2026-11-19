package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/tool"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var available, refresh, history bool
	command := &cobra.Command{
		Use:     "list [linguagem]",
		Short:   "Lista versões instaladas ou disponíveis para instalar",
		Long:    "Consulta instalações locais sem rede. Use --available para consultar o catálogo remoto da linguagem. A seleção do projeto não indica ativação no terminal.",
		Example: "  verman list\n  verman list php\n  verman list python --available\n  verman list node --available --refresh",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if history && !available {
				return fault.Usage("--history exige --available")
			}
			if refresh && !available {
				return fault.Usage("--refresh exige --available")
			}
			if available && len(args) != 1 {
				return fault.Usage("--available exige uma linguagem: list node --available")
			}
			if len(args) == 1 {
				if err := tool.ValidateLanguage(args[0]); err != nil {
					return err
				}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if available {
				result, err := installer.AvailableHistory(cmd.Context(), args[0], refresh, history)
				if err != nil {
					return err
				}
				if result.Warning != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Aviso:", result.Warning)
				}
				if jsonMode(cmd) {
					return respond(cmd, availableListData(args[0], result), nil)
				}
				return writeAvailableList(cmd.OutOrStdout(), cwd, args[0], result.Catalog)
			}
			if jsonMode(cmd) {
				data, err := localListData(cwd, args)
				if err != nil {
					return err
				}
				return respond(cmd, data, nil)
			}
			return writeLocalList(cmd.OutOrStdout(), cwd, args)
		},
	}
	command.Flags().BoolVar(&available, "available", false, "Consulta versões disponíveis para instalar na plataforma atual")
	command.Flags().BoolVar(&refresh, "refresh", false, "Atualiza o catálogo remoto, ignorando o cache (exige --available)")
	command.Flags().BoolVar(&history, "history", false, "Inclui histórico limitado de PHP/Python; informa pacotes sem checksum como indisponíveis")
	return command
}

func writeAvailableList(out io.Writer, cwd, language string, catalog installer.Catalog) error {
	cfg, err := config.LoadProjectConfig(cwd)
	if err != nil {
		return err
	}
	global, err := config.LoadGlobalConfig()
	if err != nil {
		return err
	}
	var output strings.Builder
	fmt.Fprintf(&output, "%s — versões disponíveis para %s/%s\n", language, catalog.Platform.OS, catalog.Platform.Arch)
	fmt.Fprintf(&output, "Fonte: %s\nConsultado em: %s\n", catalog.Source, catalog.FetchedAt.Format(time.RFC3339))
	if len(catalog.Artifacts) == 0 {
		output.WriteString("  Nenhuma versão compatível disponível.\n")
	}
	for _, artifact := range catalog.Artifacts {
		label := ""
		if artifact.LTS {
			label += " [LTS]"
		}
		policy := installer.Describe(artifact)
		if artifact.Build != "" {
			label += " [build " + artifact.Build + "]"
		}
		if artifact.Archived {
			label += " [histórico]"
		}
		if !policy.Installable {
			label += " [indisponível: sem SHA-256 oficial]"
		}
		if policy.Support == "eol" {
			label += " [EOL]"
		}
		if installer.IsInstalled(language, artifact.Version) {
			label += " [instalada]"
		}
		if cfg.Tools[language] == artifact.Version {
			label += " [selecionada no projeto]"
		}
		if global.Tools[language] == artifact.Version {
			label += " [selecionada globalmente]"
		}
		fmt.Fprintf(&output, "  %s%s\n", artifact.Version, label)
	}
	if selected := cfg.Tools[language]; selected != "" {
		fmt.Fprintf(&output, "Seleção do projeto: %s (não indica ativação no terminal).\n", selected)
	}
	_, err = io.WriteString(out, output.String())
	return err
}

func writeLocalList(out io.Writer, cwd string, args []string) error {
	languages := tool.Languages()
	if len(args) == 1 {
		if err := tool.ValidateLanguage(args[0]); err != nil {
			return err
		}
		languages = args
	}
	cfg, err := config.LoadProjectConfig(cwd)
	if err != nil {
		return err
	}
	global, err := config.LoadGlobalConfig()
	if err != nil {
		return err
	}
	var output strings.Builder
	for i, language := range languages {
		versions, err := installer.ListInstalled(language)
		if err != nil {
			return err
		}
		if i > 0 {
			output.WriteByte('\n')
		}
		fmt.Fprintf(&output, "%s — %d versão(ões) instalada(s)\n", language, len(versions))
		if len(versions) == 0 {
			output.WriteString("  Nenhuma versão instalada.\n")
		} else {
			for _, version := range versions {
				fmt.Fprintf(&output, "  %s\n", version)
			}
		}
		selected := cfg.Tools[language]
		if selected == "" {
			output.WriteString("  Seleção do projeto: nenhuma.\n")
		} else {
			installed := false
			for _, version := range versions {
				installed = installed || version == selected
			}
			state := "não instalada"
			if installed {
				state = "instalada"
			}
			fmt.Fprintf(&output, "  Seleção do projeto: %s (%s; não indica ativação no terminal).\n", selected, state)
		}
		if version := global.Tools[language]; version != "" {
			status := "fallback da configuração"
			if selected != "" {
				status = "substituída pelo projeto"
			}
			if !installer.IsInstalled(language, version) {
				status += "; não instalada"
			}
			fmt.Fprintf(&output, "  Seleção global: %s (%s).\n", version, status)
		}
		providers := map[string]string{"node": "Node.js oficial", "php": "PHP oficial NTS x64", "python": "Astral python-build-standalone"}
		if definition, ok := tool.Lookup(language); ok && providers[language] == "" {
			providers[language] = definition.Name + " oficial"
		}
		fmt.Fprintf(&output, "  Provedor de instalação: %s (windows/amd64).\n", providers[language])
	}
	_, err = io.WriteString(out, output.String())
	return err
}

func init() {
	rootCmd.AddCommand(newListCmd())
}
