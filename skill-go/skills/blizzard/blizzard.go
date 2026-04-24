package blizzard

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"time"
)

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

func CastBlizzard(caster spell.Caster, destX, destY, destZ float64, auraMgr *aura.Manager, bus *event.Bus) (*spell.Spell, spell.SpellCastResult) {
	s := spell.NewSpell(spell.SpellID(Info.ID), &Info, caster, spell.TriggeredNone)
	s.Targets.DestPos = [3]float64{destX, destY, destZ}
	s.Bus = bus

	result := s.Prepare()
	if result != spell.CastOK {
		return s, result
	}

	if s.State == spell.StatePreparing {
		s.Update(int32(Info.CastTime))
	}

	if s.State == spell.StateChanneling {
		ei := &Info.Effects[0]
		a := aura.NewAura(spell.SpellID(Info.ID), caster.GetID(), 0, aura.AuraPeriodicDamage, 8*time.Second)
		a.SpellName = Info.Name
		a.IsAreaAura = true
		a.AreaCenter = [3]float64{destX, destY, destZ}
		a.AreaRadius = float64(ei.MiscValue)
		a.Effects = []aura.AuraEffect{
			{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, Amount: ei.BasePoints, BonusCoeff: ei.BonusCoeff, Period: 1e9},
		}
		auraMgr.AddAura(a)

		spellID := spell.SpellID(Info.ID)
		s.OnCancel = func() {
			auraMgr.RemoveAurasBySpellID(0, spellID)
		}
	}

	return s, result
}
