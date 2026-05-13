package event

// EventType 表示事件类型的枚举，对齐 TC 的事件分类。
type EventType uint16

const (
	OnDamageDealt EventType = iota
	OnDamageTaken
	OnHealDealt
	OnHealTaken
	OnSpellHit
	OnAuraApplied
	OnAuraTick
	OnDeath
	OnSpellCastStart
	OnSpellLaunch
	OnSpellCancel
	OnSpellFinish
	OnAuraExpired
)

// Event 表示一个游戏事件，包含来源、目标、法术和数值信息。
type Event struct {
	Type     EventType
	SourceID uint64
	TargetID uint64
	SpellID  uint32
	Value    float64
	Extra    map[string]interface{}
}

// Handler 是事件处理函数的类型。
type Handler func(Event)

// Bus 是事件总线，实现发布/订阅模式。仅用于观察者（timeline、日志、UI），
// 技能逻辑应使用 script.Registry 的精确 SpellID 匹配钩子。
type Bus struct {
	handlers map[EventType][]Handler
}

// NewBus 创建一个空的事件总线。
func NewBus() *Bus {
	return &Bus{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe 为指定事件类型注册处理函数。
func (b *Bus) Subscribe(et EventType, h Handler) {
	b.handlers[et] = append(b.handlers[et], h)
}

// Publish 发布事件，通知所有订阅者。
func (b *Bus) Publish(e Event) {
	if handlers, ok := b.handlers[e.Type]; ok {
		for _, h := range handlers {
			h(e)
		}
	}
}

// UnsubscribeAll 移除指定事件类型的所有处理函数。
func (b *Bus) UnsubscribeAll(et EventType) {
	delete(b.handlers, et)
}
