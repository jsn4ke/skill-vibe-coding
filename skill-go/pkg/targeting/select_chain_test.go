package targeting

import (
	"math"
	"testing"
)

func TestSearchChainTargets(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0}, entityType: 1, alive: true}
	unit1 := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 5, y: 0, z: 0}, entityType: 2, alive: true}
	unit2 := &mockTargetUnit{id: 3, pos: &mockTargetPosition{x: 10, y: 0, z: 0}, entityType: 2, alive: true}
	unit3 := &mockTargetUnit{id: 4, pos: &mockTargetPosition{x: 15, y: 0, z: 0}, entityType: 2, alive: true}
	outOfRange := &mockTargetUnit{id: 5, pos: &mockTargetPosition{x: 100, y: 0, z: 0}, entityType: 2, alive: true}

	allUnits := []TargetUnit{unit1, unit2, unit3, outOfRange}

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

	t.Run("chains through nearby targets", func(t *testing.T) {
		result := SearchChainTargets(unit1, 3, 10, CheckEnemy, caster, spell, nil)
		// unit1(5,0)→unit2(10,0) dist=5, unit2→unit3(15,0) dist=5, unit3→outOfRange(100,0) dist=85>10
		if len(result) != 3 {
			t.Errorf("expected 3 chain results (initial+2), got %d", len(result))
		}
	})

	t.Run("nil initial returns nil", func(t *testing.T) {
		result := SearchChainTargets(nil, 3, 10, CheckEnemy, caster, spell, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("zero maxJumps returns nil", func(t *testing.T) {
		result := SearchChainTargets(unit1, 0, 10, CheckEnemy, caster, spell, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("excludes initial IDs", func(t *testing.T) {
		result := SearchChainTargets(unit1, 1, 10, CheckEnemy, caster, spell, []uint64{3})
		if len(result) != 2 { // initial + unit3 (unit2 excluded)
			t.Errorf("expected 2 results, got %d", len(result))
		}
	})

	t.Run("stops when no valid candidate", func(t *testing.T) {
		isolated := &mockTargetUnit{id: 10, pos: &mockTargetPosition{x: 50, y: 50, z: 0}, entityType: 2, alive: true}
		result := SearchChainTargets(isolated, 3, 5, CheckEnemy, caster, spell, nil)
		if len(result) != 1 { // only initial, no chain
			t.Errorf("expected 1 result (only initial), got %d", len(result))
		}
	})

	t.Run("check filter applies", func(t *testing.T) {
		result := SearchChainTargets(unit1, 3, 10, CheckAlly, caster, spell, nil)
		if len(result) != 1 { // initial only, all targets are enemies
			t.Errorf("expected 1 result (only initial, enemies filtered), got %d", len(result))
		}
	})
}
