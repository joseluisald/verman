package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/fault"
)

func safeRelative(path string) bool {
	return filepath.IsLocal(path) && !strings.ContainsAny(path, `\:`)
}
func containedFile(root, relative string) (string, error) {
	if !safeRelative(relative) {
		return "", fmt.Errorf("caminho inválido")
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, resolved)
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("arquivo fora da instalação")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("arquivo ausente ou vazio")
	}
	return path, nil
}
func mutableFile(language, relative string) bool {
	lower := strings.ToLower(relative)
	if language == "rust" && rustProxy(relative) {
		return false
	}
	for _, root := range mutableRoots(language) {
		if lower == strings.ToLower(root) || strings.HasPrefix(lower, strings.ToLower(root)+"/") {
			return true
		}
	}
	if strings.Contains(lower, "/__pycache__/") || strings.HasSuffix(lower, ".pyc") {
		return true
	}
	if language == "php" && (lower == "php.ini" || strings.HasPrefix(lower, "conf.d/")) {
		return true
	}
	return language == "python" && (strings.HasPrefix(lower, "lib/site-packages/") || strings.HasPrefix(lower, "scripts/"))
}

func rustProxy(relative string) bool {
	if !strings.HasPrefix(strings.ToLower(relative), "cargo/bin/") {
		return false
	}
	name := strings.ToLower(filepath.Base(relative))
	for _, core := range []string{"rustc", "rustdoc", "cargo", "rustup", "cargo-clippy", "clippy-driver", "rustfmt", "cargo-fmt", "rust-analyzer", "rust-gdb", "rust-lldb", "rust-gdbgui"} {
		if name == core+".exe" || name == core {
			return true
		}
	}
	return false
}
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func fingerprint(root string, meta *Metadata) error {
	meta.Files = map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("tipo não regular no runtime: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == MetadataFileName || mutableFile(meta.Language, relative) {
			return nil
		}
		digest, err := hashFile(path)
		if err != nil {
			return err
		}
		meta.Files[relative] = digest
		return nil
	})
	if err != nil {
		return err
	}
	meta.Required = []string{}
	// Só inclui dependências que existem no pacote; validação funcional já foi executada.
	for _, relative := range []string{"node_modules/npm/bin/npm-cli.js", "node_modules/npm/bin/npx-cli.js", "php.ini", "Lib/venv/__init__.py", "Lib/site-packages/pip/__init__.py"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err == nil {
			meta.Required = append(meta.Required, relative)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".dll") {
			meta.Required = append(meta.Required, entry.Name())
		}
	}
	meta.SchemaVersion = 2
	return nil
}

// Verify recalcula hashes, sem executar arquivos. Configurações e pacotes mutáveis
// estão excluídos; schema 1 não é promovido silenciosamente a integridade verificada.
func Verify(language, version string) error {
	return VerifyContext(context.Background(), language, version)
}
func VerifyContext(ctx context.Context, language, version string) error {
	lease, err := Lease(ctx, language, version)
	if err != nil {
		return err
	}
	defer lease()
	meta, err := ReadMetadata(language, version)
	if err != nil {
		return err
	}
	if meta.SchemaVersion != 2 {
		return fault.New("integrity_unavailable", 4, "Execute repair para reinstalar da origem verificada.", fmt.Errorf("instalação legada sem manifesto de arquivos; use repair para validar novamente pela origem"))
	}
	root, err := config.InstallDir(language, version)
	if err != nil {
		return err
	}
	return verifyFilesContext(ctx, root, meta)
}

func verifyFiles(root string, meta *Metadata) error {
	return verifyFilesContext(context.Background(), root, meta)
}
func verifyFilesContext(ctx context.Context, root string, meta *Metadata) error {
	for relative, expected := range meta.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		path, err := containedFile(root, relative)
		if err != nil {
			return err
		}
		actual, err := hashFile(path)
		if err != nil {
			return err
		}
		if actual != expected {
			return fault.New("integrity_error", 4, "Execute repair para reinstalar da origem verificada.", fmt.Errorf("arquivo alterado: %s; use repair", relative))
		}
	}
	return nil
}
