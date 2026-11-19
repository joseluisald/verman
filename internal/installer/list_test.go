package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func TestListInstalled(t *testing.T) {
	isolatedHome(t)
	versions, err := ListInstalled("node")
	if err != nil || len(versions) != 0 {
		t.Fatalf("diretório inexistente: %v, %v", versions, err)
	}
	if _, err := ListInstalled("../node"); err == nil {
		t.Fatal("linguagem insegura aceita")
	}
	expected := []string{"100000000000000000000000.0.0", "22.11.1", "22.11.0", "22.9.0", "9.10.0"}
	for _, version := range expected {
		root, err := config.InstallDir("node", version)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "node"), []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
		meta := Metadata{SchemaVersion: 1, Language: "node", Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/node.zip", Executables: map[string]string{"node": "node"}}
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, MetadataFileName), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	base, err := config.BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"23.0.0", "latest", "v22.0.0", "24.0.0"} {
		root := filepath.Join(base, "installs", "node", name)
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		if name == "24.0.0" {
			if err := os.WriteFile(filepath.Join(root, MetadataFileName), []byte("{invalid"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	versions, err = ListInstalled("node")
	if err != nil || !reflect.DeepEqual(versions, expected) {
		t.Fatalf("versões: %v, %v; esperado %v", versions, err, expected)
	}
	// Um executável removido invalida a instalação, mesmo com metadados presentes.
	root, err := config.InstallDir("node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "node")); err != nil {
		t.Fatal(err)
	}
	versions, err = ListInstalled("node")
	if err != nil || !reflect.DeepEqual(versions, []string{expected[0], expected[1], expected[3], expected[4]}) {
		t.Fatalf("instalação incompleta listada: %v, %v", versions, err)
	}
}

func TestListDirectoryError(t *testing.T) {
	isolatedHome(t)
	base, err := config.BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "installs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "installs", "node"), []byte("arquivo"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ListInstalled("node"); err == nil {
		t.Fatal("erro de leitura ocultado")
	}
}
