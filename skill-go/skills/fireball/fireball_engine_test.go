package fireball

import (
	"strings"
	"testing"
	"time"

	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

// runFireballEngineTimeline runs the Fireball simulation fully engine-driven.
// No manual aura creation — EffectApplyAura is handled by the effect pipeline + OnAuraCreated.
func runFireballEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	// Cast fireball — Path C (delayed: CastTime=3500, Speed=20)
	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive entire simulation — cast bar + travel + aura ticks + expiry
	eng.Simulate(15000, 100)

	return eng.Renderer().Render()
}

func TestFireball_EngineSpellInfoFields(t *testing.T) {
	if Info.ID != 25306 {
		t.Errorf("expected ID=25306, got %d", Info.ID)
	}
	if Info.CastTime != 3500 {
		t.Errorf("expected CastTime=3500, got %d", Info.CastTime)
	}
	if Info.Speed != 20.0 {
		t.Errorf("expected Speed=20, got %f", Info.Speed)
	}
	if len(Info.Effects) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(Info.Effects))
	}
	if Info.Effects[0].EffectType != spell.EffectSchoolDamage {
		t.Errorf("expected EffectSchoolDamage for effect 0")
	}
	if Info.Effects[1].EffectType != spell.EffectApplyAura {
		t.Errorf("expected EffectApplyAura for effect 1")
	}
}

func TestFireball_EngineTimelineEvents(t *testing.T) {
	output := runFireballEngineTimeline()

	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "Fireball cast start"},
		{"SpellLaunch", "Fireball launched"},
		{"SpellHit", "Fireball hit"},
		{"AuraApplied", "DoT aura applied"},
		{"Fireball PeriodicDamage applied (8s)", "Aura detail with 8s duration"},
		{"AuraTick", "Periodic tick"},
		{"AuraExpired", "Aura expired"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("engine timeline missing: %s (expected to contain %q)\nFull output:\n%s", exp.desc, exp.contains, output)
		}
	}
}

func TestFireball_EngineTickCount(t *testing.T) {
	output := runFireballEngineTimeline()
	count := strings.Count(output, "AuraTick")
	if count != 4 {
		t.Errorf("expected 4 AuraTick events (8s / 2s period), got %d\nFull output:\n%s", count, output)
	}
}

func TestFireball_EngineManaConsumed(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	manaBefore := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, &Info, engine.WithTarget(2))
	// Drive until cast completes (mana consumed at Cast() time, ~3500ms)
	eng.Simulate(5000, 100)
	manaAfter := caster.Stats.Get(stat.Mana)

	consumed := manaBefore - manaAfter
	if consumed != float64(Info.PowerCost) {
		t.Errorf("expected %d mana consumed, got %.0f", Info.PowerCost, consumed)
	}
}

func TestFireball_EngineDamage(t *testing.T) {
	output := runFireballEngineTimeline()
	// School damage should include base 678 + SP scaling
	if !strings.Contains(output, "hits for") {
		t.Errorf("expected damage hit in timeline\nFull output:\n%s", output)
	}
}

func TestFireball_EngineAuraProperties(t *testing.T) {
	eng := engine.New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive until spell hits + aura is applied
	for len(caster.GetOwnedAuras()) == 0 && len(caster.GetActiveSpells()) > 0 {
		eng.Advance(100)
	}

	if len(caster.GetOwnedAuras()) == 0 {
		t.Fatal("expected aura to be created after spell hit")
	}

	a := caster.GetOwnedAuras()[0]
	// Verify aura properties
	if a.SpellID != spell.SpellID(Info.ID) {
		t.Errorf("expected SpellID=%d, got %d", Info.ID, a.SpellID)
	}
	if a.Duration != 8*time.Second {
		t.Errorf("expected 8s duration, got %v", a.Duration)
	}
	if len(a.Effects) != 1 {
		t.Fatalf("expected 1 aura effect, got %d", len(a.Effects))
	}
	eff := a.Effects[0]
	if eff.Amount != 19 {
		t.Errorf("expected amount 19, got %.0f", eff.Amount)
	}
	if eff.Period != 2*time.Second {
		t.Errorf("expected 2s period, got %v", eff.Period)
	}
}

func TestFireball_EngineTimelineOutput(t *testing.T) {
	output := runFireballEngineTimeline()
	t.Log("\n" + output)
}
