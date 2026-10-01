// Package extract decodes the compact native game data without executing it.
package extract

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type Definition struct {
	ID, First, Last, Animation, Margin, Category, Behavior, Target, Health int
	Rate, SpeedX, SpeedY                                                   float64
	Passable, Transparent, Declared                                        bool
	Width, Height                                                          int
}
type Frame struct {
	Type, Index, Width, Height int
	File                       string
}
type Metadata struct {
	Definitions []Definition
	Frames      []Frame
}

func ExportImage(exe []byte, directory string) error {
	if len(exe) < 0x12000 || string(exe[:2]) != "MZ" {
		return fmt.Errorf("extract: incomplete native PE")
	}
	raw := func(address, size int) []byte { at := address - 0x400000; return exe[at : at+size] }
	if err := os.MkdirAll(filepath.Join(directory, "sprites"), 0755); err != nil {
		return err
	}
	f64 := func(address int) float64 { return math.Float64frombits(binary.LittleEndian.Uint64(raw(address, 8))) }
	rateScale := f64(0x409220)
	speedScale := float64(math.Float32frombits(binary.LittleEndian.Uint32(raw(0x409218, 4))))
	fmt.Println("Native scales", rateScale, speedScale)
	b := raw(0x40de98, 0x3dc0)
	p := 0
	paletteIndex := -1
	var palette [16]color.NRGBA
	var palettes [][16]color.NRGBA
	var meta Metadata
	writePNG := func(name string, img image.Image) error {
		file, e := os.Create(filepath.Join(directory, name))
		if e != nil {
			return e
		}
		e = png.Encode(file, img)
		closeErr := file.Close()
		if e != nil {
			return e
		}
		return closeErr
	}
	for p < len(b) {
		if len(b)-p < 15 {
			return fmt.Errorf("extract: truncated definition at %x", p)
		}
		h := b[p : p+15]
		p += 15
		def := Definition{ID: int(binary.LittleEndian.Uint16(h)), First: int(h[2]), Last: int(h[3]), Animation: int(h[4]), Rate: float64(h[5]) * rateScale, Passable: h[6]&2 != 0, Transparent: h[6]&1 == 0, Margin: int(h[7]), SpeedX: float64(float32(h[8]) * float32(speedScale)), SpeedY: float64(int(h[9])-127) * rateScale, Behavior: int(h[10]), Category: int(h[11]), Target: int(binary.LittleEndian.Uint16(h[12:])), Health: int(h[14]), Declared: true}
		if def.Last < def.First {
			return fmt.Errorf("extract: reversed frame range")
		}
		for frame := def.First; frame <= def.Last; frame++ {
			if p >= len(b) {
				return fmt.Errorf("extract: missing frame")
			}
			choice := int(b[p])
			p++
			if choice == 255 {
				paletteIndex++
				if len(b)-p < 48 {
					return fmt.Errorf("extract: missing palette")
				}
				for i := range palette {
					blue, green, red := b[p+i*3], b[p+i*3+1], b[p+i*3+2]
					// Native 16-bit conversion uses five-bit red/blue and even green values.
					r, g, bl := red>>1, green&62, blue>>1
					palette[i] = color.NRGBA{R: byte(uint32(r) * 255 / 31), G: byte(uint32(g) * 255 / 63), B: byte(uint32(bl) * 255 / 31), A: 255}
				}
				p += 48
				palettes = append(palettes, palette)
			} else {
				if choice >= len(palettes) {
					return fmt.Errorf("extract: invalid palette reference %d", choice)
				}
				palette = palettes[choice]
			}
			if len(b)-p < 4 {
				return fmt.Errorf("extract: missing frame size")
			}
			height, width := int(binary.LittleEndian.Uint16(b[p:])), int(binary.LittleEndian.Uint16(b[p+2:]))
			p += 4
			if width < 1 || height < 1 || width*height > 262144 || width*height%2 != 0 || len(b)-p < width*height/2 {
				return fmt.Errorf("extract: invalid frame dimensions %dx%d at %x", width, height, p)
			}
			def.Width, def.Height = width, height
			img := image.NewNRGBA(image.Rect(0, 0, width, height))
			for i := 0; i < width*height; i++ {
				index := b[p+i/2] & 15
				if i&1 != 0 {
					index = b[p+i/2] >> 4
				}
				c := palette[index]
				if c.R == 0 && c.G == 0 && c.B == 0 {
					c.A = 0
				}
				img.SetNRGBA(i%width, i/width, c)
			}
			p += width * height / 2
			name := fmt.Sprintf("sprites/%04d-%03d.png", def.ID, frame)
			if err := writePNG(name, img); err != nil {
				return err
			}
			meta.Frames = append(meta.Frames, Frame{Type: def.ID, Index: frame, Width: width, Height: height, File: name})
		}
		meta.Definitions = append(meta.Definitions, def)
	}
	if err := exportVariants(exe, directory, &meta); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "level.bin"), raw(0x40c2d8, 0x1260), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "title-mask.bin"), raw(0x40d538, 0x546), 0644); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(directory, "graphics.json"), append(encoded, '\n'), 0644); err != nil {
		return err
	}
	font := raw(0x40b6e8, 640)
	atlas := image.NewNRGBA(image.Rect(0, 0, 16*8, 5*8))
	for c := 32; c < 112; c++ {
		index := c - 32
		if c >= 72 {
			index = c + 248
		}
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				if font[index+y*40]&(1<<uint(x)) != 0 {
					atlas.SetNRGBA((c-32)%16*8+x, (c-32)/16*8+y, color.NRGBA{255, 255, 255, 255})
				}
			}
		}
	}
	if err := writePNG("font.png", atlas); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "font.bin"), font, 0644); err != nil {
		return err
	}
	if err := exportTextures(exe, directory); err != nil {
		return err
	}
	if err := exportTitle(directory, raw(0x40d538, 0x546)); err != nil {
		return err
	}
	fmt.Println("Decoded", len(meta.Definitions), "definitions and", len(meta.Frames), "native frames")
	return nil
}

func exportTitle(dir string, mask []byte) error {
	f, err := os.Open(filepath.Join(dir, "texture-7.png"))
	if err != nil {
		return err
	}
	tex, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	img := image.NewNRGBA(image.Rect(0, 0, 240, 45))
	for i := 0; i < 240*45; i++ {
		if mask[i/8]&(1<<uint(i%8)) != 0 {
			img.Set(i%240, i/240, tex.At(i%240, i/240))
		}
	}
	f, err = os.Create(filepath.Join(dir, "title.png"))
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
