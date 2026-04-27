package combat

import (
	"math/rand"

	"skill-go/pkg/entity"
	"skill-go/pkg/stat"
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

// DamageInfo 包含一次伤害的完整信息。
type DamageInfo struct {
	Attacker *entity.Entity
	Target   *entity.Entity
	Base     float64
	Result   HitResult
	Final    float64
	SpellID  uint32
}

// CombatManager 管理战斗状态。
type CombatManager struct {
	inCombat map[entity.EntityID]bool
}

// NewCombatManager 创建一个新的战斗管理器。
func NewCombatManager() *CombatManager {
	return &CombatManager{
		inCombat: make(map[entity.EntityID]bool),
	}
}

// EnterCombat 将实体置入战斗状态。
func (cm *CombatManager) EnterCombat(e *entity.Entity) {
	cm.inCombat[e.ID] = true
	e.State = e.State.Set(entity.StateInCombat)
}

// LeaveCombat 将实体移出战斗状态。
func (cm *CombatManager) LeaveCombat(e *entity.Entity) {
	delete(cm.inCombat, e.ID)
	e.State = e.State.Clear(entity.StateInCombat)
}

// IsInCombat 判断实体是否处于战斗状态。
func (cm *CombatManager) IsInCombat(e *entity.Entity) bool {
	return cm.inCombat[e.ID]
}

// CalculateDamage 根据基础伤害和攻击者属性计算最终伤害。
func CalculateDamage(base float64, attackerStats *stat.StatSet) float64 {
	ap := attackerStats.Get(stat.AttackPower)
	sp := attackerStats.Get(stat.SpellPower)
	bonus := ap*0.1 + sp*0.3
	return base + bonus
}

// RollHitResult 根据攻击者属性和目标状态进行命中判定。
func RollHitResult(attackerStats *stat.StatSet, targetState entity.UnitState) HitResult {
	if targetState.Has(entity.StateDead) {
		return HitEvade
	}

	critChance := attackerStats.Get(stat.CritChance)
	r := rand.Float64()

	switch {
	case r < 0.05:
		return HitMiss
	case r < 0.05+critChance:
		return HitCrit
	default:
		return HitNormal
	}
}

// ApplyDamageInfo 根据命中结果计算最终伤害值。
func ApplyDamageInfo(info *DamageInfo) {
	switch info.Result {
	case HitNormal:
		info.Final = info.Base
	case HitCrit:
		info.Final = info.Base * 2.0
	case HitMiss, HitEvade, HitImmune:
		info.Final = 0
	case HitDodge, HitParry, HitBlock:
		info.Final = 0
	}
}
