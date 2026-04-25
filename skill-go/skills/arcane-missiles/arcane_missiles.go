package arcanemissiles

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
	"time"
)

// Spell 5143 — Arcane Missiles (channeled)
// Engine-driven: cast via eng.CastSpell(caster, &Info, engine.WithTarget(id))
// Then call RegisterScripts to set up aura creation and missile triggering.
var Info = spell.SpellInfo{
	ID:          5143,
	Name:        "Arcane Missiles",
	CastTime:    0,
	Duration:    3000,
	PowerCost:   85,
	PowerType:   0,
	RangeMax:    30,
	IsChanneled: true,
	Attributes:     spell.AttrChanneled | spell.AttrBreakOnMove,
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

// RegisterScripts sets up Arcane Missiles' channeled aura and periodic missile triggering.
func RegisterScripts(eng *engine.Engine, caster *unit.Unit) {
	var activeAura *aura.Aura

	// On channel start: create PeriodicTriggerSpell aura
	eng.Bus().Subscribe(event.OnSpellLaunch, func(e event.Event) {
		if e.SpellID != uint32(Info.ID) {
			return
		}

		targetID := e.TargetID
		target := eng.GetUnit(targetID)
		if target == nil {
			return
		}

		ei := &Info.Effects[0]
		a := aura.NewAura(spell.SpellID(Info.ID), caster.ID(), targetID, aura.AuraPeriodicTriggerSpell, 3*time.Second)
		a.SpellName = Info.Name
		a.Effects = []aura.AuraEffect{
			{
				EffectIndex:    ei.EffectIndex,
				AuraType:       aura.AuraPeriodicTriggerSpell,
				Amount:         ei.BasePoints,
				Period:         time.Duration(ei.AuraPeriod) * time.Millisecond,
				TriggerSpellID: ei.TriggerSpellID,
			},
		}
		eng.AuraMgr().ApplyAura(caster, target, a)
		activeAura = a
	})

	// On aura tick: trigger missile spell
	eng.Bus().Subscribe(event.OnAuraTick, func(e event.Event) {
		if e.SpellID != uint32(Info.ID) {
			return
		}
		triggerID, ok := e.Extra["triggerSpellID"]
		if !ok {
			return
		}
		if triggerID != uint32(MissileInfo.ID) {
			return
		}
		eng.CastSpell(caster, &MissileInfo,
			engine.WithTarget(e.TargetID),
			engine.WithTriggered(),
		)
	})

	// Handle spell cancel — remove aura early
	eng.Bus().Subscribe(event.OnSpellCancel, func(e event.Event) {
		if e.SpellID != uint32(Info.ID) {
			return
		}
		if activeAura != nil {
			target := eng.GetUnit(activeAura.TargetID)
			if target != nil {
				eng.AuraMgr().RemoveAuraFromHosts(activeAura, caster, target, aura.RemoveByCancel)
			}
			activeAura = nil
		}
	})
}
