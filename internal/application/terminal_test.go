package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func TestTerminalKeepsVerifiedShimsBeforeDirectRuntime(t *testing.T) {
	_, project := isolatedService(t)
	fixture(t, "node", "22.11.0")
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	shims := t.TempDir()
	data := []byte("fixture: nunca executar")
	sum := sha256.Sum256(data)
	for _, name := range []string{"node.exe", "php.exe", "python.exe"} {
		if err := os.WriteFile(filepath.Join(shims, name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	manifest, _ := json.Marshal(map[string]any{"schema": 1, "sha256": hex.EncodeToString(sum[:])})
	if err := os.WriteFile(filepath.Join(shims, ".verman-shims.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	original := []string{"PATH=" + shims}
	result, err := TerminalEnvironment(project, original)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range result {
		if strings.HasPrefix(strings.ToUpper(entry), "PATH=") {
			paths := filepath.SplitList(strings.SplitN(entry, "=", 2)[1])
			if paths[0] != shims {
				t.Fatalf("shim perdeu prioridade: %v", paths)
			}
			found = true
		}
	}
	if !found || original[0] != "PATH="+shims {
		t.Fatal("ambiente original alterado/sem PATH")
	}
}
