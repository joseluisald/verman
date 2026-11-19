package installer

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func uninstallFixture(t *testing.T, version string) string {
	t.Helper()
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
	data, err := json.Marshal(Metadata{SchemaVersion: 1, Language: "node", Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/node.zip", Executables: map[string]string{"node": "node"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, MetadataFileName), data, 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestUninstallSelectionAndScope(t *testing.T) {
	isolatedHome(t)
	first := uninstallFixture(t, "22.11.0")
	second := uninstallFixture(t, "22.12.0")
	project := t.TempDir()
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), project, "node", "22.11.0", false); err == nil || !strings.Contains(err.Error(), "--allow-selected") {
		t.Fatalf("seleção não protegida: %v", err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), project, "node", "22.11.0", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("versão não removida")
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatal("outra versão afetada")
	}
	cfg, err := config.LoadProjectConfig(project)
	if err != nil || cfg.Tools["node"] != "22.11.0" {
		t.Fatalf("seleção alterada: %#v %v", cfg, err)
	}
	versions, err := ListInstalled("node")
	if err != nil || len(versions) != 1 || versions[0] != "22.12.0" {
		t.Fatalf("listagem: %v %v", versions, err)
	}
	if err := Uninstall(context.Background(), project, "node", "22.12.0", false); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRejectsUnsafeIncompleteAndLocked(t *testing.T) {
	isolatedHome(t)
	project := t.TempDir()
	root := uninstallFixture(t, "22.11.0")
	for _, pair := range [][2]string{{"../node", "22.11.0"}, {"node", "../../escape"}, {"node", "22.12.0"}} {
		if err := Uninstall(context.Background(), project, pair[0], pair[1], true); err == nil {
			t.Fatalf("entrada aceita: %v", pair)
		}
	}
	lock := filepath.Join(filepath.Dir(root), ".lock-22.11.0")
	if err := os.Mkdir(lock, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), project, "node", "22.11.0", true); err == nil || !strings.Contains(err.Error(), "bloquear") {
		t.Fatalf("bloqueio ignorado: %v", err)
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Uninstall(ctx, project, "node", "22.11.0", true); err == nil {
		t.Fatal("cancelamento ignorado")
	}
	if err := os.Remove(filepath.Join(root, MetadataFileName)); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), project, "node", "22.11.0", true); err == nil {
		t.Fatal("pasta não gerenciada aceita")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("instalação foi removida")
	}
}

func TestUninstallRejectsExternalLink(t *testing.T) {
	isolatedHome(t)
	root := uninstallFixture(t, "22.11.0")
	external := t.TempDir()
	marker := filepath.Join(external, "preservar")
	if err := os.WriteFile(marker, []byte("preservar"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(root), "22.12.0")
	makeDirectoryLink(t, external, link)
	if err := Uninstall(context.Background(), t.TempDir(), "node", "22.12.0", true); err == nil {
		t.Fatal("link externo aceito")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("destino externo afetado")
	}
	// Um link dentro de uma instalação válida é removido sem percorrer o destino.
	makeDirectoryLink(t, external, filepath.Join(root, "external"))
	if err := Uninstall(context.Background(), t.TempDir(), "node", "22.11.0", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("remoção percorreu link interno")
	}
}

func TestUninstallProtectsGlobalSelection(t *testing.T) {
	isolatedHome(t)
	uninstallFixture(t, "22.11.0")
	if err := config.SaveGlobalTool("node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	if err := Uninstall(context.Background(), project, "node", "22.11.0", false); err == nil || !strings.Contains(err.Error(), "--allow-selected") {
		t.Fatalf("global não protegido: %v", err)
	}
	if err := Uninstall(context.Background(), project, "node", "22.11.0", true); err != nil {
		t.Fatal(err)
	}
	global, err := config.LoadGlobalConfig()
	if err != nil || global.Tools["node"] != "22.11.0" {
		t.Fatal("remoção alterou seleção global")
	}
}

func makeDirectoryLink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		return
	}
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd.exe", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("criar junction: %s, %v", out, err)
		}
		return
	}
	t.Skip("sistema não permite criar links")
}
