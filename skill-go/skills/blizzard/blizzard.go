package blizzard

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
	"time"
)

// Spell 10 — Blizzard (channeled AoE)
// Engine-driven: cast via eng.CastSpell(caster, &Info, engine.WithDestPos(x,y,z))
// Then call RegisterScripts to set up aura creation on channel start.
var Info = spell.SpellInfo{
	ID:          10,
	Name:        "Blizzard",
	CastTime:    0,
	Duration:    8000,
	PowerCost:   320,
	PowerType:   0,
	RangeMax:    30,
	IsChanneled: true,
	Attributes:  spell.AttrChanneled,
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

// RegisterScripts sets up Blizzard's channeled area aura creation.
// Uses OnSpellLaunch to create the aura when channeling starts,
// and OnSpellCancel to clean it up.
func RegisterScripts(eng *engine.Engine, caster *unit.Unit) {
	var activeAura *aura.Aura

	eng.Bus().Subscribe(event.OnSpellLaunch, func(e event.Event) {
		if e.SpellID != uint32(Info.ID) {
			return
		}

		// Extract DestPos from event
		destX, _ := e.Extra["destX"].(float64)
		destY, _ := e.Extra["destY"].(float64)
		destZ, _ := e.Extra["destZ"].(float64)

		ei := &Info.Effects[0]
		a := aura.NewAura(spell.SpellID(Info.ID), caster.ID(), caster.ID(), aura.AuraPeriodicDamage, 8*time.Second)
		a.SpellName = Info.Name
		a.IsAreaAura = true
		a.AreaCenter = [3]float64{destX, destY, destZ}
		a.AreaRadius = float64(ei.MiscValue)
		a.Effects = []aura.AuraEffect{
			{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, Amount: ei.BasePoints, BonusCoeff: ei.BonusCoeff, Period: 1e9},
		}
		eng.AuraMgr().ApplyAura(caster, caster, a)
		activeAura = a

		// Find the spell in activeSpells to bind OnCancel
		for _, s := range caster.GetActiveSpells() {
			if s.ID == spell.SpellID(Info.ID) && s.State == spell.StateChanneling {
				s.OnCancel = func() {
					if activeAura != nil {
						eng.AuraMgr().RemoveAuraFromHosts(activeAura, caster, caster, aura.RemoveByCancel)
						activeAura = nil
					}
				}
				break
			}
		}
	})

	// Handle spell cancel (interrupt) — remove aura early
	eng.Bus().Subscribe(event.OnSpellCancel, func(e event.Event) {
		if e.SpellID != uint32(Info.ID) {
			return
		}
		if activeAura != nil {
			eng.AuraMgr().RemoveAuraFromHosts(activeAura, caster, caster, aura.RemoveByCancel)
			activeAura = nil
		}
	})
}
