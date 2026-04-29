package spellcore

import (
	"testing"
	"time"
)

func TestProcess_DispatchesCorrectly(t *testing.T) {
	t.Run("EffectSchoolDamage dispatches", func(t *testing.T) {
		ctx := &EffectContext{
			EffectInfo: &SpellEffectInfo{
				EffectType:   EffectSchoolDamage,
				BasePoints:   100,
				BonusCoeff:   1.0,
				BaseDieSides: 0,
			},
			Mode:             HandleLaunchTarget,
			CasterSpellPower: 50,
			Crit:             false,
			Spell:            &Spell{ID: 1, Info: &SpellInfo{}},
		}
		Process(ctx)
		if ctx.FinalDamage != 150 { // 100 + 1.0*50
			t.Errorf("FinalDamage = %v, want 150", ctx.FinalDamage)
		}
	})

	t.Run("EffectSchoolDamage with crit", func(t *testing.T) {
		ctx := &EffectContext{
			EffectInfo: &SpellEffectInfo{
				EffectType: EffectSchoolDamage,
				BasePoints: 100,
				BonusCoeff: 0,
			},
			Mode:  HandleLaunchTarget,
			Crit:  true,
			Spell: &Spell{ID: 1, Info: &SpellInfo{}},
		}
		Process(ctx)
		if ctx.FinalDamage != 150 { // 100 * 1.5
			t.Errorf("FinalDamage with crit = %v, want 150", ctx.FinalDamage)
		}
	})

	t.Run("EffectSchoolDamage wrong mode ignored", func(t *testing.T) {
		ctx := &EffectContext{
			EffectInfo: &SpellEffectInfo{
				EffectType: EffectSchoolDamage,
				BasePoints: 100,
				BonusCoeff: 1.0,
			},
			Mode:             HandleHitTarget,
			CasterSpellPower: 50,
			Spell:            &Spell{ID: 1, Info: &SpellInfo{}},
		}
		Process(ctx)
		if ctx.FinalDamage != 0 {
			t.Errorf("wrong mode should not set damage, got %v", ctx.FinalDamage)
		}
	})

	t.Run("EffectHeal dispatches", func(t *testing.T) {
		ctx := &EffectContext{
			EffectInfo: &SpellEffectInfo{
				EffectType: EffectHeal,
				BasePoints: 200,
			},
			Mode:  HandleLaunchTarget,
			Spell: &Spell{ID: 1, Info: &SpellInfo{}},
		}
		Process(ctx)
		if ctx.FinalHeal != 200 {
			t.Errorf("FinalHeal = %v, want 200", ctx.FinalHeal)
		}
	})

	t.Run("unknown effect type does nothing", func(t *testing.T) {
		ctx := &EffectContext{
			EffectInfo: &SpellEffectInfo{EffectType: EffectType(999)},
			Mode:       HandleLaunchTarget,
			Spell:      &Spell{ID: 1, Info: &SpellInfo{}},
		}
		Process(ctx) // should not panic
	})
}

func TestHandleApplyAura(t *testing.T) {
	t.Run("creates aura from effect", func(t *testing.T) {
		ctx := &EffectContext{
			Spell: &Spell{
				ID:   SpellID(100),
				Info: &SpellInfo{ID: SpellID(100), Name: "TestSpell", Duration: 5000},
				Targets: TargetData{
					DestPos: [3]float64{10, 20, 0},
				},
				Caster: &mockCaster{id: 1},
			},
			EffectInfo: &SpellEffectInfo{
				EffectType: EffectApplyAura,
				AuraType:   uint16(AuraPeriodicDamage),
				AuraPeriod: 1000,
				BasePoints: 50,
				BonusCoeff: 0.5,
				TargetA:    TargetUnitTargetEnemy,
				TargetB:    TargetNone,
			},
			CasterID: 1,
			TargetID: 2,
			Mode:     HandleHitTarget,
		}

		Process(ctx)

		if ctx.AppliedAura == nil {
			t.Fatal("expected aura to be created")
		}
		a := ctx.AppliedAura
		if a.SpellID != SpellID(100) {
			t.Errorf("aura SpellID = %v, want 100", a.SpellID)
		}
		if a.TargetID != 2 {
			t.Errorf("aura TargetID = %v, want 2", a.TargetID)
		}
		if a.AuraType != AuraPeriodicDamage {
			t.Errorf("aura AuraType = %v, want PeriodicDamage", a.AuraType)
		}
		if len(a.Effects) != 1 || a.Effects[0].Period != 1000*time.Millisecond {
			t.Errorf("aura effect period = %v, want 1s", a.Effects[0].Period)
		}
	})

	t.Run("area effect targets caster", func(t *testing.T) {
		ctx := &EffectContext{
			Spell: &Spell{
				ID:      SpellID(100),
				Info:    &SpellInfo{ID: SpellID(100), Name: "AreaSpell", Duration: 3000},
				Targets: TargetData{DestPos: [3]float64{10, 20, 0}},
				Caster:  &mockCaster{id: 1},
			},
			EffectInfo: &SpellEffectInfo{
				EffectType: EffectApplyAura,
				AuraType:   uint16(AuraModStat),
				TargetA:    TargetUnitDestAreaEnemy,
				TargetB:    TargetNone,
			},
			CasterID: 1,
			TargetID: 99,
			Mode:     HandleHitTarget,
		}

		Process(ctx)

		if ctx.AppliedAura == nil {
			t.Fatal("expected aura to be created")
		}
		// Area aura should target caster, not the hit target
		if ctx.AppliedAura.TargetID != 1 {
			t.Errorf("area aura TargetID = %v, want 1 (caster)", ctx.AppliedAura.TargetID)
		}
		if !ctx.AppliedAura.IsAreaAura {
			t.Error("area aura should have IsAreaAura = true")
		}
	})

	t.Run("none aura type skipped", func(t *testing.T) {
		ctx := &EffectContext{
			Spell: &Spell{
				ID:     SpellID(100),
				Info:   &SpellInfo{ID: SpellID(100), Name: "Test", Duration: 5000},
				Caster: &mockCaster{id: 1},
			},
			EffectInfo: &SpellEffectInfo{
				EffectType: EffectApplyAura,
				AuraType:   uint16(AuraNone),
			},
			Mode: HandleHitTarget,
		}
		Process(ctx)
		if ctx.AppliedAura != nil {
			t.Error("AuraNone should not create aura")
		}
	})
}

// mockCaster implements Caster for testing
type mockCaster struct {
	id      uint64
	alive   bool
	casting bool
	moving  bool
	pos     Position
}

func (c *mockCaster) GetID() uint64                             { return c.id }
func (c *mockCaster) IsAlive() bool                             { return c.alive }
func (c *mockCaster) CanCast() bool                             { return c.alive && !c.casting }
func (c *mockCaster) GetPosition() Position                     { return c.pos }
func (c *mockCaster) GetTargetPosition(uint64) Position         { return c.pos }
func (c *mockCaster) GetStatValue(st uint8) float64             { return 0 }
func (c *mockCaster) ModifyPower(pt uint8, amount float64) bool { return true }
func (c *mockCaster) IsMoving() bool                            { return c.moving }

type mockPosition struct{ x, y, z, facing float64 }

func (p *mockPosition) GetX() float64      { return p.x }
func (p *mockPosition) GetY() float64      { return p.y }
func (p *mockPosition) GetZ() float64      { return p.z }
func (p *mockPosition) GetFacing() float64 { return p.facing }
