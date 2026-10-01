// Package mobile attaches Turrican32 to Android's Ebitengine view.
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/olivierh59500/go-turrican32/internal/game"
	"log"
	"time"
)

type host struct {
	game     *game.Game
	requests chan int
	verify   bool
	report   time.Time
}

var h = &host{requests: make(chan int, 1)}

func init() { ebiten.SetTPS(game.FPS); enginemobile.SetGame(h) }
func (h *host) Update() error {
	if h.game == nil {
		select {
		case <-h.requests:
			h.verify = true
		default:
		}
		var err error
		h.game, err = game.New(game.Config{Mobile: true})
		if err != nil {
			return err
		}
	}
	if err := h.game.Update(); err != nil {
		return err
	}
	if h.verify && time.Since(h.report) >= 10*time.Second {
		log.Printf("turrican32_verify tick=%d tps=%.1f fps=%.1f %s", h.game.Tick(), ebiten.ActualTPS(), ebiten.ActualFPS(), h.game.VerificationState())
		h.report = time.Now()
	}
	return nil
}
func (h *host) Draw(dst *ebiten.Image) {
	if h.game != nil {
		h.game.Draw(dst)
	}
}
func (h *host) Layout(w, v int) (int, int) {
	if h.game != nil {
		return h.game.Layout(w, v)
	}
	return 960, game.Height
}

// ConfigureVerification enables cadence logging while retaining ordinary playback.
func ConfigureVerification(tick int) bool {
	select {
	case h.requests <- tick:
		return true
	default:
		return false
	}
}
func Dummy() {}
