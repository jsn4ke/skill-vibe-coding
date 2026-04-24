package arcanemissiles

import (
	"strings"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"skill-go/pkg/timeline"
)

type tlUnit struct {
	id    uint64
	alive bool
	stats *stat.StatSet
	pos   tlPos
}

func (u *tlUnit) GetID() uint64                                   { return u.id }
func (u *tlUnit) IsAlive() bool                                   { return u.alive }
func (u *tlUnit) CanCast() bool                                   { return u.alive }
func (u *tlUnit) IsMoving() bool                                  { return false }
func (u *tlUnit) GetPosition() spell.Position                     { return &u.pos }
func (u *tlUnit) GetTargetPosition(targetID uint64) spell.Position { return &testPos{} }
func (u *tlUnit) GetStatValue(st uint8) float64                   { return u.stats.Get(stat.StatType(st)) }
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

func runArcaneMissilesTimeline() string {
	caster := &tlUnit{id: 1, alive: true, stats: stat.NewStatSet(), pos: tlPos{x: 0}}
	caster.stats.SetBase(stat.SpellPower, 100)
	caster.stats.SetBase(stat.Mana, 1000)

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	s, result := CastArcaneMissiles(caster, 2, auraMgr, bus)
	if result != spell.CastOK {
		return "CAST_FAILED"
	}

	casterSP := caster.GetStatValue(3)

	totalMs := int32(Info.Duration) + 1000
	const stepMs int32 = 100
	var simMs int32
	for simMs = 0; simMs < totalMs; simMs += stepMs {
		renderer.SetTime(time.Duration(simMs) * time.Millisecond)

		auraMgr.TickPeriodic(2, time.Duration(stepMs)*time.Millisecond, casterSP,
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {
				CastTriggeredSpell(caster, a.TargetID, &MissileInfo, bus)
			})

		s.Update(stepMs)
	}

	return renderer.Render()
}

func TestArcaneMissiles_TimelineEvents(t *testing.T) {
	output := runArcaneMissilesTimeline()

	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "cast start event"},
		{"SpellLaunch", "launch event"},
		{"AuraApplied", "aura applied event"},
		{"SpellHit", "missile hit event"},
		{"AuraExpired", "aura expired event"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("timeline missing: %s (looking for %q)", exp.desc, exp.contains)
		}
	}
}

func TestArcaneMissiles_TimelineTickCount(t *testing.T) {
	output := runArcaneMissilesTimeline()

	// Each missile publishes OnSpellHit with spell ID 7268
	// Timeline renders these as SpellHit events
	hitCount := strings.Count(output, "SpellHit")
	if hitCount != 3 {
		t.Errorf("expected 3 SpellHit events, got %d", hitCount)
	}
}

func TestArcaneMissiles_TimelineOutput(t *testing.T) {
	output := runArcaneMissilesTimeline()
	t.Log("\n" + output)
	if strings.Contains(output, "No events recorded") {
		t.Error("timeline should have events")
	}
}
