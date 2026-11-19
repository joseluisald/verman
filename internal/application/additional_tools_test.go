package application

import (
	"github.com/joseluisald/verman/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYarnProjectPackageManagerPin(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "package.json")
	original := []byte(`{"packageManager":"yarn@4.18.1+sha512.hash"}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "package.json"), []byte(`{"name":"workspace-child"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkYarnPin(nested, "4.18.1"); err != nil {
		t.Fatal(err)
	}
	if err := checkYarnPin(nested, "4.17.0"); err == nil || !strings.Contains(err.Error(), "verman use yarn 4.18.1") {
		t.Fatalf("pin ignorado: %v", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != string(original) {
		t.Fatal("package.json alterado")
	}
	if err := os.WriteFile(path, []byte(`{"packageManager":`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkYarnPin(nested, "4.18.1"); err == nil {
		t.Fatal("package.json inválido ignorado")
	}
}

func TestScriptToolDiagnosisRequiresSelectedRuntime(t *testing.T) {
	service, dir := isolatedService(t)
	fixture(t, "yarn", "4.18.1")
	if err := config.SaveProjectTool(dir, "yarn", "4.18.1"); err != nil {
		t.Fatal(err)
	}
	report, err := service.Diagnose(dir, []string{"yarn"}, nil)
	if err != nil || len(report.Tools) != 1 || report.Tools[0].Status != "dependency_missing" || !report.Issues {
		t.Fatalf("dependência ausente não diagnosticada: %+v %v", report, err)
	}
	if _, _, err := service.ResolveCommand(dir, []string{"yarn", "--version"}, nil); err == nil || !strings.Contains(err.Error(), "exige node") {
		t.Fatalf("exec sem Node: %v", err)
	}
}
