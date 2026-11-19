package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joseluisald/verman/internal/tool"
)

type githubToolSpec struct {
	language, repo, format, root string
	executables                  map[string]string
}

var githubToolSpecs = map[string]githubToolSpec{
	"bun":        {"bun", "oven-sh/bun", "zip", "bun-windows-x64-baseline", map[string]string{"bun": "bun.exe", "bunx": "bun.exe"}},
	"deno":       {"deno", "denoland/deno", "zip", "", map[string]string{"deno": "deno.exe"}},
	"git":        {"git", "git-for-windows/git", "zip", "", map[string]string{"git": "cmd/git.exe"}},
	"pnpm":       {"pnpm", "pnpm/pnpm", "zip", "", map[string]string{"pnpm": "pnpm.exe"}},
	"vcpkg":      {"vcpkg", "microsoft/vcpkg-tool", "file", "vcpkg.exe", map[string]string{"vcpkg": "vcpkg.exe"}},
	"chocolatey": {"chocolatey", "chocolatey/choco", "zip", "", map[string]string{"chocolatey": "tools/chocolateyInstall/choco.exe", "choco": "tools/chocolateyInstall/choco.exe"}},
}

type githubToolProvider struct {
	spec                githubToolSpec
	apiURL, downloadURL string
	client              *http.Client
}
type githubAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}
type githubRelease struct {
	Tag        string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

var gitTag = regexp.MustCompile(`^v([0-9]+\.[0-9]+\.[0-9]+)\.windows\.([0-9]+)$`)
var dateTag = regexp.MustCompile(`^([0-9]{4})-([0-9]{2})-([0-9]{2})$`)

func normalizedDate(tag string) string {
	if _, err := time.Parse("2006-01-02", tag); err != nil {
		return ""
	}
	parts := dateTag.FindStringSubmatch(tag)
	if len(parts) != 4 {
		return ""
	}
	for i := 2; i < 4; i++ {
		parts[i] = strings.TrimLeft(parts[i], "0")
		if parts[i] == "" {
			parts[i] = "0"
		}
	}
	return strings.Join(parts[1:], ".")
}
func (p *githubToolProvider) version(tag string) string {
	switch p.spec.language {
	case "bun":
		return strings.TrimPrefix(tag, "bun-v")
	case "git":
		parts := gitTag.FindStringSubmatch(tag)
		if len(parts) == 3 {
			return parts[1]
		}
		return ""
	case "vcpkg":
		return normalizedDate(tag)
	default:
		return strings.TrimPrefix(tag, "v")
	}
}
func (p *githubToolProvider) filename(version, tag string) string {
	switch p.spec.language {
	case "bun":
		return "bun-windows-x64-baseline.zip"
	case "deno":
		return "deno-x86_64-pc-windows-msvc.zip"
	case "pnpm":
		return "pnpm-win32-x64.zip"
	case "git":
		parts := gitTag.FindStringSubmatch(tag)
		if len(parts) != 3 {
			return ""
		}
		revision := ""
		if parts[2] != "1" {
			revision = "." + parts[2]
		}
		return "MinGit-" + version + revision + "-64-bit.zip"
	case "vcpkg":
		return "vcpkg.exe"
	case "chocolatey":
		return "chocolatey." + version + ".nupkg"
	}
	return ""
}
func (p *githubToolProvider) Source() string { return p.apiURL + "?per_page=30" }
func (p *githubToolProvider) artifact(version, tag, hash string, platform Platform) Artifact {
	return Artifact{Language: p.spec.language, Version: version, Build: tag, Platform: platform, URL: p.downloadURL + "/" + tag + "/" + p.filename(version, tag), Format: p.spec.format, ArchiveRoot: p.spec.root, SHA256: hash, ChecksumURL: p.apiURL + "/tags/" + tag, Executables: p.spec.executables}
}
func (p *githubToolProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate(p.spec.language, a.Version); err != nil {
		return err
	}
	if err := windowsTarget(p.spec.language, a.Platform); err != nil {
		return err
	}
	if p.version(a.Build) != a.Version || !validSHA256(a.SHA256) || !reflect.DeepEqual(a, p.artifact(a.Version, a.Build, a.SHA256, a.Platform)) {
		return fmt.Errorf("artefato %s inválido", p.spec.language)
	}
	return nil
}
func (p *githubToolProvider) parse(release githubRelease, platform Platform) (*Artifact, error) {
	if release.Draft || release.Prerelease {
		return nil, nil
	}
	version := p.version(release.Tag)
	if tool.Validate(p.spec.language, version) != nil {
		return nil, nil
	}
	filename := p.filename(version, release.Tag)
	var result *Artifact
	for _, asset := range release.Assets {
		if asset.Name != filename {
			continue
		}
		if result != nil {
			return nil, fmt.Errorf("asset duplicado: %s", filename)
		}
		if !strings.HasPrefix(asset.Digest, "sha256:") {
			continue
		}
		a := p.artifact(version, release.Tag, strings.TrimPrefix(asset.Digest, "sha256:"), platform)
		if a.URL != asset.URL || p.ValidateArtifact(a) != nil {
			return nil, fmt.Errorf("origem/hash inválido: %s", filename)
		}
		result = &a
	}
	return result, nil
}
func (p *githubToolProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget(p.spec.language, platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.Source(), 32<<20)
	if err != nil {
		return nil, err
	}
	var releases []githubRelease
	if err := json.Unmarshal(data, &releases); err != nil || releases == nil {
		return nil, fmt.Errorf("catálogo %s inválido: %v", p.spec.language, err)
	}
	versions := map[string]Artifact{}
	for _, r := range releases {
		a, err := p.parse(r, platform)
		if err != nil {
			return nil, err
		}
		if a == nil {
			continue
		}
		if old, ok := versions[a.Version]; !ok || newerToolBuild(p.spec.language, a.Build, old.Build) {
			versions[a.Version] = *a
		}
	}
	result := make([]Artifact, 0, len(versions))
	for _, a := range versions {
		result = append(result, a)
	}
	sort.Slice(result, func(i, j int) bool { return versionGreater(result[i].Version, result[j].Version) })
	if len(result) == 0 {
		return nil, fmt.Errorf("nenhum artefato %s compatível com digest publicado nas 30 releases consultadas", p.spec.language)
	}
	return result, nil
}
func newerToolBuild(language, next, old string) bool {
	if language == "git" {
		n := gitTag.FindStringSubmatch(next)
		o := gitTag.FindStringSubmatch(old)
		if len(n) == 3 && len(o) == 3 {
			return versionGreater("0.0."+n[2], "0.0."+o[2])
		}
	}
	return next > old
}
func (p *githubToolProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	data, err := get(ctx, p.client, p.apiURL+"/tags/"+a.Build, 12<<20)
	if err != nil {
		return Artifact{}, err
	}
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return Artifact{}, err
	}
	resolved, err := p.parse(release, a.Platform)
	if err != nil {
		return Artifact{}, err
	}
	if resolved == nil || resolved.Build != a.Build || resolved.Version != a.Version {
		return Artifact{}, fmt.Errorf("release %s não corresponde ao pedido", a.Build)
	}
	return *resolved, nil
}
func (p *githubToolProvider) ResolveExact(ctx context.Context, version string, platform Platform) (Artifact, error) {
	if err := tool.Validate(p.spec.language, version); err != nil {
		return Artifact{}, err
	}
	tag := "v" + version
	switch p.spec.language {
	case "bun":
		tag = "bun-v" + version
	case "git":
		tag = "v" + version + ".windows.1"
	case "chocolatey":
		tag = version
	case "vcpkg":
		parts := strings.Split(version, ".")
		month, _ := strconv.Atoi(parts[1])
		day, _ := strconv.Atoi(parts[2])
		tag = fmt.Sprintf("%s-%02d-%02d", parts[0], month, day)
	}
	data, err := get(ctx, p.client, p.apiURL+"/tags/"+tag, 12<<20)
	if err != nil {
		return Artifact{}, err
	}
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return Artifact{}, err
	}
	a, err := p.parse(release, platform)
	if err != nil {
		return Artifact{}, err
	}
	if a == nil || a.Version != version || a.Build != tag {
		return Artifact{}, fmt.Errorf("versão %s indisponível", version)
	}
	return *a, nil
}
