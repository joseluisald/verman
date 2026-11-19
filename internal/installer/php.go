package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/tool"
)

type phpProvider struct {
	baseURL string
	client  *http.Client
	history bool
}

func (p *phpProvider) Source() string {
	if p.history {
		return p.baseURL + "/releases.json#history-v1"
	}
	return p.baseURL + "/releases.json"
}

var phpBuild = regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]+)(?:-[0-9]+)?$`)
var phpVariant = regexp.MustCompile(`^nts-(vc15|vs16|vs17)-x64$`)

func (p *phpProvider) artifact(version, build, variant, checksum string, platform Platform) Artifact {
	compiler := strings.TrimSuffix(strings.TrimPrefix(variant, "nts-"), "-x64")
	filename := "php-" + build + "-nts-Win32-" + compiler + "-x64.zip"
	return Artifact{Language: "php", Version: version, Platform: platform, URL: p.baseURL + "/" + filename, Format: "zip", SHA256: checksum, ChecksumURL: p.baseURL + "/releases.json", Executables: map[string]string{"php": "php.exe"}, Variant: variant, Build: build}
}

func (p *phpProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("php", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("PHP", a.Platform); err != nil {
		return err
	}
	match := phpBuild.FindStringSubmatch(a.Build)
	expected := p.artifact(a.Version, a.Build, a.Variant, a.SHA256, a.Platform)
	validHash := validSHA256(a.SHA256)
	if a.Archived {
		expected = p.archiveArtifact(a.Version, a.Build, a.Variant, a.SHA256, a.Platform)
		validHash = validHash || a.SHA256 == ""
	}
	if len(match) != 2 || match[1] != a.Version || !phpVariant.MatchString(a.Variant) || !validHash || !reflect.DeepEqual(a, expected) {
		return fmt.Errorf("artefato PHP inválido")
	}
	return nil
}

func (p *phpProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if p.history {
		return p.listHistory(ctx, platform)
	}
	return p.listCurrent(ctx, platform)
}
func (p *phpProvider) listCurrent(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("PHP", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.baseURL+"/releases.json", 4<<20)
	if err != nil {
		return nil, err
	}
	var releases map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &releases); err != nil || len(releases) == 0 {
		return nil, fmt.Errorf("catálogo PHP vazio ou inválido: %v", err)
	}
	artifacts := []Artifact{}
	seen := map[string]bool{}
	for branch, release := range releases {
		var build string
		if err := json.Unmarshal(release["version"], &build); err != nil {
			return nil, fmt.Errorf("versão ausente no catálogo PHP: %s", branch)
		}
		match := phpBuild.FindStringSubmatch(build)
		if len(match) != 2 {
			continue
		}
		version := match[1]
		if tool.Validate("php", version) != nil || !strings.HasPrefix(version, branch+".") {
			continue
		}
		var chosen *Artifact
		for variant, raw := range release {
			if !phpVariant.MatchString(variant) {
				continue
			}
			var item struct {
				ZIP struct {
					Path   string `json:"path"`
					SHA256 string `json:"sha256"`
				} `json:"zip"`
			}
			if err := json.Unmarshal(raw, &item); err != nil {
				return nil, fmt.Errorf("pacote PHP inválido: %w", err)
			}
			a := p.artifact(version, build, variant, strings.ToLower(item.ZIP.SHA256), platform)
			if a.URL != p.baseURL+"/"+item.ZIP.Path || item.ZIP.Path == "" || p.ValidateArtifact(a) != nil {
				return nil, fmt.Errorf("pacote ou checksum PHP inválido para %s", version)
			}
			if chosen != nil {
				return nil, fmt.Errorf("variantes PHP ambíguas para %s", version)
			}
			chosen = &a
		}
		if chosen != nil && !seen[version] {
			artifacts = append(artifacts, *chosen)
			seen[version] = true
		}
	}
	sort.Slice(artifacts, func(i, j int) bool { return versionGreater(artifacts[i].Version, artifacts[j].Version) })
	return artifacts, nil
}

func (p *phpProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	// Reconsulta a fonte confiável: checksums editados no cache nunca autorizam execução.
	items, err := p.List(ctx, a.Platform)
	if err != nil {
		return Artifact{}, err
	}
	for _, item := range items {
		if item.Version == a.Version {
			if !validSHA256(item.SHA256) {
				return Artifact{}, fmt.Errorf("PHP %s: arquivo histórico sem SHA-256 oficial; instalação recusada", a.Version)
			}
			return item, nil
		}
	}
	return Artifact{}, fmt.Errorf("PHP %s não está mais no índice atual; atualize list php --available --refresh", a.Version)
}
