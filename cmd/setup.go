package cmd

import (
	"fmt"
	"os"

	"github.com/joseluisald/verman/internal/shims"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	var dir string
	command := &cobra.Command{Use: "setup", Short: "Instala shims nativos em uma pasta fixa; configure seu PATH uma vez", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			var err error
			dir, err = shims.Directory()
			if err != nil {
				return err
			}
		}
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		if err := shims.Setup(cmd.Context(), executable, dir); err != nil {
			return err
		}
		if jsonMode(cmd) {
			return respond(cmd, map[string]any{"directory": dir, "path_changed": false}, nil)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Shims instalados em %s\nColoque essa pasta antes dos outros runtimes no PATH e reabra o terminal uma vez. Depois, use muda a seleção sem ativação.\nUse scripts/setup-path.ps1 para configurar o PATH com diagnóstico de conflitos de escopo.\n", dir)
		return err
	}}
	command.Flags().StringVar(&dir, "bin-dir", "", "Diretório dos shims (padrão ~/.verman/bin)")
	return command
}
func init() { rootCmd.AddCommand(newSetupCmd()) }
