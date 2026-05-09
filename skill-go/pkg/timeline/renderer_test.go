package timeline

import (
	"strings"
	"testing"
	"time"

	"skill-go/pkg/event"
)

func TestRenderer_NoEvents(t *testing.T) {
	r := NewRenderer()
	output := r.Render()
	if output != "No events recorded." {
		t.Errorf("expected empty message, got %q", output)
	}
}

func TestRenderer_RecordAndRender(t *testing.T) {
	r := NewRenderer()
	bus := event.NewBus()
	r.SubscribeAll(bus)

	r.SetTime(100 * time.Millisecond)
	bus.Publish(event.Event{
		Type:     event.OnSpellCastStart,
		SourceID: 1,
		TargetID: 2,
		SpellID:  133,
		Extra:    map[string]any{"castTime": uint32(1500), "spellName": "Fireball"},
	})

	output := r.Render()
	if !strings.Contains(output, "100ms") {
		t.Error("expected time 100ms in output")
	}
	if !strings.Contains(output, "SpellCastStart") {
		t.Error("expected SpellCastStart event in output")
	}
	if !strings.Contains(output, "Fireball") {
		t.Error("expected spell name Fireball in output")
	}
}

func TestRenderer_MultipleEvents(t *testing.T) {
	r := NewRenderer()
	bus := event.NewBus()
	r.SubscribeAll(bus)

	r.SetTime(0)
	bus.Publish(event.Event{
		Type:     event.OnSpellCastStart,
		SourceID: 1,
		SpellID:  10,
		Extra:    map[string]any{"castTime": uint32(0), "spellName": "Test"},
	})

	r.SetTime(100 * time.Millisecond)
	bus.Publish(event.Event{
		Type:     event.OnSpellHit,
		SourceID: 1,
		TargetID: 2,
		SpellID:  10,
		Value:    500,
		Extra:    map[string]any{"crit": false, "spellName": "Test"},
	})

	r.SetTime(200 * time.Millisecond)
	bus.Publish(event.Event{
		Type:     event.OnAuraApplied,
		SourceID: 1,
		TargetID: 2,
		SpellID:  100,
		Extra:    map[string]any{"auraType": "DoT", "duration": 5 * time.Second, "spellName": "TestDoT"},
	})

	output := r.Render()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// header + separator + 3 events = 5 lines
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d:\n%s", len(lines), output)
	}
}

func TestRenderer_SetTime(t *testing.T) {
	r := NewRenderer()
	bus := event.NewBus()
	r.SubscribeAll(bus)

	r.SetTime(5000 * time.Millisecond)
	bus.Publish(event.Event{
		Type:     event.OnSpellFinish,
		SourceID: 1,
		SpellID:  1,
		Extra:    map[string]any{"spellName": "X", "result": 0},
	})

	output := r.Render()
	if !strings.Contains(output, "5000ms") {
		t.Error("expected 5000ms timestamp")
	}
}
