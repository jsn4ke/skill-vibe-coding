package targeting

// mockTargetPosition 实现 TargetPosition 接口，用于测试。
type mockTargetPosition struct {
	x, y, z, facing float64
}

func (p *mockTargetPosition) GetX() float64      { return p.x }
func (p *mockTargetPosition) GetY() float64      { return p.y }
func (p *mockTargetPosition) GetZ() float64      { return p.z }
func (p *mockTargetPosition) GetFacing() float64 { return p.facing }

// mockTargetUnit 实现 TargetUnit 接口，用于测试。
type mockTargetUnit struct {
	id         uint64
	pos        *mockTargetPosition
	entityType uint8
	alive      bool
}

func (u *mockTargetUnit) GetID() uint64               { return u.id }
func (u *mockTargetUnit) GetPosition() TargetPosition { return u.pos }
func (u *mockTargetUnit) GetEntityType() uint8        { return u.entityType }
func (u *mockTargetUnit) IsAlive() bool               { return u.alive }

// mockSpellTargetRef 实现 SpellTargetRef 接口，用于测试。
type mockSpellTargetRef struct {
	caster        TargetUnit
	unitTargetID  uint64
	sourcePos     [3]float64
	destPos       [3]float64
	lastTargetID  uint64
	units         map[uint64]TargetUnit
	unitsInRadius func(center [3]float64, radius float64, excludeID uint64) []TargetUnit
}

func (s *mockSpellTargetRef) GetCaster() TargetUnit    { return s.caster }
func (s *mockSpellTargetRef) GetUnitTargetID() uint64  { return s.unitTargetID }
func (s *mockSpellTargetRef) GetSourcePos() [3]float64 { return s.sourcePos }
func (s *mockSpellTargetRef) GetDestPos() [3]float64   { return s.destPos }
func (s *mockSpellTargetRef) GetLastTargetID() uint64  { return s.lastTargetID }

func (s *mockSpellTargetRef) GetUnitByID(id uint64) TargetUnit {
	if s.units != nil {
		return s.units[id]
	}
	return nil
}

func (s *mockSpellTargetRef) GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []TargetUnit {
	if s.unitsInRadius != nil {
		return s.unitsInRadius(center, radius, excludeID)
	}
	return nil
}
