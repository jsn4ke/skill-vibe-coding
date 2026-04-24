package fireball

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"

	_ "skill-go/pkg/effect"
)

var Info = spell.SpellInfo{
	ID:         25306,
	Name:       "Fireball",
	CastTime:   3500,
	RangeMax:   35,
	PowerCost:  410,
	PowerType:  0,
	Duration:   8000,
	Attributes: spell.AttrBreakOnMove,
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

func CastFireball(caster spell.Caster, targetID uint64, auraMgr *aura.Manager, bus *event.Bus) (*spell.Spell, spell.SpellCastResult) {
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

	if s.State == spell.StateLaunched {
		s.Update(s.HitTimer)
	}

	if s.State == spell.StateFinished {
		for i := range Info.Effects {
			ei := &Info.Effects[i]
			if ei.EffectType == spell.EffectApplyAura {
				a := aura.NewAura(spell.SpellID(Info.ID), caster.GetID(), targetID, aura.AuraPeriodicDamage, 8e9)
				a.MaxStack = 1
				a.StackRule = aura.StackRefresh
				a.SpellName = Info.Name
				a.Effects = []aura.AuraEffect{
					{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, Amount: ei.BasePoints, BonusCoeff: ei.BonusCoeff, Period: 2e9},
				}
				auraMgr.AddAura(a)
			}
		}
	}

	return s, result
}
