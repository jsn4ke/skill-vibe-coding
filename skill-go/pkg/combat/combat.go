package combat

import (
	"skill-go/pkg/spellcore"
)

// HitResult 表示命中判定的结果类型。
type HitResult uint8

const (
	HitMiss HitResult = iota
	HitNormal
	HitCrit
	HitImmune
	HitDodge
	HitParry
	HitBlock
	HitEvade
)

// SettlementContext 伤害/治疗结算上下文，对齐 TC 的 DoDamageAndTriggers 入参。
type SettlementContext struct {
	SourceID   uint64  // 攻击者/施法者 ID
	TargetID   uint64  // 目标 ID
	SpellID    uint32  // 法术 ID
	Damage     float64 // 伤害值
	Healing    float64 // 治疗值
	IsPeriodic bool    // 是否周期性效果（光环 tick）
	IsCrit     bool    // 是否暴击
	SpellName  string  // 法术名称
}

// BuildProcEvent 从结算上下文构建攻击者侧 Proc 事件，对齐 TC 的 ProcSkillsAndAuras。
func BuildProcEvent(ctx SettlementContext) spellcore.ProcEvent {
	var flag spellcore.ProcFlag
	var typeMask spellcore.SpellTypeMask

	if ctx.Damage > 0 {
		if ctx.IsPeriodic {
			flag = spellcore.FlagPeriodicDamageDealt
		} else {
			flag = spellcore.FlagSpellDamageDealt
		}
		typeMask = spellcore.TypeMaskDamage
	} else if ctx.Healing > 0 {
		if ctx.IsPeriodic {
			flag = spellcore.FlagPeriodicHealDealt
		} else {
			flag = spellcore.FlagSpellHealDealt
		}
		typeMask = spellcore.TypeMaskHeal
	} else {
		flag = spellcore.FlagSpellHit
		typeMask = spellcore.TypeMaskNonDmgHeal
	}

	hitMask := spellcore.ProcHitNormal
	if ctx.IsCrit {
		hitMask |= spellcore.ProcHitCrit
	}

	return spellcore.ProcEvent{
		Flag:      flag,
		SpellID:   spellcore.SpellID(ctx.SpellID),
		TypeMask:  typeMask,
		PhaseMask: spellcore.PhaseHit,
		HitMask:   hitMask,
		SourceID:  ctx.SourceID,
		TargetID:  ctx.TargetID,
		Damage:    ctx.Damage,
		Healing:   ctx.Healing,
	}
}

// BuildVictimProcEvent 从结算上下文构建受击者侧 Proc 事件。
func BuildVictimProcEvent(ctx SettlementContext) spellcore.ProcEvent {
	var flag spellcore.ProcFlag
	var typeMask spellcore.SpellTypeMask

	if ctx.Damage > 0 {
		if ctx.IsPeriodic {
			flag = spellcore.FlagPeriodicDamageTaken
		} else {
			flag = spellcore.FlagSpellDamageTaken
		}
		typeMask = spellcore.TypeMaskDamage
	} else if ctx.Healing > 0 {
		if ctx.IsPeriodic {
			flag = spellcore.FlagPeriodicHealTaken
		} else {
			flag = spellcore.FlagSpellHealTaken
		}
		typeMask = spellcore.TypeMaskHeal
	} else {
		flag = spellcore.FlagSpellHit
		typeMask = spellcore.TypeMaskNonDmgHeal
	}

	hitMask := spellcore.ProcHitNormal
	if ctx.IsCrit {
		hitMask |= spellcore.ProcHitCrit
	}

	return spellcore.ProcEvent{
		Flag:      flag,
		SpellID:   spellcore.SpellID(ctx.SpellID),
		TypeMask:  typeMask,
		PhaseMask: spellcore.PhaseHit,
		HitMask:   hitMask,
		SourceID:  ctx.SourceID,
		TargetID:  ctx.TargetID,
		Damage:    ctx.Damage,
		Healing:   ctx.Healing,
	}
}
