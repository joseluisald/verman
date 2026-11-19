// Command windowsbundle prepara documentação HTML offline e payload do instalador.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/joseluisald/verman/docs"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func main() {
	out := flag.String("out", "dist/windows/payload", "Pasta do payload")
	flag.Parse()
	if err := prepare(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func prepare(out string) error {
	for _, name := range []string{"docs", "scripts"} {
		if err := os.MkdirAll(filepath.Join(out, name), 0700); err != nil {
			return err
		}
	}
	for _, name := range []string{"setup-path.ps1", "remove-path.ps1"} {
		data, err := os.ReadFile(filepath.Join("scripts", name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, "scripts", name), data, 0600); err != nil {
			return err
		}
	}
	converter := goldmark.New(goldmark.WithExtensions(extension.GFM))
	topics := []string{"PRIMEIROS_PASSOS", "SUPORTE", "CLI_JSON", "PROVEDORES", "DESKTOP", "FERRAMENTAS"}
	nav := `<nav aria-label="Documentação"><a href="index.html">Documentação</a> · <a href="PRIMEIROS_PASSOS.html">Primeiros passos</a> · <a href="SUPORTE.html">Suporte</a> · <a href="CLI_JSON.html">CLI</a> · <a href="FERRAMENTAS.html">Ferramentas</a> · <a href="PROVEDORES.html">Provedores</a> · <a href="DESKTOP.html">Desktop</a></nav>`
	for _, topic := range topics {
		content, err := docs.Read(topic)
		if err != nil {
			return err
		}
		var rendered bytes.Buffer
		if err := converter.Convert([]byte(content), &rendered); err != nil {
			return err
		}
		body := strings.ReplaceAll(rendered.String(), `.md"`, `.html"`)
		// O guia temporário aberto pelo instalador é autossuficiente; links locais ficam disponíveis após instalar.
		page := document(topic, nav+body)
		if err := os.WriteFile(filepath.Join(out, "docs", topic+".html"), []byte(page), 0600); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, "docs", topic+".md"), []byte(content), 0600); err != nil {
			return err
		}
	}
	index := `<p class="eyebrow">VERMAN · DOCUMENTAÇÃO OFFLINE</p><h1>Seu ambiente começa aqui.</h1><p>CLI e desktop usam as mesmas instalações e seleções, sem conta ou telemetria.</p><p><a class="cta" href="PRIMEIROS_PASSOS.html">Configurar e começar →</a></p><h2>Guias</h2><ul><li><a href="FERRAMENTAS.html">Ferramentas e dependências</a></li><li><a href="PRIMEIROS_PASSOS.html">Instalação e primeiro projeto</a></li><li><a href="SUPORTE.html">Offline, diagnóstico e suporte</a></li><li><a href="CLI_JSON.html">Contrato da CLI e códigos de saída</a></li><li><a href="PROVEDORES.html">Catálogos, origem e plataformas</a></li><li><a href="DESKTOP.html">Aplicativo desktop e atualização</a></li></ul>`
	return os.WriteFile(filepath.Join(out, "docs", "index.html"), []byte(document("Documentação", nav+index)), 0600)
}
func document(title, body string) string {
	return `<!doctype html><html lang="pt-BR"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'"><title>Verman — ` + html.EscapeString(title) + `</title><style>body{font:16px/1.85 "Segoe UI",sans-serif;max-width:880px;padding:40px 24px;margin:auto;color:#334237;background:#f8faf6}nav{font-size:13px;border-bottom:1px solid #dce5d7;padding-bottom:20px;margin-bottom:38px}a{color:#286742;text-underline-offset:4px}h1{font-size:34px;line-height:1.3;letter-spacing:-1px;color:#203827}h2{font-size:23px;margin-top:40px}h3{font-size:18px}p,li{overflow-wrap:anywhere}code{font:14px/1.8 Consolas,monospace}pre{padding:20px;background:#edf2e8;border:1px solid #d9e3cf;border-radius:8px;overflow:auto}table{border-collapse:collapse;display:block;overflow:auto;font-size:13px}td,th{padding:12px;border:1px solid #d9e2d1;text-align:left}.cta{display:inline-block;background:#286742;color:white;padding:10px 18px;border-radius:6px;text-decoration:none}.eyebrow{font-size:12px;letter-spacing:2px;color:#6f8065}a:focus-visible{outline:3px solid #587c48;outline-offset:4px}</style><main>` + body + `</main></html>`
}
