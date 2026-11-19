package installer

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/tool"
)

type yarnProvider struct {
	baseURL string
	client  *http.Client
}

func (p *yarnProvider) Source() string { return p.baseURL + "/@yarnpkg%2fcli-dist" }
func (p *yarnProvider) artifact(version, hash string, platform Platform) Artifact {
	return Artifact{Language: "yarn", Version: version, Build: version, Platform: platform, URL: p.baseURL + "/@yarnpkg/cli-dist/-/cli-dist-" + version + ".tgz", SHA512: hash, ChecksumURL: p.Source() + "/" + version, Format: "tar.gz", ArchiveRoot: "package", Executables: map[string]string{"yarn": "bin/yarn.js", "yarnpkg": "bin/yarn.js"}}
}
func validSHA512(value string) bool {
	data, err := hex.DecodeString(value)
	return err == nil && len(data) == 64
}

type npmVersion struct {
	Version string `json:"version"`
	Dist    struct {
		Tarball   string `json:"tarball"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}

func (p *yarnProvider) parse(v npmVersion, platform Platform) (Artifact, error) {
	if err := tool.Validate("yarn", v.Version); err != nil {
		return Artifact{}, err
	}
	if !strings.HasPrefix(v.Dist.Integrity, "sha512:") && !strings.HasPrefix(v.Dist.Integrity, "sha512-") {
		return Artifact{}, fmt.Errorf("Yarn sem integridade SHA-512 publicada")
	}
	encoded := strings.TrimPrefix(strings.TrimPrefix(v.Dist.Integrity, "sha512-"), "sha512:")
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) != 64 {
		return Artifact{}, fmt.Errorf("integridade Yarn inválida")
	}
	a := p.artifact(v.Version, hex.EncodeToString(data), platform)
	if a.URL != v.Dist.Tarball {
		return Artifact{}, fmt.Errorf("URL Yarn fora da origem esperada")
	}
	return a, p.ValidateArtifact(a)
}
func (p *yarnProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("yarn", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("yarn", a.Platform); err != nil {
		return err
	}
	if !validSHA512(a.SHA512) || !reflect.DeepEqual(a, p.artifact(a.Version, a.SHA512, a.Platform)) {
		return fmt.Errorf("artefato Yarn inválido")
	}
	return nil
}
func (p *yarnProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("Yarn", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.Source(), 24<<20)
	if err != nil {
		return nil, err
	}
	var index struct {
		Versions map[string]npmVersion `json:"versions"`
	}
	if err := json.Unmarshal(data, &index); err != nil || index.Versions == nil {
		return nil, fmt.Errorf("catálogo Yarn inválido: %v", err)
	}
	result := []Artifact{}
	for version, v := range index.Versions {
		if tool.Validate("yarn", version) != nil {
			continue
		}
		if v.Version != version {
			return nil, fmt.Errorf("versão Yarn inconsistente")
		}
		a, err := p.parse(v, platform)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	sort.Slice(result, func(i, j int) bool { return versionGreater(result[i].Version, result[j].Version) })
	return result, nil
}
func (p *yarnProvider) ResolveExact(ctx context.Context, version string, platform Platform) (Artifact, error) {
	if err := tool.Validate("yarn", version); err != nil {
		return Artifact{}, err
	}
	if err := windowsTarget("Yarn", platform); err != nil {
		return Artifact{}, err
	}
	data, err := get(ctx, p.client, p.Source()+"/"+version, 1<<20)
	if err != nil {
		return Artifact{}, err
	}
	var v npmVersion
	if err := json.Unmarshal(data, &v); err != nil {
		return Artifact{}, err
	}
	if v.Version != version {
		return Artifact{}, fmt.Errorf("Yarn retornou outra versão")
	}
	return p.parse(v, platform)
}
func (p *yarnProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	return p.ResolveExact(ctx, a.Version, a.Platform)
}
