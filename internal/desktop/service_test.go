package desktop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/installer"
)

func isolated(t *testing.T) *Service {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := New(context.Background())
	t.Cleanup(s.Close)
	return s
}
func fixture(t *testing.T, version string) {
	t.Helper()
	root, err := config.InstallDir("node", version)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node.exe"), []byte("fixture: nunca executar"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(installer.Metadata{SchemaVersion: 1, Language: "node", Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/node.zip", Executables: map[string]string{"node": "node.exe"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, installer.MetadataFileName), data, 0600); err != nil {
		t.Fatal(err)
	}
}
func awaitOperation(t *testing.T, s *Service) Operation {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ops := s.Operations()
		if len(ops) > 0 && ops[0].Status != "running" {
			return ops[0]
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("operação não terminou")
	return Operation{}
}
func TestSelectionUsesSameConfigAndPreservesOtherProject(t *testing.T) {
	s := isolated(t)
	fixture(t, "22.11.0")
	fixture(t, "22.12.0")
	a, b := t.TempDir(), t.TempDir()
	if err := config.SaveProjectTool(b, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveGlobalTool("node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Start(Request{Action: "select", Language: "node", Version: "22.12.0", Directory: a, Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if op := awaitOperation(t, s); op.Status != "succeeded" {
		t.Fatalf("seleção: %#v", op)
	}
	cli, err := config.LoadEffectiveConfig(a)
	if err != nil || cli.Tools["node"] != "22.12.0" {
		t.Fatalf("CLI não leu seleção do desktop: %#v %v", cli, err)
	}
	other, err := s.Snapshot(b)
	if err != nil || other.Tools[0].ProjectVersion != "22.11.0" || other.Tools[0].GlobalVersion != "22.11.0" {
		t.Fatalf("outro projeto alterado: %#v %v", other, err)
	}
}
func TestCancellationAndWindowConcurrency(t *testing.T) {
	s := isolated(t)
	ready := make(chan struct{})
	s.run = func(ctx context.Context, r Request, sink installer.EventSink) error {
		if err := sink(installer.Event{Kind: installer.EventProgress, Bytes: 1024, Total: 4096}); err != nil {
			return err
		}
		close(ready)
		<-ctx.Done()
		return ctx.Err()
	}
	r := Request{Action: "install", Language: "node", Version: "22.11.0"}
	id, err := s.Start(r)
	if err != nil {
		t.Fatal(err)
	}
	<-ready
	if _, err := s.Start(r); fault.Classify(err).Code != "operation_busy" {
		t.Fatalf("concorrência não bloqueada: %v", err)
	}
	if err := s.Cancel(id); err != nil {
		t.Fatal(err)
	}
	op := awaitOperation(t, s)
	if op.Status != "canceled" || op.ErrorCode != "canceled" || op.Bytes != 1024 || op.Total != 4096 || s.Busy() {
		t.Fatalf("cancelamento/progresso: %#v", op)
	}
}
func TestCLILeaseBlocksDesktopRemoval(t *testing.T) {
	s := isolated(t)
	fixture(t, "22.11.0")
	lease, err := installer.Lease(context.Background(), "node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	defer lease()
	_, err = s.Start(Request{Action: "uninstall", Language: "node", Version: "22.11.0", Directory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	op := awaitOperation(t, s)
	if op.Status != "failed" || op.ErrorCode != "operation_busy" {
		t.Fatalf("lease da CLI ignorado: %#v", op)
	}
	if !installer.IsInstalled("node", "22.11.0") {
		t.Fatal("instalação em execução foi alterada")
	}
}
func TestRejectsInvalidRequestsBeforeStarting(t *testing.T) {
	s := isolated(t)
	for _, r := range []Request{{Action: "install", Language: "../node", Version: "22.11.0"}, {Action: "select", Language: "node", Version: "22.11.0", Scope: "project", Directory: "relative"}, {Action: "execute", Language: "node", Version: "22.11.0"}, {Action: "install", Language: "node", Version: "latest"}} {
		if _, err := s.Start(r); err == nil {
			t.Fatalf("solicitação inválida aceita: %#v", r)
		}
	}
	if len(s.Operations()) != 0 {
		t.Fatal("pedido inválido criou operação")
	}
}
