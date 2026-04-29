package targeting

import (
	"math"
	"testing"
)

func TestSearchNearbyTarget(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0}, entityType: 1, alive: true}
	close := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 2, y: 0, z: 0}, entityType: 2, alive: true}
	far := &mockTargetUnit{id: 3, pos: &mockTargetPosition{x: 8, y: 0, z: 0}, entityType: 2, alive: true}

	allUnits := []TargetUnit{close, far}

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

	t.Run("returns closest target", func(t *testing.T) {
		result := SearchNearbyTarget([3]float64{0, 0, 0}, 10, CheckEnemy, caster, spell, 0)
		if result == nil || result.GetID() != 2 {
			t.Errorf("expected closest unit id=2, got %v", result)
		}
	})

	t.Run("returns nil when radius too small", func(t *testing.T) {
		result := SearchNearbyTarget([3]float64{0, 0, 0}, 1, CheckEnemy, caster, spell, 0)
		if result != nil {
			t.Errorf("expected nil for too small radius, got %v", result)
		}
	})

	t.Run("returns nil when no candidates", func(t *testing.T) {
		emptySpell := &mockSpellTargetRef{
			caster: caster,
			unitsInRadius: func(center [3]float64, radius float64, excludeID uint64) []TargetUnit {
				return nil
			},
		}
		result := SearchNearbyTarget([3]float64{0, 0, 0}, 10, CheckEnemy, caster, emptySpell, 0)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("filters by check type", func(t *testing.T) {
		result := SearchNearbyTarget([3]float64{0, 0, 0}, 10, CheckAlly, caster, spell, 0)
		if result != nil {
			t.Errorf("expected nil for ally check (all enemies), got %v", result)
		}
	})

	t.Run("excludes by ID", func(t *testing.T) {
		result := SearchNearbyTarget([3]float64{0, 0, 0}, 10, CheckEnemy, caster, spell, 2)
		if result == nil || result.GetID() != 3 {
			t.Errorf("expected id=3 after excluding id=2, got %v", result)
		}
	})
}
