package livingbomb

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
)

// Spell 44457 — Living Bomb (cast entry point)
var Info = spell.SpellInfo{
	ID:        44457,
	Name:      "Living Bomb",
	CastTime:  0,
	RangeMax:  35,
	PowerCost: 470,
	PowerType: 0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectDummy,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
	},
}

// Spell 217694 — Living Bomb Periodic (DoT)
var PeriodicInfo = spell.SpellInfo{
	ID:        217694,
	Name:      "Living Bomb Periodic",
	CastTime:  0,
	Duration:  4000,
	RangeMax:  40,
	PowerCost: 0,
	PowerType: 0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectApplyAura,
			BonusCoeff:  0.06,
			AuraType:    uint16(aura.AuraPeriodicDamage),
			AuraPeriod:  1000,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
		{
			EffectIndex: 1,
			EffectType:  spell.EffectDummy,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
	},
}

// Spell 44461 — Living Bomb Explode (AoE)
var ExplosionInfo = spell.SpellInfo{
	ID:        44461,
	Name:      "Living Bomb Explode",
	CastTime:  0,
	RangeMax:  50000,
	PowerCost: 0,
	PowerType: 0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectDummy,
			TargetA:     spell.TargetUnitAreaEnemy,
		},
		{
			EffectIndex: 1,
			EffectType:  spell.EffectSchoolDamage,
			BonusCoeff:  0.14,
			TargetA:     spell.TargetUnitAreaEnemy,
		},
	},
}

// RegisterScripts registers all Living Bomb script hooks.
// Uses engine.CastSpell(WithTriggered) for all triggered spells.
func RegisterScripts(registry *script.Registry, caster *unit.Unit, eng *engine.Engine, aoeSelector spell.AoESelector) {
	// 44457: OnEffectHit — intercept Dummy, cast periodic spell
	registry.RegisterSpellHook(44457, script.HookOnEffectHit, func(ctx *script.SpellContext) {
		ctx.PreventDefault = true
		targetID := ctx.Spell.Targets.UnitTargetID
		eng.CastSpell(caster, &PeriodicInfo,
			engine.WithTarget(targetID),
			engine.WithTriggered(),
			engine.WithSpellValues(map[uint8]float64{2: 1}),
		)
	})

	// 217694: AfterRemove — explode on expire only
	registry.RegisterAuraHook(217694, script.AuraHookAfterRemove, func(ctx *script.AuraContext) {
		if ctx.RemoveMode != uint8(aura.RemoveByExpire) {
			return
		}
		a, ok := ctx.Aura.(*aura.Aura)
		if !ok || a == nil {
			return
		}
		canSpread := float64(0)
		if a.SpellValues != nil {
			canSpread = a.SpellValues[2]
		}
		castExplosion(eng, caster, ctx.TargetID, canSpread, aoeSelector)
	})

	// 44461: OnEffectHit EFFECT_1 — after SchoolDamage, spread to hit targets
	registry.RegisterSpellHook(44461, script.HookOnEffectHit, func(ctx *script.SpellContext) {
		if ctx.EffectIndex != 1 {
			return
		}
		canSpread := ctx.Spell.SpellValues[2]
		if canSpread <= 0 {
			return
		}
		for _, ti := range ctx.Spell.TargetInfos {
			eng.CastSpell(caster, &PeriodicInfo,
				engine.WithTarget(ti.TargetID),
				engine.WithTriggered(),
				engine.WithSpellValues(map[uint8]float64{2: 0}),
			)
		}
		ctx.Spell.SpellValues[2] = 0
	})
}

// castExplosion creates and casts the explosion spell via engine.
func castExplosion(eng *engine.Engine, caster *unit.Unit, carrierTargetID uint64, canSpread float64, aoeSelector spell.AoESelector) {
	carrierPos := caster.GetTargetPosition(carrierTargetID)
	eng.CastSpell(caster, &ExplosionInfo,
		engine.WithTriggered(),
		engine.WithAoE(aoeSelector, [3]float64{carrierPos.GetX(), carrierPos.GetY(), carrierPos.GetZ()}, carrierTargetID),
		engine.WithSpellValues(map[uint8]float64{2: canSpread}),
	)
}
