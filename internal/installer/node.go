package installer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/tool"
)

type nodeProvider struct {
	baseURL string
	client  *http.Client
}

func (p *nodeProvider) Source() string { return p.baseURL + "/index.json" }

func (p *nodeProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("node", a.Version); err != nil {
		return err
	}
	if err := nodeTarget(a.Platform); err != nil {
		return err
	}
	expected := p.artifact(a.Version, a.Platform)
	expected.LTS = a.LTS
	if !reflect.DeepEqual(a, expected) {
		return fmt.Errorf("artefato Node inválido")
	}
	return nil
}

// O primeiro alvo verificado é Windows/amd64; outras plataformas são explícitas
// e só devem ser habilitadas após implementação e testes dos seus formatos.
func nodeTarget(platform Platform) error {
	return windowsTarget("Node", platform)
}

func (p *nodeProvider) artifact(version string, platform Platform) Artifact {
	root := "node-v" + version + "-win-x64"
	return Artifact{Language: "node", Version: version, Platform: platform, URL: p.baseURL + "/v" + version + "/" + root + ".zip", Format: "zip", ChecksumURL: p.baseURL + "/v" + version + "/SHASUMS256.txt", ArchiveRoot: root, Executables: map[string]string{"node": "node.exe", "npm": "npm.cmd", "npx": "npx.cmd"}}
}

func (p *nodeProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := nodeTarget(platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.Source(), 8<<20)
	if err != nil {
		return nil, err
	}
	var releases []struct {
		Version string          `json:"version"`
		Files   []string        `json:"files"`
		LTS     json.RawMessage `json:"lts"`
	}
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, fmt.Errorf("catálogo Node inválido: %w", err)
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("catálogo Node vazio ou formato inesperado")
	}
	artifacts := []Artifact{}
	seen := map[string]bool{}
	for _, release := range releases {
		if !strings.HasPrefix(release.Version, "v") || release.Files == nil {
			return nil, fmt.Errorf("release Node com formato inesperado")
		}
		version := strings.TrimPrefix(release.Version, "v")
		if tool.Validate("node", version) != nil {
			continue
		}
		for _, file := range release.Files {
			if file == "win-x64-zip" && !seen[version] {
				a := p.artifact(version, platform)
				if len(release.LTS) > 0 {
					var value any
					if err := json.Unmarshal(release.LTS, &value); err != nil {
						return nil, err
					}
					switch v := value.(type) {
					case string:
						a.LTS = v != ""
					case bool:
						if v {
							return nil, fmt.Errorf("indicador LTS inesperado")
						}
					default:
						return nil, fmt.Errorf("indicador LTS inválido")
					}
				}
				artifacts = append(artifacts, a)
				seen[version] = true
			}
		}
	}
	sort.Slice(artifacts, func(i, j int) bool { return versionGreater(artifacts[i].Version, artifacts[j].Version) })
	return artifacts, nil
}

func (p *nodeProvider) Resolve(ctx context.Context, artifact Artifact) (Artifact, error) {
	if err := tool.Validate("node", artifact.Version); err != nil {
		return Artifact{}, err
	}
	if err := nodeTarget(artifact.Platform); err != nil {
		return Artifact{}, err
	}
	// Reconstrói URLs confiáveis; nunca usa URLs arbitrárias vindas do cache.
	artifact = p.artifact(artifact.Version, artifact.Platform)
	data, err := get(ctx, p.client, artifact.ChecksumURL, 1<<20)
	if err != nil {
		return Artifact{}, err
	}
	filename := artifact.ArchiveRoot + ".zip"
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != filename {
			continue
		}
		if artifact.SHA256 != "" {
			return Artifact{}, fmt.Errorf("checksum duplicado para %s", filename)
		}
		digest, err := hex.DecodeString(fields[0])
		if err != nil || len(digest) != 32 {
			return Artifact{}, fmt.Errorf("checksum inválido para %s", filename)
		}
		artifact.SHA256 = strings.ToLower(fields[0])
	}
	if artifact.SHA256 == "" {
		return Artifact{}, fmt.Errorf("checksum de %s ausente", filename)
	}
	return artifact, nil
}
