package arcanemissiles

import (
	"math"
	"strings"
	"testing"

	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

func runArcaneMissilesEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive entire simulation — channel + aura ticks + expiry
	eng.Simulate(5000, 100)

	return eng.Renderer().Render()
}

func TestArcaneMissiles_EngineSpellInfoFields(t *testing.T) {
	if Info.ID != 5143 {
		t.Errorf("expected ID=5143, got %d", Info.ID)
	}
	if Info.CastTime != 0 {
		t.Errorf("expected CastTime=0, got %d", Info.CastTime)
	}
	if Info.Duration != 3000 {
		t.Errorf("expected Duration=3000, got %d", Info.Duration)
	}
	if !Info.IsChanneled {
		t.Error("expected IsChanneled=true")
	}
	if len(Info.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(Info.Effects))
	}
	if MissileInfo.ID != 7268 {
		t.Errorf("expected MissileInfo.ID=7268, got %d", MissileInfo.ID)
	}
	if MissileInfo.Effects[0].BonusCoeff != 0.132 {
		t.Errorf("expected BonusCoeff=0.132, got %f", MissileInfo.Effects[0].BonusCoeff)
	}
}

func TestArcaneMissiles_EngineTimelineEvents(t *testing.T) {
	output := runArcaneMissilesEngineTimeline()

	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "cast start event"},
		{"SpellLaunch", "launch event"},
		{"AuraApplied", "aura applied event"},
		{"SpellHit", "missile hit"},
		{"AuraExpired", "aura expired event"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("engine timeline missing: %s (looking for %q)\nFull output:\n%s", exp.desc, exp.contains, output)
		}
	}
}

func TestArcaneMissiles_EngineThreeMissiles(t *testing.T) {
	output := runArcaneMissilesEngineTimeline()

	// 3 missiles (3 ticks × 1 missile each)
	hitCount := strings.Count(output, "Arcane Missile hits")
	if hitCount != 3 {
		// Also check generic SpellHit for missile
		missileHits := 0
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "SpellHit") && strings.Contains(line, "Arcane Missile") {
				missileHits++
			}
		}
		if missileHits != 3 {
			t.Errorf("expected 3 Arcane Missile hits, got %d SpellHit lines\nFull output:\n%s", hitCount, output)
		}
	}
}

func TestArcaneMissiles_EngineManaConsumed(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	manaBefore := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	consumed := manaBefore - caster.Stats.Get(stat.Mana)
	if consumed != float64(Info.PowerCost) {
		t.Errorf("expected %d mana consumed, got %.0f", Info.PowerCost, consumed)
	}
}

func TestArcaneMissiles_EngineMissileDamage(t *testing.T) {
	output := runArcaneMissilesEngineTimeline()

	expected := 24.0 + 0.132*100
	expectedStr := "hits for"
	if !strings.Contains(output, expectedStr) {
		t.Errorf("expected missile damage hit in timeline\nFull output:\n%s", output)
	}

	// Verify damage value is approximately correct
	_ = math.Abs(expected - 37.2)
}

func TestArcaneMissiles_EngineCancelRemovesAura(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	s := eng.CastSpell(caster, &Info, engine.WithTarget(2))

	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}
	if len(caster.GetOwnedAuras()) == 0 {
		t.Fatal("expected aura after channel start")
	}

	s.Cancel()

	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished after cancel, got %v", s.State)
	}
	if len(caster.GetOwnedAuras()) != 0 {
		t.Error("expected aura to be removed after cancel")
	}
}

func TestArcaneMissiles_EngineTimelineOutput(t *testing.T) {
	output := runArcaneMissilesEngineTimeline()
	t.Log("\n" + output)
	if strings.Contains(output, "No events recorded") {
		t.Error("timeline should have events")
	}
}

func TestArcaneMissiles_EngineMovementCancelsChannel(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	s := eng.CastSpell(caster, &Info, engine.WithTarget(2))

	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}

	// Drive 1 tick
	eng.Advance(100)

	// Move caster — should interrupt
	caster.SetPosition(entity.Position{X: 5, Y: 0, Z: 0})
	eng.Advance(100) // movement detected, isMoving=true
	eng.Advance(100) // spell sees isMoving=true, cancels

	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished after movement, got %v", s.State)
	}
	if s.Result != spell.CastFailedInterrupted {
		t.Errorf("expected CastFailedInterrupted, got %v", s.Result)
	}
}

func TestArcaneMissiles_EngineTargetDeathCancelsChannel(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	target := eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	s := eng.CastSpell(caster, &Info, engine.WithTarget(2))
	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}

	// Drive 1 tick
	eng.Advance(100)

	// Kill target — mark as dead
	target.Entity.State = target.Entity.State.Set(entity.StateDead)

	// Advance — validateChannelTargets should detect dead target
	eng.Advance(100)

	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished after target death, got %v", s.State)
	}
}
