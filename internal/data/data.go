// Package data loads the original definitions, geometry and decoded artwork.
package data

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/olivierh59500/go-turrican32/assets"
	"github.com/olivierh59500/go-turrican32/internal/extract"
)

type Object struct {
	Type  int
	X, Y  float64
	Faded bool
}
type Data struct {
	Definitions map[int]extract.Definition
	Frames      []extract.Frame
	Objects     []Object
}

func Load() (*Data, error) {
	b, e := assets.Files.ReadFile("graphics.json")
	if e != nil {
		return nil, e
	}
	var meta extract.Metadata
	if e = json.Unmarshal(b, &meta); e != nil {
		return nil, e
	}
	d := &Data{Definitions: map[int]extract.Definition{}, Frames: meta.Frames}
	for _, v := range meta.Definitions {
		d.Definitions[v.ID] = v
	}
	b, e = assets.Files.ReadFile("level.bin")
	if e != nil {
		return nil, e
	}
	if len(b) != 0x1260 {
		return nil, fmt.Errorf("data: incomplete level")
	}
	for i := 6; i < len(b); i += 6 {
		kind := int(binary.LittleEndian.Uint16(b[i:]))
		id := kind & 32767
		if _, ok := d.Definitions[id]; !ok {
			return nil, fmt.Errorf("data: missing type %d", id)
		}
		d.Objects = append(d.Objects, Object{id, float64(int16(binary.LittleEndian.Uint16(b[i+2:]))), float64(int16(binary.LittleEndian.Uint16(b[i+4:]))), kind > 32768})
	}
	return d, nil
}
