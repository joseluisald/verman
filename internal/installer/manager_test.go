package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func isolatedHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestInstallationContract(t *testing.T) {
	isolatedHome(t)
	root, err := config.InstallDir("node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if IsInstalled("node", "22.11.0") {
		t.Fatal("pasta vazia considerada instalada")
	}
	meta := Metadata{SchemaVersion: 1, Language: "node", Version: "22.11.0", OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/node.zip", Executables: map[string]string{"node": "node.exe", "npm": "tools/npm"}}
	if err := os.Mkdir(filepath.Join(root, "tools"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, relative := range meta.Executables {
		if err := os.WriteFile(filepath.Join(root, relative), []byte("fixture, não executar"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeMeta := func(m Metadata) {
		t.Helper()
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, MetadataFileName), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeMeta(meta)
	if !IsInstalled("node", "22.11.0") {
		t.Fatal("instalação estruturalmente válida rejeitada")
	}
	dirs, err := BinDirs("node", "22.11.0")
	if err != nil || len(dirs) != 2 || dirs[0] != root || dirs[1] != filepath.Join(root, "tools") {
		t.Fatalf("layout: %v, %v", dirs, err)
	}
	for _, mutate := range []func(*Metadata){
		func(m *Metadata) { m.SchemaVersion = 2 },
		func(m *Metadata) { m.Language = "php" },
		func(m *Metadata) { m.Version = "22.12.0" },
		func(m *Metadata) { m.OS = "other" },
		func(m *Metadata) { m.Arch = "other" },
		func(m *Metadata) { m.Source = "" },
	} {
		bad := meta
		mutate(&bad)
		writeMeta(bad)
		if IsInstalled("node", "22.11.0") {
			t.Fatalf("metadados inválidos aceitos: %#v", bad)
		}
	}
	for _, path := range []string{"../outside", "/absolute", `C:\outside`, `..\outside`, "missing", "bin", ""} {
		bad := meta
		bad.Executables = map[string]string{"node": path}
		writeMeta(bad)
		if IsInstalled("node", "22.11.0") {
			t.Fatalf("caminho aceito: %q", path)
		}
	}
	writeMeta(meta)
	if err := os.WriteFile(filepath.Join(root, "node.exe"), nil, 0700); err != nil {
		t.Fatal(err)
	}
	if IsInstalled("node", "22.11.0") {
		t.Fatal("executável vazio aceito")
	}
}

func TestUnknownProviderDoesNotCreateInstallation(t *testing.T) {
	isolatedHome(t)
	for _, language := range []string{"ruby", "../node"} {
		if err := Install(language, "1.2.3"); err == nil {
			t.Fatal("instalação simulada retornou sucesso")
		}
	}
	base, err := config.BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("instalação deixou resíduos: %v", err)
	}
}
