// Package engine translates T32k's native 60 Hz platform and entity logic.
package engine

import (
	"github.com/olivierh59500/go-turrican32/internal/data"
	"github.com/olivierh59500/go-turrican32/internal/extract"
	"math"
)

const FPS = 60
const (
	Left byte = 1 << iota
	Right
	Jump
	Fire
)

type Entity struct {
	Type                int
	X, Y, Frame, DX, DY float64
	Faded               bool
	Direction, Life     int
	Deleted             bool
}
type Engine struct {
	Data                                               *data.Data
	Entities                                           []Entity
	Player                                             int
	XMin, YMin, XMax, YMax                             float64
	CameraX, CameraY                                   int
	VX, VY                                             float64
	Score, Bonus, Power, Health, Lives, Tick, DeathAge int
	CheckX, CheckY                                     float64
	Won, GameOver                                      bool
	grounded, jumpReady, fireReady                     bool
	state                                              int
	seed                                               uint32
}

func New(d *data.Data) *Engine { e := &Engine{Data: d}; e.Reset(); return e }
func (e *Engine) Reset() {
	e.Entities = e.Entities[:0]
	e.Score, e.Bonus, e.Health, e.Lives, e.Tick, e.DeathAge = 0, 0, 100, 3, 0, 0
	e.Power = 0
	e.Won, e.GameOver = false, false
	e.VX, e.VY = 0, 0
	e.jumpReady, e.fireReady = true, true
	e.seed = 1
	e.state = 1210
	e.Player = -1
	e.XMin, e.YMin = 1e5, 1e5
	e.XMax, e.YMax = -1e5, -1e5
	for _, o := range e.Data.Objects {
		d := e.def(o.Type)
		entity := Entity{Type: o.Type, X: o.X, Y: o.Y, Frame: float64(d.First), DX: d.SpeedX, DY: d.SpeedY, Direction: 1, Faded: o.Faded}
		e.Entities = append(e.Entities, entity)
		if o.Type == 1210 {
			e.Player = len(e.Entities) - 1
			e.CheckX, e.CheckY = o.X, o.Y
		}
		e.XMin = math.Min(e.XMin, o.X)
		e.YMin = math.Min(e.YMin, o.Y)
		e.XMax = math.Max(e.XMax, o.X+float64(d.Width))
		e.YMax = math.Max(e.YMax, o.Y+float64(d.Height))
	}
	e.XMax -= 320
	e.YMax -= 248
	if e.Player >= 0 {
		p := e.Entities[e.Player]
		e.CameraX = int(p.X) - 160
		e.CameraY = int(p.Y) - 120
	}
}
func (e *Engine) def(id int) extract.Definition {
	if d, ok := e.Data.Definitions[id]; ok {
		return d
	}
	return e.Data.Definitions[1500]
}
func (e *Engine) random() int { e.seed = e.seed*214013 + 2531011; return int(e.seed >> 16 & 32767) }
func (e *Engine) bounds(p Entity) (l, t, r, b int) {
	d := e.def(p.Type)
	x, y := int(p.X), int(p.Y)
	return x + d.Margin, y, x + d.Width - d.Margin, y + d.Height
}
func overlap(al, at, ar, ab, bl, bt, br, bb int) bool {
	return al <= br && ar >= bl && at <= bb && ab >= bt
}
func (e *Engine) solid(p Entity, vertical int) int {
	l, t, r, b := e.bounds(p)
	for i, q := range e.Entities {
		if i == e.Player || q.Deleted || e.def(q.Type).Passable {
			continue
		}
		d := e.def(q.Type)
		bl, bt, br, bb := int(q.X), int(q.Y), int(q.X)+d.Width, int(q.Y)+d.Height
		if abs(bl-l) >= 160 || abs(bt-t) >= 120 {
			continue
		}
		if vertical == 1 {
			if b+1 == bt && l <= br && r >= bl {
				return i
			}
		} else if vertical == -1 {
			if t-1 == bb && l <= br && r >= bl {
				return i
			}
		} else if overlap(l, t, r, b, bl, bt, br, bb) {
			return i
		}
	}
	return -1
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func (e *Engine) setPlayerType(id int) {
	if e.state != id {
		e.state = id
		e.Entities[e.Player].Type = id
		e.Entities[e.Player].Frame = float64(e.def(id).First)
		e.Entities[e.Player].Direction = 1
	}
}
func (e *Engine) spawn(id int, x, y, dx, dy float64) {
	d := e.def(id)
	e.Entities = append(e.Entities, Entity{Type: id, X: x, Y: y, DX: dx, DY: dy, Frame: float64(d.First), Direction: 1, Life: 120})
}
func (e *Engine) Step(input byte) {
	if e.Player < 0 || e.GameOver || e.Won {
		return
	}
	e.Tick++
	if e.DeathAge > 0 {
		e.DeathAge++
		e.animate()
		if e.DeathAge > 80 {
			e.DeathAge = 0
			e.Health = 100
			e.Lives--
			if e.Lives <= 0 {
				e.GameOver = true
				return
			}
			p := &e.Entities[e.Player]
			p.X, p.Y = e.CheckX, e.CheckY
			e.VX, e.VY = 0, 0
			if e.Power > 1 {
				e.Power--
			}
		}
		return
	}
	e.animate()
	e.moveEntities()
	p := e.Entities[e.Player]
	test := p
	e.grounded = e.solid(test, 1) != -1
	if !e.grounded {
		e.VY = math.Min(6, e.VY+.2)
	}
	if e.VY > 0 {
		for n := 0; n < int(e.VY); n++ {
			p.Y++
			if e.solid(p, 1) >= 0 {
				e.VY = 0
				e.grounded = true
				break
			}
		}
	}
	if e.VY < 0 {
		for n := 0; n < int(-e.VY); n++ {
			p.Y--
			if e.solid(p, -1) >= 0 {
				e.VY = -e.VY * .25
				e.grounded = true
				break
			}
		}
	}
	if e.VX > 0 {
		if e.grounded {
			e.setPlayerType(1230)
		}
		for n := 0; n < int(e.VX); n++ {
			p.X++
			if e.solid(p, 0) >= 0 {
				p.X--
				e.VX = 0
				break
			}
		}
	}
	if e.VX < 0 {
		if e.grounded {
			e.setPlayerType(1231)
		}
		for n := 0; n < int(-e.VX); n++ {
			p.X--
			if e.solid(p, 0) >= 0 {
				p.X++
				e.VX = 0
				break
			}
		}
	}
	e.Entities[e.Player].X, e.Entities[e.Player].Y = p.X, p.Y
	e.CameraX += (int(p.X - 160 - float64(e.CameraX))) >> 4
	e.CameraY += (int(p.Y - 120 - float64(e.CameraY))) >> 4
	e.CameraX = max(int(e.XMin), min(int(e.XMax), e.CameraX))
	e.CameraY = max(int(e.YMin), min(int(e.YMax), e.CameraY))
	if e.VY < 0 && input&Jump == 0 {
		e.VY *= 5.0 / 6
	}
	if math.Abs(e.VX) < 1 && math.Abs(e.VY) < 1 && e.grounded {
		if e.state == 1210 || e.state == 1230 || e.state == 1200 {
			e.setPlayerType(1200)
		} else {
			e.setPlayerType(1201)
		}
	}
	if math.Abs(e.VY) >= 1 && e.state != 1210 && e.state != 1211 {
		if e.state == 1200 || e.state == 1230 {
			e.setPlayerType(1210)
		} else {
			e.setPlayerType(1211)
		}
	}
	if input&Jump == 0 {
		e.jumpReady = true
	} else if e.grounded && e.jumpReady {
		e.jumpReady = false
		e.VY = -6
	}
	if input&Left != 0 {
		e.VX = math.Max(-2.5, e.VX-.4)
		if !e.grounded {
			e.setPlayerType(1211)
		}
	} else if input&Right != 0 {
		e.VX = math.Min(2.5, e.VX+.4)
		if !e.grounded {
			e.setPlayerType(1210)
		}
	} else {
		e.VX *= 5.0 / 6
	}
	if input&Fire == 0 {
		e.fireReady = true
	} else if e.fireReady {
		e.fireReady = false
		direction := 4.0
		if e.state == 1201 || e.state == 1211 || e.state == 1231 {
			direction = -4
		}
		id := 1300
		if direction < 0 {
			id = 1301
		}
		e.spawn(id, p.X+12+direction*6, p.Y+15, direction, 0)
		if e.Power > 0 {
			e.spawn(id, p.X+12+direction*6, p.Y+11, direction, 0)
			e.spawn(id, p.X+12+direction*6, p.Y+19, direction, 0)
		}
		if e.Power > 1 {
			e.spawn(id, p.X+12+direction*6, p.Y+7, direction, 0)
			e.spawn(id, p.X+12+direction*6, p.Y+23, direction, 0)
		}
	}
	e.pickups()
	if p.X > e.XMax+320 || p.Y > e.YMax+240 {
		e.Won = true
	}
	if e.Health < 0 {
		e.DeathAge = 1
		e.VX, e.VY = 0, 0
	}
}
func (e *Engine) animate() {
	for i := range e.Entities {
		p := &e.Entities[i]
		if p.Deleted {
			continue
		}
		d := e.def(p.Type)
		switch d.Animation {
		case 2:
			p.Frame += d.Rate
			if p.Frame > float64(d.Last) {
				p.Deleted = true
			}
		case 1:
			p.Frame += d.Rate
			if p.Frame > float64(d.Last) {
				p.Frame -= float64(d.Last - d.First)
			}
		case 0:
			if p.Direction < 0 {
				p.Frame -= d.Rate
				if p.Frame-d.Rate < float64(d.First) {
					p.Direction = 0
				}
			} else {
				p.Frame += d.Rate
				if p.Frame+d.Rate >= float64(d.Last+1) {
					p.Direction = -1
				}
			}
		}
	}
}
func (e *Engine) visible(p Entity) bool {
	return p.X > float64(e.CameraX-270) && p.X < float64(e.CameraX+320) && p.Y > float64(e.CameraY-120) && p.Y < float64(e.CameraY+240)
}
func (e *Engine) moveEntities() {
	for i := 0; i < len(e.Entities); i++ {
		if i == e.Player || e.Entities[i].Deleted {
			continue
		}
		p := e.Entities[i]
		d := e.def(p.Type)
		if !e.visible(p) {
			continue
		}
		if p.Type == 1020 && e.random()%10 == 0 {
			if e.random()%2 == 0 {
				p.DX = -p.DX
			}
			if e.random()%2 == 0 {
				p.DY = -p.DY
			}
		}
		if (p.Type == 1070 || p.Type == 1071) && e.random()%60 == 0 {
			player := e.Entities[e.Player]
			distance := math.Hypot(player.X-p.X, player.Y-p.Y)
			if distance > 0 {
				e.spawn(1350, p.X+18, p.Y+18, 2*(player.X-p.X)/distance, 2*(player.Y-p.Y)/distance)
			}
			if player.X > p.X {
				p.Type = 1070
			} else {
				p.Type = 1071
			}
		}
		if p.DX == 0 && p.DY == 0 {
			e.Entities[i] = p
			continue
		}
		if p.Life > 0 {
			p.Life--
		}
		if d.Behavior == 3 {
			old := p
			p.X += p.DX
			p.Y--
			side := e.solidOther(p, i) >= 0
			onFloor := false
			for j, q := range e.Entities {
				if j == i || j == e.Player || q.Deleted || e.def(q.Type).Passable {
					continue
				}
				l, _, r, b := e.bounds(p)
				ql, qt, qr, _ := e.bounds(q)
				if b+1 == qt && l <= qr && r >= ql {
					onFloor = true
					break
				}
			}
			if side || !onFloor {
				p.X = old.X
				p.DX = -p.DX
				p.Type = d.Target
			}
			p.Y = old.Y
			e.Entities[i] = p
			continue
		}
		old := p
		p.X += p.DX
		p.Y += p.DY
		if d.Category == 2 {
			for j, q := range e.Entities {
				if j == i || j == e.Player || q.Deleted || q.Type == 1350 {
					continue
				}
				qd := e.def(q.Type)
				if qd.Category == 3 {
					continue
				}
				a, b, c, f := e.bounds(p)
				l, t, r, s := e.bounds(q)
				if overlap(a, b, c, f, l, t, r, s) {
					if qd.Category == 1 {
						e.Score += qd.Health
						e.Entities[j].Deleted = true
						e.spawn(1500, q.X, q.Y, 0, 0)
					}
					p.Deleted = true
					break
				}
			}
			if p.Life > 0 && p.Life < 55 {
				p.Deleted = true
			}
		} else if e.solidOther(p, i) >= 0 {
			p = old
			if d.Behavior == 1 || d.Behavior == 3 {
				p.DX = -p.DX
				if d.Behavior == 1 {
					p.DY = -p.DY
				}
				p.Type = d.Target
			} else if d.Behavior == 2 {
				p.Deleted = true
			}
		}
		e.Entities[i] = p
	}
}
func (e *Engine) solidOther(p Entity, self int) int {
	l, t, r, b := e.bounds(p)
	for i, q := range e.Entities {
		if i == self || i == e.Player || q.Deleted || e.def(q.Type).Passable {
			continue
		}
		ql, qt, qr, qb := e.bounds(q)
		if overlap(l, t, r, b, ql, qt, qr, qb) {
			return i
		}
	}
	return -1
}
func (e *Engine) pickups() {
	p := e.Entities[e.Player]
	l, t, r, b := e.bounds(p)
	for i := range e.Entities {
		if i == e.Player || e.Entities[i].Deleted {
			continue
		}
		q := e.Entities[i]
		d := e.def(q.Type)
		if d.Category == 3 {
			continue
		}
		ql, qt, qr, qb := e.bounds(q)
		if !overlap(l, t, r, b, ql, qt, qr, qb) {
			continue
		}
		if !d.Transparent {
			switch q.Type {
			case 90:
				e.Bonus++
			case 1600:
				e.Power++
			case 1700:
				e.CheckX, e.CheckY = p.X, p.Y
			case 1800:
				e.Health = min(100, e.Health+50)
			}
			e.Score += d.Health
			e.Entities[i].Deleted = true
		} else if d.Passable {
			e.Health -= 2
		}
	}
}
