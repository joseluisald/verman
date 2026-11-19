package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/config"
)

const MetadataFileName = "installation.json"

// Metadata deve ser publicado pelo instalador somente após validação
// funcional e de integridade. Executables contém caminhos relativos ao destino.
type Metadata struct {
	SchemaVersion int               `json:"schema_version"`
	Language      string            `json:"language"`
	Version       string            `json:"version"`
	OS            string            `json:"os"`
	Arch          string            `json:"arch"`
	Source        string            `json:"source"`
	SHA256        string            `json:"sha256,omitempty"`
	SHA512        string            `json:"sha512,omitempty"`
	Variant       string            `json:"variant,omitempty"`
	Build         string            `json:"build,omitempty"`
	Executables   map[string]string `json:"executables"`
	TransactionID string            `json:"transaction_id,omitempty"`
	Files         map[string]string `json:"files,omitempty"`
	Required      []string          `json:"required,omitempty"`
}

func executablePath(root, relative string) (string, error) {
	if strings.ContainsAny(relative, `\:`) || !filepath.IsLocal(relative) {
		return "", fmt.Errorf("caminho de executável inválido: %q", relative)
	}
	path := filepath.Join(root, relative)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("executável fora da instalação: %q", relative)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("executável ausente ou vazio: %q", relative)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("arquivo sem permissão de execução: %q", relative)
	}
	return path, nil
}

// ReadMetadata verifica identidade, plataforma e arquivos declarados sem executar
// código. A origem registrada não substitui a verificação de integridade ao instalar.
func ReadMetadata(lang, version string) (*Metadata, error) {
	root, err := config.InstallDir(lang, version)
	if err != nil {
		return nil, err
	}
	return readMetadataAt(root, lang, version, Platform{runtime.GOOS, runtime.GOARCH})
}

func readMetadataAt(root, lang, version string, platform Platform) (*Metadata, error) {
	if info, err := os.Lstat(root); err != nil {
		return nil, err
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("instalação é link/não diretório")
	}
	anchored, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer anchored.Close()
	if info, err := anchored.Lstat(MetadataFileName); err != nil {
		return nil, err
	} else if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("metadados não são arquivo regular")
	}
	file, err := anchored.Open(MetadataFileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (8<<20)+1))
	if len(data) > 8<<20 {
		return nil, fmt.Errorf("metadados excedem 8 MiB")
	}
	if err != nil {
		return nil, fmt.Errorf("ler metadados de %s %s: %w", lang, version, err)
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("metadados inválidos: %w", err)
	}
	if (meta.SchemaVersion != 1 && meta.SchemaVersion != 2) || meta.Language != lang || meta.Version != version || meta.OS != platform.OS || meta.Arch != platform.Arch {
		return nil, fmt.Errorf("metadados incompatíveis com %s %s em %s/%s", lang, version, runtime.GOOS, runtime.GOARCH)
	}
	source, err := url.Parse(meta.Source)
	if err != nil || source.Scheme != "https" || source.Host == "" || source.User != nil || source.Fragment != "" {
		return nil, fmt.Errorf("origem da instalação inválida")
	}
	if meta.Executables[lang] == "" {
		return nil, fmt.Errorf("executável principal %s ausente dos metadados", lang)
	}
	for name, relative := range meta.Executables {
		if _, err := executablePath(root, relative); err != nil {
			return nil, fmt.Errorf("validar executável %s: %w", name, err)
		}
	}
	if meta.SchemaVersion == 2 {
		if !validSHA256(meta.SHA256) || len(meta.Files) == 0 {
			return nil, fmt.Errorf("metadados sem manifesto de integridade válido")
		}
		for relative, digest := range meta.Files {
			if !validSHA256(digest) || !safeRelative(relative) {
				return nil, fmt.Errorf("entrada de integridade inválida: %s", relative)
			}
		}
		for _, relative := range meta.Executables {
			if _, ok := meta.Files[relative]; !ok {
				return nil, fmt.Errorf("executável sem hash: %s", relative)
			}
		}
		for _, relative := range meta.Required {
			if _, err := containedFile(root, relative); err != nil {
				return nil, fmt.Errorf("dependência ausente: %s: %w", relative, err)
			}
		}
	}
	return &meta, nil
}

func IsInstalled(lang, version string) bool {
	_, err := ReadMetadata(lang, version)
	return err == nil
}

// BinDirs respeita o layout do provedor, com o runtime principal primeiro.
func BinDirs(lang, version string) ([]string, error) {
	meta, err := ReadMetadata(lang, version)
	if err != nil {
		return nil, err
	}
	root, err := config.InstallDir(lang, version)
	if err != nil {
		return nil, err
	}
	primary := filepath.Dir(filepath.Join(root, meta.Executables[lang]))
	unique := make(map[string]bool)
	for _, relative := range meta.Executables {
		dir := filepath.Dir(filepath.Join(root, relative))
		if dir != primary {
			unique[dir] = true
		}
	}
	others := make([]string, 0, len(unique))
	for dir := range unique {
		others = append(others, dir)
	}
	sort.Strings(others)
	if lang == "chocolatey" {
		others = append(others, filepath.Join(primary, "bin"))
	}
	return append([]string{primary}, others...), nil
}

func Install(lang, version string) error {
	return InstallContext(context.Background(), lang, version, os.Stdout)
}
