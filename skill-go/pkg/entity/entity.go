package entity

import "math"

type EntityID uint64

type Position struct {
	X       float64
	Y       float64
	Z       float64
	Facing  float64
}

func (p Position) DistanceTo(other Position) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	dz := p.Z - other.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (p Position) DistanceTo2D(other Position) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (p Position) IsInFront(other Position) bool {
	dx := other.X - p.X
	dy := other.Y - p.Y
	angle := math.Atan2(dy, dx)
	diff := angle - p.Facing
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	return math.Abs(diff) < math.Pi/2
}

type UnitState uint32

const (
	StateNone      UnitState = 0
	StateAlive     UnitState = 1 << iota
	StateDead
	StateInCombat
	StateStunned
	StateRooted
	StateSilenced
	StatePacified
	StateCharmed
	StateConfused
	StateFeared
	StateStealthed
)

func (s UnitState) Has(flag UnitState) bool {
	return s&flag != 0
}

func (s UnitState) Set(flag UnitState) UnitState {
	return s | flag
}

func (s UnitState) Clear(flag UnitState) UnitState {
	return s & ^flag
}

type EntityType uint8

const (
	TypeNone EntityType = iota
	TypePlayer
	TypeCreature
	TypePet
	TypeGameObject
)

type Entity struct {
	ID       EntityID
	Type     EntityType
	Pos      Position
	State    UnitState
	Level    uint32
}

func NewEntity(id EntityID, typ EntityType, pos Position) *Entity {
	return &Entity{
		ID:    id,
		Type:  typ,
		Pos:   pos,
		State: StateAlive,
	}
}

func (e *Entity) IsAlive() bool {
	return e.State.Has(StateAlive) && !e.State.Has(StateDead)
}

func (e *Entity) CanCast() bool {
	return e.IsAlive() && !e.State.Has(StateStunned) && !e.State.Has(StateSilenced)
}

func (e *Entity) CanMove() bool {
	return e.IsAlive() && !e.State.Has(StateRooted) && !e.State.Has(StateStunned)
}
