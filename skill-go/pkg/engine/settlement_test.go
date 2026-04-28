package engine

import (
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/stat"
	"skill-go/pkg/unit"
	"testing"
)

// createTestEngine 创建带两个单位的引擎，返回 (engine, caster, target)
func createTestEngine() (*Engine, *unit.Unit, *unit.Unit) {
	eng := New()
	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	target := eng.AddUnitWithID(2, entity.NewEntity(2, entity.TypeCreature, entity.Position{X: 5}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.MaxHealth, 1000)
	target.Stats.SetBase(stat.Health, 1000)
	target.Stats.SetBase(stat.MaxHealth, 1000)
	return eng, caster, target
}

// damageSpellInfo 创建一个简单的伤害法术 SpellInfo
func damageSpellInfo() *spellcore.SpellInfo {
	return &spellcore.SpellInfo{
		ID:   100,
		Name: "TestDamage",
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectType: spellcore.EffectSchoolDamage,
				BasePoints: 100,
				TargetA:    spellcore.TargetUnitTargetEnemy,
			},
		},
	}
}

// healSpellInfo 创建一个简单的治疗法术 SpellInfo
func healSpellInfo() *spellcore.SpellInfo {
	return &spellcore.SpellInfo{
		ID:   101,
		Name: "TestHeal",
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectType: spellcore.EffectHeal,
				BasePoints: 100,
				TargetA:    spellcore.TargetUnitCaster,
			},
		},
	}
}

func TestSettlement_SpellDamageReducesHealth(t *testing.T) {
	eng, caster, target := createTestEngine()

	before := target.Stats.Get(stat.Health)
	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	after := target.Stats.Get(stat.Health)
	if after >= before {
		t.Errorf("expected target health to decrease after damage spell, before=%.0f after=%.0f", before, after)
	}
}

func TestSettlement_SpellHealIncreasesHealth(t *testing.T) {
	eng, caster, _ := createTestEngine()

	// 先扣血再治疗
	caster.Stats.SetBase(stat.Health, 500)
	before := caster.Stats.Get(stat.Health)

	eng.CastSpell(caster, healSpellInfo(), WithTarget(caster.ID()))
	eng.Simulate(100, 50)

	after := caster.Stats.Get(stat.Health)
	if after <= before {
		t.Errorf("expected caster health to increase after heal spell, before=%.0f after=%.0f", before, after)
	}
}

func TestSettlement_HealthClampZero(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 目标只有 50 血，但法术造成 100 伤害
	target.Stats.SetBase(stat.Health, 50)

	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	after := target.Stats.Get(stat.Health)
	if after != 0 {
		t.Errorf("expected health to clamp to 0, got %.0f", after)
	}
}

func TestSettlement_DeathOnZeroHealth(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 监听死亡事件
	var deathEvent event.Event
	eng.Bus().Subscribe(event.OnDeath, func(e event.Event) {
		deathEvent = e
	})

	target.Stats.SetBase(stat.Health, 50)

	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	if target.IsAlive() {
		t.Error("expected target to be dead after health reaches 0")
	}
	if deathEvent.Type != event.OnDeath {
		t.Error("expected OnDeath event to be published")
	}
}

func TestSettlement_DamageBreaksAura(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 先给目标施加一个受伤打断的光环
	auraInfo := &spellcore.SpellInfo{
		ID:       200,
		Name:     "TestBreakOnDamage",
		Duration: 5000,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectType:         spellcore.EffectApplyAura,
				AuraType:           uint16(spellcore.AuraModStat),
				AuraPeriod:         0,
				AuraInterruptFlags: spellcore.AuraInterruptOnDamage,
				TargetA:            spellcore.TargetUnitTargetEnemy,
			},
		},
	}
	eng.CastSpell(caster, auraInfo, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	// 验证光环已施加
	if len(target.GetAppliedAuras()) == 0 {
		t.Fatal("expected aura to be applied before damage")
	}

	// 造成伤害
	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	// 验证光环被移除
	if len(target.GetAppliedAuras()) > 0 {
		t.Error("expected aura with AuraInterruptOnDamage to be removed after taking damage")
	}
}

func TestSettlement_PeriodicDamageReducesHealth(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 施加 DoT 光环
	dotInfo := &spellcore.SpellInfo{
		ID:       300,
		Name:     "TestDoT",
		Duration: 5000,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectType: spellcore.EffectApplyAura,
				AuraType:   uint16(spellcore.AuraPeriodicDamage),
				BasePoints: 50,
				AuraPeriod: 500,
				TargetA:    spellcore.TargetUnitTargetEnemy,
			},
		},
	}

	before := target.Stats.Get(stat.Health)
	eng.CastSpell(caster, dotInfo, WithTarget(target.ID()))
	eng.Simulate(2000, 100)

	after := target.Stats.Get(stat.Health)
	if after >= before {
		t.Errorf("expected target health to decrease from periodic damage, before=%.0f after=%.0f", before, after)
	}
}

func TestSettlement_PeriodicHealIncreasesHealth(t *testing.T) {
	eng, caster, _ := createTestEngine()

	// 先扣血
	caster.Stats.SetBase(stat.Health, 500)

	// 施加 HoT 光环（治疗自己）
	hotInfo := &spellcore.SpellInfo{
		ID:       301,
		Name:     "TestHoT",
		Duration: 5000,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectType: spellcore.EffectApplyAura,
				AuraType:   uint16(spellcore.AuraPeriodicHeal),
				BasePoints: 50,
				AuraPeriod: 500,
				TargetA:    spellcore.TargetUnitCaster,
			},
		},
	}

	before := caster.Stats.Get(stat.Health)
	eng.CastSpell(caster, hotInfo, WithTarget(caster.ID()))
	eng.Simulate(2000, 100)

	after := caster.Stats.Get(stat.Health)
	if after <= before {
		t.Errorf("expected caster health to increase from periodic heal, before=%.0f after=%.0f", before, after)
	}
}

func TestSettlement_DamageToDeadUnitIgnored(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 杀死目标
	target.Kill(1)
	if target.IsAlive() {
		t.Fatal("expected target to be dead after Kill()")
	}

	healthBefore := target.Stats.Get(stat.Health)

	// 对死亡目标造成伤害
	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	healthAfter := target.Stats.Get(stat.Health)
	if healthAfter != healthBefore {
		t.Errorf("expected no damage to dead unit, before=%.0f after=%.0f", healthBefore, healthAfter)
	}
}
