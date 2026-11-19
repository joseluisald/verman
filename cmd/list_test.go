package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/installer"
)

func TestLocalListSelection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	child := filepath.Join(project, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveProjectTool(project, "php", "8.4.1"); err != nil {
		t.Fatal(err)
	}
	root, err := config.InstallDir("node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node"), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(installer.Metadata{SchemaVersion: 1, Language: "node", Version: "22.11.0", OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/node.zip", Executables: map[string]string{"node": "node"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, installer.MetadataFileName), data, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := writeLocalList(&output, child, nil); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"node — 1", "Seleção do projeto: 22.11.0 (instalada;", "Seleção do projeto: 8.4.1 (não instalada;", "python — 0", "Seleção do projeto: nenhuma.", "Provedor de instalação: PHP oficial"} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("saída sem %q: %s", text, &output)
		}
	}
	output.Reset()
	if err := writeLocalList(&output, child, []string{"php"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "node") || !strings.Contains(output.String(), "Nenhuma versão instalada.") {
		t.Fatalf("filtro: %s", &output)
	}
	output.Reset()
	if err := writeAvailableList(&output, child, "node", installer.Catalog{Platform: installer.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, Artifacts: []installer.Artifact{{Language: "node", Version: "22.11.0"}, {Language: "node", Version: "22.12.0"}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "22.11.0 [instalada] [selecionada no projeto]") || !strings.Contains(output.String(), "22.12.0\n") {
		t.Fatalf("catálogo remoto: %s", &output)
	}
	if err := os.WriteFile(filepath.Join(project, config.ProjectFileName), []byte("[invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := writeLocalList(&output, child, nil); err == nil || output.Len() != 0 {
		t.Fatalf("TOML inválido: %q, %v", &output, err)
	}
}

func TestListCommandValidation(t *testing.T) {
	for _, args := range [][]string{{"ruby"}, {"../node"}, {"node", "extra"}, {"--available"}, {"node", "--refresh"}} {
		command := newListCmd()
		command.SetOut(&bytes.Buffer{})
		command.SetErr(&bytes.Buffer{})
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Errorf("argumentos aceitos: %q", args)
		}
	}
}
