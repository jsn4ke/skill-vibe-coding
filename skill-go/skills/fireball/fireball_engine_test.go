package fireball

import (
	"strings"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

// runFireballEngineTimeline runs the fireball simulation using the engine-driven approach.
// Key difference from legacy: the engine drives time progression, not a manual for-loop.
func runFireballEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	// Cast fireball — engine registers it to caster's activeSpells
	s := eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Phase 1: Drive cast bar + projectile travel
	// Fireball has CastTime=3500, Speed=20 → ~500ms travel at 10yd
	for s.State != spell.StateFinished {
		eng.Advance(100)
	}

	// Spell finished — create DoT aura via engine's aura manager
	for i := range Info.Effects {
		ei := &Info.Effects[i]
		if ei.EffectType == spell.EffectApplyAura {
			target := eng.GetUnit(2)
			a := aura.NewAura(spell.SpellID(Info.ID), caster.ID(), target.ID(), aura.AuraPeriodicDamage, 8*time.Second)
			a.MaxStack = 1
			a.StackRule = aura.StackRefresh
			a.SpellName = Info.Name
			a.Effects = []aura.AuraEffect{
				{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, Amount: ei.BasePoints, BonusCoeff: ei.BonusCoeff, Period: 2e9},
			}
			eng.AuraMgr().ApplyAura(caster, target, a)
		}
	}

	// Phase 2: Drive aura ticks until expired
	for i := 0; i < 100; i++ { // max 100 steps safety
		eng.Advance(100)
		if len(caster.GetOwnedAuras()) == 0 {
			break
		}
	}

	return eng.Renderer().Render()
}

func TestFireball_EngineTimelineEvents(t *testing.T) {
	output := runFireballEngineTimeline()

	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "Fireball cast start"},
		{"Fireball starts casting", "Fireball cast detail"},
		{"SpellLaunch", "Fireball launched"},
		{"SpellHit", "Fireball hit"},
		{"AuraApplied", "DoT aura applied"},
		{"Fireball PeriodicDamage applied (8s)", "Aura detail with 8s duration"},
		{"AuraTick", "Periodic tick"},
		{"Fireball ticks for", "Tick damage"},
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

func TestFireball_EngineTimelineOutput(t *testing.T) {
	output := runFireballEngineTimeline()
	t.Log("\n" + output)
}
