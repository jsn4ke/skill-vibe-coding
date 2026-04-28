package fireball

import (
	"skill-go/pkg/spellcore"
)

// 法术 25306 — 火球术
// 引擎驱动：通过 eng.CastSpell(caster, &Info, engine.WithTarget(id)) 施放
// EffectApplyAura（索引 1）通过效果管线 + OnAuraCreated 自动创建 DoT 光环。
var Info = spellcore.SpellInfo{
	ID:             25306,
	Name:           "Fireball",
	CastTime:       3500,
	RangeMax:       35,
	PowerCost:      410,
	PowerType:      0,
	Duration:       8000,
	InterruptFlags: spellcore.InterruptMovement,
	Speed:          20.0,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex:  0,
			EffectType:   spellcore.EffectSchoolDamage,
			BasePoints:   678,
			BaseDieSides: 164,
			BonusCoeff:   1.0,
			TargetA:      spellcore.TargetUnitTargetEnemy,
		},
		{
			EffectIndex: 1,
			EffectType:  spellcore.EffectApplyAura,
			BasePoints:  19,
			BonusCoeff:  0.125,
			AuraType:    uint16(spellcore.AuraPeriodicDamage),
			AuraPeriod:  2000,
			TargetA:     spellcore.TargetUnitTargetEnemy,
		},
	},
}
