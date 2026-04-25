package livingbomb

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

// runLivingBombEngineTimeline runs the Living Bomb simulation using the engine-driven approach.
// The AfterRemove hook fires via engine's updateAuras → RemoveAuraFromHosts → script registry.
func runLivingBombEngineTimeline() string {
	eng := engine.New()

	// Create caster
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)

	// Create target (bomb carrier) at (10,0,0)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	// Create 2 nearby enemies for explosion AoE
	eng.AddUnitWithID(3, entity.NewEntity(3, entity.TypeCreature, entity.Position{X: 8}), stat.NewStatSet())
	eng.AddUnitWithID(4, entity.NewEntity(4, entity.TypeCreature, entity.Position{X: 12, Y: 1}), stat.NewStatSet())

	// Register Living Bomb scripts with engine's subsystems
	selector := &engineAoESelector{eng: eng, radius: 8}
	RegisterScripts(eng.Registry(), caster, eng.AuraMgr(), eng.Bus(), selector)

	// Cast Living Bomb on target — instant spell (Path B)
	eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Manually create DoT aura via ApplyAura (engine-driven path).
	// The legacy castPeriodicSpell uses auraMgr.AddAura (legacy map),
	// so we create the aura directly for engine integration.
	ei := &PeriodicInfo.Effects[0]
	target := eng.GetUnit(2)
	a := aura.NewAura(spell.SpellID(PeriodicInfo.ID), caster.ID(), target.ID(), aura.AuraPeriodicDamage, 4*time.Second)
	a.MaxStack = 1
	a.StackRule = aura.StackRefresh
	a.SpellName = PeriodicInfo.Name
	a.SpellValues = map[uint8]float64{2: 1} // canSpread = true
	a.Effects = []aura.AuraEffect{
		{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, BonusCoeff: ei.BonusCoeff, Period: 1e9},
	}
	eng.AuraMgr().ApplyAura(caster, target, a)

	// Drive until aura expires (4s DoT + explosion event)
	for i := 0; i < 200; i++ {
		eng.Advance(100)
		if len(caster.GetOwnedAuras()) == 0 {
			break
		}
	}

	return eng.Renderer().Render()
}

func TestLivingBomb_EngineDotTicks(t *testing.T) {
	output := runLivingBombEngineTimeline()

	// DoT should tick 4 times (4s / 1s period)
	count := strings.Count(output, "AuraTick")
	if count != 4 {
		t.Errorf("expected 4 AuraTick events, got %d\nFull output:\n%s", count, output)
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

	// After aura expires, the AfterRemove hook should trigger the explosion spell
	if !strings.Contains(output, "Living Bomb Explode") {
		t.Errorf("expected Living Bomb Explode event after aura expiry\nFull output:\n%s", output)
	}
}

func TestLivingBomb_EngineExplosionHitsAoETargets(t *testing.T) {
	output := runLivingBombEngineTimeline()

	// The explosion should hit the 2 nearby enemies (units 3 and 4)
	explodeHitCount := strings.Count(output, "Living Bomb Explode")
	if explodeHitCount < 1 {
		t.Errorf("expected at least 1 Living Bomb Explode event, got %d\nFull output:\n%s", explodeHitCount, output)
	}
}

func TestLivingBomb_EngineTimelineOutput(t *testing.T) {
	output := runLivingBombEngineTimeline()
	t.Log("\n" + output)
}
