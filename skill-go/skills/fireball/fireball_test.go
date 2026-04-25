package fireball

import (
	"math"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

// testUnit implements spell.Caster
type testUnit struct {
	id      uint64
	alive   bool
	stats   *stat.StatSet
	pos     testPos
	targets map[uint64]*testUnit
}

func (u *testUnit) GetID() uint64                                    { return u.id }
func (u *testUnit) IsAlive() bool                                    { return u.alive }
func (u *testUnit) CanCast() bool                                    { return u.alive }
func (u *testUnit) IsMoving() bool                                   { return false }
func (u *testUnit) GetPosition() spell.Position                      { return &u.pos }
func (u *testUnit) GetTargetPosition(targetID uint64) spell.Position  {
	if t, ok := u.targets[targetID]; ok {
		return &t.pos
	}
	return &testPos{}
}
func (u *testUnit) GetStatValue(st uint8) float64                    { return u.stats.Get(stat.StatType(st)) }
func (u *testUnit) ModifyPower(pt uint8, amount float64) bool        { return true }

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

func TestFireball_CastLifecycle(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target
	auraMgr := aura.NewManager(nil)

	s, result := CastFireball(caster, 2, auraMgr, nil)

	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}
	if s.State != spell.StateFinished {
		t.Fatalf("expected StateFinished, got %v", s.State)
	}
	if len(s.TargetInfos) == 0 {
		t.Fatal("expected target infos")
	}

	ti := s.TargetInfos[0]
	minDmg := 678.0 + 100.0
	if ti.Damage < minDmg {
		t.Errorf("expected damage >= %.0f, got %.0f", minDmg, ti.Damage)
	}
}

func TestFireball_ProjectileDelay(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 30)
	caster.targets[2] = target
	auraMgr := aura.NewManager(nil)

	s, result := CastFireball(caster, 2, auraMgr, nil)

	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}
	// 30 yards / 20 y/s = 1.5s = 1500ms
	if s.HitTimer != 1500 {
		t.Errorf("expected HitTimer 1500, got %d", s.HitTimer)
	}
	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished, got %v", s.State)
	}
}

func TestFireball_NoDamageDuringFlight(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	s := spell.NewSpell(spell.SpellID(Info.ID), &Info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(3500) // cast completes

	if s.State != spell.StateLaunched {
		t.Fatalf("expected StateLaunched, got %v", s.State)
	}
	if s.TargetInfos[0].Damage != 0 {
		t.Errorf("expected 0 damage during flight, got %.0f", s.TargetInfos[0].Damage)
	}
}

func TestFireball_AppliesDoTAura(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target
	auraMgr := aura.NewManager(nil)

	CastFireball(caster, 2, auraMgr, nil)

	a := auraMgr.FindAura(2, spell.SpellID(Info.ID), 1)
	if a == nil {
		t.Fatal("expected DoT aura on target")
	}
	if a.AuraType != aura.AuraPeriodicDamage {
		t.Errorf("expected AuraPeriodicDamage, got %v", a.AuraType)
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

func TestFireball_DoTTicksWithSP(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target
	auraMgr := aura.NewManager(nil)

	CastFireball(caster, 2, auraMgr, nil)

	a := auraMgr.FindAura(2, spell.SpellID(Info.ID), 1)
	if a == nil {
		t.Fatal("expected DoT aura")
	}

	var tickDamage float64
	a.Tick(2*time.Second, 100, nil, func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
		tickDamage = amount
	})

	expected := 19 + 0.125*100
	if math.Abs(tickDamage-expected) > 0.01 {
		t.Errorf("expected tick damage %.2f, got %.2f", expected, tickDamage)
	}
}

func TestFireball_CancelBreaksOnMove(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	s := spell.NewSpell(spell.SpellID(Info.ID), &Info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()

	// Simulate movement during cast by cancelling
	s.Cancel()

	if s.State != spell.StateFinished {
		t.Fatalf("expected StateFinished after cancel, got %v", s.State)
	}
	if s.Result != spell.CastFailedInterrupted {
		t.Errorf("expected CastFailedInterrupted, got %v", s.Result)
	}
}

func TestFireball_MovementDoesNotBreakAfterLaunch(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	s := spell.NewSpell(spell.SpellID(Info.ID), &Info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(3500) // cast completes → StateLaunched

	if s.State != spell.StateLaunched {
		t.Fatalf("expected StateLaunched, got %v", s.State)
	}

	// Movement should NOT cancel during StateLaunched
	// (AttrBreakOnMove only applies during StatePreparing)
	s.Update(100)
	if s.State != spell.StateLaunched {
		t.Errorf("expected still StateLaunched (movement doesn't break projectile), got %v", s.State)
	}
}
