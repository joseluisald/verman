package installer

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBusyRuntimePreserved(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("bloqueio de imagem do Windows")
	}
	isolatedHome(t)
	base, err := configBase()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "installs", "node", "22.11.0")
	transactionFixture(t, root, "", "OLD")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	childPath := filepath.Join(root, "helper.exe")
	if err := os.WriteFile(childPath, data, 0700); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(childPath, "-test.run=^TestBusyHelper$")
	child.Env = append(os.Environ(), "VERMAN_BUSY_HELPER=1")
	child.Stderr = os.Stderr
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	ready, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || ready != "READY\n" {
		t.Fatalf("helper: %q %v", ready, err)
	}
	if err := Uninstall(context.Background(), t.TempDir(), "node", "22.11.0", true); err == nil {
		t.Fatal("runtime ocupado removido")
	}
	if original, err := os.ReadFile(filepath.Join(root, "node")); err != nil || string(original) != "OLD" {
		t.Fatal("remoção parcial de instalação ocupada")
	}
	if _, err := os.Stat(filepath.Join(root, MetadataFileName)); err != nil {
		t.Fatal("metadados removidos de instalação ocupada")
	}
	child.Process.Kill()
	child.Wait()
	if err := Uninstall(context.Background(), t.TempDir(), "node", "22.11.0", true); err != nil {
		t.Fatal(err)
	}
}

func TestReadonlyRuntimePreserved(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("atributo somente leitura do Windows")
	}
	isolatedHome(t)
	root := uninstallFixture(t, "22.11.0")
	path := filepath.Join(root, "node")
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0700)
	if err := Uninstall(context.Background(), t.TempDir(), "node", "22.11.0", true); err == nil || !strings.Contains(err.Error(), "somente leitura") {
		t.Fatalf("atributo não respeitado: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "fixture" {
		t.Fatal("arquivo original alterado")
	}
	if _, err := os.Stat(filepath.Join(root, MetadataFileName)); err != nil {
		t.Fatal("metadados removidos")
	}
}
func TestBusyHelper(t *testing.T) {
	if os.Getenv("VERMAN_BUSY_HELPER") == "1" {
		os.Stdout.WriteString("READY\n")
		for {
			time.Sleep(time.Second)
		}
	}
}

func TestIntegrityAndMutableConfiguration(t *testing.T) {
	isolatedHome(t)
	base, err := configBase()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "installs", "php", "8.5.12")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"php.exe": "ORIG", "php.ini": "custom=original", "empty.py": ""} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	meta := Metadata{SchemaVersion: 1, Language: "php", Version: "8.5.12", OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/php.zip", SHA256: strings.Repeat("a", 64), Executables: map[string]string{"php": "php.exe"}}
	if err := fingerprint(root, &meta); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, MetadataFileName), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify("php", "8.5.12"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "php.ini"), []byte("custom=changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Verify("php", "8.5.12"); err != nil {
		t.Fatal("configuração mutável entrou nos hashes")
	}
	if err := os.WriteFile(filepath.Join(root, "php.exe"), []byte("EVIL"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Verify("php", "8.5.12"); err == nil {
		t.Fatal("alteração de mesmo tamanho não detectada")
	}
	if err := os.WriteFile(filepath.Join(root, "php.exe"), []byte("ORIG"), 0700); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if err := preserveMutable(context.Background(), root, destination, "php"); err != nil {
		t.Fatal(err)
	}
	if saved, err := os.ReadFile(filepath.Join(destination, "php.ini")); err != nil || string(saved) != "custom=changed" {
		t.Fatal("configuração personalizada perdida")
	}
}

func TestCancelledPublishRestoresPrevious(t *testing.T) {
	isolatedHome(t)
	base, err := configBase()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "installs", "node", "22.11.0")
	transactionFixture(t, root, "", "OLD")
	parent, err := managedParent(filepath.Join(base, "installs"), "node", false)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	unlock, err := lockVersion(parent, "node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	tx, err := beginTransaction(parent, "22.11.0", "install")
	if err != nil {
		t.Fatal(err)
	}
	transactionFixture(t, filepath.Join(parent.Name(), stageName("22.11.0"), "runtime"), tx.record.ID, "NEW")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx.hook = func(phase string) {
		if phase == "old_moved" {
			cancel()
		}
	}
	if err := tx.publish(ctx); err != context.Canceled {
		t.Fatalf("cancelamento: %v", err)
	}
	if err := tx.finish(); err != nil {
		t.Fatal(err)
	}
	if saved, err := os.ReadFile(filepath.Join(root, "node")); err != nil || string(saved) != "OLD" {
		t.Fatal("instalação anterior não restaurada")
	}
}
