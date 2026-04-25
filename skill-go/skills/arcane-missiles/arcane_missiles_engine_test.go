package arcanemissiles

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

func runArcaneMissilesEngineTimeline() string {
	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.Mana, 1000)
	eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 10}), stat.NewStatSet())

	// Arcane Missiles is instant cast + channeled
	s := eng.CastSpell(caster, &Info, engine.WithTarget(2))

	// Create the PeriodicTriggerSpell aura on the target
	if s.State == spell.StateChanneling {
		ei := &Info.Effects[0]
		target := eng.GetUnit(2)
		a := aura.NewAura(spell.SpellID(Info.ID), caster.ID(), target.ID(), aura.AuraPeriodicTriggerSpell, 3*time.Second)
		a.SpellName = Info.Name
		a.Effects = []aura.AuraEffect{
			{
				EffectIndex:    ei.EffectIndex,
				AuraType:       aura.AuraPeriodicTriggerSpell,
				Amount:         ei.BasePoints,
				Period:         time.Duration(ei.AuraPeriod) * time.Millisecond,
				TriggerSpellID: ei.TriggerSpellID,
			},
		}
		eng.AuraMgr().ApplyAura(caster, target, a)
	}

	// Drive simulation — aura ticks will happen in Unit.updateAuras
	totalMs := int32(Info.Duration) + 1000
	eng.Simulate(totalMs, 100)

	return eng.Renderer().Render()
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
		{"AuraExpired", "aura expired event"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("engine timeline missing: %s (looking for %q)\nFull output:\n%s", exp.desc, exp.contains, output)
		}
	}
}

func TestArcaneMissiles_EngineTimelineOutput(t *testing.T) {
	output := runArcaneMissilesEngineTimeline()
	t.Log("\n" + output)
	if strings.Contains(output, "No events recorded") {
		t.Error("timeline should have events")
	}
}
