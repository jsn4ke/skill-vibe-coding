package arcanemissiles

import (
	"skill-go/pkg/engine"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/unit"
)

// 法术 5143 — 奥术飞弹（引导）
// 引擎驱动：通过 eng.CastSpell(caster, &Info, engine.WithTarget(id)) 施放
// 光环创建和取消清理完全自动。
// RegisterScripts 仅设置周期飞弹触发。
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

// RegisterScripts 设置周期飞弹触发。
// 光环创建由 Cast() 期间的效果管线处理。
// 取消清理由 Cancel() 的 RemoveOwnedAurasBySpellID 处理。
func RegisterScripts(registry *spellcore.Registry, caster *unit.Unit, eng *engine.Engine) {
	// 周期 tick 时：触发飞弹法术
	registry.RegisterAuraHook(spellcore.SpellID(Info.ID), spellcore.AuraHookOnPeriodic, func(ctx *spellcore.AuraContext) {
		eng.CastSpell(caster, &MissileInfo,
			engine.WithTarget(ctx.TargetID),
			engine.WithTriggered(),
		)
	})
}
