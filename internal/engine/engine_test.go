package engine

import (
	"encoding/csv"
	"github.com/olivierh59500/go-turrican32/internal/data"
	"os"
	"strconv"
	"strings"
	"testing"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	d, e := data.Load()
	if e != nil {
		t.Fatal(e)
	}
	return New(d)
}
func TestOriginalLevelAndSpawn(t *testing.T) {
	g := testEngine(t)
	if len(g.Entities) != 783 {
		t.Fatalf("level has %d objects", len(g.Entities))
	}
	p := g.Entities[g.Player]
	if p.Type != 1210 || p.X != -72 || p.Y != 80 {
		t.Fatalf("incorrect player spawn %+v", p)
	}
	for i := 0; i < 60; i++ {
		g.Step(0)
	}
	if g.Entities[g.Player].Y != 166 || g.VY != 0 {
		t.Fatalf("player did not land at native floor: %+v", g.Entities[g.Player])
	}
}
func TestJumpAndVariableRelease(t *testing.T) {
	g := testEngine(t)
	for i := 0; i < 60; i++ {
		g.Step(0)
	}
	g.Step(Jump)
	if g.VY != -6 {
		t.Fatal("jump impulse changed")
	}
	start := g.Entities[g.Player].Y
	for i := 0; i < 10; i++ {
		g.Step(Jump)
	}
	if g.Entities[g.Player].Y >= start-40 {
		t.Fatal("jump does not rise")
	}
	held := g.VY
	g.Step(0)
	if g.VY <= held {
		t.Fatal("release did not shorten jump")
	}
}
func TestNativePickupTypesKeepGemsAndWeaponsSeparate(t *testing.T) {
	g := testEngine(t)
	p := g.Entities[g.Player]
	for _, id := range []int{90, 1600, 1700} {
		g.spawn(id, p.X, p.Y, 0, 0)
	}
	g.pickups()
	if g.Bonus != 1 || g.Power != 1 || g.CheckX != p.X || g.CheckY != p.Y {
		t.Fatalf("invalid pickup state bonus=%d weapon=%d checkpoint=%v,%v", g.Bonus, g.Power, g.CheckX, g.CheckY)
	}
}
func TestFireUsesOriginalProjectileBank(t *testing.T) {
	g := testEngine(t)
	n := len(g.Entities)
	g.Step(Fire)
	if len(g.Entities) <= n {
		t.Fatal("fire did not spawn")
	}
	found := false
	for _, e := range g.Entities[n:] {
		if e.Type == 1300 || e.Type == 1301 {
			found = true
			if e.DX != 4 && e.DX != -4 {
				t.Fatal("projectile speed changed")
			}
		}
	}
	if !found {
		t.Fatal("player frame substituted for bullet")
	}
}
func TestRestartRestoresLevelAndClearsTransientObjects(t *testing.T) {
	g := testEngine(t)
	g.Step(Fire)
	g.Health = -1
	g.Step(0)
	for i := 0; i < 81; i++ {
		g.Step(0)
	}
	if g.Health != 100 || g.Lives != 2 {
		t.Fatalf("respawn state health=%d lives=%d", g.Health, g.Lives)
	}
	g.Reset()
	if len(g.Entities) != 783 || g.Score != 0 || g.Power != 0 || g.Lives != 3 {
		t.Fatal("reset retained state")
	}
}

// These landing samples are produced by executing the original x86 floor
// routine, including its one-pixel separation from the platform surface.
func TestFloorMatchesOriginalX86Routine(t *testing.T) {
	g := testEngine(t)
	b, err := os.ReadFile("testdata/native-floor.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 74 {
		t.Fatal("incomplete original fixture")
	}
	for _, row := range rows[1:] {
		y, _ := strconv.Atoi(row[1])
		want, _ := strconv.Atoi(row[2])
		p := g.Entities[g.Player]
		p.Y = float64(y)
		got := g.solid(p, 1)
		if got >= 0 {
			got++
		}
		if got != want {
			t.Fatalf("floor at %s,%s: got %d, want %d", row[0], row[1], got, want)
		}
	}
}
