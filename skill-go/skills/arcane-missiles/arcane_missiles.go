package arcanemissiles

import (
	"skill-go/pkg/spellcore"
)

// 法术 5143 — 奥术飞弹（引导）
// 引擎驱动：通过 eng.CastSpell(caster, &Info, engine.WithTarget(id)) 施放
// 光环创建和取消清理完全自动。
// 周期飞弹触发由引擎的 AuraPeriodicTriggerSpell 自动驱动，无需 RegisterScripts。
var Info = spellcore.SpellInfo{
	ID:             5143,
	Name:           "Arcane Missiles",
	CastTime:       0,
	Duration:       3000,
	PowerCost:      85,
	PowerType:      0,
	RangeMax:       30,
	IsChanneled:    true,
	Attributes:     spellcore.AttrChanneled,
	InterruptFlags: spellcore.InterruptMovement,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex:    0,
			EffectType:     spellcore.EffectApplyAura,
			BasePoints:     1,
			AuraType:       uint16(spellcore.AuraPeriodicTriggerSpell),
			AuraPeriod:     1000,
			TriggerSpellID: 7268,
			TargetA:        spellcore.TargetUnitTargetEnemy,
		},
	},
}

// 法术 7268 — 奥术飞弹（由周期 tick 触发）
var MissileInfo = spellcore.SpellInfo{
	ID:       7268,
	Name:     "Arcane Missile",
	CastTime: 0,
	RangeMax: 30,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectSchoolDamage,
			BasePoints:  24,
			BonusCoeff:  0.132,
			TargetA:     spellcore.TargetUnitTargetEnemy,
		},
	},
}

// RegisterSpells 将奥术飞弹的所有法术注册到配置表中。
func RegisterSpells(store *spellcore.SpellStore) {
	store.Register(&Info)
	store.Register(&MissileInfo)
}
