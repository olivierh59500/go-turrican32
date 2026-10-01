// Command video records Turrican32's gameplay presentation and YM soundtrack.
package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
	"github.com/olivierh59500/go-turrican32/internal/game"
	"log"
	"time"
)

func main() {
	c := video.Config{Output: "recordings/turrican32.mp4", Title: "Turrican32 Go", Width: game.Width * 2, Height: game.Height * 2, FPS: game.FPS, TPS: game.FPS, SampleRate: 48000, Duration: 3 * time.Minute, PosterAt: 15 * time.Second}
	c.Flags(flag.CommandLine)
	flag.Parse()
	if c.Duration <= 0 {
		log.Fatal("recording duration must be positive")
	}
	if err := video.Run(c, func() (ebiten.Game, error) { return game.New(game.Config{Recording: true}) }); err != nil {
		log.Fatal(err)
	}
}
