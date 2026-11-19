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

type pythonProvider struct {
	apiURL, downloadURL string
	client              *http.Client
	history             bool
}

func (p *pythonProvider) Source() string {
	if p.history {
		return p.apiURL + "#history-v1"
	}
	return p.apiURL + "/latest"
}

var pythonBuild = regexp.MustCompile(`^[0-9]{8}$`)
var pythonAsset = regexp.MustCompile(`^cpython-([0-9]+\.[0-9]+\.[0-9]+)\+([0-9]{8})-x86_64-pc-windows-msvc-install_only\.tar\.gz$`)

type pythonRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name   string `json:"name"`
		URL    string `json:"browser_download_url"`
		Digest string `json:"digest"`
	} `json:"assets"`
}

func (p *pythonProvider) artifact(version, build, checksum string, platform Platform) Artifact {
	filename := "cpython-" + version + "%2B" + build + "-x86_64-pc-windows-msvc-install_only.tar.gz"
	return Artifact{Language: "python", Version: version, Platform: platform, URL: p.downloadURL + "/" + build + "/" + filename, Format: "tar.gz", SHA256: checksum, ChecksumURL: p.apiURL + "/tags/" + build, ArchiveRoot: "python", Executables: map[string]string{"python": "python.exe", "pip": "pip.cmd"}, Variant: "install_only", Build: build}
}

func (p *pythonProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("python", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("Python", a.Platform); err != nil {
		return err
	}
	if !pythonBuild.MatchString(a.Build) || !validSHA256(a.SHA256) || !reflect.DeepEqual(a, p.artifact(a.Version, a.Build, a.SHA256, a.Platform)) {
		return fmt.Errorf("artefato Python inválido")
	}
	return nil
}

func (p *pythonProvider) readRelease(ctx context.Context, source string, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("Python", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, source, 12<<20)
	if err != nil {
		return nil, err
	}
	var release pythonRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return nil, fmt.Errorf("release Python inválido: %w", err)
	}
	return p.releaseArtifacts(release, platform, false)
}

func (p *pythonProvider) releaseArtifacts(release pythonRelease, platform Platform, historical bool) ([]Artifact, error) {
	if !pythonBuild.MatchString(release.Tag) || release.Draft || release.Prerelease || release.Assets == nil {
		return nil, fmt.Errorf("release Python sem catálogo estável válido")
	}
	artifacts := []Artifact{}
	seen := map[string]bool{}
	for _, asset := range release.Assets {
		match := pythonAsset.FindStringSubmatch(asset.Name)
		if len(match) != 3 {
			continue
		}
		if match[2] != release.Tag || tool.Validate("python", match[1]) != nil {
			return nil, fmt.Errorf("versão/build Python inconsistente")
		}
		if !strings.HasPrefix(asset.Digest, "sha256:") {
			if historical {
				continue
			}
			return nil, fmt.Errorf("checksum SHA-256 ausente no asset Python %s", asset.Name)
		}
		a := p.artifact(match[1], release.Tag, strings.TrimPrefix(asset.Digest, "sha256:"), platform)
		if p.ValidateArtifact(a) != nil || a.URL != asset.URL || seen[a.Version] {
			return nil, fmt.Errorf("asset Python inválido ou duplicado: %s", asset.Name)
		}
		seen[a.Version] = true
		artifacts = append(artifacts, a)
	}
	sort.Slice(artifacts, func(i, j int) bool { return versionGreater(artifacts[i].Version, artifacts[j].Version) })
	return artifacts, nil
}

func (p *pythonProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if p.history {
		return p.listHistory(ctx, platform)
	}
	return p.readRelease(ctx, p.Source(), platform)
}

func (p *pythonProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	items, err := p.readRelease(ctx, p.apiURL+"/tags/"+a.Build, a.Platform)
	if err != nil {
		return Artifact{}, err
	}
	for _, item := range items {
		if item.Version == a.Version && item.Build == a.Build {
			return item, nil
		}
	}
	return Artifact{}, fmt.Errorf("Python %s não encontrado no build %s", a.Version, a.Build)
}
