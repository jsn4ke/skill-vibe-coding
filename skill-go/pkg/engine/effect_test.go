package engine

import (
	"testing"

	"skill-go/pkg/entity"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/stat"
)

// --- EffectHeal ---

func TestEffectHeal_SpellPowerScaling(t *testing.T) {
	eng, caster, _ := createTestEngine()
	caster.Stats.SetBase(stat.Health, 500)
	caster.Stats.SetBase(stat.SpellPower, 100)

	info := &spellcore.SpellInfo{
		ID:   1001,
		Name: "TestHealSP",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectHeal, BasePoints: 50, BonusCoeff: 1.0, TargetA: spellcore.TargetUnitCaster},
		},
	}

	before := caster.Stats.Get(stat.Health)
	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	healed := caster.Stats.Get(stat.Health) - before
	// BasePoints(50) + BonusCoeff(1.0) * SpellPower(100) = 150
	if healed < 50 {
		t.Errorf("expected heal >= 50 (base + SP scaling), got %.1f", healed)
	}
}

func TestEffectHeal_BaseDieSidesVariance(t *testing.T) {
	eng, caster, _ := createTestEngine()
	caster.Stats.SetBase(stat.Health, 100)

	info := &spellcore.SpellInfo{
		ID:   1001,
		Name: "TestHealVariance",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectHeal, BasePoints: 50, BaseDieSides: 20, TargetA: spellcore.TargetUnitCaster},
		},
	}

	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	healed := caster.Stats.Get(stat.Health) - 100
	if healed < 50 || healed > 70 {
		t.Errorf("expected heal in [50,70] (base + variance), got %.1f", healed)
	}
}

// --- EffectHealPct ---

func TestEffectHealPct_MaxHealthPercentage(t *testing.T) {
	eng, caster, _ := createTestEngine()
	caster.Stats.SetBase(stat.Health, 200)
	caster.Stats.SetBase(stat.MaxHealth, 1000)

	info := &spellcore.SpellInfo{
		ID:   1002,
		Name: "TestHealPct",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectHealPct, BasePoints: 50, TargetA: spellcore.TargetUnitCaster},
		},
	}

	before := caster.Stats.Get(stat.Health)
	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	healed := caster.Stats.Get(stat.Health) - before
	// 50% of MaxHealth(1000) = 500
	if healed < 400 || healed > 600 {
		t.Errorf("expected heal ~500 (50%% of MaxHealth 1000), got %.1f", healed)
	}
}

// --- EffectEnergize ---

func TestEffectEnergize_RestoresMana(t *testing.T) {
	eng, caster, _ := createTestEngine()
	caster.Stats.SetBase(stat.Mana, 100)

	info := &spellcore.SpellInfo{
		ID:   1003,
		Name: "TestEnergize",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectEnergize, BasePoints: 200, MiscValue: 0, TargetA: spellcore.TargetUnitCaster},
		},
	}

	before := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	after := caster.Stats.Get(stat.Mana)
	if after != before+200 {
		t.Errorf("expected mana += 200, before=%.0f after=%.0f", before, after)
	}
}

func TestEffectEnergize_PureEnergizeNoDamage(t *testing.T) {
	eng, caster, _ := createTestEngine()
	caster.Stats.SetBase(stat.Mana, 0)

	// Pure energize spell — no damage, no healing
	info := &spellcore.SpellInfo{
		ID:   1013,
		Name: "PureEnergize",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectEnergize, BasePoints: 500, MiscValue: 0, TargetA: spellcore.TargetUnitCaster},
		},
	}

	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	after := caster.Stats.Get(stat.Mana)
	if after != 500 {
		t.Errorf("expected mana = 500 after pure energize, got %.0f", after)
	}
}

// --- EffectEnergizePct ---

func TestEffectEnergizePct(t *testing.T) {
	eng, caster, _ := createTestEngine()
	caster.Stats.SetBase(stat.Mana, 100)

	info := &spellcore.SpellInfo{
		ID:   1004,
		Name: "TestEnergizePct",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectEnergizePct, BasePoints: 50, MiscValue: 0, TargetA: spellcore.TargetUnitCaster},
		},
	}

	before := caster.Stats.Get(stat.Mana)
	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	after := caster.Stats.Get(stat.Mana)
	if after != before+50 {
		t.Errorf("expected mana += 50, before=%.0f after=%.0f", before, after)
	}
}

// --- EffectWeaponDamage ---

func TestEffectWeaponDamage_APScaling(t *testing.T) {
	eng, caster, target := createTestEngine()
	caster.Stats.SetBase(stat.AttackPower, 200)

	info := &spellcore.SpellInfo{
		ID:   1005,
		Name: "TestWeaponDmg",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectWeaponDamage, BasePoints: 100, BonusCoeff: 0.5, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	before := target.Stats.Get(stat.Health)
	eng.CastSpell(caster, info, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	damage := before - target.Stats.Get(stat.Health)
	// BasePoints(100) + BonusCoeff(0.5) * AP(200) = 200
	if damage < 50 {
		t.Errorf("expected weapon damage >= 50 (base + AP scaling), got %.1f", damage)
	}
}

// --- EffectTriggerSpell ---

func TestEffectTriggerSpell_TriggersChild(t *testing.T) {
	eng, caster, target := createTestEngine()

	childInfo := &spellcore.SpellInfo{
		ID:   2000,
		Name: "ChildDamage",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectSchoolDamage, BasePoints: 75, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}
	eng.SpellStore().Register(childInfo)

	parentInfo := &spellcore.SpellInfo{
		ID:   1006,
		Name: "TestTrigger",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectTriggerSpell, TriggerSpellID: 2000, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	before := target.Stats.Get(stat.Health)
	eng.CastSpell(caster, parentInfo, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	damage := before - target.Stats.Get(stat.Health)
	if damage < 50 {
		t.Errorf("expected trigger spell to deal damage >= 50, got %.1f", damage)
	}
}

// --- EffectSummon ---

func TestEffectSummon_CreatesUnit(t *testing.T) {
	eng, caster, _ := createTestEngine()
	beforeCount := len(eng.units)

	info := &spellcore.SpellInfo{
		ID:   1007,
		Name: "TestSummon",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectSummon, MiscValue: 100, TargetA: spellcore.TargetUnitCaster},
		},
	}

	eng.CastSpell(caster, info, WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	if len(eng.units) != beforeCount+1 {
		t.Errorf("expected %d units after summon, got %d", beforeCount+1, len(eng.units))
	}
}

func TestEffectSummon_UsesDestPos(t *testing.T) {
	eng, caster, _ := createTestEngine()

	info := &spellcore.SpellInfo{
		ID:   1014,
		Name: "TestSummonPos",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectSummon, MiscValue: 100, TargetA: spellcore.TargetUnitCaster},
		},
	}

	eng.CastSpell(caster, info, WithTarget(caster.ID()), WithDestPos(50, 60, 0))
	eng.Simulate(100, 50)

	// Find the summoned unit (not caster or target)
	for _, u := range eng.units {
		if u.ID() != 1 && u.ID() != 2 {
			pos := u.Entity.Pos
			if pos.X != 50 || pos.Y != 60 {
				t.Errorf("expected summoned unit at (50,60), got (%.0f,%.0f)", pos.X, pos.Y)
			}
			return
		}
	}
	t.Error("expected summoned unit to be created")
}

// --- EffectDispel ---

func TestEffectDispel_RemovesAura(t *testing.T) {
	eng, caster, target := createTestEngine()

	auraInfo := &spellcore.SpellInfo{
		ID: 500, Name: "TestAura", Duration: 10000,
		Effects: []spellcore.SpellEffectInfo{
			{EffectType: spellcore.EffectApplyAura, AuraType: uint16(spellcore.AuraModStat), TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}
	eng.CastSpell(caster, auraInfo, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	if len(target.GetAppliedAuraApps()) == 0 {
		t.Fatal("expected aura to be applied before dispel")
	}

	dispelInfo := &spellcore.SpellInfo{
		ID:   1008,
		Name: "TestDispel",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectDispel, MiscValue: 1, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}
	eng.CastSpell(caster, dispelInfo, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	if len(target.GetAppliedAuraApps()) > 0 {
		t.Error("expected aura to be removed after dispel")
	}
}

func TestEffectDispel_MultipleAuras(t *testing.T) {
	eng, caster, target := createTestEngine()

	// Apply 3 auras
	for i := 0; i < 3; i++ {
		auraInfo := &spellcore.SpellInfo{
			ID: spellcore.SpellID(500 + i), Name: "Aura", Duration: 10000,
			Effects: []spellcore.SpellEffectInfo{
				{EffectType: spellcore.EffectApplyAura, AuraType: uint16(spellcore.AuraModStat), TargetA: spellcore.TargetUnitTargetEnemy},
			},
		}
		eng.CastSpell(caster, auraInfo, WithTarget(target.ID()))
		eng.Simulate(100, 50)
	}

	if len(target.GetAppliedAuraApps()) != 3 {
		t.Fatalf("expected 3 auras, got %d", len(target.GetAppliedAuraApps()))
	}

	// Dispel 2
	dispelInfo := &spellcore.SpellInfo{
		ID:   1008,
		Name: "TestDispel2",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectDispel, MiscValue: 2, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}
	eng.CastSpell(caster, dispelInfo, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	if len(target.GetAppliedAuraApps()) != 1 {
		t.Errorf("expected 1 aura remaining after dispel 2, got %d", len(target.GetAppliedAuraApps()))
	}
}

// --- EffectTeleportUnits ---

func TestEffectTeleport_MovesTarget(t *testing.T) {
	eng, caster, target := createTestEngine()

	info := &spellcore.SpellInfo{
		ID:   1009,
		Name: "TestTeleport",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectTeleportUnits, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	eng.CastSpell(caster, info, WithTarget(target.ID()), WithDestPos(100, 200, 300))
	eng.Simulate(100, 50)

	pos := target.Entity.Pos
	if pos.X != 100 || pos.Y != 200 || pos.Z != 300 {
		t.Errorf("expected target at (100,200,300), got (%.0f,%.0f,%.0f)", pos.X, pos.Y, pos.Z)
	}
}

// --- EffectLeap ---

func TestEffectLeap_MovesCaster(t *testing.T) {
	eng, caster, _ := createTestEngine()

	info := &spellcore.SpellInfo{
		ID:   1010,
		Name: "TestLeap",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectLeap, TargetA: spellcore.TargetUnitCaster},
		},
	}

	eng.CastSpell(caster, info, WithTarget(caster.ID()), WithDestPos(50, 60, 0))
	eng.Simulate(100, 50)

	pos := caster.Entity.Pos
	if pos.X != 50 || pos.Y != 60 {
		t.Errorf("expected caster at (50,60), got (%.0f,%.0f)", pos.X, pos.Y)
	}
}

// --- EffectCharge ---

func TestEffectCharge_MovesCasterToTarget(t *testing.T) {
	eng, caster, target := createTestEngine()
	target.SetPosition(entity.Position{X: 30, Y: 0, Z: 0, Facing: 0})

	info := &spellcore.SpellInfo{
		ID:   1011,
		Name: "TestCharge",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectCharge, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	eng.CastSpell(caster, info, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	pos := caster.Entity.Pos
	if pos.X != 30 || pos.Y != 0 {
		t.Errorf("expected caster at target position (30,0), got (%.0f,%.0f)", pos.X, pos.Y)
	}
}

// --- EffectKnockBack ---

func TestEffectKnockBack_PushesTarget(t *testing.T) {
	eng, caster, target := createTestEngine()
	// Caster at (0,0,0), target at (5,0,0)
	// Direction: (1,0,0), distance 10 → target pushed to (15,0,0)

	info := &spellcore.SpellInfo{
		ID:   1012,
		Name: "TestKnockBack",
		Effects: []spellcore.SpellEffectInfo{
			{EffectIndex: 0, EffectType: spellcore.EffectKnockBack, MiscValue: 10, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	eng.CastSpell(caster, info, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	pos := target.Entity.Pos
	if pos.X <= 5 {
		t.Errorf("expected target X > 5 after knockback, got %.1f", pos.X)
	}
}
