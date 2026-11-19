package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/joseluisald/verman/internal/tool"
)

const GlobalFileName = "config.toml"

func LoadGlobalConfig() (*ProjectConfig, error) {
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, GlobalFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &ProjectConfig{Tools: map[string]string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler seleção global: %w", err)
	}
	cfg := &ProjectConfig{Tools: map[string]string{}}
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, configurationError(path, err)
	}
	if err := validate(cfg); err != nil {
		return nil, configurationError(path, err)
	}
	return cfg, nil
}

// LoadEffectiveConfig usa o arquivo de projeto mais próximo e completa somente
// as linguagens não declaradas nele com a seleção global. Não mescla projetos ancestrais.
func LoadEffectiveConfig(dir string) (*ProjectConfig, error) {
	global, err := LoadGlobalConfig()
	if err != nil {
		return nil, err
	}
	project, err := LoadProjectConfig(dir)
	if err != nil {
		return nil, err
	}
	for language, version := range project.Tools {
		global.Tools[language] = version
	}
	return global, nil
}

func SaveGlobalTool(language, version string) error {
	return SaveGlobalToolContext(context.Background(), language, version)
}
func SaveGlobalToolContext(ctx context.Context, language, version string) error {
	if err := tool.Validate(language, version); err != nil {
		return err
	}
	base, err := BaseDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	return saveToolContext(ctx, base, GlobalFileName, language, version, false)
}

func RemoveGlobalTool(language string) error {
	if err := tool.ValidateLanguage(language); err != nil {
		return err
	}
	base, err := BaseDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	return saveTool(base, GlobalFileName, language, "", true)
}
