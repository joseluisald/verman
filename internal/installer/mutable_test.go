package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonMutablePackagesPreservedWithoutReplacingBootstrap(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	write := func(base, name, content string) {
		t.Helper()
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "Lib/site-packages/myapp/__init__.py", "USER")
	write(source, "Lib/site-packages/pip/__init__.py", "OLD")
	write(source, "Scripts/pip.exe", "OLD")
	write(source, "Scripts/pipenv.exe", "USER")
	write(dest, "Lib/site-packages/pip/__init__.py", "NEW")
	write(dest, "Scripts/pip.exe", "NEW")
	if err := preserveMutable(context.Background(), source, dest, "python"); err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string]string{"Lib/site-packages/myapp/__init__.py": "USER", "Lib/site-packages/pip/__init__.py": "NEW", "Scripts/pip.exe": "NEW", "Scripts/pipenv.exe": "USER"} {
		data, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil || string(data) != expected {
			t.Fatalf("%s: %q %v", name, data, err)
		}
	}
}
