package combat

import (
	"math/rand"
	"skill-go/pkg/spellcore"
)

// SettlementContext 伤害/治疗结算上下文，对齐 TC 的 DoDamageAndTriggers 入参。
type SettlementContext struct {
	SourceID         uint64  // 攻击者/施法者 ID
	TargetID         uint64  // 目标 ID
	SpellID          uint32  // 法术 ID
	Damage           float64 // 原始伤害值（减伤前）
	Healing          float64 // 治疗值
	IsPeriodic       bool    // 是否周期性效果（光环 tick）
	IsCrit           bool    // 是否暴击
	SpellName        string  // 法术名称
	School           DamageSchool
	TargetArmor      float64 // 目标护甲值
	TargetResistance float64 // 目标对应学校抗性值
	TargetAbsorb     float64 // 目标当前护盾值
}

// DamageSchool 表示伤害的学校类型，对齐 TC 的 SpellSchoolMask。
type DamageSchool uint8

const (
	SchoolPhysical DamageSchool = iota
	SchoolHoly
	SchoolFire
	SchoolNature
	SchoolFrost
	SchoolShadow
	SchoolArcane
)

// AttackRollResult 保存命中判定的完整结果。
type AttackRollResult struct {
	HitResult  spellcore.HitResult
	IsCrit     bool
	DamageMult float64 // 1.0 普通命中，1.5/2.0 暴击，0.0 未命中
}

// AttackRoll 执行命中判定，对齐 TC 的 Unit::RollMeleeOutcome。
// 使用确定性 RNG 参数确保测试可复现。
// 优先级：dodge > parry > block > crit > normal。
func AttackRoll(critChance, dodgeChance, parryChance, blockChance float64, rng *rand.Rand) AttackRollResult {
	roll := func() float64 {
		if rng != nil {
			return rng.Float64()
		}
		return 0.5
	}

	if r := roll(); r < dodgeChance {
		return AttackRollResult{HitResult: spellcore.HitDodge, DamageMult: 0}
	}
	if r := roll(); r < parryChance {
		return AttackRollResult{HitResult: spellcore.HitEvade, DamageMult: 0}
	}
	if r := roll(); r < blockChance {
		return AttackRollResult{HitResult: spellcore.HitNormal, DamageMult: 1.0}
	}
	if r := roll(); r < critChance {
		return AttackRollResult{HitResult: spellcore.HitCrit, IsCrit: true, DamageMult: 1.5}
	}
	return AttackRollResult{HitResult: spellcore.HitNormal, DamageMult: 1.0}
}

// MitigateDamage 对原始伤害应用减伤计算，对齐 TC 的 CalcArmorReducedDamage + CalcSpellResistance。
// Physical 使用护甲公式，其他学校使用抗性公式。
func MitigateDamage(rawDamage, armor, resistance float64, school DamageSchool) float64 {
	if rawDamage <= 0 {
		return 0
	}
	switch school {
	case SchoolPhysical:
		if armor <= 0 {
			return rawDamage
		}
		// TC 护甲减伤公式：damage * (1 - armor / (armor + C))，C 为等级常数
		return rawDamage * (1 - armor/(armor+467.5))
	default:
		if resistance <= 0 {
			return rawDamage
		}
		// TC 抗性减伤公式：damage * (1 - resistance / (resistance + C))
		return rawDamage * (1 - resistance/(resistance+150))
	}
}

// AbsorbResult 伤害吸收结果。
type AbsorbResult struct {
	RemainingDamage float64 // 吸收后的剩余伤害
	Absorbed        float64 // 被吸收的伤害量
}

// ApplyAbsorb 处理护盾吸收，对齐 TC 的 CalcAbsorbResist 中的吸收逻辑。
func ApplyAbsorb(damage, absorbAmount float64) AbsorbResult {
	if absorbAmount <= 0 || damage <= 0 {
		return AbsorbResult{RemainingDamage: damage, Absorbed: 0}
	}
	if damage <= absorbAmount {
		return AbsorbResult{RemainingDamage: 0, Absorbed: damage}
	}
	return AbsorbResult{RemainingDamage: damage - absorbAmount, Absorbed: absorbAmount}
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
