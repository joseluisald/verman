package fslock

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCrashReleasesLock(t *testing.T) {
	dir := t.TempDir()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(binary, "-test.run=^TestLockOwnerHelper$")
	child.Env = append(os.Environ(), "VERMAN_LOCK_HELPER="+dir)
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	reader := bufio.NewReader(stdout)
	ready, err := reader.ReadString('\n')
	if err != nil || ready != "READY\n" {
		t.Fatalf("helper: %q %v", ready, err)
	}
	if _, err := Acquire(context.Background(), dir, "owner.lock", false, false); err == nil {
		t.Fatal("lock ativo ignorado")
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	child.Wait()
	unlock, err := Acquire(context.Background(), dir, "owner.lock", false, false)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if _, err := os.Stat(filepath.Join(dir, "owner.lock")); err != nil {
		t.Fatal("arquivo de identidade do lock removido")
	}
}
func TestLockOwnerHelper(t *testing.T) {
	dir := os.Getenv("VERMAN_LOCK_HELPER")
	if dir == "" {
		return
	}
	unlock, err := Acquire(context.Background(), dir, "owner.lock", false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	os.Stdout.WriteString("READY\n")
	for {
		time.Sleep(time.Second)
	}
}
