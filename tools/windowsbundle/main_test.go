package main

import (
	"strings"
	"testing"
)

func TestOfflineDocumentIsSelfContainedAndEscapesTitle(t *testing.T) {
	page := document(`<script>bad</script>`, `<h1>Guia</h1>`)
	if strings.Contains(page, `<title>Verman — <script>`) || !strings.Contains(page, `&lt;script&gt;`) || !strings.Contains(page, `<style>`) || !strings.Contains(page, `lang="pt-BR"`) {
		t.Fatal("documento não preserva escape/estilo offline")
	}
}
