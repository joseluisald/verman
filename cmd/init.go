package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/joseluisald/verman/internal/environment"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "powershell"
	}
	var disable bool
	command := &cobra.Command{Use: "init", SilenceUsage: true, Short: "Emite hook opcional para ativar versões antes de cada prompt", Long: "Emite código de integração para a sessão; não edita perfis. Use um executável instalado: init via go run apontaria para um arquivo temporário. O hook lê configurações antes de cada prompt, sem instalar runtimes nem executar scripts do projeto. Em caso de configuração inválida, limpa o ambiente anterior.", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			executable, err := os.Executable()
			if err != nil {
				return err
			}
			code, err := environment.Hook(shell, executable, disable)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), code)
			return err
		},
	}
	command.Flags().StringVar(&shell, "shell", shell, "Shell do hook: powershell ou bash (Bash 5+)")
	command.Flags().BoolVar(&disable, "disable", false, "Remove o hook da sessão e limpa a ativação")
	return command
}

func init() { rootCmd.AddCommand(newInitCmd()) }
