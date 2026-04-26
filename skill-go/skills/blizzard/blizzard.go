package blizzard

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
)

// Spell 10 — Blizzard (channeled AoE)
// Engine-driven: cast via eng.CastSpell(caster, &Info, engine.WithDestPos(x,y,z))
// The effect pipeline automatically creates the area aura during Cast().
// No RegisterScripts needed — aura creation and cancel cleanup are fully automatic.
var Info = spell.SpellInfo{
	ID:          10,
	Name:        "Blizzard",
	CastTime:    0,
	Duration:    8000,
	PowerCost:   320,
	PowerType:   0,
	RangeMax:    30,
	IsChanneled: true,
	Attributes:     spell.AttrChanneled,
	InterruptFlags: spell.InterruptMovement,
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

// RegisterScripts is a no-op for Blizzard. Aura creation and cancel cleanup
// are handled automatically by the effect pipeline and Cancel().
func RegisterScripts(registry *script.Registry, caster *unit.Unit, eng *engine.Engine) {}
