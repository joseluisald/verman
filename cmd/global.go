package cmd

import (
	"fmt"
	"os"

	"github.com/joseluisald/verman/internal/application"
	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/tool"
	"github.com/spf13/cobra"
)

func newGlobalCmd() *cobra.Command {
	var unset bool
	command := &cobra.Command{Use: "global [linguagem versao]", Short: "Consulta ou define versões globais, usadas como fallback do projeto",
		Example: "  verman global\n  verman global node lts\n  verman global --unset node",
		Args: func(cmd *cobra.Command, args []string) error {
			if (unset && len(args) == 1) || (!unset && (len(args) == 0 || len(args) == 2)) {
				return nil
			}
			return fmt.Errorf("use global, global <linguagem> <versao> ou global --unset <linguagem>")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if unset {
				if err := config.RemoveGlobalTool(args[0]); err != nil {
					return err
				}
				if jsonMode(cmd) {
					return respond(cmd, map[string]any{"language": args[0], "scope": "global", "unset": true}, nil)
				}
				return nil
			}
			if len(args) == 0 {
				cfg, err := config.LoadGlobalConfig()
				if err != nil {
					return err
				}
				if jsonMode(cmd) {
					return respond(cmd, cfg, nil)
				}
				if len(cfg.Tools) == 0 {
					_, err = fmt.Fprintln(cmd.OutOrStdout(), "Nenhuma seleção global.")
					return err
				}
				for _, language := range tool.Languages() {
					if version := cfg.Tools[language]; version != "" {
						if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s = %s\n", language, version); err != nil {
							return err
						}
					}
				}
				return nil
			}
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := application.New().Select(cmd.Context(), application.SelectionRequest{Directory: dir, Language: args[0], Version: args[1], Scope: application.Global}, commandEvents(cmd))
			if err != nil {
				return err
			}
			if jsonMode(cmd) {
				return respond(cmd, result, nil)
			}
			return writeSelection(cmd.OutOrStdout(), result)
		},
	}
	command.Flags().BoolVar(&unset, "unset", false, "Remove apenas a seleção global da linguagem, preservando a instalação")
	return command
}

func init() { rootCmd.AddCommand(newGlobalCmd()) }
