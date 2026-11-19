package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func validateRuntime(ctx context.Context, root, destination string, a Artifact) error {
	switch a.Language {
	case "node":
		return validateNode(ctx, root, destination, a)
	case "php":
		return preparePHP(ctx, root, destination, a)
	case "python":
		return preparePython(ctx, root, a)
	default:
		return prepareTool(ctx, root, a)
	}
}

func runRuntime(ctx context.Context, root, executable string, args ...string) (string, error) {
	if _, err := executablePath(root, executable); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(root, filepath.FromSlash(executable)), args...)
	cmd.Dir = root
	cmd.WaitDelay = time.Second
	for _, env := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(env, "=", 2)[0])
		if key == "PHPRC" || key == "PHP_INI_SCAN_DIR" || key == "PYTHONHOME" || key == "PYTHONPATH" || key == "VIRTUAL_ENV" || strings.HasPrefix(key, "PIP_") {
			continue
		}
		cmd.Env = append(cmd.Env, env)
	}
	var stdout, stderr limitedOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("executar %s %v: %w (%s)", executable, args, err, stderr.data)
	}
	if len(stderr.data) > 0 {
		return "", fmt.Errorf("diagnóstico inesperado de %s: %s", executable, stderr.data)
	}
	return strings.TrimSpace(string(stdout.data)), nil
}

func preparePHP(ctx context.Context, root, destination string, a Artifact) error {
	version, err := runRuntime(ctx, root, "php.exe", "-n", "-r", "echo PHP_VERSION;")
	if err != nil {
		return fmt.Errorf("PHP requer o Microsoft Visual C++ Redistributable x64 compatível: %w", err)
	}
	if version != a.Version {
		return fmt.Errorf("versão PHP executada %q difere de %s", version, a.Version)
	}
	ini, err := os.ReadFile(filepath.Join(root, "php.ini-development"))
	if err != nil {
		return fmt.Errorf("modelo php.ini-development ausente: %w", err)
	}
	extensionDir, err := phpExtensionDir(destination)
	if err != nil {
		return err
	}
	ini = append(ini, []byte("\n; Verman: local configuration; optional extensions remain disabled.\nextension_dir = \"")...)
	ini = append(ini, extensionDir...)
	ini = append(ini, []byte("\"\n")...)
	file, err := os.OpenFile(filepath.Join(root, "php.ini"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(ini)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Mkdir(filepath.Join(root, "conf.d"), 0700); err != nil {
		return err
	}
	_, err = runRuntime(ctx, root, "php.exe", "-c", filepath.Join(root, "php.ini"), "-d", "extension_dir="+filepath.Join(root, "ext"), "--version")
	return err
}

func preparePython(ctx context.Context, root string, a Artifact) error {
	version, err := runRuntime(ctx, root, "python.exe", "-I", "-c", "import platform; print(platform.python_version())")
	if err != nil {
		return err
	}
	if version != a.Version {
		return fmt.Errorf("versão Python executada %q difere de %s", version, a.Version)
	}
	if _, err := runRuntime(ctx, root, "python.exe", "-I", "-m", "pip", "--version"); err != nil {
		return fmt.Errorf("pip indisponível: %w", err)
	}
	// O wrapper relativo continua válido após publicar/mover a distribuição.
	if err := os.WriteFile(filepath.Join(root, "pip.cmd"), []byte("@echo off\r\n\"%~dp0python.exe\" -m pip %*\r\n"), 0700); err != nil {
		return err
	}
	venv, err := os.MkdirTemp(filepath.Dir(root), ".venv-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(venv)
	if _, err := runRuntime(ctx, root, "python.exe", "-I", "-m", "venv", venv); err != nil {
		return fmt.Errorf("criar venv: %w", err)
	}
	if _, err := runRuntime(ctx, venv, "Scripts/python.exe", "-I", "-c", "import sys; assert sys.prefix != sys.base_prefix; print(sys.version)"); err != nil {
		return fmt.Errorf("executar venv: %w", err)
	}
	if _, err := runRuntime(ctx, venv, "Scripts/python.exe", "-I", "-m", "pip", "--version"); err != nil {
		return fmt.Errorf("pip do venv: %w", err)
	}
	return nil
}
