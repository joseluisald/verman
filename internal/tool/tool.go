package tool

import (
	"github.com/joseluisald/verman/internal/fault"
	"regexp"
	"strings"
)

var exactVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// Languages retorna uma cópia da lista de linguagens conhecidas, em ordem estável.
func Languages() []string {
	result := []string{}
	for _, definition := range Definitions() {
		result = append(result, definition.ID)
	}
	return result
}

func ValidateLanguage(language string) error {
	for _, definition := range Definitions() {
		if language == definition.ID {
			return nil
		}
	}
	return fault.Usage("ferramenta %q desconhecida; disponíveis: %s", language, strings.Join(Languages(), ", "))
}

// Validate aceita apenas linguagens conhecidas e versões exatas estáveis.
func Validate(language, version string) error {
	if err := ValidateLanguage(language); err != nil {
		return err
	}
	if !exactVersion.MatchString(version) {
		return fault.Usage("versão %q inválida: informe uma versão exata no formato X.Y.Z", version)
	}
	return nil
}
