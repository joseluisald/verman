package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/joseluisald/verman/internal/environment"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	defaultShell := ""
	if runtime.GOOS == "windows" {
		defaultShell = "powershell"
	} else {
		defaultShell = "sh"
	}
	var shell string
	var clear bool
	command := &cobra.Command{
		SilenceUsage: true,
		Use:          "env",
		Short:        "Emite código para ativar as versões do projeto na sessão",
		Long:         "Emite código PowerShell ou POSIX para aplicar a seleção efetiva (projeto sobre global) à sessão. Não altera o ambiente persistente. No Windows, Bash/sh requerem Git Bash/cygpath. Reaplicar substitui as entradas gerenciadas anteriormente.",
		Example:      "  verman env --shell powershell\n  $code = verman env --shell powershell; if ($LASTEXITCODE -eq 0) { $code | Out-String | Invoke-Expression }",
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if shell != "powershell" && shell != "bash" && shell != "sh" {
				return fmt.Errorf("shell %q não suportado; use powershell, bash ou sh", shell)
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			var code string
			if shell == "powershell" {
				if clear {
					code, err = environment.RenderPowerShell(nil)
				} else {
					code, err = environment.PowerShell(cwd)
				}
			} else {
				if clear {
					code, err = environment.ClearPOSIX()
				} else {
					code, err = environment.POSIX(cwd)
				}
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), code)
			return err
		},
	}
	command.Flags().StringVar(&shell, "shell", defaultShell, "Shell de destino: powershell, bash ou sh")
	command.Flags().BoolVar(&clear, "clear", false, "Limpa a ativação da sessão e restaura variáveis, ignorando seleções")
	return command
}

func init() { rootCmd.AddCommand(newEnvCmd()) }
