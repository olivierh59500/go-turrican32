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

// These shots are captured by executing the original x86 firing branch and
// spawn routine for all weapon strengths in both facing directions.
func TestProjectileFanMatchesOriginalX86Routine(t *testing.T) {
	raw, err := os.ReadFile("testdata/native-fire.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 19 {
		t.Fatalf("incomplete native firing fixture: %d rows", len(rows))
	}
	for power := 0; power <= 2; power++ {
		for _, state := range []int{1200, 1201} {
			g := testEngine(t)
			g.Power = power
			g.setPlayerType(state)
			initial := len(g.Entities)
			g.Step(Fire)
			var shots []Entity
			for _, entity := range g.Entities[initial:] {
				if entity.Type == 1300 || entity.Type == 1301 {
					shots = append(shots, entity)
				}
			}
			matched := 0
			for _, row := range rows[1:] {
				readInt := func(column int) int {
					value, err := strconv.Atoi(row[column])
					if err != nil {
						t.Fatal(err)
					}
					return value
				}
				if readInt(0) != power || readInt(1) != state {
					continue
				}
				index := readInt(2)
				if index >= len(shots) {
					t.Fatalf("missing native projectile %d for power=%d state=%d", index, power, state)
				}
				shot := shots[index]
				if shot.Type != readInt(3) {
					t.Fatalf("incorrect projectile bank: %+v", shot)
				}
				got := []float64{shot.X, shot.Y, shot.DX, shot.DY}
				for n, value := range got {
					want, err := strconv.ParseFloat(row[n+4], 64)
					if err != nil {
						t.Fatal(err)
					}
					if value != want {
						t.Fatalf("power=%d state=%d shot=%d field=%s: got %v, want %v", power, state, index, rows[0][n+4], value, want)
					}
				}
				matched++
			}
			if matched != len(shots) {
				t.Fatalf("power=%d state=%d: %d shots, want %d", power, state, len(shots), matched)
			}
		}
	}
}

func TestUpgradedProjectilesKeepAllFiveBranches(t *testing.T) {
	g := testEngine(t)
	p := g.Entities[g.Player]
	g.Entities = []Entity{p}
	g.Player = 0
	g.Power = 2
	g.Step(Fire)
	for range 8 {
		g.Step(Fire)
	}
	count := 0
	for _, shot := range g.Entities {
		if (shot.Type == 1300 || shot.Type == 1301) && !shot.Deleted {
			count++
		}
	}
	if count != 5 {
		t.Fatalf("the weapon fan lost overlapping branches: got %d shots, want 5", count)
	}
}

func TestTurretProjectileBecomesHarmlessOnTerrainImpact(t *testing.T) {
	g := testEngine(t)
	g.CameraX, g.CameraY = 0, 0
	player := g.Entities[g.Player]
	g.Entities = []Entity{player, {Type: 100, X: 100, Y: 100}}
	g.Player = 0
	g.spawn(1350, 94, 105, 2, 0)
	g.moveEntities()
	impact := g.Entities[2]
	if impact.Type != 1500 || impact.X != 94 || impact.Y != 105 {
		t.Fatalf("turret shot remained a damaging projectile inside terrain: %+v", impact)
	}
	g.Entities[0].X, g.Entities[0].Y = 94, 105
	health := g.Health
	g.pickups()
	if g.Health != health {
		t.Fatal("impact animation damaged the player")
	}
	for range 12 {
		g.animate()
	}
	if !g.Entities[2].Deleted {
		t.Fatal("impact animation never expired")
	}
}

func TestOffscreenProjectilesExpireAndPreservePlayerIndex(t *testing.T) {
	g := testEngine(t)
	player := g.Entities[g.Player]
	g.Entities = []Entity{{Type: 100, Deleted: true}, player}
	g.Player = 1
	g.CameraX, g.CameraY = 0, 0
	g.spawn(1300, 600, 80, 4, 0)
	for range 66 {
		g.moveEntities()
		g.compact()
	}
	if len(g.Entities) != 1 || g.Player != 0 || g.Entities[0].X != player.X || g.Entities[0].Y != player.Y {
		t.Fatalf("spent objects were retained or player moved during compaction: player=%d entities=%+v", g.Player, g.Entities)
	}
}

func TestAnimatedTextureUsesNativeDecorationFlags(t *testing.T) {
	g := testEngine(t)
	d := g.Data.Definitions[8]
	if d.Category != 3 || !d.Passable || !d.Transparent || d.Last != 127 || d.Rate != .4 {
		t.Fatalf("incorrect native animated texture: %+v", d)
	}
	p := g.Entities[g.Player]
	g.spawn(8, p.X, p.Y, 0, 0)
	health := g.Health
	g.pickups()
	if g.Health != health {
		t.Fatal("animated decoration damaged the player")
	}
	frames := 0
	for _, f := range g.Data.Frames {
		if f.Type == 8 {
			frames++
		}
	}
	if frames != 128 {
		t.Fatalf("animated texture has %d frames, want 128", frames)
	}
}
