package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func TestEnvErrorsHaveNoScript(t *testing.T) {
	for _, args := range [][]string{{"--shell", "fish"}, {"extra"}} {
		command := newEnvCmd()
		var stdout, stderr bytes.Buffer
		command.SetOut(&stdout)
		command.SetErr(&stderr)
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Fatal("argumentos inválidos aceitos")
		}
		if stdout.Len() != 0 {
			t.Fatalf("erro produziu stdout: %s", &stdout)
		}
	}
}

func TestUseFailurePreservesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, config.ProjectFileName)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := selectProjectTool(ctx, project, "php", "8.4.1", &bytes.Buffer{}); err == nil {
		t.Fatal("cancelamento ignorado")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("falha de instalação alterou seleção")
	}
}
