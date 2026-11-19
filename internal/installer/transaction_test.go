package installer

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func transactionFixture(t *testing.T, root, id, content string) {
	t.Helper()
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node"), []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
	meta := Metadata{SchemaVersion: 1, Language: "node", Version: "22.11.0", OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/runtime", TransactionID: id, Executables: map[string]string{"node": "node"}}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, MetadataFileName), data, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestInterruptedTransactions(t *testing.T) {
	for _, phase := range []string{"building", "prepared", "old_moved", "published", "remove_detached", "remove_partial"} {
		t.Run(phase, func(t *testing.T) {
			isolatedHome(t)
			base, err := configBase()
			if err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(base, "installs", "node", "22.11.0")
			transactionFixture(t, root, "", "OLD")
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			child := exec.Command(binary, "-test.run=^TestTransactionCrashHelper$")
			child.Env = append(os.Environ(), "VERMAN_TX_PHASE="+phase)
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
				t.Fatalf("helper %q %v", ready, err)
			}
			if err := child.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			child.Wait()
			if err := Recover(context.Background(), "node", "22.11.0"); err != nil {
				t.Fatal(err)
			}
			if phase == "remove_detached" || phase == "remove_partial" {
				if _, err := os.Stat(root); !os.IsNotExist(err) {
					t.Fatal("remoção não concluída")
				}
			} else {
				expected := "OLD"
				if phase == "published" {
					expected = "NEW"
				}
				data, err := os.ReadFile(filepath.Join(root, "node"))
				if err != nil || string(data) != expected {
					t.Fatalf("versão recuperada: %q %v", data, err)
				}
			}
			for _, name := range []string{stageName("22.11.0"), journalName("22.11.0")} {
				if _, err := os.Stat(filepath.Join(filepath.Dir(root), name)); !os.IsNotExist(err) {
					t.Fatalf("resíduo %s: %v", name, err)
				}
			}
		})
	}
}
func TestTransactionCrashHelper(t *testing.T) {
	phase := os.Getenv("VERMAN_TX_PHASE")
	if phase == "" {
		return
	}
	base, err := configBase()
	if err != nil {
		t.Fatal(err)
	}
	parent, err := managedParent(filepath.Join(base, "installs"), "node", true)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	unlock, err := lockVersion(parent, "node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	operation := "install"
	if phase == "remove_detached" || phase == "remove_partial" {
		operation = "remove"
	}
	tx, err := beginTransaction(parent, "22.11.0", operation)
	if err != nil {
		t.Fatal(err)
	}
	ready := func() {
		os.Stdout.WriteString("READY\n")
		for {
			time.Sleep(time.Second)
		}
	}
	if phase == "building" {
		ready()
	}
	if operation == "remove" {
		if err := parent.Rename("22.11.0", filepath.Join(stageName("22.11.0"), "runtime")); err != nil {
			t.Fatal(err)
		}
		if phase == "remove_partial" {
			if err := parent.Remove(filepath.Join(stageName("22.11.0"), "runtime", "node")); err != nil {
				t.Fatal(err)
			}
		}
		ready()
	}
	transactionFixture(t, filepath.Join(parent.Name(), stageName("22.11.0"), "runtime"), tx.record.ID, "NEW")
	tx.hook = func(current string) {
		if current == phase {
			ready()
		}
	}
	if err := tx.publish(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Fatal("fase não atingida")
}
func configBase() (string, error) {
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".verman"), err
}
