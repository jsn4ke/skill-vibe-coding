package stat

import (
	"math"
	"testing"
)

func TestStatEntry_Value(t *testing.T) {
	tests := []struct {
		name string
		base float64
		mods []Modifier
		want float64
	}{
		{"no modifiers", 100, nil, 100},
		{"flat only", 100, []Modifier{{Flat: 20, Pct: 0}}, 120},
		{"pct only", 100, []Modifier{{Flat: 0, Pct: 0.5}}, 150},
		{"flat + pct", 100, []Modifier{{Flat: 20, Pct: 0.5}}, 180},
		{"multiple flat", 100, []Modifier{{Flat: 10}, {Flat: 20}}, 130},
		{"multiple pct", 100, []Modifier{{Pct: 0.2}, {Pct: 0.3}}, 150},
		{"negative flat", 100, []Modifier{{Flat: -30}}, 70},
		{"negative pct", 100, []Modifier{{Pct: -0.5}}, 50},
		{"zero base", 0, []Modifier{{Flat: 10}}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &StatEntry{Base: tt.base, Modifiers: tt.mods}
			got := se.Value()
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatSet_SetBase_Get(t *testing.T) {
	s := NewStatSet()
	s.SetBase(Health, 1000)
	if got := s.Get(Health); got != 1000 {
		t.Errorf("Get(Health) = %v, want 1000", got)
	}
	if got := s.Get(Mana); got != 0 {
		t.Errorf("Get(Mana) for unset = %v, want 0", got)
	}
}

func TestStatSet_AddModifier_RemoveModifierBySource(t *testing.T) {
	s := NewStatSet()
	s.SetBase(Health, 100)
	s.AddModifier(Health, Modifier{Flat: 50, Source: "buff1"})
	s.AddModifier(Health, Modifier{Flat: 30, Source: "buff2"})

	if got := s.Get(Health); got != 180 {
		t.Errorf("with modifiers = %v, want 180", got)
	}

	s.RemoveModifierBySource(Health, "buff1")
	if got := s.Get(Health); got != 130 {
		t.Errorf("after removing buff1 = %v, want 130", got)
	}

	// Removing non-existent source does nothing
	s.RemoveModifierBySource(Health, "nonexistent")
	if got := s.Get(Health); got != 130 {
		t.Errorf("after removing nonexistent = %v, want 130", got)
	}

	// Removing from unset stat does nothing
	s.RemoveModifierBySource(Mana, "buff1")
}
