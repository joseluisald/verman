package cmd

import (
	"fmt"

	"github.com/joseluisald/verman/internal/installer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{Use: "recover <linguagem> <versao>", Short: "Recupera uma transação interrompida, sem remover locks legados por suposição", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if err := installer.Recover(cmd.Context(), args[0], args[1]); err != nil {
			return err
		}
		if jsonMode(cmd) {
			return respond(cmd, map[string]any{"language": args[0], "version": args[1], "recovered": true}, nil)
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Transação recuperada; configurações preservadas.")
		return err
	}})
	rootCmd.AddCommand(&cobra.Command{Use: "verify <linguagem> <versao>", Short: "Verifica os hashes dos arquivos imutáveis, sem executar o runtime", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if err := installer.VerifyContext(cmd.Context(), args[0], args[1]); err != nil {
			return err
		}
		if jsonMode(cmd) {
			return respond(cmd, map[string]any{"language": args[0], "version": args[1], "verified": true}, nil)
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Integridade verificada para os arquivos do manifesto.")
		return err
	}})
	rootCmd.AddCommand(&cobra.Command{Use: "repair <linguagem> <versao>", Short: "Reinstala da origem verificada, preservando a anterior até validar a nova", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if err := installer.RepairWithEvents(cmd.Context(), args[0], args[1], commandEvents(cmd)); err != nil {
			return err
		}
		if jsonMode(cmd) {
			meta, err := installer.ReadMetadata(args[0], args[1])
			if err != nil {
				return err
			}
			return respond(cmd, meta, nil)
		}
		return nil
	}})
}
