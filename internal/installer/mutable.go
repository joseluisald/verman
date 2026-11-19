package installer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var pipLauncher = regexp.MustCompile(`(?i)^scripts/pip([0-9]+(\.[0-9]+)?)?\.(exe|cmd|py)$`)

func mutableRoots(language string) []string {
	switch language {
	case "rust":
		return []string{"cargo/bin", "cargo/registry", "cargo/git", "cargo/.global-cache", "cargo/.crates.toml", "cargo/.crates2.json", "cargo/config.toml"}
	case "vcpkg":
		return []string{"installed", "packages", "downloads", "buildtrees"}
	case "chocolatey":
		return []string{"tools/chocolateyInstall/config", "tools/chocolateyInstall/lib", "tools/chocolateyInstall/lib-bad", "tools/chocolateyInstall/lib-bkp", "tools/chocolateyInstall/bin", "tools/chocolateyInstall/logs", "tools/chocolateyInstall/cache"}
	}
	return nil
}

// Preserva somente dados explicitamente mutáveis, sem seguir links. O runtime
// da origem já foi validado antes disso; configuração do usuário não é executada.
func preserveMutable(ctx context.Context, source, destination, language string) error {
	roots := []string{}
	roots = append(roots, mutableRoots(language)...)
	if language == "php" {
		roots = []string{"php.ini", "conf.d"}
	}
	if language == "python" {
		roots = []string{"Lib/site-packages", "Scripts"}
	}
	var total int64
	count := 0
	for _, relativeRoot := range roots {
		start := filepath.Join(source, filepath.FromSlash(relativeRoot))
		if _, err := os.Lstat(start); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		err := filepath.WalkDir(start, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if language == "rust" && strings.HasPrefix(strings.ToLower(relative), "cargo/bin/") && rustProxy(relative) {
				return nil
			}
			if language == "python" {
				part := strings.TrimPrefix(strings.ToLower(relative), "lib/site-packages/")
				name := strings.Split(part, "/")[0]
				if name == "pip" || strings.HasPrefix(name, "pip-") || name == "setuptools" || strings.HasPrefix(name, "setuptools-") || name == "wheel" || strings.HasPrefix(name, "wheel-") || name == "_distutils_hack" {
					if entry.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if pipLauncher.MatchString(relative) {
					return nil
				}
			}
			target := filepath.Join(destination, filepath.FromSlash(relative))
			if entry.IsDir() {
				return os.MkdirAll(target, 0700)
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("dados mutáveis contêm link/tipo não regular: %s", relative)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += info.Size()
			count++
			if total > 1<<30 || count > 100000 {
				return fmt.Errorf("dados mutáveis excedem limite de cópia (1 GiB/100000 arquivos); instalação anterior preservada")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			input, err := os.Open(path)
			if err != nil {
				return err
			}
			defer input.Close()
			output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(output, &contextReader{ctx: ctx, reader: input})
			closeErr := output.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		})
		if err != nil {
			return err
		}
	}
	return nil
}
