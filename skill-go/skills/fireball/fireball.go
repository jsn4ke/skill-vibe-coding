package fireball

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/spell"
)

// Spell 25306 — Fireball
// Engine-driven: cast via eng.CastSpell(caster, &Info, engine.WithTarget(id))
// EffectApplyAura (index 1) creates DoT aura automatically via effect pipeline + OnAuraCreated.
var Info = spell.SpellInfo{
	ID:         25306,
	Name:       "Fireball",
	CastTime:   3500,
	RangeMax:   35,
	PowerCost:  410,
	PowerType:  0,
	Duration:   8000,
	Attributes:     spell.AttrBreakOnMove,
	InterruptFlags: spell.InterruptMovement,
	Speed:      20.0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex:  0,
			EffectType:   spell.EffectSchoolDamage,
			BasePoints:   678,
			BaseDieSides: 164,
			BonusCoeff:   1.0,
			TargetA:      spell.TargetUnitTargetEnemy,
		},
		{
			EffectIndex: 1,
			EffectType:  spell.EffectApplyAura,
			BasePoints:  19,
			BonusCoeff:  0.125,
			AuraType:    uint16(aura.AuraPeriodicDamage),
			AuraPeriod:  2000,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
	},
}
