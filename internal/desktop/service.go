// Package desktop adapta os mesmos casos de uso da CLI, sem interpretar stdout.
package desktop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/joseluisald/verman/internal/application"
	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/tool"
)

type Runtime struct {
	Language       string   `json:"language"`
	Installed      []string `json:"installed"`
	ProjectVersion string   `json:"project_version"`
	GlobalVersion  string   `json:"global_version"`
}
type Snapshot struct {
	Directory  string                       `json:"directory"`
	ConfigPath string                       `json:"config_path"`
	Tools      []Runtime                    `json:"tools"`
	Diagnosis  application.DiagnosticReport `json:"diagnosis"`
}
type Release struct {
	Version   string                     `json:"version"`
	Build     string                     `json:"build"`
	Variant   string                     `json:"variant"`
	Installed bool                       `json:"installed"`
	Policy    installer.DistributionInfo `json:"policy"`
}
type Catalog struct {
	Language  string    `json:"language"`
	Versions  []Release `json:"versions"`
	FetchedAt time.Time `json:"fetched_at"`
	Warning   string    `json:"warning"`
	FromCache bool      `json:"from_cache"`
	Stale     bool      `json:"stale"`
}
type Request struct {
	Action        string            `json:"action"`
	Language      string            `json:"language"`
	Version       string            `json:"version"`
	Directory     string            `json:"directory"`
	Scope         application.Scope `json:"scope"`
	AllowSelected bool              `json:"allow_selected"`
}
type Operation struct {
	ID        string     `json:"id"`
	Request   Request    `json:"request"`
	Status    string     `json:"status"`
	Phase     string     `json:"phase"`
	Bytes     int64      `json:"bytes"`
	Total     int64      `json:"total"`
	Message   string     `json:"message"`
	ErrorCode string     `json:"error_code,omitempty"`
	Hint      string     `json:"hint,omitempty"`
	Started   time.Time  `json:"started"`
	Finished  *time.Time `json:"finished,omitempty"`
}
type Service struct {
	ctx        context.Context
	mu         sync.Mutex
	operations map[string]*Operation
	cancels    map[string]context.CancelFunc
	run        func(context.Context, Request, installer.EventSink) error
}

func New(ctx context.Context) *Service {
	s := &Service{ctx: ctx, operations: map[string]*Operation{}, cancels: map[string]context.CancelFunc{}}
	s.run = execute
	return s
}
func projectDirectory(dir string) (string, error) {
	if dir == "" || !filepath.IsAbs(dir) {
		return "", fault.Usage("selecione uma pasta de projeto com caminho absoluto")
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fault.Usage("o projeto precisa ser uma pasta")
	}
	return resolved, nil
}
func (s *Service) Snapshot(dir string) (Snapshot, error) {
	var err error
	if dir == "" {
		dir, err = os.UserHomeDir()
	} else {
		dir, err = projectDirectory(dir)
	}
	if err != nil {
		return Snapshot{}, err
	}
	project, path, err := config.LoadProjectConfigWithPath(dir)
	if err != nil {
		return Snapshot{}, err
	}
	global, err := config.LoadGlobalConfig()
	if err != nil {
		return Snapshot{}, err
	}
	report, err := application.New().Diagnose(dir, nil, os.Environ())
	if err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{Directory: dir, ConfigPath: path, Tools: []Runtime{}, Diagnosis: report}
	for _, lang := range tool.Languages() {
		versions, err := installer.ListInstalled(lang)
		if err != nil {
			return Snapshot{}, err
		}
		if versions == nil {
			versions = []string{}
		}
		result.Tools = append(result.Tools, Runtime{lang, versions, project.Tools[lang], global.Tools[lang]})
	}
	return result, nil
}
func (s *Service) Catalog(language string, history, refresh, offline bool) (Catalog, error) {
	ctx := s.ctx
	if offline {
		ctx = installer.WithOffline(ctx)
	}
	result, err := installer.AvailableHistory(ctx, language, refresh, history)
	if err != nil {
		return Catalog{}, err
	}
	catalog := Catalog{Language: language, Versions: []Release{}, FetchedAt: result.Catalog.FetchedAt, Warning: result.Warning, FromCache: result.FromCache, Stale: result.Stale}
	for _, a := range result.Catalog.Artifacts {
		catalog.Versions = append(catalog.Versions, Release{a.Version, a.Build, a.Variant, installer.IsInstalled(language, a.Version), installer.Describe(a)})
	}
	return catalog, nil
}
func validateRequest(r Request) error {
	if err := tool.Validate(r.Language, r.Version); err != nil {
		return err
	}
	switch r.Action {
	case "install", "repair", "verify", "recover":
	case "select":
		if r.Scope != application.Project && r.Scope != application.Global {
			return fault.Usage("escopo inválido")
		}
		if r.Scope == application.Project {
			if _, err := projectDirectory(r.Directory); err != nil {
				return err
			}
		}
	case "uninstall":
		if _, err := projectDirectory(r.Directory); err != nil {
			return err
		}
	default:
		return fault.Usage("operação desconhecida: %s", r.Action)
	}
	return nil
}
func (s *Service) Start(request Request) (string, error) {
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	if err := validateRequest(request); err != nil {
		return "", err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(nonce[:])
	s.mu.Lock()
	// Uma operação por janela; os locks nativos também coordenam outras CLIs/janelas.
	if len(s.cancels) > 0 {
		s.mu.Unlock()
		return "", fault.New("operation_busy", 5, "Aguarde ou cancele a operação atual.", fmt.Errorf("já existe uma operação nesta janela"))
	}
	if len(s.operations) >= 50 {
		var oldest *Operation
		for _, op := range s.operations {
			if oldest == nil || op.Started.Before(oldest.Started) {
				oldest = op
			}
		}
		if oldest != nil {
			delete(s.operations, oldest.ID)
		}
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.operations[id] = &Operation{ID: id, Request: request, Status: "running", Phase: "preparing", Started: time.Now().UTC()}
	s.cancels[id] = cancel
	s.mu.Unlock()
	go func() {
		defer cancel()
		err := s.run(ctx, request, func(event installer.Event) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			op := s.operations[id]
			op.Phase = string(event.Kind)
			if event.Kind == installer.EventProgress {
				op.Bytes = event.Bytes
				op.Total = event.Total
			}
			if event.Message != "" {
				op.Message = event.Message
			}
			return nil
		})
		s.mu.Lock()
		defer s.mu.Unlock()
		op := s.operations[id]
		now := time.Now().UTC()
		op.Finished = &now
		op.Status = "succeeded"
		if err != nil {
			failure := fault.Classify(err)
			op.Status = "failed"
			op.ErrorCode = failure.Code
			op.Message = err.Error()
			op.Hint = failure.Hint
			if failure.Code == "canceled" {
				op.Status = "canceled"
			}
		}
		delete(s.cancels, id)
	}()
	return id, nil
}
func (s *Service) Operations() []Operation {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Operation, 0, len(s.operations))
	for _, op := range s.operations {
		result = append(result, *op)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Started.After(result[j].Started) })
	return result
}
func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cancel, ok := s.cancels[id]
	if !ok {
		return fmt.Errorf("operação não está em andamento")
	}
	cancel()
	return nil
}
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cancel := range s.cancels {
		cancel()
	}
}
func (s *Service) Busy() bool { s.mu.Lock(); defer s.mu.Unlock(); return len(s.cancels) > 0 }
func execute(ctx context.Context, r Request, sink installer.EventSink) error {
	ctx = installer.WithWorkingDirectory(ctx, r.Directory)
	switch r.Action {
	case "install":
		return installer.InstallWithEvents(ctx, r.Language, r.Version, sink)
	case "repair":
		return installer.RepairWithEvents(ctx, r.Language, r.Version, sink)
	case "verify":
		return installer.VerifyContext(ctx, r.Language, r.Version)
	case "recover":
		return installer.Recover(ctx, r.Language, r.Version)
	case "select":
		_, err := application.New().Select(ctx, application.SelectionRequest{Directory: r.Directory, Language: r.Language, Version: r.Version, Scope: r.Scope}, sink)
		return err
	case "uninstall":
		return installer.Uninstall(ctx, r.Directory, r.Language, r.Version, r.AllowSelected)
	}
	return fault.Usage("operação desconhecida")
}
