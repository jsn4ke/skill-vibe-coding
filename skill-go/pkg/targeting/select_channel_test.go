package targeting

import "testing"

func TestSelectChannelTargets(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0}, entityType: 1, alive: true}
	target := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 5, y: 0, z: 0}, entityType: 2, alive: true}
	units := map[uint64]TargetUnit{2: target}

	spell := &mockSpellTargetRef{
		caster:       caster,
		unitTargetID: 2,
		units:        units,
	}

	t.Run("ObjUnit returns channel target unit", func(t *testing.T) {
		info := NewImplicitTargetInfo(77) // TARGET_UNIT_CHANNEL_TARGET → ObjUnit
		result := SelectChannelTargets(info, spell)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected 1 result (id=2), got %d results", len(result))
		}
	})

	t.Run("ObjDest returns channel target as unit", func(t *testing.T) {
		info := NewImplicitTargetInfo(76) // TARGET_DEST_CHANNEL_TARGET → ObjDest
		result := SelectChannelTargets(info, spell)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected 1 result (id=2), got %d results", len(result))
		}
	})

	t.Run("returns nil when no target ID", func(t *testing.T) {
		info := NewImplicitTargetInfo(77)
		noTargetSpell := &mockSpellTargetRef{
			caster:       caster,
			unitTargetID: 0,
			units:        units,
		}
		result := SelectChannelTargets(info, noTargetSpell)
		if len(result) != 0 {
			t.Errorf("expected 0 results, got %d", len(result))
		}
	})

	t.Run("returns nil when unit not found", func(t *testing.T) {
		info := NewImplicitTargetInfo(77)
		missingSpell := &mockSpellTargetRef{
			caster:       caster,
			unitTargetID: 99,
			units:        units,
		}
		result := SelectChannelTargets(info, missingSpell)
		if len(result) != 0 {
			t.Errorf("expected 0 results, got %d", len(result))
		}
	})

	t.Run("returns nil for unknown object type", func(t *testing.T) {
		info := NewImplicitTargetInfo(10) // NYI → ObjNone
		result := SelectChannelTargets(info, spell)
		if result != nil {
			t.Errorf("expected nil for ObjNone, got %v", result)
		}
	})
}
