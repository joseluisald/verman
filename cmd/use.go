package cmd

import (
	"context"
	"io"
	"os"

	"github.com/joseluisald/verman/internal/application"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:     "use <linguagem> <versao>",
	Short:   "Define a versão de uma ferramenta para o diretório atual",
	Example: "  verman use node 22.11.0",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		result, err := application.New().Select(cmd.Context(), application.SelectionRequest{Directory: cwd, Language: args[0], Version: args[1], Scope: application.Project}, commandEvents(cmd))
		if err != nil {
			return err
		}
		if jsonMode(cmd) {
			return respond(cmd, result, nil)
		}
		return writeSelection(cmd.OutOrStdout(), result)
	},
}

func selectProjectTool(ctx context.Context, dir, lang, version string, out io.Writer) error {
	result, err := application.New().Select(ctx, application.SelectionRequest{Directory: dir, Language: lang, Version: version, Scope: application.Project}, operationEvents(out, out))
	if err != nil {
		return err
	}
	return writeSelection(out, result)
}

func init() {
	rootCmd.AddCommand(useCmd)
}
