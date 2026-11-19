package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joseluisald/verman/internal/shimmeta"
	"github.com/joseluisald/verman/internal/tool"
)

// AppDir identifica a instalação empacotada pela localização do executável.
func AppDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return AppDirForExecutable(executable)
}

func AppDirForExecutable(executable string) (string, error) {
	executable, err := filepath.Abs(executable)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(executable)
	found, err := hasAppMarker(dir)
	if err != nil || found {
		return dir, err
	}
	if strings.EqualFold(filepath.Base(dir), "bin") {
		dir = filepath.Dir(dir)
		found, err := hasAppMarker(dir)
		if err != nil || found {
			return dir, err
		}
	}
	// Um setup --bin-dir externo pode registrar a origem dos seus shims.
	if manifest, ok := shimmeta.Read(filepath.Dir(executable)); ok && filepath.IsAbs(manifest.AppDir) {
		found, err := hasAppMarker(manifest.AppDir)
		if err != nil || found {
			return manifest.AppDir, err
		}
	}
	return "", nil
}

func hasAppMarker(dir string) (bool, error) {
	path := filepath.Join(dir, ".verman-app")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("consultar %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("marcador do Verman inválido: %s", path)
	}
	return true, nil
}

// InstallRoots retorna primeiro o destino de novas instalações.
func InstallRoots() ([]string, error) {
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}
	app, err := AppDir()
	if err != nil {
		return nil, err
	}
	legacy := filepath.Join(base, "installs")
	if app == "" {
		return []string{legacy}, nil
	}
	return []string{filepath.Join(app, "tools"), legacy}, nil
}

func InstallJournalName(version string) string { return ".transaction-" + version + ".json" }
func InstallStageName(version string) string   { return ".stage-" + version }

// InstallDir mantém cada versão existente no local onde foi instalada.
func InstallDir(language, version string) (string, error) {
	if err := tool.Validate(language, version); err != nil {
		return "", err
	}
	roots, err := InstallRoots()
	if err != nil {
		return "", err
	}
	return resolveInstallDir(roots, language, version)
}

func resolveInstallDir(roots []string, language, version string) (string, error) {
	if len(roots) > 1 {
		for _, root := range roots {
			parent := filepath.Join(root, language)
			for _, name := range []string{version, InstallJournalName(version), InstallStageName(version)} {
				path := filepath.Join(parent, name)
				if _, err := os.Lstat(path); err == nil {
					return filepath.Join(parent, version), nil
				} else if !os.IsNotExist(err) {
					return "", fmt.Errorf("consultar %s: %w", path, err)
				}
			}
		}
	}
	return filepath.Join(roots[0], language, version), nil
}
