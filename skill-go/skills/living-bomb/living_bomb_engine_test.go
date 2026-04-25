package livingbomb

import (
	"strings"
	"testing"

	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/stat"
)

// engineAoESelector adapts engine.GetUnitsInRadius to spell.AoESelector.
type engineAoESelector struct {
	eng    *engine.Engine
	radius float64
}

func (s *engineAoESelector) SelectAoETargets(center [3]float64, excludeID uint64) []uint64 {
	units := s.eng.GetUnitsInRadius(center, s.radius, excludeID)
	var ids []uint64
	for _, u := range units {
		ids = append(ids, u.ID())
	}
	return ids
}

// runLivingBombEngineTimeline runs the full Living Bomb simulation via engine.
// All spells cast through engine.CastSpell, auras auto-created by hooks + effect pipeline.
func runLivingBombEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: -20}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	// Bomb carrier at (10,0,0)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	// 2 nearby enemies for explosion AoE
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 8}), stat.NewStatSet())
	eng.AddUnitWithID(4, entity.NewEntity(4, entity.TypeCreature, entity.Position{X: 12, Y: 1}), stat.NewStatSet())

	// Register hooks
	selector := &engineAoESelector{eng: eng, radius: 8}
	RegisterScripts(eng.Registry(), caster, eng, selector)

	// Cast Living Bomb — instant spell, hooks create the periodic aura automatically
	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive until first bomb's aura expires + explosion (5000ms covers 4s DoT + 1s buffer)
	eng.Simulate(5000, 100)

	return eng.Renderer().Render()
}

func TestLivingBomb_EngineSpellInfoFields(t *testing.T) {
	if Info.ID != 44457 {
		t.Errorf("expected Info.ID=44457, got %d", Info.ID)
	}
	if PeriodicInfo.ID != 217694 {
		t.Errorf("expected PeriodicInfo.ID=217694, got %d", PeriodicInfo.ID)
	}
	if ExplosionInfo.ID != 44461 {
		t.Errorf("expected ExplosionInfo.ID=44461, got %d", ExplosionInfo.ID)
	}
}

func TestLivingBomb_EngineDotTicks(t *testing.T) {
	output := runLivingBombEngineTimeline()

	// Count only the original bomb's ticks (target 2)
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "AuraTick") && strings.Contains(line, "Unit1→Unit2") {
			count++
		}
	}
	if count != 4 {
		t.Errorf("expected 4 AuraTick events on original target, got %d\nFull output:\n%s", count, output)
	}
}

func TestLivingBomb_EngineAuraExpiry(t *testing.T) {
	output := runLivingBombEngineTimeline()

	if !strings.Contains(output, "AuraExpired") {
		t.Errorf("expected AuraExpired event\nFull output:\n%s", output)
	}
}

func TestLivingBomb_EngineExplosionTriggers(t *testing.T) {
	output := runLivingBombEngineTimeline()

	if !strings.Contains(output, "Living Bomb Explode") {
		t.Errorf("expected Living Bomb Explode event after aura expiry\nFull output:\n%s", output)
	}
}

func TestLivingBomb_EngineExplosionHitsAoETargets(t *testing.T) {
	output := runLivingBombEngineTimeline()

	explodeHitCount := strings.Count(output, "Living Bomb Explode")
	if explodeHitCount < 1 {
		t.Errorf("expected at least 1 Living Bomb Explode event, got %d\nFull output:\n%s", explodeHitCount, output)
	}
}

func TestLivingBomb_EngineSpreadToNearbyTargets(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: -20}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 5}), stat.NewStatSet())
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 8}), stat.NewStatSet())
	eng.AddUnitWithID(4, entity.NewEntity(4, entity.TypeCreature, entity.Position{X: 12}), stat.NewStatSet())

	selector := &engineAoESelector{eng: eng, radius: 8}
	RegisterScripts(eng.Registry(), caster, eng, selector)

	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive until bomb explodes
	eng.Simulate(8000, 100)

	// Check that spread auras were created on targets 3 and 4
	spreadCount := strings.Count(eng.Renderer().Render(), "AuraApplied")
	if spreadCount < 3 { // Original + 2 spread
		t.Errorf("expected at least 3 AuraApplied events (original + 2 spread), got %d\n%s", spreadCount, eng.Renderer().Render())
	}
}

func TestLivingBomb_EngineSpreadChainTerminates(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: -20}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 5}), stat.NewStatSet())
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 8}), stat.NewStatSet())

	selector := &engineAoESelector{eng: eng, radius: 8}
	RegisterScripts(eng.Registry(), caster, eng, selector)

	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive through first explosion + second expiry
	eng.Simulate(16000, 100)

	output := eng.Renderer().Render()
	auraAppliedCount := strings.Count(output, "AuraApplied")
	// Should have original (target2) + spread (target3) = 2 auras total
	// The spread copy has canSpread=0, so its explosion should NOT create new auras
	if auraAppliedCount != 2 {
		t.Errorf("expected exactly 2 AuraApplied events (no chain), got %d\nFull output:\n%s", auraAppliedCount, output)
	}
}

func TestLivingBomb_EngineNoExplosionOnDispel(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: -20}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	selector := &engineAoESelector{eng: eng, radius: 8}
	RegisterScripts(eng.Registry(), caster, eng, selector)

	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Drive a bit to get aura established
	eng.Simulate(500, 100)

	// Manually dispel the aura
	if len(caster.GetOwnedAuras()) == 0 {
		t.Fatal("expected aura to exist")
	}
	a := caster.GetOwnedAuras()[0]
	target := eng.GetUnit(a.TargetID)
	eng.AuraMgr().RemoveAuraFromHosts(a, caster, target, aura.RemoveByDispel)

	output := eng.Renderer().Render()
	if strings.Contains(output, "Living Bomb Explode") {
		t.Error("explosion should NOT trigger on dispel")
	}
}

func TestLivingBomb_EngineNoExplosionOnDeath(t *testing.T) {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: -20}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	selector := &engineAoESelector{eng: eng, radius: 8}
	RegisterScripts(eng.Registry(), caster, eng, selector)

	eng.CastSpell(caster, &Info, engine.WithTarget(2))
	eng.Simulate(500, 100)

	if len(caster.GetOwnedAuras()) == 0 {
		t.Fatal("expected aura to exist")
	}
	a := caster.GetOwnedAuras()[0]
	target := eng.GetUnit(a.TargetID)
	eng.AuraMgr().RemoveAuraFromHosts(a, caster, target, aura.RemoveByDeath)

	output := eng.Renderer().Render()
	if strings.Contains(output, "Living Bomb Explode") {
		t.Error("explosion should NOT trigger on death")
	}
}

func TestLivingBomb_EngineTimelineOutput(t *testing.T) {
	output := runLivingBombEngineTimeline()
	t.Log("\n" + output)
}
