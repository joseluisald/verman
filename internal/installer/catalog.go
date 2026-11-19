package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/durable"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/fslock"
)

type Catalog struct {
	Schema    int        `json:"schema"`
	Source    string     `json:"source"`
	Platform  Platform   `json:"platform"`
	FetchedAt time.Time  `json:"fetched_at"`
	Artifacts []Artifact `json:"artifacts"`
}

type CatalogResult struct {
	Catalog   Catalog `json:"catalog"`
	Warning   string  `json:"warning,omitempty"`
	FromCache bool    `json:"from_cache"`
	Stale     bool    `json:"stale"`
}

func Available(ctx context.Context, language string, refresh bool) (CatalogResult, error) {
	return AvailableHistory(ctx, language, refresh, false)
}
func AvailableHistory(ctx context.Context, language string, refresh, history bool) (CatalogResult, error) {
	p, err := providerFor(language)
	if err != nil {
		return CatalogResult{}, err
	}
	if history {
		p = withHistory(p)
	}
	base, err := config.BaseDir()
	if err != nil {
		return CatalogResult{}, err
	}
	platform := Platform{runtime.GOOS, runtime.GOARCH}
	if err := windowsTarget(language, platform); err != nil {
		return CatalogResult{}, err
	}
	suffix := ""
	if history {
		suffix = "-history"
	}
	return loadCatalog(ctx, p, platform, filepath.Join(base, "cache", language+"-"+platform.OS+"-"+platform.Arch+suffix+".json"), refresh)
}

func loadCatalog(ctx context.Context, p Provider, platform Platform, path string, refresh bool) (CatalogResult, error) {
	if err := ctx.Err(); err != nil {
		return CatalogResult{}, err
	}
	var cached Catalog
	data, readErr := readCatalogCache(path)
	valid := readErr == nil && json.Unmarshal(data, &cached) == nil && cached.Schema == 1 && cached.Source == p.Source() && cached.Platform == platform && !cached.FetchedAt.IsZero() && !cached.FetchedAt.After(time.Now()) && cached.Artifacts != nil
	// O cache é local e editável: valide o contrato e nunca confie em suas URLs.
	if valid {
		seen := map[string]bool{}
		for _, artifact := range cached.Artifacts {
			if seen[artifact.Version] || artifact.Platform != platform || p.ValidateArtifact(artifact) != nil {
				valid = false
				break
			}
			seen[artifact.Version] = true
		}
	}
	if valid && !refresh && time.Since(cached.FetchedAt) < time.Hour {
		return CatalogResult{Catalog: cached, FromCache: true}, nil
	}
	if offline(ctx) {
		if refresh {
			return CatalogResult{}, fault.Usage("--offline não permite --refresh")
		}
		if !valid {
			return CatalogResult{}, fault.New("offline", 3, "Consulte o catálogo online uma vez para criar um cache válido.", fmt.Errorf("sem catálogo local válido"))
		}
		return CatalogResult{Catalog: cached, FromCache: true, Stale: true, Warning: "modo offline: catálogo antigo; disponibilidade pode ter mudado"}, nil
	}
	artifacts, err := p.List(ctx, platform)
	if err != nil {
		if ctx.Err() != nil {
			return CatalogResult{}, ctx.Err()
		}
		if valid {
			return CatalogResult{Catalog: cached, FromCache: true, Stale: time.Since(cached.FetchedAt) >= time.Hour, Warning: fmt.Sprintf("falha ao atualizar catálogo (%v); usando cache de %s", err, cached.FetchedAt.Format(time.RFC3339))}, nil
		}
		return CatalogResult{}, fmt.Errorf("consultar catálogo sem cache válido: %w", err)
	}
	catalog := Catalog{Schema: 1, Source: p.Source(), Platform: platform, FetchedAt: time.Now().UTC(), Artifacts: artifacts}
	data, err = json.Marshal(catalog)
	if err != nil {
		return CatalogResult{}, err
	}
	result := CatalogResult{Catalog: catalog}
	if err := writeCacheContext(ctx, path, data); err != nil {
		if ctx.Err() != nil {
			return CatalogResult{}, ctx.Err()
		}
		result.Warning = fmt.Sprintf("catálogo consultado, mas não foi possível salvar o cache: %v", err)
	}
	return result, nil
}

func writeCache(path string, data []byte) error {
	return writeCacheContext(context.Background(), path, data)
}
func writeCacheContext(parent context.Context, path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	unlock, err := fslock.Acquire(ctx, filepath.Dir(path), filepath.Base(path)+".lock", false, true)
	if err != nil {
		return err
	}
	defer unlock()
	return durable.WriteContext(ctx, path, data, 0600)
}

func readCatalogCache(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	const max = 16 << 20
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if len(data) > max {
		return nil, fmt.Errorf("cache excede 16 MiB")
	}
	return data, nil
}
