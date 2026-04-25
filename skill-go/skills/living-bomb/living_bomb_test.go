package livingbomb

import (
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/effect"
	"skill-go/pkg/event"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

type testUnit struct {
	id      uint64
	alive   bool
	stats   *stat.StatSet
	pos     testPos
	targets map[uint64]*testUnit
}

func (u *testUnit) GetID() uint64                                   { return u.id }
func (u *testUnit) IsAlive() bool                                   { return u.alive }
func (u *testUnit) CanCast() bool                                   { return u.alive }
func (u *testUnit) IsMoving() bool                                  { return false }
func (u *testUnit) GetPosition() spell.Position                     { return &u.pos }
func (u *testUnit) GetTargetPosition(targetID uint64) spell.Position {
	if t, ok := u.targets[targetID]; ok {
		return &t.pos
	}
	return &testPos{}
}
func (u *testUnit) GetStatValue(st uint8) float64 { return u.stats.Get(stat.StatType(st)) }
func (u *testUnit) ModifyPower(pt uint8, amount float64) bool {
	if pt == 0 {
		cur := u.stats.Get(stat.Mana)
		u.stats.SetBase(stat.Mana, cur+amount)
	}
	return true
}

type testPos struct{ x, y, z, facing float64 }

func (p *testPos) GetX() float64      { return p.x }
func (p *testPos) GetY() float64      { return p.y }
func (p *testPos) GetZ() float64      { return p.z }
func (p *testPos) GetFacing() float64 { return p.facing }

func newTestUnit(id uint64, x float64) *testUnit {
	u := &testUnit{
		id:      id,
		alive:   true,
		stats:   stat.NewStatSet(),
		pos:     testPos{x: x},
		targets: make(map[uint64]*testUnit),
	}
	u.stats.SetBase(stat.SpellPower, 100)
	u.stats.SetBase(stat.Mana, 1000)
	return u
}

// aoeSelector implements spell.AoESelector for testing
type aoeSelector struct {
	targets []uint64
}

func (s *aoeSelector) SelectAoETargets(center [3]float64, excludeID uint64) []uint64 {
	var result []uint64
	for _, id := range s.targets {
		if id != excludeID {
			result = append(result, id)
		}
	}
	return result
}

func setupRegistry(caster spell.Caster, auraMgr *aura.Manager, bus *event.Bus, aoe spell.AoESelector) *script.Registry {
	reg := script.NewRegistry()
	RegisterScripts(reg, caster, auraMgr, bus, aoe)
	effect.ScriptRegistry = reg
	auraMgr.SetRegistry(reg)
	return reg
}

func TestLivingBomb_SpellInfoFields(t *testing.T) {
	if Info.ID != 44457 {
		t.Errorf("expected Info.ID=44457, got %d", Info.ID)
	}
	if Info.CastTime != 0 {
		t.Errorf("expected instant cast, got %d", Info.CastTime)
	}
	if PeriodicInfo.ID != 217694 {
		t.Errorf("expected PeriodicInfo.ID=217694, got %d", PeriodicInfo.ID)
	}
	if PeriodicInfo.Duration != 4000 {
		t.Errorf("expected 4000ms duration, got %d", PeriodicInfo.Duration)
	}
	if ExplosionInfo.ID != 44461 {
		t.Errorf("expected ExplosionInfo.ID=44461, got %d", ExplosionInfo.ID)
	}
}

func TestLivingBomb_CastLifecycle(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	setupRegistry(caster, auraMgr, bus, nil)

	s, result := CastLivingBomb(caster, 2, auraMgr, bus)

	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}
	if s.State != spell.StateFinished {
		t.Fatalf("expected StateFinished, got %v", s.State)
	}
}

func TestLivingBomb_AppliesPeriodicAura(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	setupRegistry(caster, auraMgr, bus, nil)

	CastLivingBomb(caster, 2, auraMgr, bus)

	a := auraMgr.FindAura(2, 217694, 1)
	if a == nil {
		t.Fatal("expected periodic aura on target after cast")
	}
	if a.AuraType != aura.AuraPeriodicDamage {
		t.Errorf("expected AuraPeriodicDamage, got %v", a.AuraType)
	}
	if a.Duration != 4*time.Second {
		t.Errorf("expected 4s duration, got %v", a.Duration)
	}
	if a.SpellValues == nil || a.SpellValues[2] != 1 {
		t.Error("expected SpellValues[2]=1 (canSpread) on original aura")
	}
}

func TestLivingBomb_PeriodicTicks(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	setupRegistry(caster, auraMgr, bus, nil)

	CastLivingBomb(caster, 2, auraMgr, bus)

	a := auraMgr.FindAura(2, 217694, 1)
	if a == nil {
		t.Fatal("expected periodic aura")
	}

	tickCount := 0
	casterSP := caster.GetStatValue(3)

	const stepMs int32 = 100
	for simMs := int32(0); simMs < 4000; simMs += stepMs {
		a.Tick(time.Duration(stepMs)*time.Millisecond, casterSP, nil,
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
				tickCount++
				expected := 0.06 * 100
				if amount < expected-0.01 {
					t.Errorf("tick %d: expected damage >= %.2f, got %.2f", tickCount, expected, amount)
				}
			})
	}

	if tickCount != 4 {
		t.Errorf("expected 4 ticks, got %d", tickCount)
	}
}

func TestLivingBomb_NoExplosionOnDeath(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	setupRegistry(caster, auraMgr, bus, nil)

	CastLivingBomb(caster, 2, auraMgr, bus)

	a := auraMgr.FindAura(2, 217694, 1)
	if a == nil {
		t.Fatal("expected aura to exist")
	}

	explosionTriggered := false
	bus.Subscribe(event.OnSpellHit, func(e event.Event) {
		if e.SpellID == 44461 {
			explosionTriggered = true
		}
	})

	auraMgr.RemoveAura(a, aura.RemoveByDeath)

	if explosionTriggered {
		t.Error("explosion should NOT trigger on death")
	}
}

func TestLivingBomb_NoExplosionOnDispel(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	setupRegistry(caster, auraMgr, bus, nil)

	CastLivingBomb(caster, 2, auraMgr, bus)

	a := auraMgr.FindAura(2, 217694, 1)
	if a == nil {
		t.Fatal("expected aura to exist")
	}

	explosionTriggered := false
	bus.Subscribe(event.OnSpellHit, func(e event.Event) {
		if e.SpellID == 44461 {
			explosionTriggered = true
		}
	})

	auraMgr.RemoveAura(a, aura.RemoveByDispel)

	if explosionTriggered {
		t.Error("explosion should NOT trigger on dispel")
	}
}

func TestLivingBomb_ExplosionHitsAoETargets(t *testing.T) {
	caster := newTestUnit(1, 0)
	targetA := newTestUnit(2, 5)
	targetB := newTestUnit(3, 8)
	targetC := newTestUnit(4, 12)
	caster.targets[2] = targetA
	caster.targets[3] = targetB
	caster.targets[4] = targetC

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	selector := &aoeSelector{targets: []uint64{2, 3, 4}}
	setupRegistry(caster, auraMgr, bus, selector)

	CastLivingBomb(caster, 2, auraMgr, bus)

	casterSP := caster.GetStatValue(3)
	a := auraMgr.FindAura(2, 217694, 1)
	if a == nil {
		t.Fatal("expected aura")
	}

	const stepMs int32 = 100
	for simMs := int32(0); simMs < 5000; simMs += stepMs {
		a.Tick(time.Duration(stepMs)*time.Millisecond, casterSP, bus,
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {})
	}

	// Manually trigger expiry → AfterRemove hook → explosion → spread
	auraMgr.RemoveAura(a, aura.RemoveByExpire)

	// Explosion should have spread to B and C
	auraB := auraMgr.FindAura(3, 217694, 1)
	auraC := auraMgr.FindAura(4, 217694, 1)

	if auraB == nil {
		t.Error("expected spread aura on targetB after explosion")
	}
	if auraC == nil {
		t.Error("expected spread aura on targetC after explosion")
	}
	if auraB != nil && auraB.SpellValues[2] != 0 {
		t.Error("spread copy should have canSpread=0")
	}
	if auraC != nil && auraC.SpellValues[2] != 0 {
		t.Error("spread copy should have canSpread=0")
	}
}

func TestLivingBomb_SpreadChainTerminates(t *testing.T) {
	caster := newTestUnit(1, 0)
	targetA := newTestUnit(2, 5)
	targetB := newTestUnit(3, 8)
	caster.targets[2] = targetA
	caster.targets[3] = targetB

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	selector := &aoeSelector{targets: []uint64{2, 3}}
	setupRegistry(caster, auraMgr, bus, selector)

	CastLivingBomb(caster, 2, auraMgr, bus)

	auraCount := 0
	bus.Subscribe(event.OnAuraApplied, func(e event.Event) {
		if e.SpellID == 217694 {
			auraCount++
		}
	})

	casterSP := caster.GetStatValue(3)
	const stepMs int32 = 100

	// Phase 1: Tick targetA's bomb to expiry → explosion → spread to B
	a := auraMgr.FindAura(2, 217694, 1)
	if a == nil {
		t.Fatal("expected aura on targetA")
	}
	for simMs := int32(0); simMs < 5000; simMs += stepMs {
		a.Tick(time.Duration(stepMs)*time.Millisecond, casterSP, bus,
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {})
	}

	// Manually trigger expiry → AfterRemove hook → explosion → spread
	auraMgr.RemoveAura(a, aura.RemoveByExpire)

	// B should now have a bomb (canSpread=0)
	auraB := auraMgr.FindAura(3, 217694, 1)
	if auraB == nil {
		t.Fatal("expected spread aura on targetB")
	}

	// Reset count
	auraCount = 0

	// Phase 2: Tick targetB's bomb to expiry → explosion → should NOT spread further
	for simMs := int32(0); simMs < 5000; simMs += stepMs {
		auraB.Tick(time.Duration(stepMs)*time.Millisecond, casterSP, bus,
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {})
	}

	if auraCount != 0 {
		t.Errorf("expected 0 new auras from spread copy expiry, got %d — chain did not terminate", auraCount)
	}
}
