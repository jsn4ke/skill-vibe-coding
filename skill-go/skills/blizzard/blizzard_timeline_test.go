package blizzard

import (
	"strings"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"skill-go/pkg/targeting"
	"skill-go/pkg/timeline"
)

type timelineUnit struct {
	id    uint64
	alive bool
	stats *stat.StatSet
	pos   tlPos
}

func (u *timelineUnit) GetID() uint64                                   { return u.id }
func (u *timelineUnit) IsAlive() bool                                   { return u.alive }
func (u *timelineUnit) CanCast() bool                                   { return u.alive }
func (u *timelineUnit) IsMoving() bool                                  { return false }
func (u *timelineUnit) GetPosition() spell.Position                     { return &u.pos }
func (u *timelineUnit) GetTargetPosition(targetID uint64) spell.Position { return &testPos{} }
func (u *timelineUnit) GetStatValue(st uint8) float64 { return u.stats.Get(stat.StatType(st)) }
func (u *timelineUnit) ModifyPower(pt uint8, amount float64) bool {
	if pt == 0 {
		cur := u.stats.Get(stat.Mana)
		u.stats.SetBase(stat.Mana, cur+amount)
	}
	return true
}
func (u *timelineUnit) GetEntity() *entity.Entity {
	return &entity.Entity{ID: entity.EntityID(u.id), Type: entity.TypePlayer, Pos: entity.Position{X: u.pos.x, Y: u.pos.y, Z: u.pos.z}}
}

type tlPos struct{ x, y, z, facing float64 }

func (p *tlPos) GetX() float64      { return p.x }
func (p *tlPos) GetY() float64      { return p.y }
func (p *tlPos) GetZ() float64      { return p.z }
func (p *tlPos) GetFacing() float64 { return p.facing }

type tlEnemy struct {
	id    uint64
	alive bool
	pos   tlPos
}

func (e *tlEnemy) GetEntity() *entity.Entity {
	return &entity.Entity{ID: entity.EntityID(e.id), Type: entity.TypeCreature, Pos: entity.Position{X: e.pos.x, Y: e.pos.y, Z: e.pos.z}}
}
func (e *tlEnemy) IsAlive() bool { return e.alive }

func runBlizzardTimeline() string {
	caster := &timelineUnit{id: 1, alive: true, stats: stat.NewStatSet(), pos: tlPos{x: 0}}
	caster.stats.SetBase(stat.SpellPower, 100)
	caster.stats.SetBase(stat.Mana, 1000)

	e1 := &tlEnemy{id: 2, alive: true, pos: tlPos{x: 10, y: 0}}
	e2 := &tlEnemy{id: 3, alive: true, pos: tlPos{x: 12, y: 1}}
	targets := []targeting.Targetable{e1, e2}

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	// Cast Blizzard
	bs, result := CastBlizzard(caster, 10, 0, 0, auraMgr, bus)
	if result != spell.CastOK {
		return "CAST_FAILED"
	}
	_ = bs

	casterSP := caster.stats.Get(stat.SpellPower)
	casterEntity := caster.GetEntity()

	const stepMs int32 = 100
	var simMs int32
	for simMs = 0; simMs < 12000; simMs += stepMs {
		renderer.SetTime(time.Duration(simMs) * time.Millisecond)
		bs.Update(stepMs)

		auraMgr.TickPeriodicArea(time.Duration(stepMs)*time.Millisecond, casterSP,
			func() []uint64 {
				a := auraMgr.FindAreaAura(caster.id, spell.SpellID(Info.ID))
				if a == nil {
					return nil
				}
				sel := targeting.NewSelector(targets)
				desc := targeting.Descriptor{Selection: targeting.SelectArea, Check: targeting.CheckEnemy, Radius: a.AreaRadius}
				center := entity.Position{X: a.AreaCenter[0], Y: a.AreaCenter[1], Z: a.AreaCenter[2]}
				selected := sel.SelectAroundPoint(casterEntity, center, targets, desc, 0)
				var ids []uint64
				for _, e := range selected {
					ids = append(ids, uint64(e.ID))
				}
				return ids
			},
			func(a *aura.Aura, eff *aura.AuraEffect, amount float64, tid uint64) {})

		if bs.State == spell.StateFinished && auraMgr.FindAreaAura(caster.id, spell.SpellID(Info.ID)) == nil {
			break
		}
	}

	return renderer.Render()
}

func TestBlizzard_TimelineEvents(t *testing.T) {
	output := runBlizzardTimeline()

	// Verify key events present
	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "Blizzard cast start"},
		{"SpellLaunch", "Blizzard launched"},
		{"AuraApplied", "Blizzard aura applied"},
		{"Blizzard PeriodicDamage applied (8s)", "Aura detail with 8s duration"},
		{"AuraTick", "Periodic tick"},
		{"Blizzard ticks for 29.2 damage", "Tick damage amount"},
		{"AuraExpired", "Aura expired"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("timeline missing: %s (expected to contain %q)", exp.desc, exp.contains)
		}
	}
}

func TestBlizzard_TimelineTickCount(t *testing.T) {
	output := runBlizzardTimeline()
	count := strings.Count(output, "AuraTick")
	// 2 enemies × 8 ticks = 16 tick events
	if count != 16 {
		t.Errorf("expected 16 AuraTick events (2 enemies × 8 ticks), got %d", count)
	}
}

func TestBlizzard_TimelineTickDamage(t *testing.T) {
	output := runBlizzardTimeline()
	if !strings.Contains(output, "Blizzard ticks for 29.2 damage") {
		t.Error("expected tick damage 29.2 (25 + 0.042 × 100) in timeline")
	}
}

func TestBlizzard_TimelineCompletes(t *testing.T) {
	output := runBlizzardTimeline()
	if !strings.Contains(output, "AuraExpired") {
		t.Error("expected AuraExpired event in timeline")
	}
	if strings.Contains(output, "No events recorded") {
		t.Error("timeline should have events, got 'No events recorded'")
	}
}

func TestBlizzard_TimelineOutput(t *testing.T) {
	output := runBlizzardTimeline()
	t.Log("\n" + output)
}
