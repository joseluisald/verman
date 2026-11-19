package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/joseluisald/verman/internal/application"
	"github.com/joseluisald/verman/internal/environment"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/shimmeta"
	"github.com/spf13/cobra"
)

func operationEvents(out, diagnostics io.Writer) installer.EventSink {
	return renderOperationEvents(out, diagnostics, "auto")
}

func renderOperationEvents(out, diagnostics io.Writer, mode string) installer.EventSink {
	interactive := false
	if file, ok := diagnostics.(*os.File); ok {
		if info, err := file.Stat(); err == nil {
			interactive = info.Mode()&os.ModeCharDevice != 0
		}
	}
	show := interactive && mode != "never" || mode == "always"
	return func(event installer.Event) error {
		var err error
		switch event.Kind {
		case installer.EventWarning:
			_, err = fmt.Fprintln(diagnostics, "Aviso:", event.Message)
		case installer.EventDownloading:
			_, err = fmt.Fprintf(out, "Baixando %s %s...\n", event.Language, event.Version)
		case installer.EventInstalled:
			_, err = fmt.Fprintf(out, "✔ %s %s instalado em %s\n", event.Language, event.Version, event.Directory)
		case installer.EventAlreadyInstalled:
			_, err = fmt.Fprintf(out, "✔ %s %s já está instalado.\n", event.Language, event.Version)
		case installer.EventProgress:
			if show {
				prefix, suffix := "", "\n"
				if interactive {
					prefix, suffix = "\r", "   "
				}
				if event.Total > 0 {
					_, err = fmt.Fprintf(diagnostics, "%sBaixando %s: %.1f / %.1f MiB%s", prefix, event.Language, float64(event.Bytes)/(1<<20), float64(event.Total)/(1<<20), suffix)
				} else {
					_, err = fmt.Fprintf(diagnostics, "%sBaixando %s: %.1f MiB%s", prefix, event.Language, float64(event.Bytes)/(1<<20), suffix)
				}
			}
		case installer.EventExtracting, installer.EventValidating:
			if show {
				phase := "Extraindo"
				if event.Kind == installer.EventValidating {
					phase = "Validando"
				}
				_, err = fmt.Fprintf(diagnostics, "\n%s %s %s...\n", phase, event.Language, event.Version)
			}
		}
		return err
	}
}

func commandEvents(cmd *cobra.Command) installer.EventSink {
	if os.Getenv("VIRTUAL_ENV") != "" || os.Getenv("CONDA_PREFIX") != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Aviso: ambiente virtual ativo; use troca a versão base, sem migrar o venv. Desative-o antes de usar a nova seleção.")
	}
	mode, _ := cmd.Flags().GetString("progress")
	if mode == "" {
		mode = "auto"
	}
	return renderOperationEvents(resultOutput(cmd), cmd.ErrOrStderr(), mode)
}

func invocationPowerShell() string {
	executable, err := os.Executable()
	if err != nil {
		return "verman"
	}
	if environment.IsTemporaryExecutable(executable) {
		return "go run ."
	}
	return "& '" + strings.ReplaceAll(executable, "'", "''") + "'"
}

func writeSelection(out io.Writer, result application.SelectionResult) error {
	scope := "projeto"
	if result.Scope == application.Global {
		scope = "global (o projeto tem prioridade)"
	}
	_, err := fmt.Fprintf(out, "✔ Seleção salva (%s): %s %s\nConfiguração: %s\nEste comando não altera o terminal atual.\n", scope, result.Language, result.Version, result.ConfigPath)
	if err != nil {
		return err
	}
	if path, err := application.FindExecutable(result.Language, os.Environ()); err == nil && shimmeta.Matches(path) {
		_, err = fmt.Fprintln(out, "Shim ativo no PATH: a próxima execução já usará esta seleção, sem reaplicar env.")
		return err
	}
	if runtime.GOOS == "windows" {
		invocation := invocationPowerShell()
		_, err = fmt.Fprintf(out, "Para ativar agora no PowerShell:\n$code = %s env --shell powershell\nif ($LASTEXITCODE -eq 0) { $code | Out-String | Invoke-Expression }\nSem ativação: %s exec -- %s --version\n", invocation, invocation, result.Language)
	} else {
		_, err = fmt.Fprintf(out, "Ative com env --shell sh ou use verman exec -- %s --version.\n", result.Language)
	}
	return err
}
