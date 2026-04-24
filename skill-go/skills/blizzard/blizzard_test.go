package blizzard

import (
	"math"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"skill-go/pkg/targeting"
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
func (u *testUnit) GetStatValue(st uint8) float64 { return u.stats.Get(stat.StatType(st)) }
func (u *testUnit) ModifyPower(pt uint8, amount float64) bool {
	if pt == 0 {
		current := u.stats.Get(stat.Mana)
		u.stats.SetBase(stat.Mana, current+amount)
		return true
	}
	return true
}
func (u *testUnit) GetEntity() *entity.Entity {
	return &entity.Entity{ID: entity.EntityID(u.id), Type: entity.TypePlayer, Pos: entity.Position{X: u.pos.x, Y: u.pos.y, Z: u.pos.z}}
}
func (u *testUnit) casterEntity() *entity.Entity {
	return u.GetEntity()
}

type testPos struct{ x, y, z, facing float64 }

func (p *testPos) GetX() float64      { return p.x }
func (p *testPos) GetY() float64      { return p.y }
func (p *testPos) GetZ() float64      { return p.z }
func (p *testPos) GetFacing() float64 { return p.facing }

type enemyUnit struct {
	id    uint64
	alive bool
	pos   testPos
}

func (e *enemyUnit) GetEntity() *entity.Entity {
	return &entity.Entity{ID: entity.EntityID(e.id), Type: entity.TypeCreature, Pos: entity.Position{X: e.pos.x, Y: e.pos.y, Z: e.pos.z}}
}
func (e *enemyUnit) IsAlive() bool { return e.alive }

func newTestCaster(id uint64, x float64) *testUnit {
	u := &testUnit{id: id, alive: true, stats: stat.NewStatSet(), pos: testPos{x: x}}
	u.stats.SetBase(stat.SpellPower, 100)
	u.stats.SetBase(stat.Mana, 1000)
	return u
}

func newTestEnemy(id uint64, x, y float64) *enemyUnit {
	return &enemyUnit{id: id, alive: true, pos: testPos{x: x, y: y}}
}

func resolveTargets(auraMgr *aura.Manager, caster *testUnit, targets []targeting.Targetable) func() []uint64 {
	return func() []uint64 {
		a := auraMgr.FindAreaAura(caster.id, spell.SpellID(Info.ID))
		if a == nil {
			return nil
		}
		sel := targeting.NewSelector(targets)
		desc := targeting.Descriptor{Selection: targeting.SelectArea, Check: targeting.CheckEnemy, Radius: a.AreaRadius}
		center := entity.Position{X: a.AreaCenter[0], Y: a.AreaCenter[1], Z: a.AreaCenter[2]}
		selected := sel.SelectAroundPoint(caster.casterEntity(), center, targets, desc, 0)
		var ids []uint64
		for _, e := range selected {
			ids = append(ids, uint64(e.ID))
		}
		return ids
	}
}

func TestBlizzard_CastLifecycle(t *testing.T) {
	caster := newTestCaster(1, 0)
	auraMgr := aura.NewManager(nil)

	s, result := CastBlizzard(caster, 10, 0, 0, auraMgr, nil)

	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}
	if s.State != spell.StateChanneling {
		t.Fatalf("expected StateChanneling, got %v", s.State)
	}
	if s.Timer != 8000 {
		t.Errorf("expected Timer 8000, got %d", s.Timer)
	}

	s.Update(8000)
	if s.State != spell.StateFinished {
		t.Errorf("expected StateFinished, got %v", s.State)
	}
	if s.Result != spell.CastOK {
		t.Errorf("expected CastOK result, got %v", s.Result)
	}
}

func TestBlizzard_AuraApplied(t *testing.T) {
	caster := newTestCaster(1, 0)
	auraMgr := aura.NewManager(nil)

	CastBlizzard(caster, 10, 0, 0, auraMgr, nil)

	a := auraMgr.FindAreaAura(1, spell.SpellID(Info.ID))
	if a == nil {
		t.Fatal("expected area aura to be created")
	}
	if a.AuraType != aura.AuraPeriodicDamage {
		t.Errorf("expected AuraPeriodicDamage, got %v", a.AuraType)
	}
	if a.Duration != 8*time.Second {
		t.Errorf("expected 8s duration, got %v", a.Duration)
	}
	if !a.IsAreaAura {
		t.Error("expected IsAreaAura = true")
	}
	if a.AreaRadius != 8 {
		t.Errorf("expected radius 8, got %.0f", a.AreaRadius)
	}
	if a.AreaCenter != [3]float64{10, 0, 0} {
		t.Errorf("expected center [10,0,0], got %v", a.AreaCenter)
	}
	if len(a.Effects) != 1 {
		t.Fatalf("expected 1 aura effect, got %d", len(a.Effects))
	}
	eff := a.Effects[0]
	if eff.Amount != 25 {
		t.Errorf("expected amount 25, got %.0f", eff.Amount)
	}
	if eff.Period != 1*time.Second {
		t.Errorf("expected 1s period, got %v", eff.Period)
	}
}

func TestBlizzard_PeriodicDamageWithSP(t *testing.T) {
	caster := newTestCaster(1, 0)
	enemy := newTestEnemy(2, 10, 0)
	auraMgr := aura.NewManager(nil)

	CastBlizzard(caster, 10, 0, 0, auraMgr, nil)

	targets := []targeting.Targetable{enemy}
	var tickDamage float64
	var tickTargetID uint64

	auraMgr.TickPeriodicArea(1*time.Second, 100, resolveTargets(auraMgr, caster, targets),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64, tid uint64) {
			tickDamage = amount
			tickTargetID = tid
		})

	expected := 25 + 0.042*100
	if math.Abs(tickDamage-expected) > 0.01 {
		t.Errorf("expected tick damage %.2f, got %.2f", expected, tickDamage)
	}
	if tickTargetID != 2 {
		t.Errorf("expected target ID 2, got %d", tickTargetID)
	}
}

func TestBlizzard_CancelRemovesAura(t *testing.T) {
	caster := newTestCaster(1, 0)
	auraMgr := aura.NewManager(nil)

	s, result := CastBlizzard(caster, 10, 0, 0, auraMgr, nil)
	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}

	if auraMgr.FindAreaAura(1, spell.SpellID(Info.ID)) == nil {
		t.Fatal("expected aura before cancel")
	}

	s.Cancel()

	if s.State != spell.StateFinished {
		t.Fatalf("expected StateFinished after cancel, got %v", s.State)
	}
	if s.Result != spell.CastFailedInterrupted {
		t.Errorf("expected CastFailedInterrupted, got %v", s.Result)
	}
	if auraMgr.FindAreaAura(1, spell.SpellID(Info.ID)) != nil {
		t.Error("expected aura to be removed after cancel")
	}
}

func TestBlizzard_ManaConsumed(t *testing.T) {
	caster := newTestCaster(1, 0)
	auraMgr := aura.NewManager(nil)
	manaBefore := caster.stats.Get(stat.Mana)

	CastBlizzard(caster, 10, 0, 0, auraMgr, nil)

	consumed := manaBefore - caster.stats.Get(stat.Mana)
	if consumed != 320 {
		t.Errorf("expected 320 mana consumed, got %.0f", consumed)
	}
}

func TestBlizzard_EnemyLeavingArea(t *testing.T) {
	caster := newTestCaster(1, 0)
	enemy := newTestEnemy(2, 10, 0)
	auraMgr := aura.NewManager(nil)

	CastBlizzard(caster, 10, 0, 0, auraMgr, nil)

	// First tick - enemy in range
	targets := []targeting.Targetable{enemy}
	var tickCount int
	auraMgr.TickPeriodicArea(1*time.Second, 100, resolveTargets(auraMgr, caster, targets),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64, tid uint64) {
			tickCount++
		})
	if tickCount != 1 {
		t.Errorf("expected 1 tick hit (enemy in range), got %d", tickCount)
	}

	// Enemy moves out of range
	enemyMoved := &enemyUnit{id: 2, alive: true, pos: testPos{x: 50, y: 50}}
	targetsMoved := []targeting.Targetable{enemyMoved}

	tickCount = 0
	auraMgr.TickPeriodicArea(1*time.Second, 100, resolveTargets(auraMgr, caster, targetsMoved),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64, tid uint64) {
			tickCount++
		})
	if tickCount != 0 {
		t.Errorf("expected 0 tick hits (enemy out of range), got %d", tickCount)
	}
}

func TestBlizzard_MultipleEnemies(t *testing.T) {
	caster := newTestCaster(1, 0)
	e1 := newTestEnemy(2, 10, 0)
	e2 := newTestEnemy(3, 12, 1)
	e3 := newTestEnemy(4, 50, 50)
	auraMgr := aura.NewManager(nil)

	CastBlizzard(caster, 10, 0, 0, auraMgr, nil)

	targets := []targeting.Targetable{e1, e2, e3}
	var hitTargets []uint64

	auraMgr.TickPeriodicArea(1*time.Second, 100, resolveTargets(auraMgr, caster, targets),
		func(a *aura.Aura, eff *aura.AuraEffect, amount float64, tid uint64) {
			hitTargets = append(hitTargets, tid)
		})

	if len(hitTargets) != 2 {
		t.Errorf("expected 2 targets hit, got %d", len(hitTargets))
	}
	for _, id := range hitTargets {
		if id == 4 {
			t.Error("enemy 4 should not be hit (out of range)")
		}
	}
}

func TestBlizzard_SpellInfo(t *testing.T) {
	if Info.ID != 10 {
		t.Errorf("expected ID 10, got %d", Info.ID)
	}
	if Info.Name != "Blizzard" {
		t.Errorf("expected Name Blizzard, got %s", Info.Name)
	}
	if Info.CastTime != 0 {
		t.Errorf("expected CastTime 0, got %d", Info.CastTime)
	}
	if Info.Duration != 8000 {
		t.Errorf("expected Duration 8000, got %d", Info.Duration)
	}
	if Info.PowerCost != 320 {
		t.Errorf("expected PowerCost 320, got %d", Info.PowerCost)
	}
	if !Info.IsChanneled {
		t.Error("expected IsChanneled = true")
	}
	if len(Info.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(Info.Effects))
	}
	eff := Info.Effects[0]
	if eff.BonusCoeff != 0.042 {
		t.Errorf("expected BonusCoeff 0.042, got %f", eff.BonusCoeff)
	}
	if eff.AuraPeriod != 1000 {
		t.Errorf("expected AuraPeriod 1000, got %d", eff.AuraPeriod)
	}
	if eff.TargetA != spell.TargetDestTargetEnemy {
		t.Errorf("expected TargetDestTargetEnemy, got %d", eff.TargetA)
	}
	if eff.TargetB != spell.TargetUnitAreaEnemy {
		t.Errorf("expected TargetUnitAreaEnemy, got %d", eff.TargetB)
	}
}
