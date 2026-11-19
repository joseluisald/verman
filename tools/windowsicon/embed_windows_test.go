//go:build windows

package main

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsReadsEmbeddedLogoResources(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "desktop espaço.exe")
	if err := os.WriteFile(exe, original, 0700); err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 23, G: 59, B: 50, A: 255})
		}
	}
	ico, err := encodeICO(img)
	if err != nil {
		t.Fatal(err)
	}
	if err := embedIcon(exe, ico); err != nil {
		t.Fatal(err)
	}
	// Carrega exclusivamente como dados: nao executa o binario de teste copiado.
	module, err := windows.LoadLibraryEx(exe, 0, windows.LOAD_LIBRARY_AS_DATAFILE|windows.LOAD_LIBRARY_AS_IMAGE_RESOURCE)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.FreeLibrary(module)
	group, frames, err := iconResources(ico)
	if err != nil {
		t.Fatal(err)
	}
	read := func(id, kind uint16) []byte {
		t.Helper()
		resource, err := windows.FindResource(module, windows.ResourceID(id), windows.ResourceID(kind))
		if err != nil {
			t.Fatal(err)
		}
		data, err := windows.LoadResourceData(module, resource)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if !bytes.Equal(read(32512, 14), group) {
		t.Fatal("grupo IDI_APPLICATION diferente do logo")
	}
	for i, frame := range frames {
		if !bytes.Equal(read(uint16(i+1), 3), frame) {
			t.Fatalf("imagem %d diferente do ICO", i+1)
		}
	}
}
