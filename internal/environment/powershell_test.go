package environment

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func TestPowerShellSession(t *testing.T) {
	t.Setenv("PSModuleAnalysisCachePath", filepath.Join(t.TempDir(), "powershell-cache"))
	if runtime.GOOS != "windows" {
		t.Skip("validação do shell Windows")
	}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "primeiro espaço O'Brien $cash & (test)")
	second := filepath.Join(t.TempDir(), "segundo espaço")
	code1, err := RenderPowerShell([]string{first})
	if err != nil {
		t.Fatal(err)
	}
	code2, err := RenderPowerShell([]string{second})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := RenderPowerShell(nil)
	if err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "$ErrorActionPreference = 'Stop'\n$env:PATH = 'original-one;original-two'\nRemove-Item Env:VERMAN_PATH_ENTRIES -ErrorAction SilentlyContinue\n" + code1 + code1 +
		"if ($env:PATH -cne (" + quote(first) + " + ';original-one;original-two')) { throw 'duplicação ou escape incorreto' }\n" + code2 +
		"if ($env:PATH -cne (" + quote(second) + " + ';original-one;original-two')) { throw 'versão anterior não removida' }\n" + empty +
		"if ($env:PATH -cne 'original-one;original-two') { throw 'ambiente original não restaurado' }\nif ($env:VERMAN_PATH_ENTRIES) { throw 'marcador residual' }\n" +
		"$env:VERMAN_PATH_ENTRIES = '{invalid'\ntry {\n" + code1 + "throw 'falha esperada'\n} catch { if ($_.Exception.Message -notlike '*PATH nao foi alterado*') { throw } }\nif ($env:PATH -cne 'original-one;original-two') { throw 'erro alterou PATH' }\nWrite-Output 'OK'\n"
	withVars, err := renderPowerShell(nil, map[string]*string{"PHPRC": &first, "PYTHONHOME": nil})
	if err != nil {
		t.Fatal(err)
	}
	variableChecks := "Remove-Item Env:VERMAN_PATH_ENTRIES -ErrorAction SilentlyContinue\nRemove-Item Env:VERMAN_ENV_BACKUP -ErrorAction SilentlyContinue\n$env:PHPRC = 'original-php'\n$env:PYTHONHOME = 'original-python'\n" + withVars + withVars +
		"if ($env:PHPRC -ne " + quote(first) + " -or $env:PYTHONHOME) { throw 'variaveis nao aplicadas' }\n" + empty +
		"if ($env:PHPRC -ne 'original-php' -or $env:PYTHONHOME -ne 'original-python') { throw 'variaveis nao restauradas' }\nif ($env:VERMAN_ENV_BACKUP) { throw 'backup residual' }\nWrite-Output 'OK'"
	script = strings.ReplaceAll(script, "Write-Output 'OK'", variableChecks)
	path := filepath.Join(t.TempDir(), "test.ps1")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(script)...), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "OK" {
		t.Fatalf("PowerShell: %s, %v", out, err)
	}
}

func TestProjectActivationValidation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	if code, err := PowerShell(project); err == nil || code != "" {
		t.Fatalf("instalação ausente aceita: %q, %v", code, err)
	}
	root, err := config.InstallDir("node", "22.11.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node"), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	meta := map[string]any{"schema_version": 1, "language": "node", "version": "22.11.0", "os": runtime.GOOS, "arch": runtime.GOARCH, "source": "https://example.test/node.zip", "executables": map[string]string{"node": "node"}}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "installation.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	first, err := PowerShell(project)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PowerShell(project)
	if err != nil || first != second {
		t.Fatal("saída não determinística")
	}
}
