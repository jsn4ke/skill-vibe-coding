package livingbomb

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/event"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
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

// RegisterScripts registers all Living Bomb script hooks into the registry.
// aoeSelector is used by the explosion spell to find nearby enemies; may be nil.
func RegisterScripts(registry *script.Registry, caster spell.Caster, auraMgr *aura.Manager, bus *event.Bus, aoeSelector spell.AoESelector) {
	// 44457: OnEffectHit — intercept Dummy, cast periodic spell
	registry.RegisterSpellHook(44457, script.HookOnEffectHit, func(ctx *script.SpellContext) {
		ctx.PreventDefault = true
		targetID := ctx.Spell.Targets.UnitTargetID
		castPeriodicSpell(caster, targetID, auraMgr, bus, map[uint8]float64{2: 1})
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
		castExplosionSpell(caster, ctx.TargetID, a, canSpread, aoeSelector, bus)
	})

	// 44461: OnEffectHit EFFECT_1 — after SchoolDamage, spread to the hit target
	registry.RegisterSpellHook(44461, script.HookOnEffectHit, func(ctx *script.SpellContext) {
		if ctx.EffectIndex != 1 {
			return
		}
		canSpread := ctx.Spell.SpellValues[2]
		if canSpread <= 0 {
			return
		}
		// ctx.Spell.TargetInfos has all AoE targets; spread to each
		// But this hook fires once per target per effect, so track which targets we've spread to
		for _, ti := range ctx.Spell.TargetInfos {
			castPeriodicSpell(caster, ti.TargetID, auraMgr, bus, map[uint8]float64{2: 0})
		}
		// Set canSpread to 0 to prevent re-spreading on subsequent target-effect combos
		ctx.Spell.SpellValues[2] = 0
	})
}

// CastLivingBomb is the entry point — player casts spell 44457.
func CastLivingBomb(caster spell.Caster, targetID uint64, auraMgr *aura.Manager, bus *event.Bus) (*spell.Spell, spell.SpellCastResult) {
	s := spell.NewSpell(44457, &Info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = targetID
	s.Bus = bus

	result := s.Prepare()
	if result != spell.CastOK {
		return s, result
	}

	if s.State == spell.StatePreparing {
		s.Update(int32(Info.CastTime))
	}

	return s, result
}

func castPeriodicSpell(caster spell.Caster, targetID uint64, auraMgr *aura.Manager, bus *event.Bus, spellValues map[uint8]float64) {
	s := spell.NewSpell(217694, &PeriodicInfo, caster, spell.TriggeredFullMask)
	s.Targets.UnitTargetID = targetID
	s.Bus = bus
	s.SpellValues = spellValues

	result := s.Prepare()
	if result != spell.CastOK {
		return
	}

	if s.State == spell.StatePreparing {
		s.Update(int32(PeriodicInfo.CastTime))
	}

	if s.State == spell.StateFinished {
		for i := range PeriodicInfo.Effects {
			ei := &PeriodicInfo.Effects[i]
			if ei.EffectType == spell.EffectApplyAura {
				a := aura.NewAura(217694, caster.GetID(), targetID, aura.AuraPeriodicDamage, 4e9)
				a.MaxStack = 1
				a.StackRule = aura.StackRefresh
				a.SpellName = PeriodicInfo.Name
				a.SpellValues = spellValues
				a.Effects = []aura.AuraEffect{
					{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, BonusCoeff: ei.BonusCoeff, Period: 1e9},
				}
				auraMgr.AddAura(a)
			}
		}
	}
}

func castExplosionSpell(caster spell.Caster, carrierTargetID uint64, sourceAura *aura.Aura, canSpread float64, aoeSelector spell.AoESelector, bus *event.Bus) {
	s := spell.NewSpell(44461, &ExplosionInfo, caster, spell.TriggeredFullMask)
	s.Bus = bus
	s.SpellValues = map[uint8]float64{2: canSpread}
	s.AoESelector = aoeSelector
	s.AoEExcludeID = carrierTargetID

	// Set AoE center to carrier target position
	carrierPos := caster.GetTargetPosition(carrierTargetID)
	s.AoECenter = [3]float64{carrierPos.GetX(), carrierPos.GetY(), carrierPos.GetZ()}

	s.Prepare()

	if s.State == spell.StatePreparing {
		s.Update(int32(ExplosionInfo.CastTime))
	}
}
