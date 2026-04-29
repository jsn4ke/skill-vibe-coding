package targeting

import (
	"math"
	"testing"
)

func TestResolveCenter(t *testing.T) {
	casterPos := &mockTargetPosition{x: 10, y: 20, z: 0, facing: 0}
	caster := &mockTargetUnit{id: 1, pos: casterPos, entityType: 1, alive: true}
	targetPos := &mockTargetPosition{x: 30, y: 40, z: 5, facing: 0}
	target := &mockTargetUnit{id: 2, pos: targetPos, entityType: 2, alive: true}

	units := map[uint64]TargetUnit{1: caster, 2: target}
	spell := &mockSpellTargetRef{
		caster:       caster,
		unitTargetID: 2,
		sourcePos:    [3]float64{5, 10, 0},
		destPos:      [3]float64{50, 60, 0},
		lastTargetID: 2,
		units:        units,
	}

	tests := []struct {
		name string
		ref  ReferenceTypes
		want [3]float64
	}{
		{"RefCaster returns caster position", RefCaster, [3]float64{10, 20, 0}},
		{"RefTarget returns target position", RefTarget, [3]float64{30, 40, 5}},
		{"RefSrc returns source pos", RefSrc, [3]float64{5, 10, 0}},
		{"RefDest returns dest pos", RefDest, [3]float64{50, 60, 0}},
		{"RefLast returns last target position", RefLast, [3]float64{30, 40, 5}},
		{"RefNone returns zero", RefNone, [3]float64{0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCenter(tt.ref, spell)
			for i := 0; i < 3; i++ {
				if math.Abs(got[i]-tt.want[i]) > 1e-9 {
					t.Errorf("ResolveCenter(%v)[%d] = %v, want %v", tt.ref, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolveCenter_NilCaster(t *testing.T) {
	spell := &mockSpellTargetRef{caster: nil}
	got := ResolveCenter(RefCaster, spell)
	if got != [3]float64{0, 0, 0} {
		t.Errorf("ResolveCenter with nil caster = %v, want zero", got)
	}
}

func TestResolveCenter_ZeroTargetID(t *testing.T) {
	spell := &mockSpellTargetRef{unitTargetID: 0, units: map[uint64]TargetUnit{}}
	got := ResolveCenter(RefTarget, spell)
	if got != [3]float64{0, 0, 0} {
		t.Errorf("ResolveCenter with zero target ID = %v, want zero", got)
	}
}

func TestResolveCenter_ZeroLastTargetID(t *testing.T) {
	spell := &mockSpellTargetRef{lastTargetID: 0, units: map[uint64]TargetUnit{}}
	got := ResolveCenter(RefLast, spell)
	if got != [3]float64{0, 0, 0} {
		t.Errorf("ResolveCenter with zero last target ID = %v, want zero", got)
	}
}

func TestResolveReferer(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{}, entityType: 1, alive: true}
	target := &mockTargetUnit{id: 2, pos: &mockTargetPosition{}, entityType: 2, alive: true}
	units := map[uint64]TargetUnit{1: caster, 2: target}

	spell := &mockSpellTargetRef{
		caster:       caster,
		unitTargetID: 2,
		lastTargetID: 2,
		units:        units,
	}

	tests := []struct {
		name string
		ref  ReferenceTypes
		want uint64
	}{
		{"RefCaster returns caster", RefCaster, 1},
		{"RefTarget returns target unit", RefTarget, 2},
		{"RefLast returns last target unit", RefLast, 2},
		{"RefSrc returns nil", RefSrc, 0},
		{"RefDest returns nil", RefDest, 0},
		{"RefNone returns nil", RefNone, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveReferer(tt.ref, spell)
			if tt.want == 0 {
				if got != nil {
					t.Errorf("ResolveReferer(%v) = %v, want nil", tt.ref, got)
				}
			} else {
				if got == nil || got.GetID() != tt.want {
					t.Errorf("ResolveReferer(%v) ID = %v, want %v", tt.ref, got.GetID(), tt.want)
				}
			}
		})
	}
}

func TestResolveReferer_ZeroTargetID(t *testing.T) {
	spell := &mockSpellTargetRef{unitTargetID: 0, units: map[uint64]TargetUnit{}}
	got := ResolveReferer(RefTarget, spell)
	if got != nil {
		t.Errorf("ResolveReferer(RefTarget) with zero ID = %v, want nil", got)
	}
}
