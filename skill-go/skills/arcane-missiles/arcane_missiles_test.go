package arcanemissiles

import (
	"math"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

type testUnit struct {
	id    uint64
	alive bool
	stats *stat.StatSet
	pos   testPos
}

func (u *testUnit) GetID() uint64                                   { return u.id }
func (u *testUnit) IsAlive() bool                                   { return u.alive }
func (u *testUnit) CanCast() bool                                   { return u.alive }
func (u *testUnit) IsMoving() bool                                  { return false }
func (u *testUnit) GetPosition() spell.Position                     { return &u.pos }
func (u *testUnit) GetTargetPosition(targetID uint64) spell.Position { return &testPos{} }
func (u *testUnit) GetStatValue(st uint8) float64                   { return u.stats.Get(stat.StatType(st)) }
func (u *testUnit) ModifyPower(pt uint8, amount float64) bool {
	if pt == 0 {
		current := u.stats.Get(stat.Mana)
		u.stats.SetBase(stat.Mana, current+amount)
		return true
	}
	return true
}

type testPos struct{ x, y, z, facing float64 }

func (p *testPos) GetX() float64      { return p.x }
func (p *testPos) GetY() float64      { return p.y }
func (p *testPos) GetZ() float64      { return p.z }
func (p *testPos) GetFacing() float64 { return p.facing }

func newTestCaster(id uint64) *testUnit {
	u := &testUnit{id: id, alive: true, stats: stat.NewStatSet(), pos: testPos{x: 0}}
	u.stats.SetBase(stat.SpellPower, 100)
	u.stats.SetBase(stat.Mana, 1000)
	return u
}

func TestParentSpellInfo(t *testing.T) {
	if Info.ID != 5143 {
		t.Errorf("expected ID 5143, got %d", Info.ID)
	}
	if Info.Name != "Arcane Missiles" {
		t.Errorf("expected Name Arcane Missiles, got %s", Info.Name)
	}
	if Info.CastTime != 0 {
		t.Errorf("expected CastTime 0, got %d", Info.CastTime)
	}
	if Info.Duration != 3000 {
		t.Errorf("expected Duration 3000, got %d", Info.Duration)
	}
	if Info.PowerCost != 85 {
		t.Errorf("expected PowerCost 85, got %d", Info.PowerCost)
	}
	if !Info.IsChanneled {
		t.Error("expected IsChanneled = true")
	}
	if len(Info.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(Info.Effects))
	}
	eff := Info.Effects[0]
	if eff.EffectType != spell.EffectApplyAura {
		t.Errorf("expected EffectApplyAura, got %d", eff.EffectType)
	}
	if eff.AuraType != uint16(aura.AuraPeriodicTriggerSpell) {
		t.Errorf("expected AuraPeriodicTriggerSpell, got %d", eff.AuraType)
	}
	if eff.AuraPeriod != 1000 {
		t.Errorf("expected AuraPeriod 1000, got %d", eff.AuraPeriod)
	}
	if eff.TriggerSpellID != 7268 {
		t.Errorf("expected TriggerSpellID 7268, got %d", eff.TriggerSpellID)
	}
	if eff.TargetA != spell.TargetUnitTargetEnemy {
		t.Errorf("expected TargetUnitTargetEnemy, got %d", eff.TargetA)
	}
}

func TestMissileInfo(t *testing.T) {
	if MissileInfo.ID != 7268 {
		t.Errorf("expected ID 7268, got %d", MissileInfo.ID)
	}
	if MissileInfo.Name != "Arcane Missile" {
		t.Errorf("expected Name Arcane Missile, got %s", MissileInfo.Name)
	}
	if MissileInfo.CastTime != 0 {
		t.Errorf("expected CastTime 0, got %d", MissileInfo.CastTime)
	}
	if len(MissileInfo.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(MissileInfo.Effects))
	}
	eff := MissileInfo.Effects[0]
	if eff.EffectType != spell.EffectSchoolDamage {
		t.Errorf("expected EffectSchoolDamage, got %d", eff.EffectType)
	}
	if eff.BasePoints != 24 {
		t.Errorf("expected BasePoints 24, got %f", eff.BasePoints)
	}
	if eff.BonusCoeff != 0.132 {
		t.Errorf("expected BonusCoeff 0.132, got %f", eff.BonusCoeff)
	}
}

func TestCastLifecycle(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)

	s, result := CastArcaneMissiles(caster, 2, auraMgr, nil)
	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}
	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}
	if s.Timer != 3000 {
		t.Errorf("expected Timer 3000, got %d", s.Timer)
	}

	s.Update(3000)
	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished, got %v", s.State)
	}
	if s.Result != spell.CastOK {
		t.Errorf("expected CastOK result, got %v", s.Result)
	}
}

func TestAuraApplied(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)

	CastArcaneMissiles(caster, 2, auraMgr, nil)

	a := auraMgr.FindAura(2, spell.SpellID(Info.ID), caster.GetID())
	if a == nil {
		t.Fatal("expected aura to be created on target")
	}
	if a.AuraType != aura.AuraPeriodicTriggerSpell {
		t.Errorf("expected AuraPeriodicTriggerSpell, got %v", a.AuraType)
	}
	if a.Duration != 3*time.Second {
		t.Errorf("expected 3s duration, got %v", a.Duration)
	}
	if a.TargetID != 2 {
		t.Errorf("expected target ID 2, got %d", a.TargetID)
	}
	if len(a.Effects) != 1 {
		t.Fatalf("expected 1 aura effect, got %d", len(a.Effects))
	}
	eff := a.Effects[0]
	if eff.AuraType != aura.AuraPeriodicTriggerSpell {
		t.Errorf("expected AuraPeriodicTriggerSpell, got %v", eff.AuraType)
	}
	if eff.Period != 1*time.Second {
		t.Errorf("expected 1s period, got %v", eff.Period)
	}
	if eff.TriggerSpellID != 7268 {
		t.Errorf("expected TriggerSpellID 7268, got %d", eff.TriggerSpellID)
	}
}

func TestMissileDamageWithSP(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)
	bus := event.NewBus()

	CastArcaneMissiles(caster, 2, auraMgr, bus)

	var hitValues []float64
	bus.Subscribe(event.OnSpellHit, func(e event.Event) {
		if e.SpellID == 7268 {
			hitValues = append(hitValues, e.Value)
		}
	})

	// Tick once (first tick at period=1s, per TC behavior)
	auraMgr.TickPeriodic(2, 1*time.Second, caster.GetStatValue(3),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
			CastTriggeredSpell(caster, a.TargetID, &MissileInfo, bus)
		})

	if len(hitValues) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hitValues))
	}
	expected := 24 + 0.132*100
	if math.Abs(hitValues[0]-expected) > 0.01 {
		t.Errorf("expected damage %.2f, got %.2f", expected, hitValues[0])
	}
}

func TestThreeMissilesFired(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)
	bus := event.NewBus()

	CastArcaneMissiles(caster, 2, auraMgr, bus)

	var hitCount int
	bus.Subscribe(event.OnSpellHit, func(e event.Event) {
		if e.SpellID == 7268 {
			hitCount++
		}
	})

	for i := 0; i < 3; i++ {
		auraMgr.TickPeriodic(2, 1*time.Second, caster.GetStatValue(3),
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
				CastTriggeredSpell(caster, a.TargetID, &MissileInfo, bus)
			})
	}

	if hitCount != 3 {
		t.Errorf("expected 3 missile hits, got %d", hitCount)
	}
}

func TestManaConsumed(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)
	manaBefore := caster.stats.Get(stat.Mana)

	CastArcaneMissiles(caster, 2, auraMgr, nil)

	consumed := manaBefore - caster.stats.Get(stat.Mana)
	if consumed != 85 {
		t.Errorf("expected 85 mana consumed, got %.0f", consumed)
	}
}

func TestCancelRemovesAura(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)

	s, result := CastArcaneMissiles(caster, 2, auraMgr, nil)
	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}

	if auraMgr.FindAura(2, spell.SpellID(Info.ID), caster.GetID()) == nil {
		t.Fatal("expected aura before cancel")
	}

	s.Cancel()

	if s.State != spell.StateFinished {
		t.Fatalf("expected StateFinished after cancel, got %v", s.State)
	}
	if s.Result != spell.CastFailedInterrupted {
		t.Errorf("expected CastFailedInterrupted, got %v", s.Result)
	}
	if auraMgr.FindAura(2, spell.SpellID(Info.ID), caster.GetID()) != nil {
		t.Error("expected aura to be removed after cancel")
	}
}

func TestCancelAfterOneTick(t *testing.T) {
	caster := newTestCaster(1)
	auraMgr := aura.NewManager(nil)
	bus := event.NewBus()

	s, _ := CastArcaneMissiles(caster, 2, auraMgr, bus)

	var hitCount int
	bus.Subscribe(event.OnSpellHit, func(e event.Event) {
		if e.SpellID == 7268 {
			hitCount++
		}
	})

	// Tick once
	auraMgr.TickPeriodic(2, 1*time.Second, caster.GetStatValue(3),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
			CastTriggeredSpell(caster, a.TargetID, &MissileInfo, bus)
		})

	if hitCount != 1 {
		t.Errorf("expected 1 hit after first tick, got %d", hitCount)
	}

	// Cancel channel
	s.Cancel()

	// Try to tick again - aura should be gone, no more hits
	hitCount = 0
	auraMgr.TickPeriodic(2, 1*time.Second, caster.GetStatValue(3),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
			CastTriggeredSpell(caster, a.TargetID, &MissileInfo, bus)
		})

	if hitCount != 0 {
		t.Errorf("expected 0 hits after cancel, got %d", hitCount)
	}
}
