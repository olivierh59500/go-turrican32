// Command turrican32 runs the native Go/Ebitengine game.
package main

import (
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
	"github.com/olivierh59500/go-turrican32/internal/game"
	"log"
	"os"
)

func main() {
	mute := flag.Bool("mute", false, "disable music")
	directory := flag.String("capture", "", "capture directory")
	frame := flag.Int("frame", 300, "capture tick")
	playing := flag.Bool("play", false, "start in the level")
	flag.Parse()
	if *directory != "" {
		var g *game.Game
		err := capture.Run(capture.Config{Directory: *directory, Frames: []int{*frame}, Width: game.Width, Height: game.Height}, func() (ebiten.Game, error) {
			var e error
			g, e = game.New(game.Config{Mute: true})
			if *playing && g != nil {
				g.Start()
			}
			return g, e
		})
		if g != nil {
			g.Close()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	g, err := game.New(game.Config{Mute: *mute})
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()
	if *playing {
		g.Start()
	}
	ebiten.SetTPS(60)
	ebiten.SetWindowSize(960, 720)
	ebiten.SetWindowTitle("Turrican32 Go / Mekka & Symposium 2000")
	if err = ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
