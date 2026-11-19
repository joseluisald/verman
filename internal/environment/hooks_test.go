package environment

import (
	"strings"
	"testing"
)

func TestHookRejectsTemporaryGoRunExecutable(t *testing.T) {
	for _, path := range []string{`C:\Users\test\AppData\Local\Temp\go-build12345\b001\exe\verman.exe`, "/tmp/go-build12345/b001/exe/verman", `C:\Users\test\AppData\Local\go-build\aa\` + strings.Repeat("a", 64) + `-d\verman.exe`} {
		if _, err := Hook("powershell", path, false); err == nil {
			t.Fatalf("hook temporário aceito: %s", path)
		}
	}
	if _, err := Hook("powershell", `C:\tools\verman.exe`, false); err != nil {
		t.Fatal(err)
	}
}
