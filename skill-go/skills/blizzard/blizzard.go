package blizzard

import (
	"skill-go/pkg/engine"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/unit"
)

// 法术 10 — 暴风雪（引导 AoE）
// 引擎驱动：通过 eng.CastSpell(caster, &Info, engine.WithDestPos(x,y,z)) 施放
// 效果管线在 Cast() 时自动创建区域光环。
// 无需 RegisterScripts — 光环创建和取消清理完全自动。
var Info = spellcore.SpellInfo{
	ID:             10,
	Name:           "Blizzard",
	CastTime:       0,
	Duration:       8000,
	PowerCost:      320,
	PowerType:      0,
	RangeMax:       30,
	IsChanneled:    true,
	Attributes:     spellcore.AttrChanneled,
	InterruptFlags: spellcore.InterruptMovement,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectApplyAura,
			BasePoints:  25,
			BonusCoeff:  0.042,
			AuraType:    uint16(spellcore.AuraPeriodicDamage),
			AuraPeriod:  1000,
			TargetA:     spellcore.TargetDestTargetEnemy,
			TargetB:     spellcore.TargetUnitDestAreaEnemy,
			Radius:      8.0,
			MiscValue:   8,
		},
	},
}

// RegisterScripts 对暴风雪是空操作。光环创建和取消清理由效果管线和 Cancel() 自动处理。
func RegisterScripts(registry *spellcore.Registry, caster *unit.Unit, eng *engine.Engine) {}

// RegisterSpells 将暴风雪的法术注册到配置表中。
func RegisterSpells(store *spellcore.SpellStore) {
	store.Register(&Info)
}
