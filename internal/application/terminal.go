package application

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joseluisald/verman/internal/environment"
	"github.com/joseluisald/verman/internal/shimmeta"
)

// TerminalEnvironment prepara o processo novo, mantendo shims verificados antes
// dos caminhos diretos. Assim um use posterior continua dinâmico nessa sessão.
func TerminalEnvironment(dir string, environ []string) ([]string, error) {
	result, err := environment.Compute(dir, environ)
	if err != nil {
		return nil, err
	}
	var shimDir string
	for _, entry := range environ {
		pair := strings.SplitN(entry, "=", 2)
		if len(pair) != 2 || !strings.EqualFold(pair[0], "PATH") {
			continue
		}
		for _, candidate := range filepath.SplitList(pair[1]) {
			candidate = strings.Trim(candidate, `"`)
			if !filepath.IsAbs(candidate) {
				continue
			}
			if shimmeta.Matches(filepath.Join(candidate, "node.exe")) && shimmeta.Matches(filepath.Join(candidate, "php.exe")) && shimmeta.Matches(filepath.Join(candidate, "python.exe")) {
				shimDir = candidate
				break
			}
		}
	}
	if shimDir == "" {
		return result, nil
	}
	for i, entry := range result {
		pair := strings.SplitN(entry, "=", 2)
		if len(pair) != 2 || !strings.EqualFold(pair[0], "PATH") {
			continue
		}
		paths := []string{shimDir}
		for _, path := range filepath.SplitList(pair[1]) {
			if !strings.EqualFold(filepath.Clean(path), filepath.Clean(shimDir)) {
				paths = append(paths, path)
			}
		}
		result[i] = pair[0] + "=" + strings.Join(paths, string(os.PathListSeparator))
	}
	return result, nil
}
