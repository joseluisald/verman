package environment

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAdditionalToolVariablesRestoreWithoutChangingParent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "espaço O'Brien $cash")
	vars := map[string]*string{}
	environ := []string{"PATH=original"}
	for _, key := range managedVariables {
		copy := dir
		vars[key] = &copy
		environ = append(environ, key+"=original-"+key)
	}
	activated, err := applySelection([]string{dir}, vars, environ)
	if err != nil {
		t.Fatal(err)
	}
	cleared, err := applySelection(nil, map[string]*string{}, activated)
	if err != nil {
		t.Fatal(err)
	}
	values := envMap(cleared)
	for _, key := range managedVariables {
		if values[envKey(key)] != "original-"+key {
			t.Fatalf("%s não restaurada", key)
		}
	}
	if runtime.GOOS != "windows" {
		return
	}
	t.Setenv("PSModuleAnalysisCachePath", filepath.Join(t.TempDir(), "ps-cache"))
	code, err := renderPowerShell([]string{dir}, vars)
	if err != nil {
		t.Fatal(err)
	}
	clear, err := renderPowerShell(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	script := "$ErrorActionPreference='Stop'\nRemove-Item Env:VERMAN_ENV_BACKUP,Env:VERMAN_PATH_ENTRIES -ErrorAction SilentlyContinue\n"
	for _, key := range managedVariables {
		script += "$env:" + key + "='original-" + key + "'\n"
	}
	script += code + code + clear
	for _, key := range managedVariables {
		script += "if ($env:" + key + " -cne 'original-" + key + "') { throw 'restauração " + key + "' }\n"
	}
	script += "Write-Output 'OK'"
	path := filepath.Join(t.TempDir(), "test.ps1")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(script)...), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "OK" {
		t.Fatalf("PowerShell: %v %s", err, output)
	}
}
