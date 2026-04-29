package spellcore

import (
	"testing"
	"time"
)

func TestProcManager_Check_FlagMatching(t *testing.T) {
	m := NewProcManager()
	m.Register(Entry{
		SpellID:      1,
		Flags:        FlagSpellDamageDealt,
		SpellType:    TypeMaskDamage,
		SpellPhase:   PhaseHit,
		HitFlags:     ProcHitNormal,
		Chance:       1.0,
		TriggerSpell: 1,
	})

	t.Run("matching event triggers", func(t *testing.T) {
		results := m.Check(ProcEvent{
			Flag:      FlagSpellDamageDealt,
			SpellID:   1,
			TypeMask:  TypeMaskDamage,
			PhaseMask: PhaseHit,
			HitMask:   ProcHitNormal,
		})
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].TriggeredSpell != 1 {
			t.Errorf("TriggeredSpell = %v, want 1", results[0].TriggeredSpell)
		}
	})

	t.Run("wrong flag no trigger", func(t *testing.T) {
		results := m.Check(ProcEvent{
			Flag:      FlagSpellHealDealt,
			TypeMask:  TypeMaskHeal,
			PhaseMask: PhaseHit,
			HitMask:   ProcHitNormal,
		})
		if len(results) != 0 {
			t.Errorf("expected 0 results for wrong flag, got %d", len(results))
		}
	})

	t.Run("wrong type mask no trigger", func(t *testing.T) {
		results := m.Check(ProcEvent{
			Flag:      FlagSpellDamageDealt,
			TypeMask:  TypeMaskHeal,
			PhaseMask: PhaseHit,
			HitMask:   ProcHitNormal,
		})
		if len(results) != 0 {
			t.Errorf("expected 0 results for wrong type mask, got %d", len(results))
		}
	})

	t.Run("wrong phase no trigger", func(t *testing.T) {
		results := m.Check(ProcEvent{
			Flag:      FlagSpellDamageDealt,
			TypeMask:  TypeMaskDamage,
			PhaseMask: PhaseCast,
			HitMask:   ProcHitNormal,
		})
		if len(results) != 0 {
			t.Errorf("expected 0 results for wrong phase, got %d", len(results))
		}
	})

	t.Run("wrong hit mask no trigger", func(t *testing.T) {
		results := m.Check(ProcEvent{
			Flag:      FlagSpellDamageDealt,
			TypeMask:  TypeMaskDamage,
			PhaseMask: PhaseHit,
			HitMask:   ProcHitCrit,
		})
		if len(results) != 0 {
			t.Errorf("expected 0 results for wrong hit mask, got %d", len(results))
		}
	})
}

func TestProcManager_Check_Cooldown(t *testing.T) {
	m := NewProcManager()
	m.Register(Entry{
		SpellID:      2,
		Flags:        FlagSpellHit,
		Chance:       1.0,
		Cooldown:     1 * time.Hour,
		TriggerSpell: 2,
	})

	results := m.Check(ProcEvent{Flag: FlagSpellHit})
	if len(results) != 1 {
		t.Fatalf("first trigger: expected 1 result, got %d", len(results))
	}

	// Second trigger within cooldown → no result
	results = m.Check(ProcEvent{Flag: FlagSpellHit})
	if len(results) != 0 {
		t.Errorf("within cooldown: expected 0 results, got %d", len(results))
	}
}

func TestProcManager_Check_Charges(t *testing.T) {
	m := NewProcManager()
	m.Register(Entry{
		SpellID:      3,
		Flags:        FlagSpellHit,
		Chance:       1.0,
		Charges:      2,
		TriggerSpell: 3,
	})

	// First trigger: charges 2→1
	results := m.Check(ProcEvent{Flag: FlagSpellHit})
	if len(results) != 1 {
		t.Fatalf("trigger 1: expected 1, got %d", len(results))
	}

	// Second trigger: charges 1→0, entry removed
	results = m.Check(ProcEvent{Flag: FlagSpellHit})
	if len(results) != 1 {
		t.Fatalf("trigger 2: expected 1, got %d", len(results))
	}

	// Third trigger: entry gone
	results = m.Check(ProcEvent{Flag: FlagSpellHit})
	if len(results) != 0 {
		t.Errorf("trigger 3: expected 0 (charges spent), got %d", len(results))
	}
}

func TestProcManager_Unregister(t *testing.T) {
	m := NewProcManager()
	m.Register(Entry{SpellID: 10, Flags: FlagSpellHit, Chance: 1.0, TriggerSpell: 10})
	m.Register(Entry{SpellID: 20, Flags: FlagSpellHit, Chance: 1.0, TriggerSpell: 20})

	m.Unregister(10)

	results := m.Check(ProcEvent{Flag: FlagSpellHit})
	if len(results) != 1 || results[0].TriggeredSpell != 20 {
		t.Errorf("after unregistering 10, expected only 20, got %v", results)
	}
}

func TestProcManager_Check_NoFilters(t *testing.T) {
	m := NewProcManager()
	// Entry with no SpellType/SpellPhase/HitFlags filters
	m.Register(Entry{
		SpellID:      5,
		Flags:        FlagSpellHit,
		SpellType:    TypeMaskNone,
		SpellPhase:   PhaseNone,
		HitFlags:     ProcHitNone,
		Chance:       1.0,
		TriggerSpell: 5,
	})

	results := m.Check(ProcEvent{
		Flag:      FlagSpellHit,
		TypeMask:  TypeMaskDamage,
		PhaseMask: PhaseHit,
		HitMask:   ProcHitCrit,
	})
	if len(results) != 1 {
		t.Errorf("no-filter entry should match any, got %d results", len(results))
	}
}
