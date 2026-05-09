package arcaneintellect

import (
	"strings"
	"testing"

	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/stat"
)

func TestArcaneIntellect_IncreasesSpellPower(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.Mana, 1000)

	RegisterSpells(eng.SpellStore())

	spBefore := caster.Stats.Get(stat.SpellPower)

	eng.CastSpell(caster, &Info)
	eng.Simulate(100, 50)

	spAfter := caster.Stats.Get(stat.SpellPower)
	if spAfter != spBefore+60 {
		t.Errorf("expected SpellPower to increase by 60, got increase of %.0f (before=%.0f after=%.0f)", spAfter-spBefore, spBefore, spAfter)
	}
}

func TestArcaneIntellect_ManaConsumed(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.Mana, 1000)

	RegisterSpells(eng.SpellStore())

	manaBefore := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, &Info)
	eng.Simulate(100, 50)

	manaAfter := caster.Stats.Get(stat.Mana)
	consumed := manaBefore - manaAfter
	if consumed != float64(Info.PowerCost) {
		t.Errorf("expected %d mana consumed, got %.0f", Info.PowerCost, consumed)
	}
}

func TestArcaneIntellect_BuffExpires(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.Mana, 1000)

	RegisterSpells(eng.SpellStore())

	spBase := caster.Stats.Get(stat.SpellPower)

	eng.CastSpell(caster, &Info)
	eng.Simulate(100, 50)

	if caster.Stats.Get(stat.SpellPower) != spBase+60 {
		t.Fatal("expected buff applied")
	}

	// Wait for expiry (30s duration)
	eng.Simulate(31000, 100)

	if caster.Stats.Get(stat.SpellPower) != spBase {
		t.Errorf("expected SpellPower to return to base after expiry, got %.0f (base=%.0f)", caster.Stats.Get(stat.SpellPower), spBase)
	}
}

func TestArcaneIntellect_EngineTimelineOutput(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.Mana, 1000)

	RegisterSpells(eng.SpellStore())

	eng.CastSpell(caster, &Info)
	eng.Simulate(100, 50)

	output := eng.Renderer().Render()
	if !strings.Contains(output, "AuraApplied") {
		t.Errorf("expected AuraApplied in timeline\n%s", output)
	}
	t.Log("\n" + output)
}
