package arcaneintellect

import (
	"skill-go/pkg/engine"
	"skill-go/pkg/spellcore"
)

// 法术 1459 — 奥术智慧（纯瞬发自增益 buff）
// 效果管线在 Cast() 时自动创建 AuraModSpellPower 光环。
// 无需 RegisterScripts — 光环创建和移除完全自动。
var Info = spellcore.SpellInfo{
	ID:        1459,
	Name:      "Arcane Intellect",
	CastTime:  0,
	Duration:  30000,
	PowerCost: 120,
	PowerType: 0,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectApplyAura,
			AuraType:    uint16(spellcore.AuraModSpellPower),
			BasePoints:  60,
			TargetA:     spellcore.TargetUnitCaster,
		},
	},
}

// RegisterScripts 对奥术智慧是空操作。
func RegisterScripts(registry *spellcore.Registry, caster interface{}, eng *engine.Engine) {}

// RegisterSpells 将奥术智慧注册到配置表。
func RegisterSpells(store *spellcore.SpellStore) {
	store.Register(&Info)
}
