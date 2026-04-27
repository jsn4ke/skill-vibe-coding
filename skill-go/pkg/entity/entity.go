package entity

import "math"

// EntityID 是实体的唯一标识符。
type EntityID uint64

// Position 表示三维空间中的位置，包含朝向信息。
type Position struct {
	X       float64
	Y       float64
	Z       float64
	Facing  float64
}

// DistanceTo 计算到另一个位置的三维欧几里得距离。
func (p Position) DistanceTo(other Position) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	dz := p.Z - other.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// DistanceTo2D 计算到另一个位置的二维欧几里得距离（忽略 Z 轴）。
func (p Position) DistanceTo2D(other Position) float64 {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// IsInFront 判断 other 是否在当前实体的正前方（朝向 ±90° 范围内）。
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

// UnitState 是单位状态的位掩码，对齐 TC 的 UnitState 枚举。
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

// Has 检查是否包含指定状态标志。
func (s UnitState) Has(flag UnitState) bool {
	return s&flag != 0
}

// Set 设置指定状态标志，返回新的状态值（不可变模式）。
func (s UnitState) Set(flag UnitState) UnitState {
	return s | flag
}

// Clear 清除指定状态标志，返回新的状态值（不可变模式）。
func (s UnitState) Clear(flag UnitState) UnitState {
	return s & ^flag
}

// EntityType 表示实体的类型分类。
type EntityType uint8

const (
	TypeNone EntityType = iota
	TypePlayer
	TypeCreature
	TypePet
	TypeGameObject
)

// Entity 是游戏世界中的基础实体，包含位置、状态和等级信息。
type Entity struct {
	ID       EntityID
	Type     EntityType
	Pos      Position
	State    UnitState
	Level    uint32
}

// NewEntity 创建一个新的实体，默认状态为存活。
func NewEntity(id EntityID, typ EntityType, pos Position) *Entity {
	return &Entity{
		ID:    id,
		Type:  typ,
		Pos:   pos,
		State: StateAlive,
	}
}

// IsAlive 判断实体是否存活（拥有 StateAlive 且没有 StateDead）。
func (e *Entity) IsAlive() bool {
	return e.State.Has(StateAlive) && !e.State.Has(StateDead)
}

// CanCast 判断实体是否可以施法（存活且未昏迷/沉默）。
func (e *Entity) CanCast() bool {
	return e.IsAlive() && !e.State.Has(StateStunned) && !e.State.Has(StateSilenced)
}

// CanMove 判断实体是否可以移动（存活且未定身/昏迷）。
func (e *Entity) CanMove() bool {
	return e.IsAlive() && !e.State.Has(StateRooted) && !e.State.Has(StateStunned)
}
