package cmd

import (
	"encoding/json"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/spf13/cobra"
	"io"
)

// Schema 2 unifica a antiga saída exclusiva de doctor e os demais comandos.
type response struct {
	Schema  int            `json:"schema"`
	Command string         `json:"command"`
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Error   *errorResponse `json:"error"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func jsonMode(cmd *cobra.Command) bool { value, _ := cmd.Flags().GetBool("json"); return value }
func writeResponse(cmd *cobra.Command, data any, err error) error {
	result := response{Schema: 2, Command: cmd.Name(), Success: err == nil, Data: data}
	if err != nil {
		failure := fault.Classify(err)
		result.Error = &errorResponse{failure.Code, err.Error(), failure.Hint}
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
}
func resultOutput(cmd *cobra.Command) io.Writer {
	if jsonMode(cmd) {
		return io.Discard
	}
	return cmd.OutOrStdout()
}

type renderedError struct{ error }

func (e *renderedError) Unwrap() error { return e.error }
func respond(cmd *cobra.Command, data any, err error) error {
	if writeErr := writeResponse(cmd, data, err); writeErr != nil {
		return writeErr
	}
	if err != nil {
		return &renderedError{err}
	}
	return nil
}
