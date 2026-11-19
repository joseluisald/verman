package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/tool"
)

// ListInstalled consulta apenas arquivos locais. Instalações incompletas,
// incompatíveis e diretórios com nomes inválidos não entram no resultado.
func ListInstalled(language string) ([]string, error) {
	if err := tool.ValidateLanguage(language); err != nil {
		return nil, err
	}
	base, err := config.BaseDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "installs", language)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consultar instalações de %s: %w", language, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("diretório de instalações de %s inválido: %s", language, dir)
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("listar instalações de %s: %w", language, err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || tool.Validate(language, entry.Name()) != nil {
			continue
		}
		if _, err := ReadMetadata(language, entry.Name()); err != nil {
			// Erros de acesso devem ser visíveis, não apresentados como ausência.
			if errors.Is(err, os.ErrPermission) {
				return nil, err
			}
			continue
		}
		versions = append(versions, entry.Name())
	}
	sort.Slice(versions, func(i, j int) bool { return versionGreater(versions[i], versions[j]) })
	return versions, nil
}

// Os componentes já foram validados como decimais sem zeros à esquerda.
// Comparar comprimento antes do texto evita overflow em versões muito grandes.
func versionGreater(a, b string) bool {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := range left {
		if len(left[i]) != len(right[i]) {
			return len(left[i]) > len(right[i])
		}
		if left[i] != right[i] {
			return left[i] > right[i]
		}
	}
	return false
}
