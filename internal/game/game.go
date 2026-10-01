// Package game composes the native level with DCK sprite and background layers.
package game

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/sound"
	music "github.com/olivierh59500/democonstructionkit/sound/ebiten"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/go-turrican32/assets"
	"github.com/olivierh59500/go-turrican32/internal/attract"
	"github.com/olivierh59500/go-turrican32/internal/controls"
	"github.com/olivierh59500/go-turrican32/internal/data"
	"github.com/olivierh59500/go-turrican32/internal/engine"
	"image"
	"image/color"
	_ "image/png"
	"math"
)

const Width, Height, FPS = 320, 240, 60

type Config struct{ Mute, Mobile, Recording, Demo bool }
type Game struct {
	core                       *engine.Engine
	config                     Config
	images                     []*ebiten.Image
	frame                      map[[2]int]int
	fontIndex, backgroundIndex int
	batch                      *sprites.ImageSlots
	background                 *composite.Background
	canvas                     *ebiten.Image
	slots                      []sprites.ImageSlot
	title                      bool
	tick, uiWidth              int
	paused, muted, closed      bool
	player                     *music.Player
	joystick                   controls.Joystick
	contacts                   []controls.Touch
	ids                        []ebiten.TouchID
	demo                       *attract.Sequence
	demoActive, autoDemo       bool
	demoFrame, titleAge        int
}

func New(config Config) (_ *Game, err error) {
	g := &Game{config: config, title: true, uiWidth: Width, frame: map[[2]int]int{}, muted: config.Mute}
	defer func() {
		if err != nil {
			g.Close()
		}
	}()
	d, err := data.Load()
	if err != nil {
		return nil, err
	}
	g.core = engine.New(d)
	g.demo, err = attract.Load()
	if err != nil {
		return nil, err
	}
	g.autoDemo = config.Demo || config.Recording
	load := func(name string) (*ebiten.Image, error) {
		b, e := assets.Files.ReadFile(name)
		if e != nil {
			return nil, e
		}
		img, _, e := image.Decode(bytes.NewReader(b))
		if e != nil {
			return nil, e
		}
		texture := ebiten.NewImageFromImage(img)
		g.images = append(g.images, texture)
		return texture, nil
	}
	for _, f := range d.Frames {
		g.frame[[2]int{f.Type, f.Index}] = len(g.images)
		if _, err = load(f.File); err != nil {
			return nil, err
		}
	}
	g.backgroundIndex = g.frame[[2]int{9, 0}]
	g.fontIndex = len(g.images)
	if _, err = load("font.png"); err != nil {
		return nil, err
	}
	if _, err = load("title.png"); err != nil {
		return nil, err
	}
	g.batch, err = sprites.NewImageSlots(sprites.ImageSlotsConfig{Images: g.images, MaxSlots: 2048, Filter: ebiten.FilterNearest})
	if err != nil {
		return nil, err
	}
	g.background, err = composite.NewBackground(composite.BackgroundConfig{PeriodX: 256, PeriodY: 128, Filter: ebiten.FilterNearest})
	if err != nil {
		return nil, err
	}
	g.canvas = ebiten.NewImage(Width, Height)
	g.slots = make([]sprites.ImageSlot, 0, 1400)
	if !config.Mute {
		if err = g.playTrack("title.ym"); err != nil {
			return nil, err
		}
	}
	return g, nil
}
func (g *Game) playTrack(name string) error {
	if g.player != nil {
		g.player.Close()
		g.player = nil
	}
	if g.config.Mute {
		return nil
	}
	b, e := assets.Files.ReadFile(name)
	if e != nil {
		return e
	}
	g.player, e = music.Open(nil, name, b, sound.Options{SampleRate: 48000, Loop: true})
	if e == nil {
		g.player.SetVolume(.65)
		if !g.muted {
			g.player.Play()
		}
	}
	return e
}
func (g *Game) label(text string, x, y int) {
	for _, c := range text {
		if c >= 32 && c < 112 && c != ' ' {
			index := int(c) - 32
			g.slots = append(g.slots, sprites.ImageSlot{Image: g.fontIndex, Source: image.Rect(index%16*8, index/16*8, index%16*8+8, index/16*8+8), X: float64(x), Y: float64(y)})
		}
		x += 8
	}
}
func (g *Game) input() (byte, string) {
	var mask byte
	action := ""
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		mask |= engine.Left
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		mask |= engine.Right
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeySpace) {
		mask |= engine.Jump
	}
	if ebiten.IsKeyPressed(ebiten.KeyControlRight) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) {
		mask |= engine.Fire
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		action = "play"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		action = "demo"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		action = "reset"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		action = "pause"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		action = "mute"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.title {
			action = "quit"
		} else {
			action = "reset"
		}
	}
	if g.config.Mobile {
		g.ids = ebiten.AppendTouchIDs(g.ids[:0])
		g.contacts = g.contacts[:0]
		for _, id := range g.ids {
			x, y := ebiten.TouchPosition(id)
			g.contacts = append(g.contacts, controls.Touch{ID: int(id), X: float64(x), Y: float64(y), Pressed: inpututil.TouchPressDuration(id) == 1})
		}
		g.joystick.Place(45, 180, 35)
		g.joystick.Update(g.contacts)
		if g.joystick.X < 0 {
			mask |= engine.Left
		}
		if g.joystick.X > 0 {
			mask |= engine.Right
		}
		if g.joystick.Y < 0 {
			mask |= engine.Jump
		}
		for _, t := range g.contacts {
			if g.joystick.Owns(t.ID) {
				continue
			}
			if math.Hypot(t.X-float64(g.uiWidth-43), t.Y-180) < 35 {
				mask |= engine.Fire
			}
			if t.Pressed && t.Y < 45 {
				if t.X < 90 {
					action = "reset"
				} else if t.X > float64(g.uiWidth-90) {
					action = "pause"
				} else if g.title {
					action = "play"
				}
			}
			if t.Pressed && t.X < 90 && t.Y >= 45 && t.Y < 85 {
				action = "demo"
			}
		}
	}
	return mask, action
}
func (g *Game) Update() error {
	mask, action := g.input()
	if g.config.Recording {
		mask, action = 0, ""
	}
	if g.title && !g.paused {
		g.titleAge++
		if (g.autoDemo && g.titleAge >= FPS*6) || g.titleAge >= FPS*15 {
			action = "demo"
		}
	}
	switch action {
	case "quit":
		return ebiten.Termination
	case "reset":
		g.joystick.Reset()
		if err := g.showTitle(false); err != nil {
			return err
		}
	case "demo":
		if g.demoActive {
			g.demoActive, g.autoDemo = false, false
		} else {
			if err := g.Start(); err != nil {
				return err
			}
			g.demoActive, g.autoDemo = true, true
			g.demoFrame = 0
		}
	case "pause":
		g.paused = !g.paused
		if g.player != nil {
			if g.paused || g.muted {
				g.player.Pause()
			} else {
				g.player.Play()
			}
		}
	case "mute":
		g.muted = !g.muted
		if g.player != nil {
			if g.muted || g.paused {
				g.player.Pause()
			} else {
				g.player.Play()
			}
		}
	}
	if g.title && (action == "play" || mask&engine.Fire != 0) {
		if err := g.Start(); err != nil {
			return err
		}
	}
	if g.paused {
		return nil
	}
	if g.demoActive {
		manual := mask != 0 || (g.config.Mobile && g.joystick.Active())
		if !g.config.Recording && manual {
			g.demoActive, g.autoDemo = false, false
		} else {
			input, ok := g.demo.Input(g.demoFrame)
			if !ok || g.core.GameOver || g.core.Won {
				return g.showTitle(true)
			}
			mask = input
			g.demoFrame++
		}
	}
	g.tick++
	if !g.title {
		g.core.Step(mask)
	}
	return nil
}
func (g *Game) Draw(dst *ebiten.Image) {
	g.canvas.Fill(color.RGBA{A: 255})
	g.slots = g.slots[:0]
	overlayAt := -1
	g.background.Draw(g.canvas, g.images[g.backgroundIndex], composite.BackgroundPose{X: -float64(g.core.CameraX) / 4, Y: -float64(g.core.CameraY) / 4})
	if g.title {
		g.slots = append(g.slots, sprites.ImageSlot{Image: len(g.images) - 1, X: 40, Y: 15})
		g.label("PRESENTED BY", 112, 102)
		g.label("MYTH, TMBINC, ARTHUS,", 80, 122)
		g.label("KB, RYG AND KOJOTE", 88, 134)
		g.label("INSERT COIN TO START", 88, 168)
		g.label("F1: DEMO", 128, 182)
		frame := int(float64(g.tick)*.2) % 6
		g.slots = append(g.slots, sprites.ImageSlot{Image: g.frame[[2]int{1230, frame}], X: float64(140 + int(35*math.Sin(float64(g.tick)/60))), Y: 191})
	} else {
		for _, e := range g.core.Entities {
			if e.Deleted {
				continue
			}
			index, ok := g.frame[[2]int{e.Type, int(e.Frame)}]
			if !ok {
				index, ok = g.frame[[2]int{e.Type, g.core.Data.Definitions[e.Type].First}]
			}
			if !ok {
				continue
			}
			d := g.core.Data.Definitions[e.Type]
			x, y := int(e.X)-g.core.CameraX, int(e.Y)-g.core.CameraY
			if x+d.Width < 0 || y+d.Height < 0 || x >= Width || y >= Height {
				continue
			}
			opacity := 1.0
			if e.Faded {
				opacity = .5
			}
			var tint ebiten.ColorScale
			tint.ScaleAlpha(float32(opacity))
			g.slots = append(g.slots, sprites.ImageSlot{Image: index, X: float64(x), Y: float64(y), Tint: tint})
		}
		overlayAt = len(g.slots)
		g.label(fmt.Sprintf("SCORE: %05d LIVES:%d BONUS:%d", g.core.Score, g.core.Lives, g.core.Bonus), 8, 3)
		if g.demoActive {
			g.label("DEMO - MOVE TO PLAY", 88, 228)
		}
		if g.core.GameOver {
			g.label("GAME OVER", 124, 108)
			g.label("PRESS R TO RESTART", 92, 126)
		}
		if g.core.Won {
			g.label("WELL DONE!", 120, 108)
			g.label("PRESS R TO RESTART", 92, 126)
		}
	}
	worldSlots := g.slots
	if overlayAt >= 0 {
		worldSlots = g.slots[:overlayAt]
	}
	if err := g.batch.SetSlots(worldSlots); err != nil {
		panic(err)
	}
	g.batch.Draw(g.canvas)
	if overlayAt >= 0 {
		vector.FillRect(g.canvas, 0, 0, Width, 15, color.RGBA{A: 200}, false)
		if g.demoActive {
			vector.FillRect(g.canvas, 84, 225, 156, 14, color.RGBA{A: 180}, false)
		}
		if err := g.batch.SetSlots(g.slots[overlayAt:]); err != nil {
			panic(err)
		}
		g.batch.Draw(g.canvas)
	}
	dst.Fill(color.RGBA{A: 255})
	offset := (g.uiWidth - Width) / 2
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(offset), 0)
	dst.DrawImage(g.canvas, &op)
	if g.config.Mobile {
		vector.FillCircle(dst, 45, 180, 35, color.RGBA{25, 36, 62, 230}, false)
		vector.StrokeCircle(dst, 45, 180, 35, 2, color.RGBA{115, 180, 240, 255}, false)
		vector.FillCircle(dst, float32(45+g.joystick.OffsetX), float32(180+g.joystick.OffsetY), 14, color.RGBA{90, 148, 226, 255}, false)
		vector.FillCircle(dst, float32(g.uiWidth-43), 180, 32, color.RGBA{150, 60, 58, 235}, false)
		ebitenutil.DebugPrintAt(dst, "FIRE", g.uiWidth-59, 177)
		ebitenutil.DebugPrintAt(dst, "RESET", 20, 15)
		demoLabel := "DEMO"
		if g.demoActive {
			demoLabel = "PLAY"
		}
		ebitenutil.DebugPrintAt(dst, demoLabel, 20, 58)
		ebitenutil.DebugPrintAt(dst, "PAUSE", g.uiWidth-63, 15)
	}
}
func (g *Game) Layout(width, height int) (int, int) {
	g.uiWidth = Width
	if g.config.Mobile && height > 0 {
		g.uiWidth = max(Width+180, int(float64(Height)*float64(width)/float64(height)))
	}
	return g.uiWidth, Height
}
func (g *Game) Close() {
	if g == nil || g.closed {
		return
	}
	g.closed = true
	if g.player != nil {
		g.player.Close()
	}
	if g.batch != nil {
		g.batch.Close()
	}
	for _, im := range g.images {
		im.Deallocate()
	}
	if g.canvas != nil {
		g.canvas.Deallocate()
	}
}

// Start enters the original level from its initial player position.
func (g *Game) Start() error {
	g.title = false
	g.paused = false
	g.demoActive, g.autoDemo = false, false
	g.core.Reset()
	return g.playTrack("game.ym")
}

func (g *Game) showTitle(auto bool) error {
	g.core.Reset()
	g.title, g.paused = true, false
	g.demoActive, g.autoDemo = false, auto
	g.demoFrame, g.titleAge = 0, 0
	return g.playTrack("title.ym")
}

func (g *Game) Tick() int { return g.tick }
func (g *Game) VerificationState() string {
	p := g.core.Entities[g.core.Player]
	return fmt.Sprintf("player=(%.1f,%.1f) health=%d entities=%d demo=%v frame=%d", p.X, p.Y, g.core.Health, len(g.core.Entities), g.demoActive, g.demoFrame)
}
