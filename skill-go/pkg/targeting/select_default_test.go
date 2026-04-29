package targeting

import (
	"fmt"
	"math"
	"testing"
)

func TestResolveDefaultTargets(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0, facing: 0}, entityType: 1, alive: true}
	target := &mockTargetUnit{id: 2, pos: &mockTargetPosition{x: 5, y: 0, z: 0, facing: 0}, entityType: 2, alive: true}
	units := map[uint64]TargetUnit{1: caster, 2: target}

	spell := &mockSpellTargetRef{
		caster:       caster,
		unitTargetID: 2,
		sourcePos:    [3]float64{0, 0, 0},
		destPos:      [3]float64{10, 0, 0},
		units:        units,
	}

	t.Run("ObjUnit with RefCaster returns caster", func(t *testing.T) {
		info := NewImplicitTargetInfo(1) // TARGET_UNIT_CASTER
		result := ResolveDefaultTargets(info, spell)
		if len(result) != 1 || result[0].GetID() != 1 {
			t.Errorf("expected caster (id=1), got %v", result)
		}
	})

	t.Run("ObjUnit with RefTarget returns target", func(t *testing.T) {
		info := NewImplicitTargetInfo(6) // TARGET_UNIT_TARGET_ENEMY
		result := ResolveDefaultTargets(info, spell)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected target (id=2), got %v", result)
		}
	})

	t.Run("ObjDest returns nil (position-only)", func(t *testing.T) {
		info := NewImplicitTargetInfo(18) // TARGET_DEST_CASTER
		result := ResolveDefaultTargets(info, spell)
		if result != nil {
			t.Errorf("expected nil for ObjDest, got %v", result)
		}
	})

	t.Run("ObjSrc returns nil (position-only)", func(t *testing.T) {
		info := NewImplicitTargetInfo(22) // TARGET_SRC_CASTER
		result := ResolveDefaultTargets(info, spell)
		if result != nil {
			t.Errorf("expected nil for ObjSrc, got %v", result)
		}
	})

	t.Run("ObjUnitAndDest returns referer unit", func(t *testing.T) {
		info := NewImplicitTargetInfo(25) // TARGET_UNIT_TARGET_ANY
		result := ResolveDefaultTargets(info, spell)
		if len(result) != 1 || result[0].GetID() != 2 {
			t.Errorf("expected target (id=2), got %v", result)
		}
	})

	t.Run("NYI object type returns nil", func(t *testing.T) {
		info := NewImplicitTargetInfo(10) // unset → ObjNone
		result := ResolveDefaultTargets(info, spell)
		if result != nil {
			t.Errorf("expected nil for ObjNone, got %v", result)
		}
	})
}

func TestApplyDirectionOffset(t *testing.T) {
	caster := &mockTargetUnit{id: 1, pos: &mockTargetPosition{x: 0, y: 0, z: 0, facing: 0}, entityType: 1, alive: true}
	spell := &mockSpellTargetRef{caster: caster}

	t.Run("DirNone returns original position", func(t *testing.T) {
		pos := [3]float64{5, 5, 0}
		result := ApplyDirectionOffset(pos, DirNone, spell)
		if result != pos {
			t.Errorf("expected same position, got %v", result)
		}
	})

	t.Run("DirFront with zero facing and zero offset returns same position", func(t *testing.T) {
		pos := [3]float64{5, 5, 0}
		result := ApplyDirectionOffset(pos, DirFront, spell)
		if result != pos {
			t.Errorf("expected same position (offset=0), got %v", result)
		}
	})

	t.Run("nil caster still works", func(t *testing.T) {
		nilCasterSpell := &mockSpellTargetRef{caster: nil}
		pos := [3]float64{5, 5, 0}
		// With DirBack (angle=π) and no caster, facing=0, offset=0
		result := ApplyDirectionOffset(pos, DirBack, nilCasterSpell)
		// offsetDist=0 so result should be same as pos
		if result[0] != 5 || result[1] != 5 || result[2] != 0 {
			t.Errorf("expected same position (offset=0), got %v", result)
		}
	})
}

func TestCalcDirectionAngle(t *testing.T) {
	tests := []struct {
		dir  DirectionTypes
		want float64
	}{
		{DirFront, 0.0},
		{DirBack, math.Pi},
		{DirRight, -math.Pi / 2},
		{DirLeft, math.Pi / 2},
		{DirFrontRight, -math.Pi / 4},
		{DirBackRight, -3 * math.Pi / 4},
		{DirBackLeft, 3 * math.Pi / 4},
		{DirFrontLeft, math.Pi / 4},
		{DirNone, 0.0},
		{DirRandom, 0.0}, // Just check it doesn't crash; value is random
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("dir=%d", tt.dir), func(t *testing.T) {
			got := CalcDirectionAngle(tt.dir)
			if tt.dir == DirRandom {
				if got < 0 || got >= 2*math.Pi {
					t.Errorf("DirRandom = %v, want in [0, 2π)", got)
				}
				return
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("CalcDirectionAngle(%v) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}
