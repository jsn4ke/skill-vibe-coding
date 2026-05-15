package engine

import (
	"skill-go/pkg/combat"
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
	if len(target.GetAppliedAuraApps()) == 0 {
		t.Fatal("expected aura to be applied before damage")
	}

	// 造成伤害
	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	// 验证光环被移除
	if len(target.GetAppliedAuraApps()) > 0 {
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

func TestSettlement_ArmorReducesPhysicalDamage(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 给目标设置护甲
	target.Stats.SetBase(stat.Armor, 1000)

	// 物理伤害法术（School 默认 0 = Physical）
	info := &spellcore.SpellInfo{
		ID:     102,
		Name:   "PhysicalHit",
		School: uint8(combat.SchoolPhysical),
		Effects: []spellcore.SpellEffectInfo{
			{EffectType: spellcore.EffectSchoolDamage, BasePoints: 1000, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	eng.CastSpell(caster, info, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	damageTaken := 1000 - target.Stats.Get(stat.Health)
	// 无护甲时 damageTaken = 1000，有护甲 1000 时 damageTaken < 1000
	if damageTaken >= 1000 {
		t.Errorf("expected armor to reduce damage, but took %.0f damage", damageTaken)
	}
	// 验证减伤公式：1000 * armor/(armor+467.5) = 1000 * 1000/1467.5 ≈ 681.4
	// 实际伤害 = 1000 - 681.4 ≈ 318.6
	if damageTaken < 300 || damageTaken > 350 {
		t.Errorf("expected ~318 damage after armor, got %.0f", damageTaken)
	}
}

func TestSettlement_ResistanceReducesMagicDamage(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 给目标设置抗性
	target.Stats.SetBase(stat.Resistance, 300)

	// 火焰伤害法术
	info := &spellcore.SpellInfo{
		ID:     103,
		Name:   "Fireball",
		School: uint8(combat.SchoolFire),
		Effects: []spellcore.SpellEffectInfo{
			{EffectType: spellcore.EffectSchoolDamage, BasePoints: 1000, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	eng.CastSpell(caster, info, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	damageTaken := 1000 - target.Stats.Get(stat.Health)
	// 无抗性时 damageTaken = 1000，有抗性时 damageTaken < 1000
	if damageTaken >= 1000 {
		t.Errorf("expected resistance to reduce damage, but took %.0f damage", damageTaken)
	}
	// 减伤公式：1000 * (1 - 300/450) ≈ 333
	if damageTaken < 320 || damageTaken > 345 {
		t.Errorf("expected ~333 damage after resistance, got %.0f", damageTaken)
	}
}

func TestSettlement_NoArmorNoResistance_FullDamage(t *testing.T) {
	eng, caster, target := createTestEngine()

	info := &spellcore.SpellInfo{
		ID:     104,
		Name:   "Fireball",
		School: uint8(combat.SchoolFire),
		Effects: []spellcore.SpellEffectInfo{
			{EffectType: spellcore.EffectSchoolDamage, BasePoints: 200, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}

	eng.CastSpell(caster, info, WithTarget(target.ID()))
	eng.Simulate(100, 50)

	damageTaken := 1000 - target.Stats.Get(stat.Health)
	if damageTaken != 200 {
		t.Errorf("expected full 200 damage with no armor/resistance, got %.0f", damageTaken)
	}
}

func TestSettlement_ProcTriggersSpell(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 注册一个 proc：攻击者造成法术伤害时，100% 触发一个额外伤害法术
	procDamageSpell := &spellcore.SpellInfo{
		ID:   500,
		Name: "ProcDamage",
		Effects: []spellcore.SpellEffectInfo{
			{EffectType: spellcore.EffectSchoolDamage, BasePoints: 50, TargetA: spellcore.TargetUnitTargetEnemy},
		},
	}
	eng.SpellStore().Register(procDamageSpell)

	eng.ProcMgr().Register(spellcore.Entry{
		SpellID:      100,
		Flags:        spellcore.FlagSpellDamageDealt,
		Chance:       1.0, // 100%
		TriggerSpell: spellcore.SpellID(procDamageSpell.ID),
	})

	// 施放主法术
	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(100, 50)

	// 主法术伤害 100 + Proc 伤害 50 = 150 总伤害
	damageTaken := 1000 - target.Stats.Get(stat.Health)
	if damageTaken < 140 {
		t.Errorf("expected ~150 damage (100 main + 50 proc), got %.0f", damageTaken)
	}
}

func TestSettlement_AuraModifiesStat(t *testing.T) {
	eng, caster, _ := createTestEngine()

	// 施加一个增加法术强度的光环
	buffInfo := &spellcore.SpellInfo{
		ID:       600,
		Name:     "SpellPowerBuff",
		Duration: 5000,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectType: spellcore.EffectApplyAura,
				AuraType:   uint16(spellcore.AuraModSpellPower),
				BasePoints: 200,
				TargetA:    spellcore.TargetUnitCaster,
			},
		},
	}

	spBefore := caster.Stats.Get(stat.SpellPower)
	eng.CastSpell(caster, buffInfo, WithTarget(caster.ID()))
	eng.Simulate(100, 50)
	spAfter := caster.Stats.Get(stat.SpellPower)

	if spAfter <= spBefore {
		t.Errorf("expected SpellPower to increase after buff, before=%.0f after=%.0f", spBefore, spAfter)
	}
	if spAfter != spBefore+200 {
		t.Errorf("expected SpellPower to increase by 200, got increase of %.0f", spAfter-spBefore)
	}

	// 等待光环过期
	eng.Simulate(6000, 100)
	spExpired := caster.Stats.Get(stat.SpellPower)
	if spExpired != spBefore {
		t.Errorf("expected SpellPower to return to %.0f after expiry, got %.0f", spBefore, spExpired)
	}
}

func TestDamageCancelsPreparingSpell(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 目标正在施法一个有 InterruptDamageCancels 的法术
	castInfo := &spellcore.SpellInfo{
		ID:             400,
		Name:           "LongCast",
		CastTime:       3000,
		InterruptFlags: spellcore.InterruptDamageCancels,
	}
	s := eng.CastSpell(target, castInfo, WithTarget(caster.ID()))
	if s == nil || s.State != spellcore.StatePreparing {
		t.Fatal("spell should be in preparing state")
	}

	// 对目标造成伤害
	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(200, 100)

	if s.State != spellcore.StateFinished {
		t.Fatal("preparing spell with InterruptDamageCancels should be cancelled when target takes damage")
	}
}

func TestDamageNoCancelWithoutFlag(t *testing.T) {
	eng, caster, target := createTestEngine()

	// 目标正在施法一个没有 InterruptDamageCancels 的法术
	castInfo := &spellcore.SpellInfo{
		ID:       401,
		Name:     "ToughCast",
		CastTime: 3000,
	}
	s := eng.CastSpell(target, castInfo, WithTarget(caster.ID()))
	if s == nil || s.State != spellcore.StatePreparing {
		t.Fatal("spell should be in preparing state")
	}

	eng.CastSpell(caster, damageSpellInfo(), WithTarget(target.ID()))
	eng.Simulate(200, 100)

	if s.State == spellcore.StateFinished {
		t.Fatal("spell without InterruptDamageCancels should NOT be cancelled by damage")
	}
}
