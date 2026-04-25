package blizzard

import (
	"strings"
	"testing"

	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

// runBlizzardEngineTimeline runs the Blizzard simulation fully engine-driven.
// RegisterScripts sets up the area aura on channel start.
func runBlizzardEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	// Two enemies within the Blizzard area (center=10,0,0 radius=8)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10, Y: 0}), stat.NewStatSet())
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 12, Y: 1}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	// Cast Blizzard — channeled instant, Path B
	eng.CastSpell(caster, &Info, engine.WithDestPos(10, 0, 0))

	// Drive until aura expires
	eng.Simulate(10000, 100)

	return eng.Renderer().Render()
}

func TestBlizzard_EngineSpellInfoFields(t *testing.T) {
	if Info.ID != 10 {
		t.Errorf("expected ID=10, got %d", Info.ID)
	}
	if Info.CastTime != 0 {
		t.Errorf("expected CastTime=0, got %d", Info.CastTime)
	}
	if Info.Duration != 8000 {
		t.Errorf("expected Duration=8000, got %d", Info.Duration)
	}
	if !Info.IsChanneled {
		t.Error("expected IsChanneled=true")
	}
	if len(Info.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(Info.Effects))
	}
	eff := Info.Effects[0]
	if eff.BonusCoeff != 0.042 {
		t.Errorf("expected BonusCoeff=0.042, got %f", eff.BonusCoeff)
	}
	if eff.AuraPeriod != 1000 {
		t.Errorf("expected AuraPeriod=1000, got %d", eff.AuraPeriod)
	}
}

func TestBlizzard_EngineTimelineEvents(t *testing.T) {
	output := runBlizzardEngineTimeline()

	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "Blizzard cast start"},
		{"SpellLaunch", "Blizzard launched"},
		{"AuraApplied", "Blizzard aura applied"},
		{"AuraTick", "Periodic tick"},
		{"AuraExpired", "Aura expired"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("engine timeline missing: %s (expected to contain %q)\nFull output:\n%s", exp.desc, exp.contains, output)
		}
	}
}

func TestBlizzard_EngineTickCount(t *testing.T) {
	output := runBlizzardEngineTimeline()
	count := strings.Count(output, "AuraTick")
	// 2 enemies × 8 ticks = 16 tick events
	if count != 16 {
		t.Errorf("expected 16 AuraTick events (2 enemies × 8 ticks), got %d\nFull output:\n%s", count, output)
	}
}

func TestBlizzard_EngineTickDamage(t *testing.T) {
	output := runBlizzardEngineTimeline()
	if !strings.Contains(output, "Blizzard ticks for 29.2 damage") {
		t.Errorf("expected tick damage 29.2 (25 + 0.042 × 100) in timeline\nFull output:\n%s", output)
	}
}

func TestBlizzard_EngineManaConsumed(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	manaBefore := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, &Info, engine.WithDestPos(10, 0, 0))

	consumed := manaBefore - caster.Stats.Get(stat.Mana)
	if consumed != float64(Info.PowerCost) {
		t.Errorf("expected %d mana consumed, got %.0f", Info.PowerCost, consumed)
	}
}

func TestBlizzard_EngineChannelCancel(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	s := eng.CastSpell(caster, &Info, engine.WithDestPos(10, 0, 0))

	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}
	if len(caster.GetOwnedAuras()) == 0 {
		t.Fatal("expected area aura after channel start")
	}

	// Cancel the channel
	s.Cancel()

	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished after cancel, got %v", s.State)
	}
	if len(caster.GetOwnedAuras()) != 0 {
		t.Error("expected aura to be removed after cancel")
	}
}

func TestBlizzard_EngineTimelineOutput(t *testing.T) {
	output := runBlizzardEngineTimeline()
	t.Log("\n" + output)
}

func TestBlizzard_EngineMovementCancelsChannel(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	s := eng.CastSpell(caster, &Info, engine.WithDestPos(10, 0, 0))

	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}

	// Drive 1 tick to establish position
	eng.Advance(100)

	// Move caster — should interrupt the channel
	caster.SetPosition(entity.Position{X: 5, Y: 0, Z: 0})
	eng.Advance(100) // movement detected, isMoving=true
	eng.Advance(100) // spell sees isMoving=true, cancels

	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished after movement, got %v", s.State)
	}
	if s.Result != spell.CastFailedInterrupted {
		t.Errorf("expected CastFailedInterrupted, got %v", s.Result)
	}
	if len(caster.GetOwnedAuras()) != 0 {
		t.Error("expected aura to be removed after movement interrupt")
	}
}

func TestBlizzard_EngineTargetLeavesRange(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	// Target near Blizzard center
	target := eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())
	// Second target also near center
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 12}), stat.NewStatSet())

	RegisterScripts(eng, caster)

	s := eng.CastSpell(caster, &Info, engine.WithDestPos(10, 0, 0))
	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}

	// Drive until aura is created
	eng.Advance(100)

	if len(caster.GetOwnedAuras()) == 0 {
		t.Fatal("expected area aura to be created")
	}

	// Move target 2 out of range (RangeMax=30, caster at X=0, so target needs to be > 33 away)
	target.SetPosition(entity.Position{X: 50, Y: 0, Z: 0})

	// Advance — validateChannelTargets should detect target 2 out of range
	eng.Advance(100)

	// Spell should still be channeling (target 3 still in range)
	if s.State != spell.StateChanneling {
		t.Errorf("expected channel to continue with remaining targets, got state %v", s.State)
	}
}
