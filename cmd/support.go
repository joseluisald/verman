package cmd

import (
	"fmt"
	"github.com/joseluisald/verman/internal/application"
	"github.com/spf13/cobra"
	"os"
	"runtime"
)

func newSupportCmd() *cobra.Command {
	return &cobra.Command{Use: "support", Short: "Gera diagnóstico offline para um relato de problema, com caminhos omitidos", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		report, err := application.New().Diagnose(dir, nil, os.Environ())
		if err != nil {
			return err
		}
		// Reporte estado, nunca diretórios pessoais, PATH completo ou variáveis.
		type item struct {
			Language  string `json:"language"`
			Version   string `json:"version,omitempty"`
			Status    string `json:"status"`
			Installed bool   `json:"installed"`
		}
		tools := []item{}
		for _, tool := range report.Tools {
			tools = append(tools, item{tool.Language, tool.Version, tool.Status, tool.Installed})
		}
		data := struct {
			Version                  string `json:"version"`
			Platform                 string `json:"platform"`
			VirtualEnvironmentActive bool   `json:"virtual_environment_active"`
			Tools                    []item `json:"tools"`
		}{Version, runtime.GOOS + "/" + runtime.GOARCH, report.VirtualEnvironment != "", tools}
		if jsonMode(cmd) {
			return respond(cmd, data, nil)
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Verman %s — %s\nAmbiente virtual ativo: %t\n", data.Version, data.Platform, data.VirtualEnvironmentActive); err != nil {
			return err
		}
		for _, tool := range tools {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s: %s\n", tool.Language, tool.Version, tool.Status); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Relato JSON: verman support --json. Veja docs/SUPORTE.md. Nenhum diagnóstico foi enviado.")
		return err
	}}
}

func init() { rootCmd.AddCommand(newSupportCmd()) }
