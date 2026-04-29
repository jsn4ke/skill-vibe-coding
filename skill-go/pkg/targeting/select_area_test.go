package targeting

import (
	"math"
	"testing"
)

func TestSearchAreaTargets(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0}, entityType: 1, alive: true}
	ally := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 3, y: 0, z: 0}, entityType: 1, alive: true}
	enemy := &mockTargetUnit{id: 3, pos: &mockTargetPosition{x: 4, y: 0, z: 0}, entityType: 2, alive: true}

	allUnits := []TargetUnit{ally, enemy}

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

	t.Run("returns all passing units in radius", func(t *testing.T) {
		result := SearchAreaTargets([3]float64{0, 0, 0}, 5, CheckDefault, caster, spell, 0)
		if len(result) != 2 {
			t.Errorf("expected 2 results, got %d", len(result))
		}
	})

	t.Run("filters by enemy check", func(t *testing.T) {
		result := SearchAreaTargets([3]float64{0, 0, 0}, 5, CheckEnemy, caster, spell, 0)
		if len(result) != 1 || result[0].GetID() != 3 {
			t.Errorf("expected 1 enemy result (id=3), got %d results", len(result))
		}
	})

	t.Run("filters by ally check", func(t *testing.T) {
		result := SearchAreaTargets([3]float64{0, 0, 0}, 5, CheckAlly, caster, spell, 0)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected 1 ally result (id=2), got %d results", len(result))
		}
	})

	t.Run("excludes by ID", func(t *testing.T) {
		result := SearchAreaTargets([3]float64{0, 0, 0}, 5, CheckDefault, caster, spell, 2)
		for _, u := range result {
			if u.GetID() == 2 {
				t.Error("unit 2 should have been excluded")
			}
		}
	})

	t.Run("empty when no candidates", func(t *testing.T) {
		emptySpell := &mockSpellTargetRef{
			caster: caster,
			unitsInRadius: func(center [3]float64, radius float64, excludeID uint64) []TargetUnit {
				return nil
			},
		}
		result := SearchAreaTargets([3]float64{0, 0, 0}, 5, CheckDefault, caster, emptySpell, 0)
		if len(result) != 0 {
			t.Errorf("expected 0 results, got %d", len(result))
		}
	})
}
