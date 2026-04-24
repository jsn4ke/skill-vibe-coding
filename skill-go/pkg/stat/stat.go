package stat

type StatType uint8

const (
	Health StatType = iota
	Mana
	AttackPower
	SpellPower
	CritChance
	Haste
	Armor
	Resistance
)

type Modifier struct {
	Flat   float64
	Pct    float64
	Source string
}

type StatEntry struct {
	Base       float64
	Modifiers  []Modifier
}

func (se *StatEntry) Value() float64 {
	totalFlat := se.Base
	totalPct := 1.0
	for _, m := range se.Modifiers {
		totalFlat += m.Flat
		totalPct += m.Pct
	}
	return totalFlat * totalPct
}

type StatSet struct {
	entries map[StatType]*StatEntry
}

func NewStatSet() *StatSet {
	return &StatSet{
		entries: make(map[StatType]*StatEntry),
	}
}

func (s *StatSet) SetBase(st StatType, value float64) {
	if _, ok := s.entries[st]; !ok {
		s.entries[st] = &StatEntry{}
	}
	s.entries[st].Base = value
}

func (s *StatSet) Get(st StatType) float64 {
	if e, ok := s.entries[st]; ok {
		return e.Value()
	}
	return 0
}

func (s *StatSet) AddModifier(st StatType, mod Modifier) uint64 {
	if _, ok := s.entries[st]; !ok {
		s.entries[st] = &StatEntry{}
	}
	s.entries[st].Modifiers = append(s.entries[st].Modifiers, mod)
	return uint64(len(s.entries[st].Modifiers) - 1)
}

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
