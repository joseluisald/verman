package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/joseluisald/verman/internal/application"
	"github.com/spf13/cobra"
)

type processExit struct{ code int }

func (e *processExit) Error() string { return fmt.Sprintf("processo terminou com código %d", e.code) }

func newExecCmd() *cobra.Command {
	command := &cobra.Command{Use: "exec -- <comando> [argumentos...]", Short: "Executa um processo com as versões efetivas do projeto", SilenceUsage: true, Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			lease, err := application.ExecutionLease(cmd.Context(), dir)
			if err != nil {
				return err
			}
			defer lease()
			plan, err := application.New().PlanExecution(dir, args, os.Environ())
			if err != nil {
				return err
			}
			process := exec.CommandContext(cmd.Context(), plan.Executable, plan.Arguments...)
			process.Env = plan.Environment
			process.Dir = plan.Directory
			process.Stdin = cmd.InOrStdin()
			process.Stdout = cmd.OutOrStdout()
			process.Stderr = cmd.ErrOrStderr()
			if err := process.Run(); err != nil {
				var exit *exec.ExitError
				if errors.As(err, &exit) {
					code := exit.ExitCode()
					if code < 1 {
						code = 1
					}
					return &processExit{code: code}
				}
				return err
			}
			return nil
		},
	}
	command.Flags().SetInterspersed(false)
	return command
}

func init() { rootCmd.AddCommand(newExecCmd()) }
