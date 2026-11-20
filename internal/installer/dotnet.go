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

type dotnetProvider struct {
	baseURL string
	client  *http.Client
}

type dotnetSDK struct {
	Version string
	Files   []struct {
		RID, URL, Hash string
	}
}

func (p *dotnetProvider) Source() string {
	return p.baseURL + "/release-metadata/releases-index.json"
}

func (p *dotnetProvider) channelURL(channel string) string {
	return p.baseURL + "/release-metadata/" + channel + "/releases.json"
}

func dotnetChannel(version string) string {
	return version[:strings.LastIndexByte(version, '.')]
}

func (p *dotnetProvider) artifact(version, hash string, platform Platform) Artifact {
	return Artifact{
		Language: "dotnet", Version: version, Build: version, Variant: "sdk", Platform: platform,
		URL:    p.baseURL + "/Sdk/" + version + "/dotnet-sdk-" + version + "-win-x64.zip",
		SHA512: hash, ChecksumURL: p.channelURL(dotnetChannel(version)), Format: "zip",
		Executables: map[string]string{"dotnet": "dotnet.exe"},
	}
}

func (p *dotnetProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("dotnet", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("dotnet", a.Platform); err != nil {
		return err
	}
	if versionGreater("5.0.0", a.Version) || !validSHA512(a.SHA512) || !reflect.DeepEqual(a, p.artifact(a.Version, a.SHA512, a.Platform)) {
		return fmt.Errorf("artefato do SDK .NET inválido")
	}
	return nil
}

func (p *dotnetProvider) listChannel(ctx context.Context, channel string, platform Platform) ([]Artifact, error) {
	data, err := get(ctx, p.client, p.channelURL(channel), 16<<20)
	if err != nil {
		return nil, err
	}
	var catalog struct {
		Channel  string `json:"channel-version"`
		Releases []struct {
			SDK  dotnetSDK
			SDKs []dotnetSDK
		}
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("ler catálogo do .NET %s: %w", channel, err)
	}
	if catalog.Channel != channel || catalog.Releases == nil {
		return nil, fmt.Errorf("catálogo do .NET %s inválido", channel)
	}
	var items []Artifact
	seen := map[string]bool{}
	for _, release := range catalog.Releases {
		for _, sdk := range append(release.SDKs, release.SDK) {
			if tool.Validate("dotnet", sdk.Version) != nil || dotnetChannel(sdk.Version) != channel || seen[sdk.Version] {
				continue
			}
			for _, file := range sdk.Files {
				if file.RID != "win-x64" || !validSHA512(file.Hash) {
					continue
				}
				a := p.artifact(sdk.Version, strings.ToLower(file.Hash), platform)
				if file.URL != a.URL {
					continue
				}
				items = append(items, a)
				seen[sdk.Version] = true
				break
			}
		}
	}
	return items, nil
}

func (p *dotnetProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("dotnet", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.Source(), 1<<20)
	if err != nil {
		return nil, err
	}
	var index struct {
		Channels []struct {
			Version string `json:"channel-version"`
			URL     string `json:"releases.json"`
		} `json:"releases-index"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("ler índice do .NET: %w", err)
	}
	items := []Artifact{}
	seen := map[string]bool{}
	for _, channel := range index.Channels {
		version := channel.Version + ".0"
		if tool.Validate("dotnet", version) != nil || dotnetChannel(version) != channel.Version ||
			versionGreater("5.0.0", version) || seen[channel.Version] || channel.URL != p.channelURL(channel.Version) {
			continue
		}
		seen[channel.Version] = true
		releases, err := p.listChannel(ctx, channel.Version, platform)
		if err != nil {
			return nil, err
		}
		items = append(items, releases...)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("índice do .NET sem SDKs estáveis para Windows x64")
	}
	sort.Slice(items, func(i, j int) bool { return versionGreater(items[i].Version, items[j].Version) })
	return items, nil
}

func (p *dotnetProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	items, err := p.listChannel(ctx, dotnetChannel(a.Version), a.Platform)
	if err != nil {
		return Artifact{}, err
	}
	for _, item := range items {
		if item.Version == a.Version {
			return item, nil
		}
	}
	return Artifact{}, fmt.Errorf("SDK .NET %s não está no catálogo oficial", a.Version)
}
