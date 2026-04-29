package targeting

import (
	"math"
	"testing"
)

func TestPointToSegmentDistance2D(t *testing.T) {
	tests := []struct {
		name       string
		px, py     float64
		ax, ay     float64
		bx, by     float64
		wantApprox float64
	}{
		{"point on segment", 5, 0, 0, 0, 10, 0, 0},
		{"point off segment perpendicular", 5, 3, 0, 0, 10, 0, 3},
		{"point beyond A endpoint", -2, 1, 0, 0, 10, 0, math.Sqrt(5)},
		{"point beyond B endpoint", 12, 1, 0, 0, 10, 0, math.Sqrt(5)},
		{"degenerate segment (point)", 3, 4, 0, 0, 0, 0, 5},
		{"point exactly at A", 0, 0, 0, 0, 10, 0, 0},
		{"point exactly at B", 10, 0, 0, 0, 10, 0, 0},
		{"diagonal segment", 1, 2, 0, 0, 4, 4, math.Sqrt(2) / 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pointToSegmentDistance2D(tt.px, tt.py, tt.ax, tt.ay, tt.bx, tt.by)
			if math.Abs(got-tt.wantApprox) > 1e-9 {
				t.Errorf("pointToSegmentDistance2D(%v,%v,%v,%v,%v,%v) = %v, want ~%v",
					tt.px, tt.py, tt.ax, tt.ay, tt.bx, tt.by, got, tt.wantApprox)
			}
		})
	}
}

func TestSearchLineTargets(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0}, entityType: 1, alive: true}
	onLine := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 5, y: 0.5, z: 0}, entityType: 2, alive: true}
	offLine := &mockTargetUnit{id: 3, pos: &mockTargetPosition{x: 5, y: 10, z: 0}, entityType: 2, alive: true}

	allUnits := []TargetUnit{onLine, offLine}

	spell := &mockSpellTargetRef{
		caster: caster,
		unitsInRadius: func(center [3]float64, radius float64, excludeID uint64) []TargetUnit {
			var result []TargetUnit
			for _, u := range allUnits {
				if u.GetID() == excludeID {
					continue
				}
				p := u.GetPosition()
				dx := p.GetX() - center[0]
				dy := p.GetY() - center[1]
				dz := p.GetZ() - center[2]
				dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if dist <= radius {
					result = append(result, u)
				}
			}
			return result
		},
	}

	from := [3]float64{0, 0, 0}
	to := [3]float64{10, 0, 0}
	width := 2.0

	t.Run("selects targets within line width", func(t *testing.T) {
		result := SearchLineTargets(from, to, width, CheckEnemy, caster, spell, 0)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected 1 result (id=2 on line), got %d results", len(result))
		}
	})

	t.Run("wide width includes more targets", func(t *testing.T) {
		result := SearchLineTargets(from, to, 25, CheckEnemy, caster, spell, 0)
		if len(result) != 2 {
			t.Errorf("expected 2 results with wide width, got %d", len(result))
		}
	})

	t.Run("check filter excludes allies", func(t *testing.T) {
		result := SearchLineTargets(from, to, width, CheckAlly, caster, spell, 0)
		if len(result) != 0 {
			t.Errorf("expected 0 ally results (all enemies), got %d", len(result))
		}
	})
}
