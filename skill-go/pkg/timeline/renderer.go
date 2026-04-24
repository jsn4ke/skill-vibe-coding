package timeline

import (
	"fmt"
	"strings"
	"time"

	"skill-go/pkg/event"
)

type TimelineEvent struct {
	Index    int
	Time     time.Duration
	EvtType  event.EventType
	SourceID uint64
	TargetID uint64
	SpellID  uint32
	Value    float64
	Extra    map[string]any
}

type TimelineRenderer struct {
	events  []TimelineEvent
	simTime time.Duration
	index   int
}

func NewRenderer() *TimelineRenderer {
	return &TimelineRenderer{}
}

func (r *TimelineRenderer) SetTime(t time.Duration) {
	r.simTime = t
}

func (r *TimelineRenderer) record(e event.Event) {
	r.index++
	r.events = append(r.events, TimelineEvent{
		Index:    r.index,
		Time:     r.simTime,
		EvtType:  e.Type,
		SourceID: e.SourceID,
		TargetID: e.TargetID,
		SpellID:  e.SpellID,
		Value:    e.Value,
		Extra:    e.Extra,
	})
}

func (r *TimelineRenderer) SubscribeAll(bus *event.Bus) {
	types := []event.EventType{
		event.OnSpellCastStart,
		event.OnSpellLaunch,
		event.OnSpellHit,
		event.OnSpellCancel,
			event.OnSpellFinish,
		event.OnAuraApplied,
		event.OnAuraTick,
		event.OnAuraExpired,
	}
	for _, et := range types {
		et := et
		bus.Subscribe(et, func(e event.Event) {
			r.record(e)
		})
	}
}

func (r *TimelineRenderer) Render() string {
	if len(r.events) == 0 {
		return "No events recorded."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-8s %-18s %-16s %s\n", "Time", "Event", "Src→Tgt", "Detail"))
	sb.WriteString(strings.Repeat("─", 70) + "\n")

	for _, e := range r.events {
		timeStr := formatMs(e.Time)
		evtName := eventName(e.EvtType)
		srcTgt := formatSrcTgt(e.SourceID, e.TargetID)
		detail := formatDetail(e)
		sb.WriteString(fmt.Sprintf("%-8s %-18s %-16s %s\n", timeStr, evtName, srcTgt, detail))
	}

	return sb.String()
}

func formatMs(d time.Duration) string {
	ms := d.Milliseconds()
	return fmt.Sprintf("%dms", ms)
}

func formatSrcTgt(src, tgt uint64) string {
	if tgt == 0 {
		return fmt.Sprintf("Unit%d", src)
	}
	return fmt.Sprintf("Unit%d→Unit%d", src, tgt)
}

func eventName(et event.EventType) string {
	names := map[event.EventType]string{
		event.OnSpellCastStart: "SpellCastStart",
		event.OnSpellLaunch:    "SpellLaunch",
		event.OnSpellHit:       "SpellHit",
		event.OnSpellCancel:    "SpellCancel",
		event.OnSpellFinish:   "SpellFinish",
		event.OnAuraApplied:    "AuraApplied",
		event.OnAuraTick:       "AuraTick",
		event.OnAuraExpired:    "AuraExpired",
	}
	if n, ok := names[et]; ok {
		return n
	}
	return fmt.Sprintf("Event(%d)", et)
}

func formatDetail(e TimelineEvent) string {
	name := spellName(e)
	switch e.EvtType {
	case event.OnSpellCastStart:
		return fmt.Sprintf("%s starts casting (%dms)", name, e.Extra["castTime"])
	case event.OnSpellLaunch:
		return fmt.Sprintf("%s launched (speed=%.0f)", name, e.Extra["speed"])
	case event.OnSpellHit:
		crit := ""
		if c, ok := e.Extra["crit"]; ok && c.(bool) {
			crit = " CRIT!"
		}
		return fmt.Sprintf("%s hits for %.0f damage%s", name, e.Value, crit)
	case event.OnSpellCancel:
		return fmt.Sprintf("%s cast cancelled", name)
	case event.OnSpellFinish:
		return fmt.Sprintf("%s finished (result=%v)", name, e.Extra["result"])
	case event.OnAuraApplied:
		return fmt.Sprintf("%s %s applied (%.0fs)", name, e.Extra["auraType"], e.Extra["duration"].(time.Duration).Seconds())
	case event.OnAuraTick:
		if tid, ok := e.Extra["triggerSpellID"]; ok {
			return fmt.Sprintf("%s tick (triggers spell %v)", name, tid)
		}
		return fmt.Sprintf("%s ticks for %.1f damage", name, e.Value)
	case event.OnAuraExpired:
		return fmt.Sprintf("%s aura fades", name)
	default:
		return fmt.Sprintf("%s value=%.0f", name, e.Value)
	}
}

func spellName(e TimelineEvent) string {
	if n, ok := e.Extra["spellName"]; ok {
		if s, ok := n.(string); ok && s != "" {
			return s
		}
	}
	return fmt.Sprintf("Spell(%d)", e.SpellID)
}
