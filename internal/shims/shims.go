package shims

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joseluisald/verman/internal/application"
	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/durable"
	"github.com/joseluisald/verman/internal/fslock"
	"github.com/joseluisald/verman/internal/shimmeta"
	"github.com/joseluisald/verman/internal/tool"
)

const Manifest = shimmeta.FileName

var Names = func() []string {
	names := []string{"verman"}
	for _, d := range tool.Definitions() {
		names = append(names, d.Commands...)
	}
	return names
}()

func Tool(executable string) string {
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(executable), filepath.Ext(executable)))
	for _, known := range Names[1:] {
		if name == known {
			return name
		}
	}
	return ""
}

func Directory() (string, error) {
	app, err := config.AppDir()
	if err != nil {
		return "", err
	}
	if app != "" {
		return filepath.Join(app, "bin"), nil
	}
	base, err := config.BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "bin"), nil
}

// Setup grava cópias nativas do próprio executável; nenhum shell interpreta argumentos.
func Setup(ctx context.Context, source, dir string) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	unlock, err := fslock.Acquire(ctx, dir, ".setup.lock", false, false)
	if err != nil {
		return err
	}
	defer unlock()
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	next := shimmeta.Manifest{Schema: 1, SHA256: hex.EncodeToString(hash[:])}
	app, err := config.AppDirForExecutable(source)
	if err != nil {
		return err
	}
	if app != "" && !strings.EqualFold(dir, filepath.Join(app, "bin")) {
		next.AppDir = app
	}
	var previous shimmeta.Manifest
	previousData, readErr := os.ReadFile(filepath.Join(dir, Manifest))
	if readErr == nil {
		if json.Unmarshal(previousData, &previous) != nil || previous.Schema != 1 {
			return fmt.Errorf("manifesto de shims inválido")
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	backups := map[string][]byte{}
	for _, name := range Names {
		path := filepath.Join(dir, name+".exe")
		if info, err := os.Lstat(path); err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("destino não é arquivo regular: %s", path)
			}
			old, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(old)
			if previous.Schema != 1 || hex.EncodeToString(sum[:]) != previous.SHA256 {
				return fmt.Errorf("arquivo existente não pertence aos shims gerenciados: %s", path)
			}
			backups[path] = old
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	changed := []string{}
	rollback := func(cause error) error {
		for _, path := range changed {
			if old, ok := backups[path]; ok {
				cause = errors.Join(cause, durable.Write(path, old, 0700))
			} else {
				cause = errors.Join(cause, os.Remove(path))
			}
		}
		return cause
	}
	for _, name := range Names {
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		path := filepath.Join(dir, name+".exe")
		if old, ok := backups[path]; ok {
			sum := sha256.Sum256(old)
			if hex.EncodeToString(sum[:]) == next.SHA256 {
				continue
			}
		}
		if err := durable.Write(path, data, 0700); err != nil {
			return rollback(err)
		}
		changed = append(changed, path)
	}
	encoded, err := json.Marshal(next)
	if err != nil {
		return rollback(err)
	}
	if err := durable.Write(filepath.Join(dir, Manifest), encoded, 0600); err != nil {
		return rollback(err)
	}
	return nil
}

func IsDirectory(dir string) bool {
	_, ok := shimmeta.Read(dir)
	return ok
}

func cleanPATH(environ []string) []string {
	result := append([]string{}, environ...)
	for i, value := range result {
		pair := strings.SplitN(value, "=", 2)
		if len(pair) != 2 || !strings.EqualFold(pair[0], "PATH") {
			continue
		}
		paths := []string{}
		for _, dir := range filepath.SplitList(pair[1]) {
			if !IsDirectory(strings.Trim(dir, `"`)) {
				paths = append(paths, dir)
			}
		}
		result[i] = pair[0] + "=" + strings.Join(paths, string(os.PathListSeparator))
	}
	return result
}

func Run(ctx context.Context, executable, tool string, args, environ []string, in io.Reader, out, diagnostics io.Writer) int {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	setupLease, err := fslock.Acquire(ctx, filepath.Dir(executable), ".setup.lock", true, false)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	defer setupLease()
	lease, err := application.ExecutionLease(ctx, dir)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	defer lease()
	plan, err := application.New().PlanExecution(dir, append([]string{tool}, args...), cleanPATH(environ))
	if err != nil {
		fmt.Fprintf(diagnostics, "Verman %s: %v\n", tool, err)
		return 1
	}
	if strings.EqualFold(filepath.Clean(plan.Executable), filepath.Clean(executable)) {
		fmt.Fprintln(diagnostics, "recursão de shim recusada")
		return 1
	}
	process := exec.CommandContext(ctx, plan.Executable, plan.Arguments...)
	process.Env = plan.Environment
	process.Dir = plan.Directory
	process.Stdin = in
	process.Stdout = out
	process.Stderr = diagnostics
	if err := process.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code := exit.ExitCode()
			if code > 0 {
				return code
			}
		}
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	return 0
}
