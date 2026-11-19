// Package docs disponibiliza a documentação offline no desktop e no instalador.
package docs

import (
	"embed"
	"fmt"
)

//go:embed *.md
var Files embed.FS

func Read(topic string) (string, error) {
	switch topic {
	case "PRIMEIROS_PASSOS", "SUPORTE", "CLI_JSON", "PROVEDORES", "DESKTOP", "FERRAMENTAS":
	default:
		return "", fmt.Errorf("documentação desconhecida")
	}
	data, err := Files.ReadFile(topic + ".md")
	return string(data), err
}
