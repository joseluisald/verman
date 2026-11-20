package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/shims"
	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "verman",
	Short:         "Gerenciador de versões de runtimes e ferramentas por projeto",
	Long:          "Instale e selecione Node, PHP, Python, Bun, Go, .NET, Deno, Git, Rust, Composer, Yarn, pnpm, vcpkg e Chocolatey por projeto (Windows x64).",
	Version:       Version,
	Example:       "  verman list php --available\n  verman use php latest\n  verman setup\n  verman doctor php\n  verman exec -- php -v\n  verman support --json",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if mode, _ := cmd.Flags().GetString("progress"); mode != "auto" && mode != "always" && mode != "never" {
			return fault.Usage("--progress aceita auto, always ou never")
		}
		if disabled, _ := cmd.Flags().GetBool("offline"); disabled {
			cmd.SetContext(installer.WithOffline(cmd.Context()))
		}
		if jsonMode(cmd) && (cmd.Name() == "env" || cmd.Name() == "init" || cmd.Name() == "exec" || cmd.Name() == "completion") {
			return fault.Usage("--json não é compatível com %s", cmd.Name())
		}
		return nil
	},
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if executable, err := os.Executable(); err == nil {
		if tool := shims.Tool(executable); tool != "" {
			os.Exit(shims.Run(ctx, executable, tool, os.Args[1:], os.Environ(), os.Stdin, os.Stdout, os.Stderr))
		}
	}
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error { return fault.Usage("%v", err) })
	wrapArguments(rootCmd)
	started := false
	preRun := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { started = true; return preRun(cmd, args) }
	command, err := rootCmd.ExecuteContextC(ctx)
	if err != nil {
		var typed *fault.Error
		if !started && !errors.As(err, &typed) && ctx.Err() == nil {
			err = fault.Usage("%v", err)
		}
		var exit *processExit
		if errors.As(err, &exit) {
			os.Exit(exit.code)
		}
		failure := fault.Classify(err)
		var rendered *renderedError
		if errors.As(err, &rendered) {
			os.Exit(failure.Exit)
		}
		if command == nil {
			command = rootCmd
		}
		if !started && requestedJSON(os.Args[1:]) {
			command.Flags().Set("json", "true")
		}
		if jsonMode(command) {
			if writeErr := writeResponse(command, nil, err); writeErr != nil {
				fmt.Fprintln(os.Stderr, writeErr)
			}
		} else {
			fmt.Fprintf(command.ErrOrStderr(), "Erro [%s]: %v\n%s\n", failure.Code, err, failure.Hint)
		}
		os.Exit(failure.Exit)
	}
}

func requestedJSON(args []string) bool {
	value := false
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "--json" || arg == "--json=true" {
			value = true
		}
		if strings.HasPrefix(arg, "--json=") && arg != "--json=true" {
			value = false
		}
	}
	return value
}

func wrapArguments(command *cobra.Command) {
	if original := command.Args; original != nil {
		command.Args = func(cmd *cobra.Command, args []string) error {
			if err := original(cmd, args); err != nil {
				return fault.Usage("%v", err)
			}
			return nil
		}
	}
	for _, child := range command.Commands() {
		wrapArguments(child)
	}
}
func init() {
	rootCmd.PersistentFlags().Bool("json", false, "Emite resposta JSON versionada (schema 2) nos comandos de consulta e gerenciamento")
	rootCmd.PersistentFlags().Bool("offline", false, "Não consulta rede; usa instalações e catálogos locais")
	rootCmd.PersistentFlags().String("progress", "auto", "Progresso em stderr: auto, always (também em pipes) ou never")
}
