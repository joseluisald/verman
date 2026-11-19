// Package application contém casos de uso compartilhados pela CLI e pelo futuro desktop.
package application

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/tool"
)

type Scope string

const (
	Project Scope = "project"
	Global  Scope = "global"
)

type SelectionRequest struct {
	Directory, Language, Version string
	Scope                        Scope
}
type SelectionResult struct {
	Language       string `json:"language"`
	Version        string `json:"version"`
	Scope          Scope  `json:"scope"`
	ConfigPath     string `json:"config_path"`
	Executable     string `json:"executable"`
	SessionChanged bool   `json:"session_changed"`
}

type runtimeBackend interface {
	Resolve(context.Context, string, string, installer.EventSink) (string, error)
	Install(context.Context, string, string, installer.EventSink) error
}
type backend struct{}

func (backend) Resolve(ctx context.Context, lang, version string, sink installer.EventSink) (string, error) {
	return installer.ResolveVersionEvents(ctx, lang, version, sink)
}
func (backend) Install(ctx context.Context, lang, version string, sink installer.EventSink) error {
	return installer.InstallWithEvents(ctx, lang, version, sink)
}

type Service struct{ runtime runtimeBackend }

func New() *Service { return &Service{runtime: backend{}} }

// Select prepara a seleção sem mudar o ambiente do chamador. Eventos opcionais
// permitem progresso/diagnóstico sem vincular o caso de uso a stdout ou a um shell.
func (s *Service) Select(ctx context.Context, request SelectionRequest, sink installer.EventSink) (SelectionResult, error) {
	if err := ctx.Err(); err != nil {
		return SelectionResult{}, err
	}
	if err := tool.ValidateLanguage(request.Language); err != nil {
		return SelectionResult{}, err
	}
	if request.Scope != Project && request.Scope != Global {
		return SelectionResult{}, fmt.Errorf("escopo de seleção inválido: %q", request.Scope)
	}
	if request.Scope == Project && request.Directory == "" {
		return SelectionResult{}, fmt.Errorf("informe o diretório do projeto")
	}
	dir, err := filepath.Abs(request.Directory)
	if err != nil {
		return SelectionResult{}, err
	}
	// Não faça downloads para uma seleção que já sabemos não poder gravar.
	ctx = installer.WithWorkingDirectory(ctx, dir)
	if request.Scope == Global {
		_, err = config.LoadGlobalConfig()
	} else {
		_, err = config.LoadProjectConfig(dir)
	}
	if err != nil {
		return SelectionResult{}, err
	}
	version, err := s.runtime.Resolve(ctx, request.Language, request.Version, sink)
	if err != nil {
		return SelectionResult{}, err
	}
	if err := tool.Validate(request.Language, version); err != nil {
		return SelectionResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return SelectionResult{}, err
	}
	meta, readErr := installer.ReadMetadata(request.Language, version)
	if readErr != nil {
		if err := s.runtime.Install(ctx, request.Language, version, sink); err != nil {
			return SelectionResult{}, err
		}
		meta, err = installer.ReadMetadata(request.Language, version)
		if err != nil {
			return SelectionResult{}, fmt.Errorf("instalador não produziu uma instalação válida: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return SelectionResult{}, err
	}
	configPath := filepath.Join(dir, config.ProjectFileName)
	lease, err := installer.Lease(ctx, request.Language, version)
	if err != nil {
		return SelectionResult{}, err
	}
	defer lease()
	if _, err := installer.ReadMetadata(request.Language, version); err != nil {
		return SelectionResult{}, err
	}
	if request.Scope == Global {
		base, baseErr := config.BaseDir()
		if baseErr != nil {
			return SelectionResult{}, baseErr
		}
		configPath = filepath.Join(base, config.GlobalFileName)
		err = config.SaveGlobalToolContext(ctx, request.Language, version)
	} else {
		err = config.SaveProjectToolContext(ctx, dir, request.Language, version)
	}
	if err != nil {
		return SelectionResult{}, err
	}
	root, err := config.InstallDir(request.Language, version)
	if err != nil {
		return SelectionResult{}, err
	}
	return SelectionResult{Language: request.Language, Version: version, Scope: request.Scope, ConfigPath: configPath, Executable: filepath.Join(root, meta.Executables[request.Language]), SessionChanged: false}, nil
}
