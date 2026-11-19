package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/tool"
)

type goProvider struct {
	indexURL, baseURL string
	client            *http.Client
}

func (p *goProvider) Source() string { return p.indexURL }
func goBuild(version string) string {
	if strings.HasSuffix(version, ".0") && versionGreater("1.21.0", version) {
		return "go" + strings.TrimSuffix(version, ".0")
	}
	return "go" + version
}
func normalizeGo(version string) string {
	version = strings.TrimPrefix(version, "go")
	if strings.Count(version, ".") == 1 {
		version += ".0"
	}
	return version
}
func (p *goProvider) artifact(version, hash string, platform Platform) Artifact {
	return Artifact{Language: "go", Version: version, Build: goBuild(version), Platform: platform, URL: p.baseURL + "/" + goBuild(version) + ".windows-amd64.zip", SHA256: hash, ChecksumURL: p.indexURL, Format: "zip", ArchiveRoot: "go", Executables: map[string]string{"go": "bin/go.exe", "gofmt": "bin/gofmt.exe"}}
}
func (p *goProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("go", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("go", a.Platform); err != nil {
		return err
	}
	if !validSHA256(a.SHA256) || !reflect.DeepEqual(a, p.artifact(a.Version, a.SHA256, a.Platform)) {
		return fmt.Errorf("artefato Go inválido")
	}
	return nil
}
func (p *goProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("Go", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.indexURL, 16<<20)
	if err != nil {
		return nil, err
	}
	var releases []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
		Files   []struct{ Filename, OS, Arch, SHA256, Kind string }
	}
	if err := json.Unmarshal(data, &releases); err != nil || releases == nil {
		return nil, fmt.Errorf("catálogo Go inválido: %v", err)
	}
	result := []Artifact{}
	seen := map[string]bool{}
	for _, r := range releases {
		version := normalizeGo(r.Version)
		if !r.Stable || tool.Validate("go", version) != nil {
			continue
		}
		for _, f := range r.Files {
			if f.SHA256 == "" {
				continue
			}
			if f.OS != "windows" || f.Arch != "amd64" || f.Kind != "archive" || !strings.HasSuffix(f.Filename, ".zip") {
				continue
			}
			a := p.artifact(version, f.SHA256, platform)
			if f.Filename != goBuild(version)+".windows-amd64.zip" || p.ValidateArtifact(a) != nil {
				return nil, fmt.Errorf("arquivo Go inválido")
			}
			if !seen[version] {
				result = append(result, a)
				seen[version] = true
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return versionGreater(result[i].Version, result[j].Version) })
	return result, nil
}
func (p *goProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	items, err := p.List(ctx, a.Platform)
	if err != nil {
		return Artifact{}, err
	}
	for _, item := range items {
		if item.Version == a.Version {
			return item, nil
		}
	}
	return Artifact{}, fmt.Errorf("Go %s não está no catálogo oficial", a.Version)
}

type composerProvider struct {
	baseURL string
	client  *http.Client
}

func (p *composerProvider) Source() string { return p.baseURL + "/versions" }
func (p *composerProvider) artifact(version, hash string, platform Platform) Artifact {
	return Artifact{Language: "composer", Version: version, Build: version, Platform: platform, URL: p.baseURL + "/download/" + version + "/composer.phar", SHA256: hash, ChecksumURL: p.baseURL + "/download/" + version + "/composer.phar.sha256sum", Format: "file", ArchiveRoot: "composer.phar", Executables: map[string]string{"composer": "composer.phar"}}
}
func (p *composerProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("composer", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("composer", a.Platform); err != nil {
		return err
	}
	if !validSHA256(a.SHA256) || !reflect.DeepEqual(a, p.artifact(a.Version, a.SHA256, a.Platform)) {
		return fmt.Errorf("artefato Composer inválido")
	}
	return nil
}
func (p *composerProvider) ResolveExact(ctx context.Context, version string, platform Platform) (Artifact, error) {
	if err := tool.Validate("composer", version); err != nil {
		return Artifact{}, err
	}
	if err := windowsTarget("Composer", platform); err != nil {
		return Artifact{}, err
	}
	a := p.artifact(version, "", platform)
	data, err := get(ctx, p.client, a.ChecksumURL, 4096)
	if err != nil {
		return Artifact{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return Artifact{}, fmt.Errorf("checksum Composer ausente")
	}
	a.SHA256 = strings.ToLower(fields[0])
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	return a, nil
}
func (p *composerProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("Composer", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.Source(), 1<<20)
	if err != nil {
		return nil, err
	}
	var index struct {
		Stable []struct{ Version, Path string }
	}
	if err := json.Unmarshal(data, &index); err != nil || len(index.Stable) == 0 {
		return nil, fmt.Errorf("catálogo Composer inválido: %v", err)
	}
	result := []Artifact{}
	for _, r := range index.Stable {
		if tool.Validate("composer", r.Version) != nil {
			continue
		}
		if r.Path != "/download/"+r.Version+"/composer.phar" {
			return nil, fmt.Errorf("caminho Composer inesperado")
		}
		a, err := p.ResolveExact(ctx, r.Version, platform)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	sort.Slice(result, func(i, j int) bool { return versionGreater(result[i].Version, result[j].Version) })
	return result, nil
}
func (p *composerProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	return p.ResolveExact(ctx, a.Version, a.Platform)
}
