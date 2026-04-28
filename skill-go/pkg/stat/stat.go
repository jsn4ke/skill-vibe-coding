package stat

// StatType 表示属性类型的枚举。
type StatType uint8

const (
	Health StatType = iota
	MaxHealth
	Mana
	AttackPower
	SpellPower
	CritChance
	Haste
	Armor
	Resistance
)

// Modifier 表示对属性的修改，包含固定值加成和百分比加成。
type Modifier struct {
	Flat   float64
	Pct    float64
	Source string
}

// StatEntry 表示一个属性条目，包含基础值和修改器列表。
type StatEntry struct {
	Base      float64
	Modifiers []Modifier
}

// Value 计算属性的最终值：基础值 + 所有固定值加成，再乘以 (1 + 所有百分比加成)。
func (se *StatEntry) Value() float64 {
	totalFlat := se.Base
	totalPct := 1.0
	for _, m := range se.Modifiers {
		totalFlat += m.Flat
		totalPct += m.Pct
	}
	return totalFlat * totalPct
}

// StatSet 管理一组属性条目。
type StatSet struct {
	entries map[StatType]*StatEntry
}

// NewStatSet 创建一个空的属性集合。
func NewStatSet() *StatSet {
	return &StatSet{
		entries: make(map[StatType]*StatEntry),
	}
}

// SetBase 设置指定属性类型的基础值。
func (s *StatSet) SetBase(st StatType, value float64) {
	if _, ok := s.entries[st]; !ok {
		s.entries[st] = &StatEntry{}
	}
	s.entries[st].Base = value
}

// Get 获取指定属性类型的最终计算值。
func (s *StatSet) Get(st StatType) float64 {
	if e, ok := s.entries[st]; ok {
		return e.Value()
	}
	return 0
}

// AddModifier 为指定属性类型添加修改器，返回修改器的索引。
func (s *StatSet) AddModifier(st StatType, mod Modifier) uint64 {
	if _, ok := s.entries[st]; !ok {
		s.entries[st] = &StatEntry{}
	}
	s.entries[st].Modifiers = append(s.entries[st].Modifiers, mod)
	return uint64(len(s.entries[st].Modifiers) - 1)
}

// RemoveModifierBySource 移除指定属性类型中来源匹配的所有修改器。
func (s *StatSet) RemoveModifierBySource(st StatType, source string) {
	if e, ok := s.entries[st]; ok {
		filtered := e.Modifiers[:0]
		for _, m := range e.Modifiers {
			if m.Source != source {
				filtered = append(filtered, m)
			}
		}
		e.Modifiers = filtered
	}
}
