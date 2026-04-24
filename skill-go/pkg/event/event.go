package event

type EventType uint16

const (
	OnDamageDealt EventType = iota
	OnDamageTaken
	OnHealDealt
	OnHealTaken
	OnSpellCast
	OnSpellHit
	OnSpellMiss
	OnAuraApplied
	OnAuraRemoved
	OnAuraTick
	OnDeath
	OnKill
	OnMovement
	OnCombatEnter
	OnCombatLeave
	OnAttackSwing
	OnPeriodicTick
	OnSpellCastStart
	OnSpellLaunch
	OnSpellCancel
	OnSpellFinish
	OnAuraExpired
)

type Event struct {
	Type     EventType
	SourceID uint64
	TargetID uint64
	SpellID  uint32
	Value    float64
	Extra    map[string]interface{}
}

type Handler func(Event)

type Bus struct {
	handlers map[EventType][]Handler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[EventType][]Handler),
	}
}

func (b *Bus) Subscribe(et EventType, h Handler) {
	b.handlers[et] = append(b.handlers[et], h)
}

func (b *Bus) Publish(e Event) {
	if handlers, ok := b.handlers[e.Type]; ok {
		for _, h := range handlers {
			h(e)
		}
	}
}

func (b *Bus) UnsubscribeAll(et EventType) {
	delete(b.handlers, et)
}
