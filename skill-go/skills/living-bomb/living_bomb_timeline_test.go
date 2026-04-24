package livingbomb

import (
	"strings"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/effect"
	"skill-go/pkg/event"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"skill-go/pkg/timeline"
)

// ── shared test helpers ──────────────────────────────────────────

type tlUnit struct {
	id      uint64
	alive   bool
	stats   *stat.StatSet
	pos     tlPos
	targets map[uint64]*tlUnit
}

func (u *tlUnit) GetID() uint64                                   { return u.id }
func (u *tlUnit) IsAlive() bool                                   { return u.alive }
func (u *tlUnit) CanCast() bool                                   { return u.alive }
func (u *tlUnit) IsMoving() bool                                  { return false }
func (u *tlUnit) GetPosition() spell.Position                     { return &u.pos }
func (u *tlUnit) GetTargetPosition(targetID uint64) spell.Position {
	if t, ok := u.targets[targetID]; ok {
		return &t.pos
	}
	return &tlPos{}
}
func (u *tlUnit) GetStatValue(st uint8) float64 { return u.stats.Get(stat.StatType(st)) }
func (u *tlUnit) ModifyPower(pt uint8, amount float64) bool {
	if pt == 0 {
		cur := u.stats.Get(stat.Mana)
		u.stats.SetBase(stat.Mana, cur+amount)
	}
	return true
}

type tlPos struct{ x, y, z, facing float64 }

func (p *tlPos) GetX() float64      { return p.x }
func (p *tlPos) GetY() float64      { return p.y }
func (p *tlPos) GetZ() float64      { return p.z }
func (p *tlPos) GetFacing() float64 { return p.facing }

func newTLUnit(id uint64, x float64) *tlUnit {
	u := &tlUnit{
		id:      id,
		alive:   true,
		stats:   stat.NewStatSet(),
		pos:     tlPos{x: x},
		targets: make(map[uint64]*tlUnit),
	}
	u.stats.SetBase(stat.SpellPower, 100)
	u.stats.SetBase(stat.Mana, 1000)
	return u
}

type tlSelector struct {
	targets []uint64
}

func (s *tlSelector) SelectAoETargets(center [3]float64, excludeID uint64) []uint64 {
	var result []uint64
	for _, id := range s.targets {
		if id != excludeID {
			result = append(result, id)
		}
	}
	return result
}

// tickTarget advances simulation for one aura target, updating renderer time.
func tickTarget(r *timeline.TimelineRenderer, auraMgr *aura.Manager, targetID uint64, startMs int32, totalMs int32, sp float64) int32 {
	const stepMs int32 = 100
	simMs := startMs
	for ; simMs < startMs+totalMs; simMs += stepMs {
		r.SetTime(time.Duration(simMs) * time.Millisecond)
		auraMgr.TickPeriodic(targetID, time.Duration(stepMs)*time.Millisecond, sp,
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {})
	}
	return simMs
}

// ── Timeline 1: 基础生命周期（单体，无 AoE） ─────────────────────

func runBasicLifecycleTimeline() string {
	caster := newTLUnit(1, 0)
	target := newTLUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	reg := script.NewRegistry()
	RegisterScripts(reg, caster, auraMgr, bus, nil)
	effect.ScriptRegistry = reg
	auraMgr.SetRegistry(reg)

	// t=0: cast
	CastLivingBomb(caster, 2, auraMgr, bus)

	// t=0..5000: DoT ticks then expiry → explosion
	sp := caster.GetStatValue(3)
	tickTarget(renderer, auraMgr, 2, 0, 5000, sp)

	return renderer.Render()
}

func TestLivingBomb_T1_BasicLifecycle(t *testing.T) {
	output := runBasicLifecycleTimeline()
	t.Log("\n[Timeline 1: 基础生命周期]\n" + output)

	expected := []string{"SpellCastStart", "SpellLaunch", "AuraApplied", "AuraTick", "AuraExpired", "Living Bomb Explode"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("missing: %s", exp)
		}
	}

	if strings.Count(output, "AuraTick") != 4 {
		t.Errorf("expected 4 ticks, got %d", strings.Count(output, "AuraTick"))
	}
}

// ── Timeline 2: 完整传染链（3 目标，AoE 爆炸 + 传播） ────────────

func runFullSpreadTimeline() string {
	caster := newTLUnit(1, 0)
	targetA := newTLUnit(2, 5)  // bomb carrier
	targetB := newTLUnit(3, 8)  // nearby enemy
	targetC := newTLUnit(4, 12) // nearby enemy
	caster.targets[2] = targetA
	caster.targets[3] = targetB
	caster.targets[4] = targetC

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	selector := &tlSelector{targets: []uint64{2, 3, 4}}
	reg := script.NewRegistry()
	RegisterScripts(reg, caster, auraMgr, bus, selector)
	effect.ScriptRegistry = reg
	auraMgr.SetRegistry(reg)

	sp := caster.GetStatValue(3)

	// t=0: cast on A
	CastLivingBomb(caster, 2, auraMgr, bus)

	// t=0..5000: A's DoT → expires → explosion → spreads to B+C
	t := tickTarget(renderer, auraMgr, 2, 0, 5000, sp)

	// t continues: B's spread copy → expires → explodes → no spread
	t = tickTarget(renderer, auraMgr, 3, t, 5000, sp)

	// t continues: C's spread copy → expires → explodes → no spread
	t = tickTarget(renderer, auraMgr, 4, t, 5000, sp)

	return renderer.Render()
}

func TestLivingBomb_T2_FullSpreadChain(t *testing.T) {
	output := runFullSpreadTimeline()
	t.Log("\n[Timeline 2: 完整传染链（A→B+C, B+C各自爆炸不再传染）]\n" + output)

	if !strings.Contains(output, "Living Bomb launched") {
		t.Error("missing original cast")
	}

	explodeCount := strings.Count(output, "Living Bomb Explode starts casting")
	if explodeCount != 3 {
		t.Errorf("expected 3 explosions, got %d", explodeCount)
	}

	hitCount := strings.Count(output, "Living Bomb Explode hits")
	if hitCount < 4 {
		t.Errorf("expected at least 4 explosion hits, got %d", hitCount)
	}

	auraApplied := strings.Count(output, "AuraApplied")
	if auraApplied < 3 {
		t.Errorf("expected at least 3 aura applications (original + 2 spread), got %d", auraApplied)
	}
}

// ── Timeline 3: 死亡不触发爆炸 ───────────────────────────────────

func runDeathNoExplosionTimeline() string {
	caster := newTLUnit(1, 0)
	target := newTLUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	reg := script.NewRegistry()
	RegisterScripts(reg, caster, auraMgr, bus, nil)
	effect.ScriptRegistry = reg
	auraMgr.SetRegistry(reg)

	CastLivingBomb(caster, 2, auraMgr, bus)

	sp := caster.GetStatValue(3)

	// t=0..2000: tick 2 times
	t := tickTarget(renderer, auraMgr, 2, 0, 2000, sp)

	// t=2000: target dies
	renderer.SetTime(time.Duration(t) * time.Millisecond)
	a := auraMgr.FindAura(2, 217694, 1)
	if a != nil {
		auraMgr.RemoveAura(a, aura.RemoveByDeath)
	}

	return renderer.Render()
}

func TestLivingBomb_T3_DeathNoExplosion(t *testing.T) {
	output := runDeathNoExplosionTimeline()
	t.Log("\n[Timeline 3: 目标死亡不触发爆炸]\n" + output)

	if strings.Count(output, "AuraTick") != 2 {
		t.Errorf("expected 2 ticks before death, got %d", strings.Count(output, "AuraTick"))
	}

	if strings.Contains(output, "Living Bomb Explode") {
		t.Error("explosion should NOT trigger on death")
	}
}

// ── Timeline 4: 驱散不触发爆炸 ───────────────────────────────────

func runDispelNoExplosionTimeline() string {
	caster := newTLUnit(1, 0)
	target := newTLUnit(2, 10)
	caster.targets[2] = target

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	reg := script.NewRegistry()
	RegisterScripts(reg, caster, auraMgr, bus, nil)
	effect.ScriptRegistry = reg
	auraMgr.SetRegistry(reg)

	CastLivingBomb(caster, 2, auraMgr, bus)

	sp := caster.GetStatValue(3)

	// t=0..1000: tick 1 time
	t := tickTarget(renderer, auraMgr, 2, 0, 1000, sp)

	// t=1000: dispel
	renderer.SetTime(time.Duration(t) * time.Millisecond)
	a := auraMgr.FindAura(2, 217694, 1)
	if a != nil {
		auraMgr.RemoveAura(a, aura.RemoveByDispel)
	}

	return renderer.Render()
}

func TestLivingBomb_T4_DispelNoExplosion(t *testing.T) {
	output := runDispelNoExplosionTimeline()
	t.Log("\n[Timeline 4: 驱散不触发爆炸]\n" + output)

	if strings.Count(output, "AuraTick") != 1 {
		t.Errorf("expected 1 tick before dispel, got %d", strings.Count(output, "AuraTick"))
	}

	if strings.Contains(output, "Living Bomb Explode") {
		t.Error("explosion should NOT trigger on dispel")
	}
}
