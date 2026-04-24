package diminishing

import (
	"time"
)

type Group uint8

const (
	GroupNone Group = iota
	GroupStun
	GroupFear
	GroupRoot
	GroupSilence
	GroupCharm
	GroupConfuse
	GroupDisarm
	GroupBanish
	GroupKnockback
	GroupPolymorph
	GroupSlow
)

type ReturnType uint8

const (
	ReturnNone ReturnType = iota
	ReturnStandard
	ReturnHalf
	ReturnQuarter
	ReturnImmune
)

type Level struct {
	Group      Group
	ReturnType ReturnType
	MaxLevel   uint8
	DurLimit   time.Duration
}

type Entry struct {
	Group    Group
	CasterID uint64
	TargetID uint64
	ApplyAt  time.Time
	Level    uint8
	MaxDur   time.Duration
}

type Manager struct {
	levels  map[Group]Level
	entries map[uint64][]Entry
}

func NewManager() *Manager {
	return &Manager{
		levels:  make(map[Group]Level),
		entries: make(map[uint64][]Entry),
	}
}

func (m *Manager) RegisterLevel(level Level) {
	m.levels[level.Group] = level
}

func (m *Manager) ApplyDiminishing(targetID, casterID uint64, group Group, duration time.Duration) (time.Duration, bool) {
	if group == GroupNone {
		return duration, false
	}

	level, ok := m.levels[group]
	if !ok {
		return duration, false
	}

	entries := m.entries[targetID]
	now := time.Now()

	var active []Entry
	for _, e := range entries {
		if e.Group == group && now.Sub(e.ApplyAt) < 18*time.Second {
			active = append(active, e)
		}
	}

	currentLevel := uint8(len(active))
	if level.MaxLevel > 0 && currentLevel >= level.MaxLevel {
		return 0, true
	}

	var modifiedDuration time.Duration
	switch {
	case currentLevel == 0:
		modifiedDuration = duration
	case currentLevel == 1:
		modifiedDuration = duration / 2
	case currentLevel == 2:
		modifiedDuration = duration / 4
	default:
		return 0, true
	}

	if level.DurLimit > 0 && modifiedDuration > level.DurLimit {
		modifiedDuration = level.DurLimit
	}

	newEntry := Entry{
		Group:    group,
		CasterID: casterID,
		TargetID: targetID,
		ApplyAt:  now,
		Level:    currentLevel,
		MaxDur:   modifiedDuration,
	}
	m.entries[targetID] = append(entries, newEntry)

	return modifiedDuration, false
}

func (m *Manager) GetLevel(targetID uint64, group Group) uint8 {
	entries := m.entries[targetID]
	now := time.Now()
	var count uint8
	for _, e := range entries {
		if e.Group == group && now.Sub(e.ApplyAt) < 18*time.Second {
			count++
		}
	}
	return count
}

func (m *Manager) Clear(targetID uint64) {
	delete(m.entries, targetID)
}
