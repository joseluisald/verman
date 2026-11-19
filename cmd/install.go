package cmd

import (
	"github.com/joseluisald/verman/internal/installer"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install <linguagem> <versao>",
	Short:   "Instala uma versão de uma ferramenta",
	Example: "  verman list bun --available\n  verman install bun latest\n  verman install git latest\n  verman install rust latest",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		lang := args[0]
		version := args[1]
		sink := commandEvents(cmd)
		resolved, err := installer.ResolveVersionEvents(cmd.Context(), lang, version, sink)
		if err != nil {
			return err
		}
		if err := installer.InstallWithEvents(cmd.Context(), lang, resolved, sink); err != nil {
			return err
		}
		if jsonMode(cmd) {
			meta, err := installer.ReadMetadata(lang, resolved)
			if err != nil {
				return err
			}
			return respond(cmd, meta, nil)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
