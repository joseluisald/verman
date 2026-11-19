package application

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/installer"
)

type fakeRuntime struct {
	resolve func(string) string
	install func(context.Context, string, string, installer.EventSink) error
	called  bool
}

func (f *fakeRuntime) Resolve(ctx context.Context, language, requested string, sink installer.EventSink) (string, error) {
	f.called = true
	if f.resolve != nil {
		return f.resolve(requested), nil
	}
	return requested, nil
}
func (f *fakeRuntime) Install(ctx context.Context, language, version string, sink installer.EventSink) error {
	if f.install != nil {
		return f.install(ctx, language, version, sink)
	}
	return nil
}

func isolatedService(t *testing.T) (*Service, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return New(), t.TempDir()
}
func fixture(t *testing.T, language, version string) string {
	t.Helper()
	root, err := config.InstallDir(language, version)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	name := language
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(root, name), []byte("fixture: nunca executar"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(installer.Metadata{SchemaVersion: 1, Language: language, Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/runtime", Executables: map[string]string{language: name}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, installer.MetadataFileName), data, 0600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, name)
}

func TestSelectPersistsExactVersionAndReturnsData(t *testing.T) {
	svc, project := isolatedService(t)
	backend := &fakeRuntime{resolve: func(string) string { return "8.5.12" }, install: func(ctx context.Context, lang, version string, sink installer.EventSink) error {
		fixture(t, lang, version)
		return nil
	}}
	svc.runtime = backend
	result, err := svc.Select(context.Background(), SelectionRequest{Directory: project, Language: "php", Version: "latest", Scope: Project}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadProjectConfig(project)
	if err != nil || cfg.Tools["php"] != "8.5.12" || result.Version != "8.5.12" || result.SessionChanged || result.ConfigPath != filepath.Join(project, config.ProjectFileName) {
		t.Fatalf("seleção: %#v %#v %v", result, cfg, err)
	}
}

func TestSelectPreflightAndIncompleteInstallation(t *testing.T) {
	svc, project := isolatedService(t)
	fake := &fakeRuntime{}
	svc.runtime = fake
	path := filepath.Join(project, config.ProjectFileName)
	if err := os.WriteFile(path, []byte("[broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Select(context.Background(), SelectionRequest{Directory: project, Language: "php", Version: "8.5.12", Scope: Project}, nil); err == nil || fake.called {
		t.Fatal("configuração inválida chegou ao instalador")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Select(context.Background(), SelectionRequest{Directory: project, Language: "php", Version: "8.5.12", Scope: Project}, nil); err == nil {
		t.Fatal("instalação incompleta considerada válida")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("seleção gravada sem instalação válida")
	}
}

func TestSelectCancelledAfterInstallationPreservesConfig(t *testing.T) {
	svc, project := isolatedService(t)
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(project, config.ProjectFileName))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.runtime = &fakeRuntime{install: func(ctx context.Context, lang, version string, sink installer.EventSink) error {
		fixture(t, lang, version)
		cancel()
		return nil
	}}
	if _, err := svc.Select(ctx, SelectionRequest{Directory: project, Language: "php", Version: "8.5.12", Scope: Project}, nil); err != context.Canceled {
		t.Fatalf("cancelamento ignorado: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(project, config.ProjectFileName))
	if err != nil || string(after) != string(before) {
		t.Fatal("cancelamento alterou configuração")
	}
}

func TestSelectReturnsGlobalSaveError(t *testing.T) {
	svc, project := isolatedService(t)
	base, err := config.BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	svc.runtime = &fakeRuntime{install: func(ctx context.Context, lang, version string, sink installer.EventSink) error {
		fixture(t, lang, version)
		// Simula mudança do destino depois do preflight e antes de gravar.
		return os.Mkdir(filepath.Join(base, config.GlobalFileName), 0700)
	}}
	if _, err := svc.Select(context.Background(), SelectionRequest{Directory: project, Language: "php", Version: "8.5.12", Scope: Global}, nil); err == nil {
		t.Fatal("erro de gravação global retornou sucesso")
	}
}

func TestDiagnosisAndExecutionPlanDetectExternalPHP(t *testing.T) {
	svc, project := isolatedService(t)
	expected := fixture(t, "php", "8.5.12")
	if err := config.SaveProjectTool(project, "php", "8.5.12"); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	name := filepath.Base(expected)
	if err := os.WriteFile(filepath.Join(external, name), []byte("fixture externa: nunca executar"), 0700); err != nil {
		t.Fatal(err)
	}
	environ := []string{"PATH=" + external}
	report, err := svc.Diagnose(project, []string{"php"}, environ)
	if err != nil || !report.Issues || report.ShellAliasesInspected || report.Tools[0].PATHExecutable != filepath.Join(external, name) || report.Tools[0].ExpectedExecutable != expected || report.Tools[0].Status != "activation_required" {
		t.Fatalf("diagnóstico: %#v %v", report, err)
	}
	plan, err := svc.PlanExecution(project, []string{"php", "-v"}, environ)
	if err != nil || plan.Executable != expected || plan.Arguments[0] != "-v" {
		t.Fatalf("plano: %#v %v", plan, err)
	}
	report, err = svc.Diagnose(project, []string{"php"}, plan.Environment)
	if err != nil || report.Issues || !report.Tools[0].PATHMatches {
		t.Fatalf("ambiente ativado: %#v %v", report, err)
	}
	if environ[0] != "PATH="+external {
		t.Fatal("ambiente pai alterado")
	}
}
