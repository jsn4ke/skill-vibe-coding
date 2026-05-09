package dragonbreath

import (
	"testing"

	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/stat"
)

func runDragonBreathEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0, Y: 0, Z: 0, Facing: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	caster.Stats.SetBase(stat.Health, 1000)

	// Two enemies in front of caster (facing = 0, positive X direction)
	target1 := eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 5, Y: 0}), stat.NewStatSet())
	target1.Stats.SetBase(stat.Health, 1000)
	target2 := eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 4, Y: 2}), stat.NewStatSet())
	target2.Stats.SetBase(stat.Health, 1000)

	RegisterSpells(eng.SpellStore())

	eng.CastSpell(caster, &Info, engine.WithTarget(2))
	eng.Simulate(100, 50)

	return eng.Renderer().Render()
}

func TestDragonBreath_SpellInfoFields(t *testing.T) {
	if Info.ID != 31661 {
		t.Errorf("expected ID=31661, got %d", Info.ID)
	}
	if Info.CastTime != 0 {
		t.Errorf("expected instant cast, got CastTime=%d", Info.CastTime)
	}
	if len(Info.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(Info.Effects))
	}
	if Info.Effects[0].TargetA != spellcore.TargetUnitConeEnemy24 {
		t.Errorf("expected ConeEnemy target, got %d", Info.Effects[0].TargetA)
	}
}

func TestDragonBreath_HitsMultipleTargets(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0, Y: 0, Z: 0, Facing: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	target1 := eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 5, Y: 0}), stat.NewStatSet())
	target1.Stats.SetBase(stat.Health, 1000)
	target2 := eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 4, Y: 2}), stat.NewStatSet())
	target2.Stats.SetBase(stat.Health, 1000)

	RegisterSpells(eng.SpellStore())

	eng.CastSpell(caster, &Info, engine.WithTarget(2))
	eng.Simulate(100, 50)

	damagePerTarget := 150.0 + 0.15*100.0

	hp1Lost := 1000.0 - target1.Stats.Get(stat.Health)
	hp2Lost := 1000.0 - target2.Stats.Get(stat.Health)

	if hp1Lost < damagePerTarget-1 {
		t.Errorf("target1: expected ~%.0f damage, got %.0f", damagePerTarget, hp1Lost)
	}
	if hp2Lost < damagePerTarget-1 {
		t.Errorf("target2: expected ~%.0f damage, got %.0f", damagePerTarget, hp2Lost)
	}
}

func TestDragonBreath_ManaConsumed(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 5}), stat.NewStatSet())

	RegisterSpells(eng.SpellStore())

	manaBefore := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, &Info, engine.WithTarget(2))
	eng.Simulate(100, 50)

	consumed := manaBefore - caster.Stats.Get(stat.Mana)
	if consumed != float64(Info.PowerCost) {
		t.Errorf("expected %d mana consumed, got %.0f", Info.PowerCost, consumed)
	}
}

func TestDragonBreath_EngineTimelineOutput(t *testing.T) {
	output := runDragonBreathEngineTimeline()
	t.Log("\n" + output)
}
