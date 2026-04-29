package targeting

import (
	"math"
	"testing"
)

func TestIsAngleInCone(t *testing.T) {
	tests := []struct {
		name      string
		angle     float64
		direction float64
		arcAngle  float64
		want      bool
	}{
		{"exact center", 0, 0, math.Pi, true},
		{"just inside left edge", math.Pi/2 - 0.01, 0, math.Pi, true},
		{"just outside left edge", math.Pi/2 + 0.01, 0, math.Pi, false},
		{"just inside right edge", -math.Pi/2 + 0.01, 0, math.Pi, true},
		{"just outside right edge", -math.Pi/2 - 0.01, 0, math.Pi, false},
		{"narrow cone inside", 0.1, 0, math.Pi / 4, true},
		{"narrow cone outside", math.Pi / 2, 0, math.Pi / 4, false},
		{"full circle always true", math.Pi, 0, 2 * math.Pi, true},
		{"backward direction center", math.Pi, math.Pi, math.Pi / 2, true},
		{"wrap-around positive to negative", -math.Pi + 0.01, math.Pi, math.Pi / 4, true},
		{"wrap-around negative to positive", math.Pi - 0.01, -math.Pi, math.Pi / 4, true},
		{"direction Pi/2 cone Pi/2", math.Pi / 4, math.Pi / 2, math.Pi / 2, true},
		{"boundary exact half arc", math.Pi / 4, 0, math.Pi / 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAngleInCone(tt.angle, tt.direction, tt.arcAngle)
			if got != tt.want {
				t.Errorf("isAngleInCone(%v, %v, %v) = %v, want %v", tt.angle, tt.direction, tt.arcAngle, got, tt.want)
			}
		})
	}
}

func TestSearchConeTargets(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0, facing: 0}, entityType: 1, alive: true}
	inFront := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 5, y: 0.1, z: 0}, entityType: 2, alive: true}
	behind := &mockTargetUnit{id: 3, pos: &mockTargetPosition{x: -5, y: 0, z: 0}, entityType: 2, alive: true}

	allUnits := []TargetUnit{inFront, behind}

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

	t.Run("front cone includes only front targets", func(t *testing.T) {
		result := SearchConeTargets([3]float64{0, 0, 0}, 0, math.Pi/2, 10, CheckEnemy, caster, spell, 0)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected 1 result (id=2), got %d results", len(result))
		}
	})

	t.Run("full circle includes all", func(t *testing.T) {
		result := SearchConeTargets([3]float64{0, 0, 0}, 0, 2*math.Pi, 10, CheckEnemy, caster, spell, 0)
		if len(result) != 2 {
			t.Errorf("expected 2 results, got %d", len(result))
		}
	})

	t.Run("backward cone includes only behind targets", func(t *testing.T) {
		result := SearchConeTargets([3]float64{0, 0, 0}, math.Pi, math.Pi/2, 10, CheckEnemy, caster, spell, 0)
		if len(result) != 1 || result[0].GetID() != 3 {
			t.Errorf("expected 1 result (id=3), got %d results", len(result))
		}
	})

	t.Run("ally check filters enemies", func(t *testing.T) {
		result := SearchConeTargets([3]float64{0, 0, 0}, 0, math.Pi/2, 10, CheckAlly, caster, spell, 0)
		if len(result) != 0 {
			t.Errorf("expected 0 ally results (all enemies), got %d", len(result))
		}
	})
}
