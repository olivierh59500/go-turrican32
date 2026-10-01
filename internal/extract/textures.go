package extract

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func exportTextures(exe []byte, dir string) error {
	for n, at := range []int{0x40da88, 0x40db14, 0x40dba0} {
		id := []int{9, 8, 7}[n]
		data := exe[at-0x400000:]
		count := int(binary.LittleEndian.Uint32(data[4:]))
		pixels := make([]byte, 256*256*4)
		for layer := 0; layer < count; layer++ {
			var args [7]int
			for i := range args {
				args[i] = int(int32(binary.LittleEndian.Uint32(data[8+(layer*7+i)*4:])))
			}
			seed := uint32(args[0]) * 0x1010101
			noise := make([]byte, 0x1000c0)
			for i := range noise {
				v := seed ^ 0xd87172
				seed = (seed<<11 | seed>>21) ^ 0x25f7b8 ^ (v>>13 | v<<20) ^ 0x1de2b57
				seed = (seed<<23 | seed>>10) ^ (v>>5 | v<<28)
				noise[i] = byte(seed)
			}
			var smooth [256]int
			for i := range smooth {
				smooth[i] = 255 - int(math.Cos(float64(i)*math.Float64frombits(binary.LittleEndian.Uint64(exe[0x9148:])))*255)
			}
			sample := func(x, y, size int) int { return int(noise[(x&(size-1))*size+(y&(size-1))]) }
			interpolated := func(x, y, size int) int {
				ix, iy := x>>8, y>>8
				a, b, c, d := sample(ix, iy, size), sample(ix+1, iy, size), sample(ix, iy+1, size), sample(ix+1, iy+1, size)
				top := ((b - a) * smooth[x&255] >> 8) + a
				return ((((d - c) * smooth[x&255] >> 8) + c - top) * smooth[y&255] >> 8) + top
			}
			level := make([]byte, 256*256)
			for y := 0; y < 256; y++ {
				for x := 0; x < 256; x++ {
					sum := 0
					for octave := 2; octave <= 10; octave++ {
						weight := max(0, 9-octave)
						sum += interpolated(x<<uint(octave), y<<uint(octave), 1<<uint(octave)) << uint(weight)
					}
					level[y*256+x] = byte(sum >> 8)
				}
			}
			local := make([]byte, len(pixels))
			for i, v := range level {
				for c := 0; c < 3; c++ {
					local[i*4+c] = byte(int(v) * args[1+c] >> 8)
				}
			}
			remap := func(dst []byte, contrast, bias int, mirror bool) {
				for i := 0; i < len(dst); i += 4 {
					for c := 0; c < 3; c++ {
						v := ((-128*contrast + int(dst[i+c])*contrast) >> 6) + 128 + bias
						v = max(0, min(255, v))
						if mirror && v > 127 {
							v ^= 255
						}
						dst[i+c] = byte(v)
					}
				}
			}
			remap(local, args[4], args[5], args[6]&1 != 0)
			for i := range pixels {
				switch {
				case args[6]&2 != 0:
					pixels[i] = byte(min(255, int(pixels[i])+int(local[i])))
				case args[6]&4 != 0:
					pixels[i] = byte(int(pixels[i]) * int(local[i]) >> 8)
				case args[6]&8 != 0:
					pixels[i] ^= local[i]
				}
			}
		}
		after := data[8+count*28:]
		var post [5]int
		for i := range post {
			post[i] = int(int32(binary.LittleEndian.Uint32(after[i*4:])))
		}
		remapAll := func(contrast, bias int, mirror bool) {
			for i := 0; i < len(pixels); i += 4 {
				for c := 0; c < 3; c++ {
					v := ((-128*contrast + int(pixels[i+c])*contrast) >> 6) + 128 + bias
					v = max(0, min(255, v))
					if mirror && v > 127 {
						v ^= 255
					}
					pixels[i+c] = byte(v)
				}
			}
		}
		remapAll(post[0], post[1], post[2] != 0)
		remapAll(post[3], post[4], false)
		img := image.NewNRGBA(image.Rect(0, 0, 256, 128))
		for y := 0; y < 128; y++ {
			for x := 0; x < 256; x++ {
				p := (y*512 + x) * 4
				r, g, b := pixels[p]>>3, (pixels[p+1]>>2)&62, pixels[p+2]>>3
				img.SetNRGBA(x, y, color.NRGBA{byte(uint32(r) * 255 / 31), byte(uint32(g) * 255 / 63), byte(uint32(b) * 255 / 31), 255})
			}
		}
		file, e := os.Create(filepath.Join(dir, fmt.Sprintf("texture-%d.png", id)))
		if e != nil {
			return e
		}
		e = png.Encode(file, img)
		closeErr := file.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
