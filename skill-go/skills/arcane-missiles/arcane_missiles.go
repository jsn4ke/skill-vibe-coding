package arcanemissiles

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/effect"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"time"
)

// ensure effect handlers are registered
var _ = effect.Process

var Info = spell.SpellInfo{
	ID:          5143,
	Name:        "Arcane Missiles",
	CastTime:    0,
	Duration:    3000,
	PowerCost:   85,
	PowerType:   0,
	RangeMax:    30,
	IsChanneled: true,
	Attributes:  spell.AttrChanneled | spell.AttrBreakOnMove,
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

func CastTriggeredSpell(caster spell.Caster, targetID uint64, info *spell.SpellInfo, bus *event.Bus) {
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredFullMask)
	s.Targets.UnitTargetID = targetID
	s.Bus = bus
	s.Prepare()
}

func CastArcaneMissiles(caster spell.Caster, targetID uint64, auraMgr *aura.Manager, bus *event.Bus) (*spell.Spell, spell.SpellCastResult) {
	s := spell.NewSpell(spell.SpellID(Info.ID), &Info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = targetID
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
		a := aura.NewAura(spell.SpellID(Info.ID), caster.GetID(), targetID, aura.AuraPeriodicTriggerSpell, 3*time.Second)
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
		auraMgr.AddAura(a)

		s.OnCancel = func() {
			auraMgr.RemoveAurasBySpellID(targetID, spell.SpellID(Info.ID))
		}
	}

	return s, result
}
