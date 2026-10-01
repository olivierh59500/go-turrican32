package extract

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"math"
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
		// The texture converter copies this native 128-byte definition template.
		template := exe[0xdc2c:0xdcac]
		integer := func(offset int) int { return int(binary.LittleEndian.Uint32(template[offset:])) }
		def := Definition{ID: id, Width: integer(76), Height: integer(80), Passable: template[64] != 0, Transparent: template[24] == 0, Category: integer(72), First: integer(4), Last: integer(8), Animation: integer(0), Rate: math.Float64frombits(binary.LittleEndian.Uint64(template[16:])), Margin: integer(84), Declared: template[124] != 0, Target: integer(116)}
		if id == 8 {
			def.Last = 127
			def.Rate = math.Float64frombits(binary.LittleEndian.Uint64(exe[0x91c8:]))
			if err := exportWaveFrames(exe, dir, meta); err != nil {
				return err
			}
		} else {
			meta.Frames = append(meta.Frames, Frame{id, 0, def.Width, def.Height, fmt.Sprintf("texture-%d.png", id)})
		}
		meta.Definitions = append(meta.Definitions, def)
	}
	return nil
}

// Bake the original row shifts once; playback uses ordinary DCK image slots.
func exportWaveFrames(exe []byte, dir string, meta *Metadata) error {
	f, err := os.Open(filepath.Join(dir, "texture-8.png"))
	if err != nil {
		return err
	}
	base, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	width, height := base.Bounds().Dx(), base.Bounds().Dy()
	angle := math.Float64frombits(binary.LittleEndian.Uint64(exe[0x9210:]))
	amplitude := math.Float64frombits(binary.LittleEndian.Uint64(exe[0x9118:]))
	for frame := range 128 {
		img := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := range height {
			shift := int((math.Cos(float64(y+frame)*angle)+1)*amplitude+float64(width)) % width
			for x := range width {
				// The native source is two complete copies of the texture, so
				// crossing a row reads the next row before wrapping the image.
				source := (y*width + shift + x) % (width * height)
				img.Set(x, y, base.At(source%width, source/width))
			}
		}
		name := fmt.Sprintf("sprites/0008-%03d.png", frame)
		out, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		err = png.Encode(out, img)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		meta.Frames = append(meta.Frames, Frame{8, frame, width, height, name})
	}
	return nil
}
