package extract

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

func exportVariants(exe []byte, dir string, meta *Metadata) error {
	definitions := map[int]Definition{}
	frames := map[int][]Frame{}
	for _, d := range meta.Definitions {
		definitions[d.ID] = d
	}
	for _, f := range meta.Frames {
		frames[f.Type] = append(frames[f.Type], f)
	}
	type variant struct {
		id, source, x, y int
		mirror           bool
	}
	var variants []variant
	for p := 0xdcac; p < 0xdce4; p += 8 {
		variants = append(variants, variant{id: int(binary.LittleEndian.Uint32(exe[p+4:])), source: int(binary.LittleEndian.Uint32(exe[p:])), x: 1, y: 1, mirror: true})
	}
	for p := 0xdce4; p < 0xdd84; p += 16 {
		variants = append(variants, variant{id: int(binary.LittleEndian.Uint32(exe[p:])), source: int(binary.LittleEndian.Uint32(exe[p+4:])), x: int(binary.LittleEndian.Uint32(exe[p+8:])), y: int(binary.LittleEndian.Uint32(exe[p+12:]))})
	}
	for _, v := range variants {
		d := definitions[v.source]
		d.ID = v.id
		d.Width *= v.x
		d.Height *= v.y
		if v.id == 1011 || v.id == 1041 {
			d.SpeedX = -d.SpeedX
			d.Target = v.source
		}
		meta.Definitions = append(meta.Definitions, d)
		for _, original := range frames[v.source] {
			f, e := os.Open(filepath.Join(dir, original.File))
			if e != nil {
				return e
			}
			im, e := png.Decode(f)
			f.Close()
			if e != nil {
				return e
			}
			dst := image.NewNRGBA(image.Rect(0, 0, d.Width, d.Height))
			for y := 0; y < d.Height; y++ {
				for x := 0; x < d.Width; x++ {
					sx, sy := x%original.Width, y%original.Height
					if v.mirror {
						sx = original.Width - 1 - sx
					}
					dst.Set(x, y, im.At(sx, sy))
				}
			}
			name := fmt.Sprintf("sprites/%04d-%03d.png", v.id, original.Index)
			f, e = os.Create(filepath.Join(dir, name))
			if e != nil {
				return e
			}
			e = png.Encode(f, dst)
			f.Close()
			if e != nil {
				return e
			}
			meta.Frames = append(meta.Frames, Frame{v.id, original.Index, d.Width, d.Height, name})
		}
	}
	for id := 7; id <= 9; id++ {
		meta.Definitions = append(meta.Definitions, Definition{ID: id, Width: 256, Height: 128, Passable: true, First: 0, Last: 0, Animation: 1, Rate: 0, Declared: true, Target: id})
		meta.Frames = append(meta.Frames, Frame{id, 0, 256, 128, fmt.Sprintf("texture-%d.png", id)})
	}
	return nil
}
