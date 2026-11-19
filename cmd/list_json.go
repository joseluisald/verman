package cmd

import (
	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/installer"
	"github.com/joseluisald/verman/internal/tool"
)

type localLanguage struct {
	Language       string   `json:"language"`
	Installed      []string `json:"installed"`
	ProjectVersion string   `json:"project_version"`
	GlobalVersion  string   `json:"global_version"`
}

func localListData(dir string, args []string) ([]localLanguage, error) {
	project, err := config.LoadProjectConfig(dir)
	if err != nil {
		return nil, err
	}
	global, err := config.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}
	languages := tool.Languages()
	if len(args) > 0 {
		languages = args
	}
	result := []localLanguage{}
	for _, lang := range languages {
		versions, err := installer.ListInstalled(lang)
		if err != nil {
			return nil, err
		}
		if versions == nil {
			versions = []string{}
		}
		result = append(result, localLanguage{lang, versions, project.Tools[lang], global.Tools[lang]})
	}
	return result, nil
}

type availableVersion struct {
	installer.Artifact
	Policy         installer.DistributionInfo `json:"distribution"`
	Installed      bool                       `json:"installed"`
	InstalledBuild string                     `json:"installed_build,omitempty"`
	DifferentBuild bool                       `json:"different_build"`
}

func availableListData(language string, result installer.CatalogResult) any {
	versions := []availableVersion{}
	for _, artifact := range result.Catalog.Artifacts {
		entry := availableVersion{Artifact: artifact, Policy: installer.Describe(artifact)}
		if meta, err := installer.ReadMetadata(language, artifact.Version); err == nil {
			entry.Installed = true
			entry.InstalledBuild = meta.Build
			entry.DifferentBuild = meta.Build != artifact.Build || meta.Variant != artifact.Variant
		}
		versions = append(versions, entry)
	}
	return struct {
		Language string                  `json:"language"`
		Catalog  installer.CatalogResult `json:"catalog_result"`
		Versions []availableVersion      `json:"versions"`
	}{language, result, versions}
}
