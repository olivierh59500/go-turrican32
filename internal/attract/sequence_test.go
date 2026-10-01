package attract

import (
	"github.com/olivierh59500/go-turrican32/internal/data"
	"github.com/olivierh59500/go-turrican32/internal/engine"
	"testing"
)

func TestRecordedDemoTraversesLevelWithoutLosingALife(t *testing.T) {
	sequence, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	d, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	g := engine.New(d)
	maxX, minY := -1000.0, 1000.0
	left, right := false, false
	for frame := 0; frame < sequence.Len(); frame++ {
		input, ok := sequence.Input(frame)
		if !ok {
			t.Fatal("missing recorded input")
		}
		left = left || input&engine.Left != 0
		right = right || input&engine.Right != 0
		g.Step(input)
		if g.DeathAge > 0 || g.Lives != 3 || g.GameOver || g.Won {
			t.Fatalf("demo interrupted at frame %d: health=%d lives=%d", frame, g.Health, g.Lives)
		}
		p := g.Entities[g.Player]
		maxX = max(maxX, p.X)
		minY = min(minY, p.Y)
		if len(g.Entities) > 1100 {
			t.Fatalf("unbounded transient objects at frame %d: %d", frame, len(g.Entities))
		}
	}
	if !left || !right || maxX < 1200 || minY > -350 {
		t.Fatalf("demo did not cover its route: left=%v right=%v maxX=%v minY=%v", left, right, maxX, minY)
	}
	if _, ok := sequence.Input(-1); ok {
		t.Fatal("negative input frame accepted")
	}
	if _, ok := sequence.Input(sequence.Len()); ok {
		t.Fatal("recording loops without returning to title")
	}
}
