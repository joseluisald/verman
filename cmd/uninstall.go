package cmd

import (
	"fmt"
	"os"

	"github.com/joseluisald/verman/internal/installer"
	"github.com/spf13/cobra"
)

func newUninstallCmd() *cobra.Command {
	var allowSelected bool
	command := &cobra.Command{
		Use:     "uninstall <linguagem> <versao>",
		Short:   "Remove uma versão instalada pelo Verman",
		Long:    "Remove somente a instalação informada. Não altera o TOML nem o PATH da sessão. Outros projetos podem referenciar essa versão; o Verman não procura referências em todo o disco.",
		Example: "  verman uninstall node 22.11.0\n  verman uninstall node 22.11.0 --allow-selected",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if err := installer.Uninstall(cmd.Context(), cwd, args[0], args[1], allowSelected); err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "Aviso: outros projetos podem referenciar a versão removida. As seleções e o PATH da sessão foram mantidos.")
			if jsonMode(cmd) {
				return respond(cmd, map[string]any{"language": args[0], "version": args[1], "removed": true}, nil)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "✔ %s %s removido.\n", args[0], args[1])
			return err
		},
	}
	command.Flags().BoolVar(&allowSelected, "allow-selected", false, "Permite remover uma versão selecionada no projeto ou globalmente, mantendo as configurações")
	return command
}

func init() { rootCmd.AddCommand(newUninstallCmd()) }
