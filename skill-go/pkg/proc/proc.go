package proc

import (
	"math/rand"
	"time"

	"skill-go/pkg/spell"
)

type ProcFlag uint32

const (
	FlagNone          ProcFlag = 0
	FlagKill          ProcFlag = 1 << iota
	FlagMeleeSwing
	FlagSpellDamageDealt
	FlagSpellDamageTaken
	FlagSpellHealDealt
	FlagSpellHealTaken
	FlagPeriodicDamageDealt
	FlagPeriodicDamageTaken
	FlagPeriodicHealDealt
	FlagPeriodicHealTaken
	FlagSpellCast
	FlagSpellHit
	FlagDeath
	FlagCombatEnter
	FlagAttackSwing
)

type SpellTypeMask uint8

const (
	TypeMaskNone      SpellTypeMask = 0
	TypeMaskDamage    SpellTypeMask = 1 << iota
	TypeMaskHeal
	TypeMaskNonDmgHeal
	TypeMaskAll SpellTypeMask = TypeMaskDamage | TypeMaskHeal | TypeMaskNonDmgHeal
)

type SpellPhaseMask uint8

const (
	PhaseNone   SpellPhaseMask = 0
	PhaseCast   SpellPhaseMask = 1 << iota
	PhaseHit
	PhaseFinish
)

type HitMask uint8

const (
	HitNone   HitMask = 0
	HitNormal HitMask = 1 << iota
	HitCrit
	HitMiss
	HitImmune
)

type Entry struct {
	SpellID      spell.SpellID
	Flags        ProcFlag
	SpellType    SpellTypeMask
	SpellPhase   SpellPhaseMask
	HitFlags     HitMask
	Chance       float64
	PPM          float64
	Cooldown     time.Duration
	Charges      int32
	TriggerSpell spell.SpellID
	lastProc     time.Time
}

type Manager struct {
	entries []Entry
	rng     *rand.Rand
}

func NewManager() *Manager {
	return &Manager{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *Manager) Register(entry Entry) {
	m.entries = append(m.entries, entry)
}

func (m *Manager) Unregister(spellID spell.SpellID) {
	var remaining []Entry
	for _, e := range m.entries {
		if e.SpellID != spellID {
			remaining = append(remaining, e)
		}
	}
	m.entries = remaining
}

type ProcEvent struct {
	Flag       ProcFlag
	SpellID    spell.SpellID
	TypeMask   SpellTypeMask
	PhaseMask  SpellPhaseMask
	HitMask    HitMask
	SourceID   uint64
	TargetID   uint64
	Damage     float64
	Healing    float64
}

type ProcResult struct {
	TriggeredSpell spell.SpellID
	SourceID       uint64
	TargetID       uint64
}

func (m *Manager) Check(event ProcEvent) []ProcResult {
	var results []ProcResult
	now := time.Now()

	for i := range m.entries {
		e := &m.entries[i]

		if e.Flags&event.Flag == 0 {
			continue
		}
		if e.SpellType != TypeMaskNone && e.SpellType&event.TypeMask == 0 {
			continue
		}
		if e.SpellPhase != PhaseNone && e.SpellPhase&event.PhaseMask == 0 {
			continue
		}
		if e.HitFlags != HitNone && e.HitFlags&event.HitMask == 0 {
			continue
		}

		if e.Cooldown > 0 && now.Sub(e.lastProc) < e.Cooldown {
			continue
		}

		if !m.rollChance(e) {
			continue
		}

		e.lastProc = now

		if e.Charges > 0 {
			e.Charges--
			if e.Charges == 0 {
				m.entries = append(m.entries[:i], m.entries[i+1:]...)
				i--
			}
		}

		results = append(results, ProcResult{
			TriggeredSpell: e.TriggerSpell,
			SourceID:       event.SourceID,
			TargetID:       event.TargetID,
		})
	}

	return results
}

func (m *Manager) rollChance(e *Entry) bool {
	if e.PPM > 0 {
		return m.rng.Float64() < e.PPM
	}
	if e.Chance > 0 {
		return m.rng.Float64() < e.Chance
	}
	return true
}
