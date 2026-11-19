package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func withHistory(provider Provider) Provider {
	switch p := provider.(type) {
	case *phpProvider:
		copy := *p
		copy.history = true
		return &copy
	case *pythonProvider:
		copy := *p
		copy.history = true
		return &copy
	default:
		return provider
	}
}

var archivedPHP = regexp.MustCompile(`(?i)href="(php-([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9]+)?)-nts-Win32-(vc15|vs16|vs17)-x64\.zip)"`)

func (p *phpProvider) archiveArtifact(version, build, variant, hash string, platform Platform) Artifact {
	a := p.artifact(version, build, variant, hash, platform)
	a.URL = p.baseURL + "/archives/" + strings.TrimPrefix(a.URL, p.baseURL+"/")
	a.ChecksumURL = p.baseURL + "/sha256sum.txt"
	a.Archived = true
	return a
}
func (p *phpProvider) listHistory(ctx context.Context, platform Platform) ([]Artifact, error) {
	current, err := p.listCurrent(ctx, platform)
	if err != nil {
		return nil, err
	}
	index, err := get(ctx, p.client, p.baseURL+"/archives/", 8<<20)
	if err != nil {
		return nil, err
	}
	sums, err := get(ctx, p.client, p.baseURL+"/sha256sum.txt", 1<<20)
	if err != nil {
		return nil, err
	}
	hashes := map[string]string{}
	for _, line := range strings.Split(string(sums), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 || !validSHA256(parts[0]) {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if old, ok := hashes[name]; ok && old != strings.ToLower(parts[0]) {
			return nil, fmt.Errorf("checksums PHP conflitantes: %s", name)
		}
		hashes[name] = strings.ToLower(parts[0])
	}
	byVersion := map[string]Artifact{}
	for _, match := range archivedPHP.FindAllStringSubmatch(string(index), -1) {
		build := match[2]
		version := strings.SplitN(build, "-", 2)[0]
		a := p.archiveArtifact(version, build, "nts-"+strings.ToLower(match[3])+"-x64", hashes[match[1]], platform)
		if err := p.ValidateArtifact(a); err != nil {
			return nil, err
		}
		old, ok := byVersion[version]
		if !ok || preferPHP(a, old) {
			byVersion[version] = a
		}
	}
	if len(byVersion) == 0 {
		return nil, fmt.Errorf("índice histórico PHP vazio ou formato alterado")
	}
	for _, a := range current {
		byVersion[a.Version] = a
	}
	return sortedArtifacts(byVersion), nil
}
func preferPHP(a, b Artifact) bool {
	if validSHA256(a.SHA256) != validSHA256(b.SHA256) {
		return validSHA256(a.SHA256)
	}
	revision := func(build string) string {
		parts := strings.SplitN(build, "-", 2)
		if len(parts) == 2 {
			return "0.0." + parts[1]
		}
		return "0.0.0"
	}
	if a.Build != b.Build {
		return versionGreater(revision(a.Build), revision(b.Build))
	}
	return a.Variant > b.Variant
}
func sortedArtifacts(values map[string]Artifact) []Artifact {
	result := make([]Artifact, 0, len(values))
	for _, a := range values {
		result = append(result, a)
	}
	sort.Slice(result, func(i, j int) bool { return versionGreater(result[i].Version, result[j].Version) })
	return result
}
func (p *pythonProvider) listHistory(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("Python", platform); err != nil {
		return nil, err
	}
	latest, err := p.readRelease(ctx, p.apiURL+"/latest", platform)
	if err != nil {
		return nil, err
	}
	versions := map[string]Artifact{}
	for _, a := range latest {
		versions[a.Version] = a
	}
	// Limite explícito de 30 releases, sem seguir URLs Link arbitrárias.
	for page := 1; page <= 3; page++ {
		data, err := get(ctx, p.client, fmt.Sprintf("%s?per_page=10&page=%d", p.apiURL, page), 32<<20)
		if err != nil {
			return nil, err
		}
		var releases []pythonRelease
		if err := json.Unmarshal(data, &releases); err != nil || releases == nil {
			return nil, fmt.Errorf("histórico Python inválido: %v", err)
		}
		for _, release := range releases {
			if release.Draft || release.Prerelease || !pythonBuild.MatchString(release.Tag) {
				continue
			}
			items, err := p.releaseArtifacts(release, platform, true)
			if err != nil {
				return nil, err
			}
			for _, a := range items {
				if old, ok := versions[a.Version]; !ok || a.Build > old.Build {
					versions[a.Version] = a
				}
			}
		}
		if len(releases) < 10 {
			break
		}
	}
	return sortedArtifacts(versions), nil
}
