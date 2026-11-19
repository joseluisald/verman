// Command windowsicon exporta o logo e incorpora seu ícone ao desktop Windows.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var iconSizes = []int{16, 24, 32, 48, 64, 128, 256}

func main() {
	source := flag.String("source", "desktop/frontend/src/assets/verman-logo.png", "Logo PNG original")
	out := flag.String("out", "dist/windows/payload/assets", "Pasta dos recursos gerados")
	exe := flag.String("exe", "", "Executável Windows recém-compilado para receber o ícone")
	icon := flag.String("icon", "", "Arquivo ICO a incorporar")
	flag.Parse()
	var err error
	if *exe != "" {
		if *icon == "" {
			err = fmt.Errorf("informe -icon")
		} else {
			var data []byte
			data, err = os.ReadFile(*icon)
			if err == nil {
				err = embedIcon(*exe, data)
			}
		}
	} else {
		err = prepare(*source, *out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepare(source, out string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if len(data) > 32<<20 {
		return fmt.Errorf("logo excede 32 MiB")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if cfg.Width != cfg.Height || cfg.Width < 256 || cfg.Width > 4096 {
		return fmt.Errorf("logo deve ser quadrado, entre 256 e 4096 pixels")
	}
	logo, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	ico, err := encodeICO(logo)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "verman.ico"), ico, 0644); err != nil {
		return err
	}
	if err := savePNG(filepath.Join(out, "verman-logo.png"), resize(logo, 256)); err != nil {
		return err
	}
	banner := image.NewNRGBA(image.Rect(0, 0, 328, 628))
	draw.Draw(banner, banner.Bounds(), image.NewUniform(color.NRGBA{R: 23, G: 59, B: 50, A: 255}), image.Point{}, draw.Src)
	draw.Draw(banner, image.Rect(36, 186, 292, 442), resize(logo, 256), image.Point{}, draw.Src)
	return savePNG(filepath.Join(out, "wizard-logo.png"), banner)
}
func savePNG(path string, img image.Image) error {
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		return err
	}
	return os.WriteFile(path, data.Bytes(), 0644)
}

// Redução por área preserva as faixas finas do logo em ícones pequenos.
func resize(src image.Image, size int) *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, size, size))
	bounds := src.Bounds()
	scale := float64(bounds.Dx()) / float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			left, right := float64(x)*scale, float64(x+1)*scale
			top, bottom := float64(y)*scale, float64(y+1)*scale
			var r, g, b, a, total float64
			for sy := int(top); sy < int(math.Ceil(bottom)); sy++ {
				for sx := int(left); sx < int(math.Ceil(right)); sx++ {
					weight := (math.Min(right, float64(sx+1)) - math.Max(left, float64(sx))) * (math.Min(bottom, float64(sy+1)) - math.Max(top, float64(sy)))
					c := color.NRGBAModel.Convert(src.At(bounds.Min.X+sx, bounds.Min.Y+sy)).(color.NRGBA)
					alpha := float64(c.A) / 255
					r += float64(c.R) * alpha * weight
					g += float64(c.G) * alpha * weight
					b += float64(c.B) * alpha * weight
					a += float64(c.A) * weight
					total += weight
				}
			}
			pixel := color.NRGBA{}
			if a > 0 {
				pixel = color.NRGBA{R: uint8(math.Round(r * 255 / a)), G: uint8(math.Round(g * 255 / a)), B: uint8(math.Round(b * 255 / a)), A: uint8(math.Round(a / total))}
			}
			result.SetNRGBA(x, y, pixel)
		}
	}
	return result
}

func encodeICO(logo image.Image) ([]byte, error) {
	header := make([]byte, 6+16*len(iconSizes))
	binary.LittleEndian.PutUint16(header[2:], 1)
	binary.LittleEndian.PutUint16(header[4:], uint16(len(iconSizes)))
	data := header
	for i, size := range iconSizes {
		var frame bytes.Buffer
		if err := png.Encode(&frame, resize(logo, size)); err != nil {
			return nil, err
		}
		entry := data[6+i*16 : 6+(i+1)*16]
		entry[0] = byte(size % 256)
		entry[1] = byte(size % 256)
		binary.LittleEndian.PutUint16(entry[4:], 1)
		binary.LittleEndian.PutUint16(entry[6:], 32)
		binary.LittleEndian.PutUint32(entry[8:], uint32(frame.Len()))
		binary.LittleEndian.PutUint32(entry[12:], uint32(len(data)))
		data = append(data, frame.Bytes()...)
	}
	return data, nil
}

func iconResources(data []byte) ([]byte, [][]byte, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data) != 0 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil, nil, fmt.Errorf("cabeçalho ICO inválido")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	if count < 1 || count > 32 || len(data) < 6+16*count {
		return nil, nil, fmt.Errorf("diretório ICO inválido")
	}
	group := make([]byte, 6+14*count)
	copy(group, data[:6])
	frames := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		entry := data[6+i*16 : 6+(i+1)*16]
		size := uint64(binary.LittleEndian.Uint32(entry[8:]))
		offset := uint64(binary.LittleEndian.Uint32(entry[12:]))
		if size == 0 || offset < uint64(6+16*count) || offset+size > uint64(len(data)) {
			return nil, nil, fmt.Errorf("imagem ICO fora do arquivo")
		}
		frame := data[offset : offset+size]
		cfg, err := png.DecodeConfig(bytes.NewReader(frame))
		width, height := int(entry[0]), int(entry[1])
		if width == 0 {
			width = 256
		}
		if height == 0 {
			height = 256
		}
		if err != nil || cfg.Width != width || cfg.Height != height {
			return nil, nil, fmt.Errorf("dimensão PNG/ICO inconsistente")
		}
		copy(group[6+i*14:], entry[:12])
		binary.LittleEndian.PutUint16(group[6+i*14+12:], uint16(i+1))
		frames = append(frames, frame)
	}
	return group, frames, nil
}
