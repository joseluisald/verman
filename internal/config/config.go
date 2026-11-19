package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/joseluisald/verman/internal/durable"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/fslock"
	"github.com/joseluisald/verman/internal/tool"
)

const ProjectFileName = ".verman.toml"

type ProjectConfig struct {
	Tools map[string]string `toml:"tools" json:"tools"`
}

func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".verman"), nil
}

// InstallDir não impõe um layout de executáveis ao provedor.
func InstallDir(lang, version string) (string, error) {
	if err := tool.Validate(lang, version); err != nil {
		return "", err
	}
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "installs", lang, version), nil
}

func validate(cfg *ProjectConfig) error {
	for lang, version := range cfg.Tools {
		if err := tool.Validate(lang, version); err != nil {
			return err
		}
	}
	return nil
}

// LoadProjectConfig lê apenas o primeiro arquivo encontrado, sem mesclar os pais.
func LoadProjectConfig(startDir string) (*ProjectConfig, error) {
	cfg, _, err := LoadProjectConfigWithPath(startDir)
	return cfg, err
}

// LoadProjectConfigWithPath também informa qual arquivo definiu a seleção.
func LoadProjectConfigWithPath(startDir string) (*ProjectConfig, string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", err
	}
	for {
		path := filepath.Join(dir, ProjectFileName)
		data, err := os.ReadFile(path)
		if err == nil {
			cfg := &ProjectConfig{Tools: make(map[string]string)}
			if _, err := toml.Decode(string(data), cfg); err != nil {
				return nil, "", configurationError(path, err)
			}
			if err := validate(cfg); err != nil {
				return nil, "", configurationError(path, err)
			}
			return cfg, path, nil
		}
		if !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("ler %s: %w", path, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return &ProjectConfig{Tools: make(map[string]string)}, "", nil
		}
		dir = parent
	}
}

// SaveProjectTool grava somente no diretório informado, preparando o novo
// conteúdo antes de substituir o anterior. Preserva chaves, mas não comentários.
func SaveProjectTool(dir, lang, version string) error {
	return SaveProjectToolContext(context.Background(), dir, lang, version)
}
func SaveProjectToolContext(ctx context.Context, dir, lang, version string) error {
	return saveToolContext(ctx, dir, ProjectFileName, lang, version, false)
}

func saveTool(dir, name, lang, version string, remove bool) error {
	return saveToolContext(context.Background(), dir, name, lang, version, remove)
}
func saveToolContext(parent context.Context, dir, name, lang, version string, remove bool) error {
	if remove {
		version = "0.0.0"
	}
	if err := tool.Validate(lang, version); err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	unlock, err := fslock.Acquire(ctx, dir, name+".lock", false, true)
	if err != nil {
		return err
	}
	defer unlock()
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("configuração é link; gravação recusada: %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	document := make(map[string]any)
	mode := os.FileMode(0600)
	data, err := os.ReadFile(path)
	if err == nil {
		var cfg ProjectConfig
		if _, err := toml.Decode(string(data), &cfg); err != nil {
			return configurationError(path, err)
		}
		if err := validate(&cfg); err != nil {
			return configurationError(path, err)
		}
		if _, err := toml.Decode(string(data), &document); err != nil {
			return configurationError(path, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("ler %s: %w", path, err)
	}
	tools, ok := document["tools"].(map[string]any)
	if !ok {
		tools = make(map[string]any)
	}
	if remove {
		delete(tools, lang)
	} else {
		tools[lang] = version
	}
	document["tools"] = tools
	var encoded bytes.Buffer
	if err := toml.NewEncoder(&encoded).Encode(document); err != nil {
		return fmt.Errorf("codificar configuração: %w", err)
	}
	return durable.WriteContext(ctx, path, encoded.Bytes(), mode)
}

func configurationError(path string, err error) error {
	return fault.New("configuration_invalid", 6, "Corrija o arquivo TOML informado; nenhuma seleção nova foi salva.", fmt.Errorf("validar configuração %s: %w", path, err))
}
