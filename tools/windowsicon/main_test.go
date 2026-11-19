package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLogoExportsWindowsSizesAndInstallerImages(t *testing.T) {
	out := t.TempDir()
	source := filepath.Join("..", "..", "desktop", "frontend", "src", "assets", "verman-logo.png")
	if err := prepare(source, out); err != nil {
		t.Fatal(err)
	}
	ico, err := os.ReadFile(filepath.Join(out, "verman.ico"))
	if err != nil {
		t.Fatal(err)
	}
	group, frames, err := iconResources(ico)
	if err != nil || len(frames) != len(iconSizes) {
		t.Fatalf("ICO: %v", err)
	}
	for i, size := range iconSizes {
		cfg, err := png.DecodeConfig(bytes.NewReader(frames[i]))
		if err != nil || cfg.Width != size || cfg.Height != size {
			t.Fatalf("tamanho %d: %+v %v", size, cfg, err)
		}
		if binary.LittleEndian.Uint16(group[6+i*14+12:]) != uint16(i+1) {
			t.Fatal("grupo nao referencia a imagem correta")
		}
	}
	for name, want := range map[string]image.Point{"verman-logo.png": {256, 256}, "wizard-logo.png": {328, 628}} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width != want.X || cfg.Height != want.Y {
			t.Fatalf("%s: %+v %v", name, cfg, err)
		}
	}
	broken := append([]byte{}, ico...)
	binary.LittleEndian.PutUint32(broken[6+12:], uint32(len(broken)))
	if _, _, err := iconResources(broken); err == nil {
		t.Fatal("offset ICO fora do arquivo aceito")
	}
	if _, _, err := iconResources(ico[:8]); err == nil {
		t.Fatal("ICO truncado aceito")
	}
}

func TestIconReductionKeepsOpaqueAndTransparentEdges(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 255})
	result := resize(src, 1).NRGBAAt(0, 0)
	if result.R != 255 || result.G != 0 || result.B != 0 || result.A != 128 {
		t.Fatalf("redução com halo/cor incorreta: %+v", result)
	}
}
