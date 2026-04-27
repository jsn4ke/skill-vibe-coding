package blizzard

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
)

// 法术 10 — 暴风雪（引导 AoE）
// 引擎驱动：通过 eng.CastSpell(caster, &Info, engine.WithDestPos(x,y,z)) 施放
// 效果管线在 Cast() 时自动创建区域光环。
// 无需 RegisterScripts — 光环创建和取消清理完全自动。
var Info = spell.SpellInfo{
	ID:          10,
	Name:        "Blizzard",
	CastTime:    0,
	Duration:    8000,
	PowerCost:   320,
	PowerType:   0,
	RangeMax:    30,
	IsChanneled: true,
	Attributes:     spell.AttrChanneled,
	InterruptFlags: spell.InterruptMovement,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectApplyAura,
			BasePoints:  25,
			BonusCoeff:  0.042,
			AuraType:    uint16(aura.AuraPeriodicDamage),
			AuraPeriod:  1000,
			TargetA:     spell.TargetDestTargetEnemy,
			TargetB:     spell.TargetUnitAreaEnemy,
			MiscValue:   8,
		},
	},
}

// RegisterScripts 对暴风雪是空操作。光环创建和取消清理由效果管线和 Cancel() 自动处理。
func RegisterScripts(registry *script.Registry, caster *unit.Unit, eng *engine.Engine) {}
