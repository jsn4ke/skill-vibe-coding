package blizzard

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

// runBlizzardEngineTimeline runs the Blizzard simulation using the engine-driven approach.
// Key difference from legacy: Engine.Advance drives both spell update and aura tick,
// area targets resolved via engine.GetUnitsInRadius.
func runBlizzardEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	// Two enemies within the Blizzard area (center=10,0,0 radius=8)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10, Y: 0}), stat.NewStatSet())
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 12, Y: 1}), stat.NewStatSet())

	// Cast Blizzard — channeled instant, Path C (delayed, driven by Advance)
	s := eng.CastSpell(caster, &Info, engine.WithDestPos(10, 0, 0))

	// Spell enters StateChanneling — create area aura on caster
	if s.State == spell.StateChanneling {
		ei := &Info.Effects[0]
		a := aura.NewAura(spell.SpellID(Info.ID), caster.ID(), caster.ID(), aura.AuraPeriodicDamage, 8*time.Second)
		a.SpellName = Info.Name
		a.IsAreaAura = true
		a.AreaCenter = [3]float64{10, 0, 0}
		a.AreaRadius = float64(ei.MiscValue)
		a.Effects = []aura.AuraEffect{
			{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, Amount: ei.BasePoints, BonusCoeff: ei.BonusCoeff, Period: 1e9},
		}
		eng.AuraMgr().ApplyAura(caster, caster, a)
	}

	// Drive until aura expires
	for i := 0; i < 200; i++ {
		eng.Advance(100)
		if len(caster.GetOwnedAuras()) == 0 {
			break
		}
	}

	return eng.Renderer().Render()
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
		{"Blizzard PeriodicDamage applied (8s)", "Aura detail with 8s duration"},
		{"AuraTick", "Periodic tick"},
		{"Blizzard ticks for 29.2 damage", "Tick damage amount"},
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

func TestBlizzard_EngineTimelineOutput(t *testing.T) {
	output := runBlizzardEngineTimeline()
	t.Log("\n" + output)
}
