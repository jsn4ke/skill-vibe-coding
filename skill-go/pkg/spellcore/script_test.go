package spellcore

import "testing"

func TestRegistry_SpellHook(t *testing.T) {
	r := NewRegistry()
	spellID := SpellID(1337)

	var called bool
	r.RegisterSpellHook(spellID, HookOnEffectHit, func(ctx *SpellContext) {
		called = true
	})

	if !r.HasSpellHook(spellID, HookOnEffectHit) {
		t.Error("HasSpellHook should return true after registration")
	}
	if r.HasSpellHook(spellID, HookOnCast) {
		t.Error("HasSpellHook should return false for unregistered hook")
	}

	r.CallSpellHook(spellID, HookOnEffectHit, &SpellContext{})
	if !called {
		t.Error("handler should have been called")
	}
}

func TestRegistry_SpellHook_PreventDefault(t *testing.T) {
	r := NewRegistry()
	spellID := SpellID(42)

	order := []int{}
	r.RegisterSpellHook(spellID, HookOnHit, func(ctx *SpellContext) {
		order = append(order, 1)
		ctx.PreventDefault = true
	})
	r.RegisterSpellHook(spellID, HookOnHit, func(ctx *SpellContext) {
		order = append(order, 2)
	})

	r.CallSpellHook(spellID, HookOnHit, &SpellContext{})
	if len(order) != 1 || order[0] != 1 {
		t.Errorf("PreventDefault should stop chain, got order=%v", order)
	}
}

func TestRegistry_AuraHook(t *testing.T) {
	r := NewRegistry()
	spellID := SpellID(100)

	var called bool
	r.RegisterAuraHook(spellID, AuraHookAfterApply, func(ctx *AuraContext) {
		called = true
	})

	r.CallAuraHook(spellID, AuraHookAfterApply, &AuraContext{})
	if !called {
		t.Error("aura handler should have been called")
	}
}

func TestRegistry_UnregisteredHook(t *testing.T) {
	r := NewRegistry()
	// Should not panic
	r.CallSpellHook(SpellID(999), HookOnCast, &SpellContext{})
	r.CallAuraHook(SpellID(999), AuraHookOnApply, &AuraContext{})
}

func TestRegistry_UnregisterAll(t *testing.T) {
	r := NewRegistry()
	spellID := SpellID(50)

	r.RegisterSpellHook(spellID, HookOnCast, func(ctx *SpellContext) {})
	r.RegisterAuraHook(spellID, AuraHookOnApply, func(ctx *AuraContext) {})

	r.UnregisterAll(spellID)

	if r.HasSpellHook(spellID, HookOnCast) {
		t.Error("spell hook should be removed after UnregisterAll")
	}
	if r.HasAuraHook(spellID, AuraHookOnApply) {
		t.Error("aura hook should be removed after UnregisterAll")
	}
}
