package spell_test

import (
	"math"
	"testing"

	"skill-go/pkg/aura"
	"skill-go/pkg/effect"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
)

type testUnit struct {
	id       uint64
	alive    bool
	stats    *stat.StatSet
	pos      testPos
	targets  map[uint64]*testUnit
}

func (u *testUnit) GetID() uint64                          { return u.id }
func (u *testUnit) IsAlive() bool                          { return u.alive }
func (u *testUnit) CanCast() bool                          { return u.alive }
func (u *testUnit) IsMoving() bool                         { return false }
func (u *testUnit) GetPosition() spell.Position            { return &u.pos }
func (u *testUnit) GetTargetPosition(targetID uint64) spell.Position {
	if t, ok := u.targets[targetID]; ok {
		return &t.pos
	}
	return &u.pos
}
func (u *testUnit) GetStatValue(st uint8) float64          { return u.stats.Get(stat.StatType(st)) }
func (u *testUnit) ModifyPower(pt uint8, amount float64) bool { return true }

type testPos struct {
	x, y, z, facing float64
}

func (p *testPos) GetX() float64      { return p.x }
func (p *testPos) GetY() float64      { return p.y }
func (p *testPos) GetZ() float64      { return p.z }
func (p *testPos) GetFacing() float64 { return p.facing }

func fireballInfo() *spell.SpellInfo {
	return &spell.SpellInfo{
		ID:         25306,
		Name:       "Fireball",
		CastTime:   3500,
		RangeMax:   35,
		PowerCost:  410,
		PowerType:  0,
		Duration:   8000,
		Speed:      20.0,
		Effects: []spell.SpellEffectInfo{
			{
				EffectIndex: 0,
				EffectType:  spell.EffectSchoolDamage,
				BasePoints:  678,
				BonusCoeff:  1.0,
				TargetA:     spell.TargetUnitTargetEnemy,
			},
			{
				EffectIndex: 1,
				EffectType:  spell.EffectApplyAura,
				BasePoints:  19,
				BonusCoeff:  0.125,
				AuraType:    uint16(aura.AuraPeriodicDamage),
				AuraPeriod:  2000,
				TargetA:     spell.TargetUnitTargetEnemy,
			},
		},
	}
}

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

// Task 5.1: Updated full cast lifecycle with projectile delay
func TestFireball_FullCastLifecycle(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	info := fireballInfo()
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2

	result := s.Prepare()
	if result != spell.CastOK {
		t.Fatalf("expected CastOK, got %v", result)
	}
	if s.State != spell.StatePreparing {
		t.Fatalf("expected StatePreparing, got %v", s.State)
	}

	// Cast time completes
	s.Update(3500)
	if s.State != spell.StateLaunched {
		t.Fatalf("expected StateLaunched after cast, got %v", s.State)
	}

	// No damage yet
	if s.TargetInfos[0].Damage != 0 {
		t.Fatalf("expected 0 damage during flight, got %.0f", s.TargetInfos[0].Damage)
	}

	// Projectile arrives
	hitDelay := s.HitTimer
	s.Update(hitDelay)
	if s.State != spell.StateFinished {
		t.Fatalf("expected StateFinished, got %v", s.State)
	}

	ti := s.TargetInfos[0]
	minDmg := 678.0 + 100.0
	if ti.Damage < minDmg {
		t.Errorf("expected damage >= %.0f, got %.0f", minDmg, ti.Damage)
	}
}

// Task 5.2: No damage before projectile arrives
func TestFireball_NoDamageDuringFlight(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 30)
	caster.targets[2] = target

	info := fireballInfo()
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(3500) // cast completes

	if s.State != spell.StateLaunched {
		t.Fatalf("expected StateLaunched, got %v", s.State)
	}
	if s.HitTimer <= 0 {
		t.Fatal("expected positive HitTimer for projectile spell")
	}

	// Partial update — not enough for projectile to arrive
	s.Update(100)

	if s.State != spell.StateLaunched {
		t.Fatalf("expected still StateLaunched, got %v", s.State)
	}
	if s.TargetInfos[0].Damage != 0 {
		t.Errorf("expected 0 damage during flight, got %.0f", s.TargetInfos[0].Damage)
	}
}

// Task 5.3: Hit delay calculation with known distance and speed
func TestFireball_HitDelayCalculation(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 30)
	caster.targets[2] = target

	info := fireballInfo() // Speed = 20.0
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(3500) // cast completes, enters StateLaunched

	// distance = 30, Speed = 20 → hitDelay = 30/20 * 1000 = 1500ms
	expected := int32(1500)
	if s.HitTimer != expected {
		t.Errorf("expected HitTimer %d, got %d", expected, s.HitTimer)
	}
}

// Task 5.4: MinDuration clamp and 5-yard distance clamp
func TestFireball_DistanceClamp(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 2) // only 2 yards away
	caster.targets[2] = target

	info := fireballInfo() // Speed = 20, MinDuration = 0
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(3500)

	// distance clamped to 5 yards → 5/20 * 1000 = 250ms
	expected := int32(250)
	if s.HitTimer != expected {
		t.Errorf("expected HitTimer %d (5yd clamp), got %d", expected, s.HitTimer)
	}
}

func TestFireball_MinDurationClamp(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 2)
	caster.targets[2] = target

	info := fireballInfo()
	info.MinDuration = 500 // min 500ms flight time
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(3500)

	// dist/Speed = 250ms, but MinDuration = 500ms → 500 wins
	if s.HitTimer != 500 {
		t.Errorf("expected HitTimer 500 (MinDuration clamp), got %d", s.HitTimer)
	}
}

func TestFireball_CritIncreasesDamage(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	info := fireballInfo()

	totalNormal := 0.0
	totalCrit := 0.0
	runs := 100

	for i := 0; i < runs; i++ {
		s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
		s.Targets.UnitTargetID = 2
		s.Prepare()
		s.Update(3500) // cast
		s.Update(s.HitTimer) // projectile
		totalNormal += s.TargetInfos[0].Damage
	}

	for i := 0; i < runs; i++ {
		s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
		s.Targets.UnitTargetID = 2
		s.Prepare()
		s.SelectTargets()
		for j := range s.TargetInfos {
			s.TargetInfos[j].Crit = true
		}
		effect.ProcessAll(s, spell.HandleHit)
		totalCrit += s.TargetInfos[0].Damage
	}

	avgNormal := totalNormal / float64(runs)
	avgCrit := totalCrit / float64(runs)

	ratio := avgCrit / avgNormal
	if math.Abs(ratio-1.5) > 0.1 {
		t.Errorf("expected crit/normal ratio ~1.5, got %.2f", ratio)
	}
}

func TestFireball_AppliesDoTAura(t *testing.T) {
	caster := newTestUnit(1, 0)
	target := newTestUnit(2, 10)
	caster.targets[2] = target

	info := fireballInfo()
	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()

	origFn := spell.ProcessEffectsFn
	spell.ProcessEffectsFn = func(sp *spell.Spell, mode spell.EffectHandleMode) {
		effect.ProcessAll(sp, mode)
	}
	defer func() { spell.ProcessEffectsFn = origFn }()

	s.Update(3500)       // cast
	s.Update(s.HitTimer) // projectile

	ctx := &effect.Context{
		Spell:            s,
		EffectInfo:       &info.Effects[1],
		CasterID:         1,
		TargetID:         2,
		Mode:             spell.HandleHit,
		CasterSpellPower: 100,
	}
	effect.Process(ctx)

	if ctx.AppliedAura == nil {
		t.Fatal("expected DoT aura to be created")
	}
	if ctx.AppliedAura.AuraType != aura.AuraPeriodicDamage {
		t.Errorf("expected AuraPeriodicDamage, got %v", ctx.AppliedAura.AuraType)
	}
}
