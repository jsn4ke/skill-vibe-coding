package entity

import (
	"math"
	"testing"
)

func TestPosition_DistanceTo(t *testing.T) {
	tests := []struct {
		name   string
		p, q   Position
		want   float64
		want2D float64
	}{
		{"same point", Position{0, 0, 0, 0}, Position{0, 0, 0, 0}, 0, 0},
		{"unit distance X", Position{0, 0, 0, 0}, Position{1, 0, 0, 0}, 1, 1},
		{"3D diagonal", Position{0, 0, 0, 0}, Position{3, 4, 0, 0}, 5, 5},
		{"with Z axis", Position{0, 0, 0, 0}, Position{1, 1, 1, 0}, math.Sqrt(3), math.Sqrt(2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.DistanceTo(tt.q); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("DistanceTo() = %v, want %v", got, tt.want)
			}
			if got := tt.p.DistanceTo2D(tt.q); math.Abs(got-tt.want2D) > 1e-9 {
				t.Errorf("DistanceTo2D() = %v, want %v", got, tt.want2D)
			}
		})
	}
}

func TestPosition_IsInFront(t *testing.T) {
	tests := []struct {
		name    string
		facing  float64
		self    Position
		other   Position
		inFront bool
	}{
		{"facing +X, target at +X", 0, Position{0, 0, 0, 0}, Position{5, 0, 0, 0}, true},
		{"facing +X, target at -X", 0, Position{0, 0, 0, 0}, Position{-5, 0, 0, 0}, false},
		{"facing +X, target at +Y (90 deg)", 0, Position{0, 0, 0, 0}, Position{0, 5, 0, 0}, false},
		{"facing +X, target at 45 deg", 0, Position{0, 0, 0, 0}, Position{5, 4.9, 0, 0}, true},
		{"facing +Y, target at +Y", math.Pi / 2, Position{0, 0, 0, 0}, Position{0, 5, 0, 0}, true},
		{"facing +Y, target at -Y", math.Pi / 2, Position{0, 0, 0, 0}, Position{0, -5, 0, 0}, false},
		{"facing -X, target at -X", math.Pi, Position{0, 0, 0, 0}, Position{-5, 0, 0, 0}, true},
		{"same position", 0, Position{0, 0, 0, 0}, Position{0, 0, 0, 0}, true}, // angle=0, diff=0 < π/2
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Position{X: tt.self.X, Y: tt.self.Y, Z: tt.self.Z, Facing: tt.facing}
			got := p.IsInFront(tt.other)
			if got != tt.inFront {
				t.Errorf("IsInFront() = %v, want %v", got, tt.inFront)
			}
		})
	}
}

func TestUnitState_HasSetClear(t *testing.T) {
	s := StateAlive
	if !s.Has(StateAlive) {
		t.Error("StateAlive should have StateAlive")
	}
	if s.Has(StateDead) {
		t.Error("StateAlive should not have StateDead")
	}

	s = s.Set(StateInCombat)
	if !s.Has(StateAlive) || !s.Has(StateInCombat) {
		t.Error("after Set(StateInCombat), should have both")
	}

	s = s.Clear(StateAlive)
	if s.Has(StateAlive) {
		t.Error("after Clear(StateAlive), should not have StateAlive")
	}
	if !s.Has(StateInCombat) {
		t.Error("after Clear(StateAlive), should still have StateInCombat")
	}
}

func TestEntity_IsAlive_CanCast_CanMove(t *testing.T) {
	e := NewEntity(1, TypePlayer, Position{0, 0, 0, 0})
	if !e.IsAlive() {
		t.Error("new entity should be alive")
	}
	if !e.CanCast() {
		t.Error("alive entity should be able to cast")
	}
	if !e.CanMove() {
		t.Error("alive entity should be able to move")
	}

	e.State = e.State.Set(StateStunned)
	if e.CanCast() {
		t.Error("stunned entity should not be able to cast")
	}
	if e.CanMove() {
		t.Error("stunned entity should not be able to move")
	}

	e.State = StateAlive
	if !e.CanCast() {
		t.Error("after clearing stun, should be able to cast")
	}

	e.State = e.State.Set(StateDead)
	if e.IsAlive() {
		t.Error("dead entity should not be alive")
	}
}
