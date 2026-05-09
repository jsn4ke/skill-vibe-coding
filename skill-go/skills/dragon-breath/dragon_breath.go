package dragonbreath

import (
	"skill-go/pkg/engine"
	"skill-go/pkg/spellcore"
)

// 法术 31661 — 龙息术（锥形即时伤害）
// 效果管线通过 TargetUnitConeEnemy24 解析施法者前方锥形区域内的敌方目标。
// 无需 RegisterScripts — 纯即时伤害。
var Info = spellcore.SpellInfo{
	ID:        31661,
	Name:      "Dragon's Breath",
	CastTime:  0,
	PowerCost: 185,
	PowerType: 0,
	RangeMax:  10,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectSchoolDamage,
			BasePoints:  150,
			BonusCoeff:  0.15,
			TargetA:     spellcore.TargetUnitConeEnemy24,
			MiscValue:   10, // 锥形距离
		},
	},
}

// RegisterScripts 对龙息术是空操作。
func RegisterScripts(registry *spellcore.Registry, caster interface{}, eng *engine.Engine) {}

// RegisterSpells 将龙息术注册到配置表。
func RegisterSpells(store *spellcore.SpellStore) {
	store.Register(&Info)
}
