package application

import "testing"

func TestDiagnoseVirtualEnvironmentWithoutChangingIt(t *testing.T) {
	svc, dir := isolatedService(t)
	environment := []string{"PATH=", "VIRTUAL_ENV=C:\\project\\.venv"}
	report, err := svc.Diagnose(dir, []string{"python"}, environment)
	if err != nil || report.VirtualEnvironment != "C:\\project\\.venv" || len(report.Warnings) == 0 {
		t.Fatalf("diagnóstico: %#v %v", report, err)
	}
	if environment[1] != "VIRTUAL_ENV=C:\\project\\.venv" {
		t.Fatal("ambiente alterado")
	}
}
