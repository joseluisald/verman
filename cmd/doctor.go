package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/joseluisald/verman/internal/application"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	command := &cobra.Command{Use: "doctor [linguagem]", Short: "Diagnostica seleção, instalação e conflitos de PATH, sem alterar arquivos", SilenceUsage: true, Args: cobra.MaximumNArgs(1),
		Long: "Compara a versão selecionada com a instalação e o executável encontrado no PATH atual. Não executa runtimes, não acessa a rede e não inspeciona aliases/funções do shell pai. Código de saída 1 indica uma seleção ausente no disco ou não ativada no PATH.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			report, err := application.New().Diagnose(dir, args, os.Environ())
			if err != nil {
				return err
			}
			var issue error
			if report.Issues {
				issue = fault.New("diagnostic_issue", 1, "Confira a ação indicada para cada ferramenta.", fmt.Errorf("diagnóstico encontrou seleção sem instalação válida ou ativação pendente"))
			}
			if jsonMode(cmd) {
				return respond(cmd, report, issue)
			}
			err = writeDiagnosis(cmd.OutOrStdout(), report)
			if err != nil {
				return err
			}
			return issue
		},
	}
	return command
}

func writeDiagnosis(out io.Writer, report application.DiagnosticReport) error {
	if _, err := fmt.Fprintln(out, "Diagnóstico do Verman — seleção e resolução por PATH"); err != nil {
		return err
	}
	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintln(out, "Aviso:", warning); err != nil {
			return err
		}
	}
	for _, item := range report.Tools {
		if _, err := fmt.Fprintf(out, "\n%s\n", item.Language); err != nil {
			return err
		}
		if item.Version == "" {
			if _, err := fmt.Fprintln(out, "  Nenhuma versão selecionada pelo Verman."); err != nil {
				return err
			}
		} else {
			source := item.Source
			if source == "project" {
				source = "projeto"
			}
			if _, err := fmt.Fprintf(out, "  Seleção: %s (%s)\n  Configuração: %s\n", item.Version, source, item.ConfigPath); err != nil {
				return err
			}
		}
		if item.ExpectedExecutable != "" {
			if _, err := fmt.Fprintln(out, "  Esperado:", item.ExpectedExecutable); err != nil {
				return err
			}
		}
		found := item.PATHExecutable
		if found == "" {
			found = "não encontrado"
		}
		if _, err := fmt.Fprintln(out, "  Encontrado no PATH:", found); err != nil {
			return err
		}
		if item.Detail != "" {
			if _, err := fmt.Fprintln(out, "  Ação:", item.Detail); err != nil {
				return err
			}
		}
		if item.PATHMatches {
			if _, err := fmt.Fprintln(out, "  O caminho corresponde à seleção; o binário não foi executado."); err != nil {
				return err
			}
		}
		if item.UsesShim {
			if _, err := fmt.Fprintln(out, "  Shim do Verman no PATH: a seleção será resolvida a cada execução."); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(out, "\nAliases/funções do terminal não são inspecionados. No PowerShell, confira Get-Command php -All se ainda houver divergência.\nAtive a sessão com env ou habilite init; para executar sem ativação, use exec.")
	return err
}

func init() { rootCmd.AddCommand(newDoctorCmd()) }
