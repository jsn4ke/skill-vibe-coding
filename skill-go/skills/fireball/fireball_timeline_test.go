package fireball

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

type fbTLUnit struct {
	id    uint64
	alive bool
	stats *stat.StatSet
	pos   fbTLPos
}

func (u *fbTLUnit) GetID() uint64                                   { return u.id }
func (u *fbTLUnit) IsAlive() bool                                   { return u.alive }
func (u *fbTLUnit) CanCast() bool                                   { return u.alive }
func (u *fbTLUnit) IsMoving() bool                                  { return false }
func (u *fbTLUnit) GetPosition() spell.Position                     { return &u.pos }
func (u *fbTLUnit) GetTargetPosition(targetID uint64) spell.Position { return &fbTLPos{x: 10} }
func (u *fbTLUnit) GetStatValue(st uint8) float64 { return u.stats.Get(stat.StatType(st)) }
func (u *fbTLUnit) ModifyPower(pt uint8, amount float64) bool {
	if pt == 0 {
		cur := u.stats.Get(stat.Mana)
		u.stats.SetBase(stat.Mana, cur+amount)
	}
	return true
}

type fbTLPos struct{ x, y, z, facing float64 }

func (p *fbTLPos) GetX() float64      { return p.x }
func (p *fbTLPos) GetY() float64      { return p.y }
func (p *fbTLPos) GetZ() float64      { return p.z }
func (p *fbTLPos) GetFacing() float64 { return p.facing }

func runFireballTimeline() string {
	caster := &fbTLUnit{id: 1, alive: true, stats: stat.NewStatSet(), pos: fbTLPos{x: 0}}
	caster.stats.SetBase(stat.SpellPower, 100)
	caster.stats.SetBase(stat.Mana, 1000)

	bus := event.NewBus()
	auraMgr := aura.NewManager(bus)
	renderer := timeline.NewRenderer()
	renderer.SubscribeAll(bus)

	// Manual spell lifecycle for proper timeline timestamps
	s := spell.NewSpell(spell.SpellID(Info.ID), &Info, caster, spell.TriggeredNone)
	s.Targets.UnitTargetID = 2
	s.Bus = bus
	s.Prepare()

	const stepMs int32 = 100
	casterSP := caster.stats.Get(stat.SpellPower)
	spellDone := false

	for simMs := int32(0); simMs < 15000; simMs += stepMs {
		renderer.SetTime(time.Duration(simMs) * time.Millisecond)
		s.Update(stepMs)

		if s.State == spell.StateFinished && !spellDone {
			spellDone = true
			for i := range Info.Effects {
				ei := &Info.Effects[i]
				if ei.EffectType == spell.EffectApplyAura {
					a := aura.NewAura(spell.SpellID(Info.ID), caster.GetID(), 2, aura.AuraPeriodicDamage, 8*time.Second)
					a.MaxStack = 1
					a.StackRule = aura.StackRefresh
					a.SpellName = Info.Name
					a.Effects = []aura.AuraEffect{
						{EffectIndex: 0, AuraType: aura.AuraPeriodicDamage, Amount: ei.BasePoints, BonusCoeff: ei.BonusCoeff, Period: 2e9},
					}
					renderer.SetTime(time.Duration(simMs) * time.Millisecond)
					auraMgr.AddAura(a)
				}
			}
		}

		if spellDone {
			renderer.SetTime(time.Duration(simMs) * time.Millisecond)
			auraMgr.TickPeriodic(2, time.Duration(stepMs)*time.Millisecond, casterSP,
				func(a *aura.Aura, eff *aura.AuraEffect, amount float64) {})
		}

		if spellDone && len(auraMgr.GetAuras(2)) == 0 {
			break
		}
	}

	return renderer.Render()
}

func TestFireball_TimelineEvents(t *testing.T) {
	output := runFireballTimeline()

	expectedEvents := []struct {
		contains string
		desc     string
	}{
		{"SpellCastStart", "Fireball cast start"},
		{"Fireball starts casting", "Fireball cast detail"},
		{"SpellLaunch", "Fireball launched"},
		{"SpellHit", "Fireball hit"},
		{"AuraApplied", "DoT aura applied"},
		{"Fireball PeriodicDamage applied (8s)", "Aura detail with 8s duration"},
		{"AuraTick", "Periodic tick"},
		{"Fireball ticks for", "Tick damage"},
		{"AuraExpired", "Aura expired"},
	}

	for _, exp := range expectedEvents {
		if !strings.Contains(output, exp.contains) {
			t.Errorf("timeline missing: %s (expected to contain %q)", exp.desc, exp.contains)
		}
	}
}

func TestFireball_TimelineTickCount(t *testing.T) {
	output := runFireballTimeline()
	count := strings.Count(output, "AuraTick")
	if count != 4 {
		t.Errorf("expected 4 AuraTick events (8s / 2s period), got %d", count)
	}
}

func TestFireball_TimelineOutput(t *testing.T) {
	output := runFireballTimeline()
	t.Log("\n" + output)
}
