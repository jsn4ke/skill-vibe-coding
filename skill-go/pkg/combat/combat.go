package combat

import (
	"skill-go/pkg/proc"
	"skill-go/pkg/spell"
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
func BuildProcEvent(ctx SettlementContext) proc.ProcEvent {
	var flag proc.ProcFlag
	var typeMask proc.SpellTypeMask

	if ctx.Damage > 0 {
		if ctx.IsPeriodic {
			flag = proc.FlagPeriodicDamageDealt
		} else {
			flag = proc.FlagSpellDamageDealt
		}
		typeMask = proc.TypeMaskDamage
	} else if ctx.Healing > 0 {
		if ctx.IsPeriodic {
			flag = proc.FlagPeriodicHealDealt
		} else {
			flag = proc.FlagSpellHealDealt
		}
		typeMask = proc.TypeMaskHeal
	} else {
		flag = proc.FlagSpellHit
		typeMask = proc.TypeMaskNonDmgHeal
	}

	hitMask := proc.HitNormal
	if ctx.IsCrit {
		hitMask |= proc.HitCrit
	}

	return proc.ProcEvent{
		Flag:      flag,
		SpellID:   spell.SpellID(ctx.SpellID),
		TypeMask:  typeMask,
		PhaseMask: proc.PhaseHit,
		HitMask:   hitMask,
		SourceID:  ctx.SourceID,
		TargetID:  ctx.TargetID,
		Damage:    ctx.Damage,
		Healing:   ctx.Healing,
	}
}

// BuildVictimProcEvent 从结算上下文构建受击者侧 Proc 事件。
func BuildVictimProcEvent(ctx SettlementContext) proc.ProcEvent {
	var flag proc.ProcFlag
	var typeMask proc.SpellTypeMask

	if ctx.Damage > 0 {
		if ctx.IsPeriodic {
			flag = proc.FlagPeriodicDamageTaken
		} else {
			flag = proc.FlagSpellDamageTaken
		}
		typeMask = proc.TypeMaskDamage
	} else if ctx.Healing > 0 {
		if ctx.IsPeriodic {
			flag = proc.FlagPeriodicHealTaken
		} else {
			flag = proc.FlagSpellHealTaken
		}
		typeMask = proc.TypeMaskHeal
	} else {
		flag = proc.FlagSpellHit
		typeMask = proc.TypeMaskNonDmgHeal
	}

	hitMask := proc.HitNormal
	if ctx.IsCrit {
		hitMask |= proc.HitCrit
	}

	return proc.ProcEvent{
		Flag:      flag,
		SpellID:   spell.SpellID(ctx.SpellID),
		TypeMask:  typeMask,
		PhaseMask: proc.PhaseHit,
		HitMask:   hitMask,
		SourceID:  ctx.SourceID,
		TargetID:  ctx.TargetID,
		Damage:    ctx.Damage,
		Healing:   ctx.Healing,
	}
}
