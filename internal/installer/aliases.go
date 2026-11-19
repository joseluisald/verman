package installer

import (
	"context"
	"fmt"
	"io"

	"github.com/joseluisald/verman/internal/tool"
)

// Aliases são resolvidos apenas na seleção/instalação; arquivos TOML continuam exatos.
func ResolveVersion(ctx context.Context, language, requested string, warnings io.Writer) (string, error) {
	return ResolveVersionEvents(ctx, language, requested, legacyEvents(warnings))
}

func ResolveVersionEvents(ctx context.Context, language, requested string, sink EventSink) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := tool.ValidateLanguage(language); err != nil {
		return "", err
	}
	if requested != "latest" && requested != "lts" {
		if err := tool.Validate(language, requested); err != nil {
			return "", err
		}
		return requested, nil
	}
	if requested == "lts" && language != "node" {
		return "", fmt.Errorf("alias lts disponível somente para node")
	}
	// Atualize LTS para não depender de caches antigos sem esse metadado.
	result, err := Available(ctx, language, requested == "lts" && !offline(ctx))
	if err != nil {
		return "", err
	}
	if result.Warning != "" {
		if err := emit(sink, Event{Kind: EventWarning, Language: language, Version: requested, Message: result.Warning}); err != nil {
			return "", err
		}
	}
	return resolveAlias(language, requested, result.Catalog.Artifacts)
}

func resolveAlias(language, alias string, items []Artifact) (string, error) {
	best := ""
	for _, item := range items {
		if !Describe(item).Installable {
			continue
		}
		if item.Language != language || (alias == "lts" && !item.LTS) {
			continue
		}
		if best == "" || versionGreater(item.Version, best) {
			best = item.Version
		}
	}
	if best == "" {
		return "", fmt.Errorf("nenhuma versão compatível para %s %s", language, alias)
	}
	return best, nil
}
