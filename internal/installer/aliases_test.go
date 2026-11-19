package installer

import (
	"context"
	"io"
	"testing"
)

func TestVersionAliases(t *testing.T) {
	items := []Artifact{{Language: "node", Version: "9.9.0", LTS: true}, {Language: "node", Version: "24.1.0", LTS: true}, {Language: "node", Version: "25.1.0"}, {Language: "php", Version: "99.0.0"}}
	for alias, expected := range map[string]string{"latest": "25.1.0", "lts": "24.1.0"} {
		got, err := resolveAlias("node", alias, items)
		if err != nil || got != expected {
			t.Fatalf("%s: %s %v", alias, got, err)
		}
	}
	if _, err := ResolveVersion(context.Background(), "php", "lts", io.Discard); err == nil {
		t.Fatal("lts PHP aceito")
	}
	if _, err := ResolveVersion(context.Background(), "node", "../../escape", io.Discard); err == nil {
		t.Fatal("entrada insegura aceita")
	}
	if got, err := ResolveVersion(context.Background(), "node", "22.11.0", io.Discard); err != nil || got != "22.11.0" {
		t.Fatal("versão exata dependeu de rede")
	}
	if _, err := resolveAlias("python", "latest", nil); err == nil {
		t.Fatal("catálogo vazio aceito")
	}
}
