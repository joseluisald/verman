package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joseluisald/verman/internal/application"
)

func TestDoctorJSONAndActivationInstructions(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("instruções PowerShell para o alvo Windows")
	}
	binary, project, _ := step9Setup(t)
	// Não execute as fixtures; doctor só compara arquivos e PATH.
	var stdout, stderr bytes.Buffer
	// Teste a CLI real com seu diretório de trabalho, sem alterar o cwd do teste.
	process := exec.Command(binary, "doctor", "node", "--json")
	process.Dir = project
	process.Stdout = &stdout
	process.Stderr = &stderr
	if err := process.Run(); err == nil {
		t.Fatal("doctor não retornou erro para PATH não ativado")
	}
	var report application.DiagnosticReport
	var envelope struct {
		Schema  int                          `json:"schema"`
		Success bool                         `json:"success"`
		Data    application.DiagnosticReport `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("JSON contaminado: %s %v", &stdout, err)
	}
	report = envelope.Data
	if envelope.Schema != 2 || envelope.Success {
		t.Fatalf("envelope: %#v", envelope)
	}
	if report.Schema != 1 || report.Tools[0].Status != "activation_required" || report.ShellAliasesInspected {
		t.Fatalf("relatório: %#v", report)
	}
	stdout.Reset()
	stderr.Reset()
	process = exec.Command(binary, "use", "node", "22.12.0")
	process.Dir = project
	process.Stdout = &stdout
	process.Stderr = &stderr
	if err := process.Run(); err != nil {
		t.Fatalf("use: %s %v", &stderr, err)
	}
	for _, text := range []string{"Seleção salva", "não altera o terminal atual", "env --shell powershell", "exec -- node --version"} {
		if !bytes.Contains(stdout.Bytes(), []byte(text)) {
			t.Fatalf("instrução ausente %q: %s", text, &stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(project, ".verman.toml")); err != nil {
		t.Fatal(err)
	}
}
