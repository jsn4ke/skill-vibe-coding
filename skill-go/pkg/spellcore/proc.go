package spellcore

import (
	"math/rand"
	"time"
)

// ProcFlag 表示触发条件的位掩码，对齐 TC 的 ProcFlags。
type ProcFlag uint32

const (
	FlagNone ProcFlag = 0
	FlagKill ProcFlag = 1 << iota
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

// SpellTypeMask 表示法术类型的位掩码。
type SpellTypeMask uint8

const (
	TypeMaskNone   SpellTypeMask = 0
	TypeMaskDamage SpellTypeMask = 1 << iota
	TypeMaskHeal
	TypeMaskNonDmgHeal
	TypeMaskAll SpellTypeMask = TypeMaskDamage | TypeMaskHeal | TypeMaskNonDmgHeal
)

// SpellPhaseMask 表示法术阶段的位掩码。
type SpellPhaseMask uint8

const (
	PhaseNone SpellPhaseMask = 0
	PhaseCast SpellPhaseMask = 1 << iota
	PhaseHit
	PhaseFinish
)

// ProcHitMask 表示命中结果的位掩码，用于过滤触发条件。
// 使用 Proc 前缀避免与 spell.HitResult 常量冲突。
type ProcHitMask uint8

const (
	ProcHitNone   ProcHitMask = 0
	ProcHitNormal ProcHitMask = 1 << iota
	ProcHitCrit
	ProcHitMiss
	ProcHitImmune
)

// Entry 表示一个触发器条目，对齐 TC 的 ProcTriggerEntry。
type Entry struct {
	SpellID      SpellID
	Flags        ProcFlag
	SpellType    SpellTypeMask
	SpellPhase   SpellPhaseMask
	HitFlags     ProcHitMask
	Chance       float64
	PPM          float64
	Cooldown     time.Duration
	Charges      int32
	TriggerSpell SpellID
	lastProc     time.Time
}

// ProcManager 管理触发系统，对齐 TC 的 ProcSystem。
type ProcManager struct {
	entries []Entry
	rng     *rand.Rand
}

// NewProcManager 创建一个新的触发器管理器。
func NewProcManager() *ProcManager {
	return &ProcManager{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Register 注册一个触发器条目。
func (m *ProcManager) Register(entry Entry) {
	m.entries = append(m.entries, entry)
}

// Unregister 按 SpellID 移除所有匹配的触发器条目。
func (m *ProcManager) Unregister(spellID SpellID) {
	var remaining []Entry
	for _, e := range m.entries {
		if e.SpellID != spellID {
			remaining = append(remaining, e)
		}
	}
	m.entries = remaining
}

// ProcEvent 表示一次触发事件。
type ProcEvent struct {
	Flag      ProcFlag
	SpellID   SpellID
	TypeMask  SpellTypeMask
	PhaseMask SpellPhaseMask
	HitMask   ProcHitMask
	SourceID  uint64
	TargetID  uint64
	Damage    float64
	Healing   float64
}

// ProcResult 表示触发判定的结果。
type ProcResult struct {
	TriggeredSpell SpellID
	SourceID       uint64
	TargetID       uint64
}

// Check 检查所有触发器条目是否匹配当前事件，返回所有触发的结果。
func (m *ProcManager) Check(event ProcEvent) []ProcResult {
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
		if e.HitFlags != ProcHitNone && e.HitFlags&event.HitMask == 0 {
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

func (m *ProcManager) rollChance(e *Entry) bool {
	if e.PPM > 0 {
		return m.rng.Float64() < e.PPM
	}
	if e.Chance > 0 {
		return m.rng.Float64() < e.Chance
	}
	return true
}
