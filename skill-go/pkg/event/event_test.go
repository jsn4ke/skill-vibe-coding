package event

import "testing"

func TestBus_SubscribePublish(t *testing.T) {
	bus := NewBus()
	var received []Event

	bus.Subscribe(OnDamageDealt, func(e Event) {
		received = append(received, e)
	})

	bus.Publish(Event{Type: OnDamageDealt, SourceID: 1, TargetID: 2, Value: 100})
	bus.Publish(Event{Type: OnHealDealt, SourceID: 1, TargetID: 2, Value: 50})

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Value != 100 {
		t.Errorf("event Value = %v, want 100", received[0].Value)
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()
	count := 0

	bus.Subscribe(OnDeath, func(e Event) { count++ })
	bus.Subscribe(OnDeath, func(e Event) { count++ })

	bus.Publish(Event{Type: OnDeath})
	if count != 2 {
		t.Errorf("expected 2 invocations, got %d", count)
	}
}

func TestBus_UnsubscribeAll(t *testing.T) {
	bus := NewBus()
	count := 0

	bus.Subscribe(OnSpellHit, func(e Event) { count++ })
	bus.UnsubscribeAll(OnSpellHit)

	bus.Publish(Event{Type: OnSpellHit})
	if count != 0 {
		t.Errorf("expected 0 after unsubscribe, got %d", count)
	}
}

func TestBus_PublishNoSubscribers(t *testing.T) {
	bus := NewBus()
	bus.Publish(Event{Type: OnDamageDealt}) // should not panic
}
