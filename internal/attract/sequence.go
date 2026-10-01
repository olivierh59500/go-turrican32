// Package attract replays a verified gamepad recording through ordinary gameplay.
package attract

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/olivierh59500/go-turrican32/assets"
	"github.com/olivierh59500/go-turrican32/internal/engine"
)

type Sequence struct{ frames []byte }

func Load() (*Sequence, error) {
	frames, err := assets.Files.ReadFile("demo-inputs.bin")
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("attract: empty gamepad recording")
	}
	info, err := assets.Files.ReadFile("demo-info.json")
	if err != nil {
		return nil, err
	}
	var metadata struct {
		Frames int    `json:"frames"`
		FPS    int    `json:"fps"`
		Level  string `json:"level_sha256"`
	}
	if err := json.Unmarshal(info, &metadata); err != nil {
		return nil, err
	}
	level, err := assets.Files.ReadFile("level.bin")
	if err != nil {
		return nil, err
	}
	if metadata.Frames != len(frames) || metadata.FPS != engine.FPS || metadata.Level != fmt.Sprintf("%x", sha256.Sum256(level)) {
		return nil, fmt.Errorf("attract: recording does not match the level or tick rate")
	}
	for _, input := range frames {
		if input & ^byte(engine.Left|engine.Right|engine.Jump|engine.Fire) != 0 {
			return nil, fmt.Errorf("attract: invalid gamepad input %d", input)
		}
	}
	return &Sequence{frames: frames}, nil
}

func (s *Sequence) Len() int { return len(s.frames) }

func (s *Sequence) Input(frame int) (byte, bool) {
	if frame < 0 || frame >= len(s.frames) {
		return 0, false
	}
	return s.frames[frame], true
}
