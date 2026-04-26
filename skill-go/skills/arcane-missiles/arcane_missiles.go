package arcanemissiles

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
)

// Spell 5143 — Arcane Missiles (channeled)
// Engine-driven: cast via eng.CastSpell(caster, &Info, engine.WithTarget(id))
// Aura creation and cancel cleanup are fully automatic.
// RegisterScripts only sets up periodic missile triggering.
var Info = spell.SpellInfo{
	ID:          5143,
	Name:        "Arcane Missiles",
	CastTime:    0,
	Duration:    3000,
	PowerCost:   85,
	PowerType:   0,
	RangeMax:    30,
	IsChanneled: true,
	Attributes:     spell.AttrChanneled,
	InterruptFlags: spell.InterruptMovement,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex:    0,
			EffectType:     spell.EffectApplyAura,
			BasePoints:     1,
			AuraType:       uint16(aura.AuraPeriodicTriggerSpell),
			AuraPeriod:     1000,
			TriggerSpellID: 7268,
			TargetA:        spell.TargetUnitTargetEnemy,
		},
	},
}

// Spell 7268 — Arcane Missile (triggered by periodic tick)
var MissileInfo = spell.SpellInfo{
	ID:        7268,
	Name:      "Arcane Missile",
	CastTime:  0,
	RangeMax:  30,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectSchoolDamage,
			BasePoints:  24,
			BonusCoeff:  0.132,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
	},
}

// RegisterScripts sets up periodic missile triggering.
// Aura creation is handled by the effect pipeline during Cast().
// Cancel cleanup is handled by Cancel()'s RemoveOwnedAurasBySpellID.
func RegisterScripts(registry *script.Registry, caster *unit.Unit, eng *engine.Engine) {
	// On periodic tick: trigger missile spell
	registry.RegisterAuraHook(spell.SpellID(Info.ID), script.AuraHookOnPeriodic, func(ctx *script.AuraContext) {
		eng.CastSpell(caster, &MissileInfo,
			engine.WithTarget(ctx.TargetID),
			engine.WithTriggered(),
		)
	})
}
