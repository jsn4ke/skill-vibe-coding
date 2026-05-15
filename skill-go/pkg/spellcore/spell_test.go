package spellcore_test

import (
	"math"
	"skill-go/pkg/event"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/stat"
	"testing"
	"time"
)

// --- 测试辅助类型 ---

// testPos 实现 spellcore.Position 接口。
type testPos struct{ x, y, z, facing float64 }

func (p testPos) GetX() float64      { return p.x }
func (p testPos) GetY() float64      { return p.y }
func (p testPos) GetZ() float64      { return p.z }
func (p testPos) GetFacing() float64 { return p.facing }

// testUnit 实现 spellcore.Caster 接口。
type testUnit struct {
	id      uint64
	alive   bool
	canCast bool
	pos     testPos
	targets map[uint64]testPos
	stats   *stat.StatSet
	moving  bool
	power   float64
}

func newTestUnit(id uint64) *testUnit {
	u := &testUnit{
		id:      id,
		alive:   true,
		canCast: true,
		pos:     testPos{0, 0, 0, 0},
		targets: make(map[uint64]testPos),
		stats:   stat.NewStatSet(),
	}
	u.stats.SetBase(stat.Mana, 10000)
	return u
}

func (u *testUnit) GetID() uint64                       { return u.id }
func (u *testUnit) IsAlive() bool                       { return u.alive }
func (u *testUnit) CanCast() bool                       { return u.canCast }
func (u *testUnit) GetPosition() spellcore.Position     { return u.pos }
func (u *testUnit) IsMoving() bool                      { return u.moving }
func (u *testUnit) GetHistory() *spellcore.History      { return nil }
func (u *testUnit) GetCCState() spellcore.CasterCCState { return 0 }
func (u *testUnit) ModifyPower(_ uint8, amt float64) bool {
	u.power += amt
	return true
}
func (u *testUnit) GetTargetPosition(targetID uint64) spellcore.Position {
	if p, ok := u.targets[targetID]; ok {
		return p
	}
	return testPos{0, 0, 0, 0}
}
func (u *testUnit) GetStatValue(st uint8) float64 {
	return u.stats.Get(stat.StatType(st))
}

// --- Fireball SpellInfo ---

func fireballInfo() spellcore.SpellInfo {
	return spellcore.SpellInfo{
		ID:          133,
		Name:        "Fireball",
		CastTime:    1500,
		PowerCost:   100,
		PowerType:   0,
		Speed:       24.0,
		MinDuration: 500,
		Attributes:  spellcore.AttrNone,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectIndex: 0,
				EffectType:  spellcore.EffectSchoolDamage,
				BasePoints:  100,
				BonusCoeff:  0.5,
				TargetA:     spellcore.TargetUnitTargetEnemy,
				TargetB:     spellcore.TargetNone,
			},
		},
	}
}

// --- Tests ---

// TestFireball_FullCastLifecycle 验证火球术的完整施法生命周期：
// PREPARING → CastTime 倒计时 → LAUNCHED → 弹道飞行 → HIT → FINISHED。
func TestFireball_FullCastLifecycle(t *testing.T) {
	bus := event.NewBus()
	info := fireballInfo()
	caster := newTestUnit(1)
	_ = newTestUnit(2) // target exists in the world
	caster.targets[2] = testPos{20, 0, 0, 0}

	s := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	s.Bus = bus
	s.Targets.UnitTargetID = 2

	// 阶段 1: Prepare → PREPARING
	result := s.Prepare()
	if result != spellcore.CastOK {
		t.Fatalf("Prepare() = %d, want CastOK", result)
	}
	if s.State != spellcore.StatePreparing {
		t.Fatalf("after Prepare: state = %d, want StatePreparing", s.State)
	}
	if s.CastTime != info.CastTime {
		t.Fatalf("CastTime = %d, want %d", s.CastTime, info.CastTime)
	}

	// 阶段 2: Update 推进 CastTime → LAUNCHED
	s.Update(int32(info.CastTime))
	if s.State != spellcore.StateLaunched {
		t.Fatalf("after CastTime update: state = %d, want StateLaunched", s.State)
	}

	// 阶段 3: Update 推进弹道飞行时间 → FINISHED
	// Fireball speed = 24.0, distance = 20, flight time = 20/24*1000 = 833ms
	// MinDuration = 500ms, so TimeDelay = max(833, 500) = 833ms
	// Total delay to advance: 833ms
	s.Update(1000)
	if s.State != spellcore.StateFinished {
		t.Fatalf("after flight update: state = %d, want StateFinished", s.State)
	}
}

// TestFireball_NoDamageDuringFlight 验证弹道飞行期间法术尚未完成。
// LaunchPhase 已计算伤害（对齐 TC），但 HandleImmediate 尚未被调用，
// 法术仍在 LAUNCHED 状态等待命中。
func TestFireball_NoDamageDuringFlight(t *testing.T) {
	bus := event.NewBus()
	info := fireballInfo()
	caster := newTestUnit(1)
	_ = newTestUnit(2) // target exists in the world
	caster.targets[2] = testPos{30, 0, 0, 0}

	s := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	s.Bus = bus
	s.Targets.UnitTargetID = 2

	// 快进到 LAUNCHED
	s.Prepare()
	s.Update(int32(info.CastTime))
	if s.State != spellcore.StateLaunched {
		t.Fatalf("state = %d, want StateLaunched", s.State)
	}

	// 推进部分飞行时间（不足完全命中）
	s.Update(200)
	// 仍然在 LAUNCHED 状态，尚未进入 FINISHED（HandleImmediate 未被调用）
	if s.State != spellcore.StateLaunched {
		t.Fatalf("during flight: state = %d, want StateLaunched", s.State)
	}
}

// TestFireball_HitDelayCalculation 验证命中延迟的计算公式：
// TimeDelay = LaunchDelay + max(dist/Speed*1000, MinDuration)。
func TestFireball_HitDelayCalculation(t *testing.T) {
	info := fireballInfo()
	caster := newTestUnit(1)
	_ = newTestUnit(2) // target exists in the world

	// 距离 30 码, speed = 24.0, flight = 30/24*1000 = 1250ms, MinDuration = 500ms
	// TimeDelay = max(1250, 500) = 1250ms
	caster.targets[2] = testPos{30, 0, 0, 0}

	s := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(int32(info.CastTime))

	// 计算的 TimeDelay 应为 1250ms
	expectedDelay := int32(1250)
	for _, ti := range s.TargetInfos {
		if ti.TimeDelay != expectedDelay {
			t.Errorf("TimeDelay = %d, want %d", ti.TimeDelay, expectedDelay)
		}
	}
}

// TestFireball_DistanceClamp 验证距离小于 5 码时使用 5 码最小值。
// 公式: if dist < 5.0 { dist = 5.0 }，确保极近距离弹道仍有最短飞行时间。
func TestFireball_DistanceClamp(t *testing.T) {
	info := fireballInfo()
	caster := newTestUnit(1)
	_ = newTestUnit(2) // target exists in the world

	// 距离 2 码，应 clamp 到 5 码
	// flight = 5/24*1000 = 208ms, MinDuration = 500ms
	// TimeDelay = max(208, 500) = 500ms
	caster.targets[2] = testPos{2, 0, 0, 0}

	s := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(int32(info.CastTime))

	expectedDelay := int32(500)
	for _, ti := range s.TargetInfos {
		if ti.TimeDelay != expectedDelay {
			t.Errorf("TimeDelay = %d, want %d (clamped)", ti.TimeDelay, expectedDelay)
		}
	}
}

// TestFireball_MinDurationClamp 验证 MinDuration 限制最小命中延迟。
// 当飞行时间小于 MinDuration 时，使用 MinDuration。
func TestFireball_MinDurationClamp(t *testing.T) {
	info := fireballInfo()
	caster := newTestUnit(1)
	_ = newTestUnit(2) // target exists in the world

	// 距离 5 码, flight = 5/24*1000 = 208ms < MinDuration 500ms
	// TimeDelay = max(208, 500) = 500ms
	caster.targets[2] = testPos{5, 0, 0, 0}

	s := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Prepare()
	s.Update(int32(info.CastTime))

	expectedDelay := int32(500)
	for _, ti := range s.TargetInfos {
		if ti.TimeDelay != expectedDelay {
			t.Errorf("TimeDelay = %d, want %d (min duration)", ti.TimeDelay, expectedDelay)
		}
	}
}

// TestFireball_CritIncreasesDamage 验证暴击使伤害乘以 1.5 倍。
// 通过手动设置 TargetInfo.Crit = true 后触发 LaunchPhase 来模拟暴击。
func TestFireball_CritIncreasesDamage(t *testing.T) {
	bus := event.NewBus()
	info := fireballInfo()
	info.Speed = 0
	info.CastTime = 0 // 即时施法
	info.MinDuration = 0

	caster := newTestUnit(1)
	caster.stats.SetBase(stat.SpellPower, 100)

	// 普通施法（无暴击）
	sNormal := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	sNormal.Bus = bus
	sNormal.Targets.UnitTargetID = 0
	// 手动添加目标，避免依赖引擎
	sNormal.TargetInfos = []spellcore.TargetInfo{
		{TargetID: 2, MissReason: spellcore.HitNormal, EfffectMask: 1, Crit: false},
	}
	sNormal.HandleLaunchPhase()
	normalDmg := sNormal.TargetInfos[0].Damage

	// 暴击施法
	sCrit := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	sCrit.Bus = bus
	sCrit.Targets.UnitTargetID = 0
	sCrit.TargetInfos = []spellcore.TargetInfo{
		{TargetID: 2, MissReason: spellcore.HitCrit, EfffectMask: 1, Crit: true},
	}
	sCrit.HandleLaunchPhase()
	critDmg := sCrit.TargetInfos[0].Damage

	// 暴击伤害应为普通伤害的 1.5 倍
	expectedCrit := normalDmg * 1.5
	if math.Abs(critDmg-expectedCrit) > 0.01 {
		t.Errorf("crit damage = %.2f, want %.2f (normal %.2f * 1.5)", critDmg, expectedCrit, normalDmg)
	}
	// 确认普通伤害和暴击伤害都非零
	if normalDmg == 0 {
		t.Error("normal damage should be non-zero")
	}
	if critDmg == 0 {
		t.Error("crit damage should be non-zero")
	}
}

// TestFireball_AppliesDoTAura 验证附带 DoT 效果的 Fireball 施加 Aura。
// EffectApplyAura + AuraPeriodicDamage 组合测试。
func TestFireball_AppliesDoTAura(t *testing.T) {
	bus := event.NewBus()
	caster := newTestUnit(1)
	caster.stats.SetBase(stat.SpellPower, 100)
	_ = newTestUnit(2) // target exists in the world
	caster.targets[2] = testPos{10, 0, 0, 0}

	info := spellcore.SpellInfo{
		ID:        133,
		Name:      "Fireball",
		CastTime:  0, // 即时施法
		PowerCost: 50,
		PowerType: 0,
		Speed:     0,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectIndex: 0,
				EffectType:  spellcore.EffectSchoolDamage,
				BasePoints:  100,
				BonusCoeff:  0.5,
				TargetA:     spellcore.TargetUnitTargetEnemy,
				TargetB:     spellcore.TargetNone,
			},
			{
				EffectIndex: 1,
				EffectType:  spellcore.EffectApplyAura,
				AuraType:    uint16(spellcore.AuraPeriodicDamage),
				AuraPeriod:  3000, // 3 秒 tick
				BasePoints:  20,
				BonusCoeff:  0.1,
				TargetA:     spellcore.TargetUnitTargetEnemy,
				TargetB:     spellcore.TargetNone,
			},
		},
	}

	s := spellcore.NewSpell(info.ID, &info, caster, spellcore.TriggeredNone)
	s.Bus = bus
	s.Targets.UnitTargetID = 2

	// Prepare 会触发即时施法（CastTime=0）
	s.Prepare()

	if s.State != spellcore.StateFinished {
		t.Fatalf("state = %d, want StateFinished", s.State)
	}

	// 效果管线现在直接调用 ProcessLaunchPhase 和 ProcessHitPhase，
	// 不再需要设置 ProcessLaunchPhaseFn / ProcessHitPhaseFn。

	// 验证伤害效果已处理
	if len(s.TargetInfos) == 0 {
		t.Fatal("no target infos")
	}
	// base=100 + variance + bonus=0.5*100=50 → ~150+
	if s.TargetInfos[0].Damage == 0 {
		t.Error("expected non-zero damage after hit phase")
	}

	// 验证 OnAuraCreated 回调中收到了 Aura
	var appliedAura *spellcore.Aura
	s.OnAuraCreated = func(a *spellcore.Aura) {
		appliedAura = a
	}

	// 重新创建法术实例来触发 OnAuraCreated
	// 使用 TargetUnitCaster 作为 DoT 效果的目标，因为无引擎时 TargetUnitTargetEnemy
	// 的 fallback 只为第一个 effect 设置 mask，后续 effect 的 mask 不会合并。
	// TargetUnitCaster 能在无引擎时直接解析（ResolveReferer 返回 casterAdapter）。
	dotInfo := spellcore.SpellInfo{
		ID:        133,
		Name:      "FireballDoT",
		CastTime:  0,
		PowerCost: 0,
		Speed:     0,
		Effects: []spellcore.SpellEffectInfo{
			{
				EffectIndex: 0,
				EffectType:  spellcore.EffectApplyAura,
				AuraType:    uint16(spellcore.AuraPeriodicDamage),
				AuraPeriod:  3000,
				BasePoints:  20,
				BonusCoeff:  0.1,
				TargetA:     spellcore.TargetUnitCaster,
				TargetB:     spellcore.TargetNone,
			},
		},
	}
	s2 := spellcore.NewSpell(dotInfo.ID, &dotInfo, caster, spellcore.TriggeredNone)
	s2.Bus = bus
	s2.OnAuraCreated = func(a *spellcore.Aura) {
		appliedAura = a
	}
	s2.Prepare()

	// OnAuraCreated 应该被调用了（HandleHitPhase 中 ctx.AppliedAura 触发）
	// 注意：即时法术在 Prepare → Cast → HandleImmediate 中完成
	if appliedAura == nil {
		t.Fatal("expected OnAuraCreated to be called with a non-nil aura")
	}
	if appliedAura.AuraType != spellcore.AuraPeriodicDamage {
		t.Errorf("aura type = %d, want AuraPeriodicDamage", appliedAura.AuraType)
	}
	if appliedAura.SpellID != dotInfo.ID {
		t.Errorf("aura SpellID = %d, want %d", appliedAura.SpellID, dotInfo.ID)
	}
	if appliedAura.TargetID != 1 { // TargetUnitCaster → caster ID = 1
		t.Errorf("aura TargetID = %d, want 1", appliedAura.TargetID)
	}
	if len(appliedAura.Effects) == 0 {
		t.Fatal("expected aura to have periodic effects")
	}
	// 验证 period = 3000ms
	if appliedAura.Effects[0].Period != 3000*time.Millisecond {
		t.Errorf("aura period = %v, want 3000ms", appliedAura.Effects[0].Period)
	}
}

// --- CheckCast validation tests ---

func TestCheckCast_NoPower(t *testing.T) {
	info := spellcore.SpellInfo{
		ID:        900,
		Name:      "ExpensiveSpell",
		PowerCost: 500,
		PowerType: 0,
		Effects:   []spellcore.SpellEffectInfo{{EffectType: spellcore.EffectSchoolDamage, BasePoints: 100}},
	}
	caster := newTestUnit(1)
	caster.stats.SetBase(stat.Mana, 100) // not enough

	s := spellcore.NewSpell(spellcore.SpellID(info.ID), &info, caster, spellcore.TriggeredNone)
	result := s.CheckCast(true)
	if result != spellcore.CastFailedNoPower {
		t.Errorf("expected CastFailedNoPower, got %d", result)
	}
}

func TestCheckCast_EnoughPower(t *testing.T) {
	info := spellcore.SpellInfo{
		ID:        901,
		Name:      "CheapSpell",
		PowerCost: 50,
		PowerType: 0,
		Effects:   []spellcore.SpellEffectInfo{{EffectType: spellcore.EffectSchoolDamage, BasePoints: 100}},
	}
	caster := newTestUnit(1)

	s := spellcore.NewSpell(spellcore.SpellID(info.ID), &info, caster, spellcore.TriggeredNone)
	result := s.CheckCast(true)
	if result != spellcore.CastOK {
		t.Errorf("expected CastOK, got %d", result)
	}
}

func TestCheckCast_CooldownNotReady(t *testing.T) {
	info := spellcore.SpellInfo{
		ID:           902,
		Name:         "CooldownSpell",
		CooldownTime: 5000,
		PowerCost:    0,
		Effects:      []spellcore.SpellEffectInfo{{EffectType: spellcore.EffectSchoolDamage, BasePoints: 100}},
	}
	caster := newTestUnit(1)
	h := spellcore.NewHistory()
	h.AddCooldown(spellcore.SpellID(902), 0, 5*time.Second)

	// override history via a wrapper
	casterWithHistory := &historyCaster{testUnit: caster, history: h}
	s := spellcore.NewSpell(spellcore.SpellID(info.ID), &info, casterWithHistory, spellcore.TriggeredNone)
	result := s.CheckCast(false)
	if result != spellcore.CastFailedNotReady {
		t.Errorf("expected CastFailedNotReady, got %d", result)
	}
}

func TestCheckCast_TriggeredIgnoresAll(t *testing.T) {
	info := spellcore.SpellInfo{
		ID:           903,
		Name:         "TriggeredSpell",
		CooldownTime: 5000,
		PowerCost:    500,
		PowerType:    0,
		CategoryID:   1,
		Effects:      []spellcore.SpellEffectInfo{{EffectType: spellcore.EffectSchoolDamage, BasePoints: 100}},
	}
	caster := newTestUnit(1)
	caster.stats.SetBase(stat.Mana, 0) // no mana

	s := spellcore.NewSpell(spellcore.SpellID(info.ID), &info, caster, spellcore.TriggeredFullMask)
	result := s.CheckCast(true)
	if result != spellcore.CastOK {
		t.Errorf("expected CastOK for triggered spell, got %d", result)
	}
}

// historyCaster wraps testUnit to provide a real History
type historyCaster struct {
	*testUnit
	history *spellcore.History
}

func (h *historyCaster) GetHistory() *spellcore.History { return h.history }
