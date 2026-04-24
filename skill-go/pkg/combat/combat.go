package combat

import (
	"math/rand"

	"skill-go/pkg/entity"
	"skill-go/pkg/stat"
)

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

type DamageInfo struct {
	Attacker *entity.Entity
	Target   *entity.Entity
	Base     float64
	Result   HitResult
	Final    float64
	SpellID  uint32
}

type CombatManager struct {
	inCombat map[entity.EntityID]bool
}

func NewCombatManager() *CombatManager {
	return &CombatManager{
		inCombat: make(map[entity.EntityID]bool),
	}
}

func (cm *CombatManager) EnterCombat(e *entity.Entity) {
	cm.inCombat[e.ID] = true
	e.State = e.State.Set(entity.StateInCombat)
}

func (cm *CombatManager) LeaveCombat(e *entity.Entity) {
	delete(cm.inCombat, e.ID)
	e.State = e.State.Clear(entity.StateInCombat)
}

func (cm *CombatManager) IsInCombat(e *entity.Entity) bool {
	return cm.inCombat[e.ID]
}

func CalculateDamage(base float64, attackerStats *stat.StatSet) float64 {
	ap := attackerStats.Get(stat.AttackPower)
	sp := attackerStats.Get(stat.SpellPower)
	bonus := ap*0.1 + sp*0.3
	return base + bonus
}

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
