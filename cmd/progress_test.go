package cmd

import (
	"bytes"
	"github.com/joseluisald/verman/internal/installer"
	"testing"
)

func TestProgressDoesNotContaminatePipelines(t *testing.T) {
	for _, mode := range []string{"auto", "always", "never"} {
		var output, diagnostics bytes.Buffer
		sink := renderOperationEvents(&output, &diagnostics, mode)
		if err := sink(installer.Event{Kind: installer.EventProgress, Language: "php", Bytes: 1024, Total: 2048}); err != nil {
			t.Fatal(err)
		}
		if output.Len() != 0 || bytes.Contains(diagnostics.Bytes(), []byte("\r")) {
			t.Fatal("progresso contaminou stdout/pipe")
		}
		if (mode == "always") != (diagnostics.Len() > 0) {
			t.Fatalf("modo %s: %s", mode, &diagnostics)
		}
	}
}
