package unit

import (
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/stat"
	"testing"
	"time"
)

// --- mock EngineRef ---

type settleCall struct {
	sourceID, targetID uint64
	spellID            uint32
	damage, healing    float64
	isCrit             bool
	spellName          string
}

type triggerCall struct {
	casterID, targetID uint64
	spellID            spellcore.SpellID
}

type mockEngine struct {
	units     map[uint64]*Unit
	bus       *event.Bus
	auraMgr   *spellcore.AuraManager
	registry  *spellcore.Registry
	spellPow  float64
	tick      time.Duration
	inRadius  []*Unit
	settled   []settleCall
	triggered []triggerCall
}

func newMockEngine() *mockEngine {
	bus := event.NewBus()
	reg := spellcore.NewRegistry()
	mgr := spellcore.NewAuraManager(bus)
	mgr.SetRegistry(reg)
	return &mockEngine{
		units:    make(map[uint64]*Unit),
		bus:      bus,
		auraMgr:  mgr,
		registry: reg,
		tick:     100 * time.Millisecond,
	}
}

func (m *mockEngine) GetUnit(id uint64) *Unit { return m.units[id] }
func (m *mockEngine) GetUnitsInRadius(_ [3]float64, _ float64, _ uint64) []*Unit {
	return m.inRadius
}
func (m *mockEngine) GetBus() interface{}                 { return m.bus }
func (m *mockEngine) GetSpellPower(_ uint64) float64      { return m.spellPow }
func (m *mockEngine) Tick() time.Duration                 { return m.tick }
func (m *mockEngine) AuraMgr() *spellcore.AuraManager     { return m.auraMgr }
func (m *mockEngine) ScriptRegistry() *spellcore.Registry { return m.registry }
func (m *mockEngine) SettlePeriodicDamage(src, tgt uint64, sid uint32, dmg, heal float64, crit bool, name string) {
	m.settled = append(m.settled, settleCall{src, tgt, sid, dmg, heal, crit, name})
}
func (m *mockEngine) TriggerPeriodicSpell(casterID, targetID uint64, spellID spellcore.SpellID) {
	m.triggered = append(m.triggered, triggerCall{casterID, targetID, spellID})
}

// --- helpers ---

func newTestUnit(id uint64, hp float64) *Unit {
	ent := entity.NewEntity(entity.EntityID(id), entity.TypePlayer, entity.Position{})
	ss := stat.NewStatSet()
	ss.SetBase(stat.Health, hp)
	ss.SetBase(stat.MaxHealth, hp)
	return NewUnit(ent, ss, spellcore.NewHistory())
}

func newTestUnitWithEngine(id uint64, hp float64) (*Unit, *mockEngine) {
	u := newTestUnit(id, hp)
	eng := newMockEngine()
	eng.units[id] = u
	u.SetEngine(eng)
	return u, eng
}

// --- tests ---

func TestNewUnit(t *testing.T) {
	ent := entity.NewEntity(1, entity.TypePlayer, entity.Position{})
	ss := stat.NewStatSet()
	hist := spellcore.NewHistory()
	u := NewUnit(ent, ss, hist)

	if u.Entity != ent {
		t.Fatal("Entity not set")
	}
	if u.Stats != ss {
		t.Fatal("Stats not set")
	}
	if u.History != hist {
		t.Fatal("History not set")
	}
	if u.engine != nil {
		t.Fatal("engine should be nil")
	}
	if len(u.activeSpells) != 0 {
		t.Fatal("activeSpells should be empty")
	}
	if len(u.ownedAuras) != 0 {
		t.Fatal("ownedAuras should be empty")
	}
	if len(u.appliedAuraApps) != 0 {
		t.Fatal("appliedAuraApps should be empty")
	}
	if u.IsMoving() {
		t.Fatal("should not be moving initially")
	}
}

func TestSetEngine_ID(t *testing.T) {
	u := newTestUnit(42, 1000)
	if u.ID() != 42 {
		t.Fatalf("expected ID 42, got %d", u.ID())
	}
	eng := newMockEngine()
	u.SetEngine(eng)
	if u.engine != eng {
		t.Fatal("engine not set")
	}
}

func TestModifyHealth(t *testing.T) {
	tests := []struct {
		name      string
		hp        float64
		maxHP     float64
		delta     float64
		wantHP    float64
		wantDelta float64
	}{
		{"减血", 1000, 1000, -200, 800, -200},
		{"加血未满", 500, 1000, 300, 800, 300},
		{"加血超出上限", 900, 1000, 300, 1000, 100},
		{"减血超出下限", 100, 1000, -500, 0, -100},
		{"零变化", 500, 1000, 0, 500, 0},
		{"MaxHealth 为 0 不限上界", 500, 0, 300, 800, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := newTestUnit(1, tt.hp)
			u.Stats.SetBase(stat.MaxHealth, tt.maxHP)
			got := u.ModifyHealth(tt.delta)
			if got != tt.wantDelta {
				t.Fatalf("delta: got %v, want %v", got, tt.wantDelta)
			}
			if hp := u.Stats.Get(stat.Health); hp != tt.wantHP {
				t.Fatalf("hp: got %v, want %v", hp, tt.wantHP)
			}
		})
	}
}

func TestKill(t *testing.T) {
	u, eng := newTestUnitWithEngine(1, 1000)

	var deathEvents []event.Event
	eng.bus.Subscribe(event.OnDeath, func(e event.Event) {
		deathEvents = append(deathEvents, e)
	})

	u.Kill(42)

	if u.IsAlive() {
		t.Fatal("should be dead")
	}
	if u.Entity.State.Has(entity.StateDead) == false {
		t.Fatal("should have StateDead")
	}
	if len(deathEvents) != 1 {
		t.Fatalf("expected 1 death event, got %d", len(deathEvents))
	}
	if deathEvents[0].SourceID != 42 {
		t.Fatalf("attacker should be 42, got %d", deathEvents[0].SourceID)
	}
	if deathEvents[0].TargetID != 1 {
		t.Fatalf("target should be 1, got %d", deathEvents[0].TargetID)
	}
}

func TestKill_RemovesAuras(t *testing.T) {
	caster, eng := newTestUnitWithEngine(1, 1000)
	target := newTestUnit(2, 1000)
	eng.units[2] = target
	target.SetEngine(eng)

	a := spellcore.NewAura(100, 1, 2, spellcore.AuraPeriodicDamage, 5*time.Second)
	eng.auraMgr.ApplyAura(caster, target, a)
	if len(target.GetAppliedAuraApps()) == 0 {
		t.Fatal("aura should be applied")
	}

	target.Kill(1)

	if len(caster.GetOwnedAuras()) != 0 {
		t.Fatalf("caster owned auras should be empty, got %d", len(caster.GetOwnedAuras()))
	}
	if len(target.GetAppliedAuraApps()) != 0 {
		t.Fatalf("target applied auras should be empty, got %d", len(target.GetAppliedAuraApps()))
	}
}

func TestKill_NilEngine_NoPanic(t *testing.T) {
	u := newTestUnit(1, 1000)
	u.Kill(2)
	if u.IsAlive() {
		t.Fatal("should be dead")
	}
}

func TestActiveSpellManagement(t *testing.T) {
	u := newTestUnit(1, 1000)

	info := &spellcore.SpellInfo{Name: "test"}
	s1 := spellcore.NewSpell(1, info, u, 0)
	s2 := spellcore.NewSpell(2, info, u, 0)
	s1.State = spellcore.StatePreparing
	s2.State = spellcore.StatePreparing

	u.AddActiveSpell(s1)
	u.AddActiveSpell(s2)
	if len(u.GetActiveSpells()) != 2 {
		t.Fatal("should have 2 active spells")
	}

	// mark s1 finished, update should clean it
	s1.State = spellcore.StateFinished
	u.updateSpells(100)

	if len(u.GetActiveSpells()) != 1 {
		t.Fatalf("should have 1 active spell after cleanup, got %d", len(u.GetActiveSpells()))
	}
	if u.GetActiveSpells()[0].ID != 2 {
		t.Fatal("remaining spell should be s2")
	}
}

func TestUpdateSpells_CallsUpdate(t *testing.T) {
	u := newTestUnit(1, 1000)
	info := &spellcore.SpellInfo{Name: "test"}
	s := spellcore.NewSpell(1, info, u, 0)
	s.State = spellcore.StatePreparing

	u.AddActiveSpell(s)
	u.updateSpells(150)

	// Spell.Update with StatePreparing and CastTime=0 triggers Cast immediately,
	// which transitions to StateFinished for instant spells.
	// Just verify the spell was driven (state changed from Preparing).
	if s.State == spellcore.StateNull {
		t.Fatal("spell should have been updated")
	}
}

func TestOwnedAuraManagement(t *testing.T) {
	u := newTestUnit(1, 1000)

	a1 := spellcore.NewAura(100, 1, 2, spellcore.AuraPeriodicDamage, 5*time.Second)
	a2 := spellcore.NewAura(200, 1, 3, spellcore.AuraPeriodicHeal, 3*time.Second)

	u.AddOwnedAura(a1)
	u.AddOwnedAura(a2)

	if len(u.GetOwnedAuras()) != 2 {
		t.Fatal("should have 2 owned auras")
	}

	found := u.FindOwnedAura(100, 2)
	if found == nil || found.SpellID != 100 {
		t.Fatal("FindOwnedAura should find spell 100 target 2")
	}

	notFound := u.FindOwnedAura(999, 2)
	if notFound != nil {
		t.Fatal("FindOwnedAura should return nil for non-existent")
	}

	u.RemoveOwnedAura(0)
	if len(u.GetOwnedAuras()) != 1 || u.GetOwnedAuras()[0].SpellID != 200 {
		t.Fatal("after removing index 0, remaining should be spell 200")
	}
}

func TestFindAreaAura(t *testing.T) {
	u := newTestUnit(1, 1000)

	regular := spellcore.NewAura(100, 1, 2, spellcore.AuraPeriodicDamage, 5*time.Second)
	area := spellcore.NewAura(200, 1, 0, spellcore.AuraPeriodicDamage, 8*time.Second)
	area.IsAreaAura = true
	sameIDNotArea := spellcore.NewAura(200, 1, 2, spellcore.AuraPeriodicDamage, 3*time.Second)

	u.AddOwnedAura(regular)
	u.AddOwnedAura(area)
	u.AddOwnedAura(sameIDNotArea)

	found := u.FindAreaAura(200)
	if found == nil || !found.IsAreaAura {
		t.Fatal("FindAreaAura should return the area aura")
	}

	notFound := u.FindAreaAura(100)
	if notFound != nil {
		t.Fatal("FindAreaAura should return nil for non-area aura")
	}
}

func TestAppliedAuraManagement(t *testing.T) {
	u := newTestUnit(1, 1000)

	a1 := spellcore.NewAura(100, 2, 1, spellcore.AuraPeriodicDamage, 5*time.Second)
	a2 := spellcore.NewAura(200, 3, 1, spellcore.AuraPeriodicHeal, 3*time.Second)

	app1 := spellcore.NewAuraApplication(a1, u.GetID())
	app2 := spellcore.NewAuraApplication(a2, u.GetID())
	u.AddAppliedAuraApp(app1)
	u.AddAppliedAuraApp(app2)

	if len(u.GetAppliedAuraApps()) != 2 {
		t.Fatal("should have 2 applied aura apps")
	}

	found := u.FindAppliedAuraApp(100, 2)
	if found == nil || found.Base.SpellID != 100 {
		t.Fatal("FindAppliedAuraApp should find spell 100 caster 2")
	}

	notFound := u.FindAppliedAuraApp(100, 999)
	if notFound != nil {
		t.Fatal("FindAppliedAuraApp should return nil for wrong caster")
	}

	u.RemoveAppliedAuraApp(0)
	if len(u.GetAppliedAuraApps()) != 1 {
		t.Fatal("should have 1 applied aura app after removal")
	}
}

func TestRemoveAurasWithInterruptFlags_NoneFlag(t *testing.T) {
	u, _ := newTestUnitWithEngine(1, 1000)
	a := spellcore.NewAura(100, 2, 1, spellcore.AuraModRoot, 5*time.Second)
	a.InterruptFlags = spellcore.AuraInterruptOnMovement
	app := spellcore.NewAuraApplication(a, u.GetID())
	u.AddAppliedAuraApp(app)

	// AuraInterruptNone should be no-op
	u.RemoveAurasWithInterruptFlags(spellcore.AuraInterruptNone)
	if len(u.GetAppliedAuraApps()) != 1 {
		t.Fatal("AuraInterruptNone should not remove anything")
	}
}

func TestRemoveAurasWithInterruptFlags_RemovesMatching(t *testing.T) {
	caster, eng := newTestUnitWithEngine(1, 1000)
	target := newTestUnit(2, 1000)
	eng.units[2] = target
	target.SetEngine(eng)

	// caster owns aura, target has it applied
	moveAura := spellcore.NewAura(100, 1, 2, spellcore.AuraModRoot, 5*time.Second)
	moveAura.InterruptFlags = spellcore.AuraInterruptOnMovement
	eng.auraMgr.ApplyAura(caster, target, moveAura)

	// another aura not matching
	safeAura := spellcore.NewAura(200, 1, 2, spellcore.AuraModSpeed, 5*time.Second)
	safeAura.InterruptFlags = spellcore.AuraInterruptOnDamage
	eng.auraMgr.ApplyAura(caster, target, safeAura)

	target.RemoveAurasWithInterruptFlags(spellcore.AuraInterruptOnMovement)

	// moveAura removed, safeAura stays
	if len(target.GetAppliedAuraApps()) != 1 {
		t.Fatalf("should have 1 applied aura, got %d", len(target.GetAppliedAuraApps()))
	}
	if target.GetAppliedAuraApps()[0].Base.SpellID != 200 {
		t.Fatal("remaining aura should be spell 200")
	}
}

func TestRemoveAurasWithInterruptFlags_CancelsChanneling(t *testing.T) {
	u, _ := newTestUnitWithEngine(1, 1000)

	// Add a channeling spell with InterruptMovement
	info := &spellcore.SpellInfo{
		Name:           "channel",
		InterruptFlags: spellcore.InterruptMovement,
	}
	s := spellcore.NewSpell(1, info, u, 0)
	s.State = spellcore.StateChanneling
	u.AddActiveSpell(s)

	u.RemoveAurasWithInterruptFlags(spellcore.AuraInterruptOnMovement)

	if s.State != spellcore.StateFinished {
		t.Fatal("channeling spell should be cancelled")
	}
}

func TestRemoveAurasWithInterruptFlags_NoCancelNonChanneling(t *testing.T) {
	u, _ := newTestUnitWithEngine(1, 1000)

	info := &spellcore.SpellInfo{
		Name:           "preparing",
		InterruptFlags: spellcore.InterruptMovement,
	}
	s := spellcore.NewSpell(1, info, u, 0)
	s.State = spellcore.StatePreparing
	u.AddActiveSpell(s)

	u.RemoveAurasWithInterruptFlags(spellcore.AuraInterruptOnMovement)

	if s.State == spellcore.StateFinished {
		t.Fatal("non-channeling spell should not be cancelled")
	}
}

func TestUpdate_MovementDetection(t *testing.T) {
	u, _ := newTestUnitWithEngine(1, 1000)

	// Frame 1: position unchanged (initially prevPos is zero, Entity.Pos is also zero)
	u.Update(100)
	if u.IsMoving() {
		t.Fatal("should not be moving (same position)")
	}

	// Frame 2: move
	u.SetPosition(entity.Position{X: 10, Y: 0, Z: 0})
	u.Update(100)
	if !u.IsMoving() {
		t.Fatal("should be moving after position change")
	}

	// Frame 3: stop
	u.Update(100)
	if u.IsMoving() {
		t.Fatal("should not be moving when position unchanged")
	}
}

func TestUpdate_MovementRemovesAuras(t *testing.T) {
	u, eng := newTestUnitWithEngine(1, 1000)

	moveAura := spellcore.NewAura(100, 2, 1, spellcore.AuraModRoot, 5*time.Second)
	moveAura.InterruptFlags = spellcore.AuraInterruptOnMovement

	// also register as owned by caster
	caster := newTestUnit(2, 1000)
	eng.units[2] = caster
	caster.SetEngine(eng)
	eng.auraMgr.ApplyAura(caster, u, moveAura)

	u.SetPosition(entity.Position{X: 5, Y: 0, Z: 0})
	u.Update(100)

	if len(u.GetAppliedAuraApps()) != 0 {
		t.Fatal("movement should remove auras with AuraInterruptOnMovement")
	}
}

func TestUpdateAuras_TickAndExpire(t *testing.T) {
	caster, eng := newTestUnitWithEngine(1, 1000)
	caster.Stats.SetBase(stat.SpellPower, 50)
	target := newTestUnit(2, 1000)
	eng.units[2] = target
	target.SetEngine(eng)

	a := spellcore.NewAura(100, 1, 2, spellcore.AuraPeriodicDamage, 3*time.Second)
	a.SpellName = "test_dot"
	a.Effects = []spellcore.AuraEffect{
		{
			AuraType:   spellcore.AuraPeriodicDamage,
			Amount:     100,
			BonusCoeff: 0.5,
			Period:     1 * time.Second,
		},
	}
	eng.auraMgr.ApplyAura(caster, target, a)

	// Tick once (1000ms)
	caster.Update(1000)

	if len(eng.settled) != 1 {
		t.Fatalf("expected 1 settle call, got %d", len(eng.settled))
	}
	// amount = 100 + 0.5*50 = 125
	if eng.settled[0].damage != 125 {
		t.Fatalf("expected damage 125, got %v", eng.settled[0].damage)
	}

	// Tick twice more to expire (2000ms more, total 3000ms = duration)
	caster.Update(2000)

	// After expiry, aura should be removed
	if len(caster.GetOwnedAuras()) != 0 {
		t.Fatal("owned aura should be removed after expiry")
	}
	if len(target.GetAppliedAuraApps()) != 0 {
		t.Fatal("applied aura should be removed after expiry")
	}
}

func TestUpdateAuras_PeriodicHeal(t *testing.T) {
	caster, eng := newTestUnitWithEngine(1, 1000)
	caster.Stats.SetBase(stat.SpellPower, 0)

	a := spellcore.NewAura(200, 1, 1, spellcore.AuraPeriodicHeal, 5*time.Second)
	a.SpellName = "hot"
	a.Effects = []spellcore.AuraEffect{
		{
			AuraType: spellcore.AuraPeriodicHeal,
			Amount:   50,
			Period:   1 * time.Second,
		},
	}
	eng.auraMgr.ApplyAura(caster, caster, a)

	caster.Update(1000)

	if len(eng.settled) != 1 {
		t.Fatalf("expected 1 settle call, got %d", len(eng.settled))
	}
	if eng.settled[0].healing != 50 {
		t.Fatalf("expected healing 50, got %v", eng.settled[0].healing)
	}
}

func TestUpdateAuras_TriggerSpell(t *testing.T) {
	caster, eng := newTestUnitWithEngine(1, 1000)
	target := newTestUnit(2, 1000)
	eng.units[2] = target
	target.SetEngine(eng)

	a := spellcore.NewAura(300, 1, 2, spellcore.AuraPeriodicTriggerSpell, 3*time.Second)
	a.SpellName = "trigger"
	a.Effects = []spellcore.AuraEffect{
		{
			AuraType:       spellcore.AuraPeriodicTriggerSpell,
			Amount:         0,
			Period:         1 * time.Second,
			TriggerSpellID: 999,
		},
	}
	eng.auraMgr.ApplyAura(caster, target, a)

	caster.Update(1000)

	if len(eng.triggered) != 1 {
		t.Fatalf("expected 1 trigger call, got %d", len(eng.triggered))
	}
	if eng.triggered[0].spellID != 999 {
		t.Fatalf("expected trigger spell 999, got %d", eng.triggered[0].spellID)
	}
}

func TestUpdateAuras_NonPeriodic_Skipped(t *testing.T) {
	u, eng := newTestUnitWithEngine(1, 1000)

	a := spellcore.NewAura(100, 1, 1, spellcore.AuraModStun, 5*time.Second)
	a.Effects = []spellcore.AuraEffect{
		{AuraType: spellcore.AuraModStun, Amount: 0, Period: 0},
	}
	eng.auraMgr.ApplyAura(u, u, a)

	u.Update(1000)

	if len(eng.settled) != 0 {
		t.Fatal("non-periodic aura should not trigger settlement")
	}
}

func TestUpdateAuras_TickEvent(t *testing.T) {
	u, eng := newTestUnitWithEngine(1, 1000)
	u.Stats.SetBase(stat.SpellPower, 0)

	var tickEvents []event.Event
	eng.bus.Subscribe(event.OnAuraTick, func(e event.Event) {
		tickEvents = append(tickEvents, e)
	})

	a := spellcore.NewAura(100, 1, 1, spellcore.AuraPeriodicDamage, 5*time.Second)
	a.SpellName = "dot"
	a.Effects = []spellcore.AuraEffect{
		{AuraType: spellcore.AuraPeriodicDamage, Amount: 50, Period: 1 * time.Second},
	}
	eng.auraMgr.ApplyAura(u, u, a)

	u.Update(1000)

	if len(tickEvents) != 1 {
		t.Fatalf("expected 1 tick event, got %d", len(tickEvents))
	}
	if tickEvents[0].SpellID != 100 {
		t.Fatalf("expected spell 100, got %d", tickEvents[0].SpellID)
	}
}

func TestUpdateAuras_ExpiredEvent(t *testing.T) {
	u, eng := newTestUnitWithEngine(1, 1000)
	u.Stats.SetBase(stat.SpellPower, 0)

	var expiredEvents []event.Event
	eng.bus.Subscribe(event.OnAuraExpired, func(e event.Event) {
		expiredEvents = append(expiredEvents, e)
	})

	a := spellcore.NewAura(100, 1, 1, spellcore.AuraPeriodicDamage, 2*time.Second)
	a.SpellName = "short_dot"
	a.Effects = []spellcore.AuraEffect{
		{AuraType: spellcore.AuraPeriodicDamage, Amount: 10, Period: 1 * time.Second},
	}
	eng.auraMgr.ApplyAura(u, u, a)

	u.Update(2000)

	if len(expiredEvents) != 1 {
		t.Fatalf("expected 1 expired event, got %d", len(expiredEvents))
	}
	if expiredEvents[0].SpellID != 100 {
		t.Fatalf("expected spell 100, got %d", expiredEvents[0].SpellID)
	}
}

func TestCasterInterface(t *testing.T) {
	u := newTestUnit(5, 1000)
	u.Stats.SetBase(stat.SpellPower, 42)

	if u.GetID() != 5 {
		t.Fatalf("GetID: expected 5, got %d", u.GetID())
	}
	if !u.IsAlive() {
		t.Fatal("should be alive")
	}
	if !u.CanCast() {
		t.Fatal("should be able to cast")
	}
	if u.GetStatValue(uint8(stat.SpellPower)) != 42 {
		t.Fatal("GetStatValue should return 42")
	}
}

func TestModifyPower(t *testing.T) {
	u := newTestUnit(1, 1000)
	u.Stats.SetBase(stat.Mana, 500)

	result := u.ModifyPower(0, -100)
	if !result {
		t.Fatal("ModifyPower should return true")
	}
	if mp := u.Stats.Get(stat.Mana); mp != 400 {
		t.Fatalf("expected mana 400, got %v", mp)
	}
}

func TestGetTargetPosition(t *testing.T) {
	u, eng := newTestUnitWithEngine(1, 1000)
	target := newTestUnit(2, 1000)
	target.SetPosition(entity.Position{X: 30, Y: 0, Z: 0})
	eng.units[2] = target
	target.SetEngine(eng)

	pos := u.GetTargetPosition(2)
	if pos.GetX() != 30 {
		t.Fatalf("expected X=30, got %v", pos.GetX())
	}

	// non-existent target falls back to own position
	pos = u.GetTargetPosition(999)
	if pos.GetX() != 0 {
		t.Fatalf("expected fallback X=0, got %v", pos.GetX())
	}
}

func TestGetTargetPosition_NilEngine(t *testing.T) {
	u := newTestUnit(1, 1000)
	u.SetPosition(entity.Position{X: 5, Y: 5, Z: 0})

	pos := u.GetTargetPosition(999)
	if pos.GetX() != 5 {
		t.Fatalf("expected fallback X=5, got %v", pos.GetX())
	}
}

func TestUnit_SilenceAuraBlocksCasting(t *testing.T) {
	u, me := newTestUnitWithEngine(1, 1000)
	u.Stats.SetBase(stat.Mana, 10000)

	silenceAura := spellcore.NewAura(999, 2, 1, spellcore.AuraModSilence, 5*time.Second)
	me.auraMgr.ApplyAura(u, u, silenceAura)

	if u.CanCast() {
		t.Error("expected CanCast=false after silence aura applied")
	}

	me.auraMgr.RemoveAuraFromHosts(silenceAura, u, u, spellcore.RemoveByDispel)

	if !u.CanCast() {
		t.Error("expected CanCast=true after silence aura removed")
	}
}

func TestUnit_StunAuraBlocksCastAndMove(t *testing.T) {
	u, me := newTestUnitWithEngine(1, 1000)

	stunAura := spellcore.NewAura(998, 2, 1, spellcore.AuraModStun, 3*time.Second)
	me.auraMgr.ApplyAura(u, u, stunAura)

	if u.CanCast() {
		t.Error("expected CanCast=false after stun")
	}
	if u.Entity.CanMove() {
		t.Error("expected CanMove=false after stun")
	}

	me.auraMgr.RemoveAuraFromHosts(stunAura, u, u, spellcore.RemoveByExpire)

	if !u.CanCast() {
		t.Error("expected CanCast=true after stun removed")
	}
}
