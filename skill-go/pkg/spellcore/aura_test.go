package spellcore

import (
	"testing"
	"time"
)

func TestAura_IsExpired(t *testing.T) {
	t.Run("zero duration never expires", func(t *testing.T) {
		a := NewAura(1, 1, 2, AuraModStat, 0)
		if a.IsExpired() {
			t.Error("zero-duration aura should not expire")
		}
	})

	t.Run("elapsed-based expiry", func(t *testing.T) {
		a := NewAura(1, 1, 2, AuraModStat, 5*time.Second)
		a.Elapsed = 6 * time.Second
		if !a.IsExpired() {
			t.Error("aura past duration should be expired")
		}
	})

	t.Run("not yet expired", func(t *testing.T) {
		a := NewAura(1, 1, 2, AuraModStat, 5*time.Second)
		a.Elapsed = 3 * time.Second
		if a.IsExpired() {
			t.Error("aura not yet at duration should not be expired")
		}
	})
}

func TestAura_AddStack(t *testing.T) {
	a := NewAura(1, 1, 2, AuraModStat, 10*time.Second)
	a.MaxStack = 3

	a.AddStack()
	if a.StackAmount != 1 {
		t.Errorf("after AddStack, StackAmount = %d, want 1", a.StackAmount)
	}

	a.AddStack()
	a.AddStack()
	a.AddStack() // exceeds MaxStack
	if a.StackAmount != 3 {
		t.Errorf("StackAmount capped at MaxStack: got %d, want 3", a.StackAmount)
	}
}

func TestAura_AddStack_ZeroMax(t *testing.T) {
	a := NewAura(1, 1, 2, AuraModStat, 10*time.Second)
	a.MaxStack = 0
	a.AddStack()
	if a.StackAmount != 0 {
		t.Errorf("zero MaxStack should not stack, got %d", a.StackAmount)
	}
}

func TestAura_RemoveStack(t *testing.T) {
	a := NewAura(1, 1, 2, AuraModStat, 10*time.Second)
	a.MaxStack = 5
	a.StackAmount = 3

	a.RemoveStack(1)
	if a.StackAmount != 2 {
		t.Errorf("after RemoveStack(1), got %d", a.StackAmount)
	}

	a.RemoveStack(10) // remove more than available
	if a.StackAmount != 0 {
		t.Errorf("after RemoveStack(10), got %d, want 0", a.StackAmount)
	}
}

func TestAura_CalcAmount(t *testing.T) {
	a := &Aura{
		StackAmount: 3,
		Effects: []AuraEffect{
			{Amount: 50},
			{Amount: 100},
		},
	}

	if got := a.CalcAmount(0); got != 150 {
		t.Errorf("CalcAmount(0) = %v, want 150", got)
	}
	if got := a.CalcAmount(1); got != 300 {
		t.Errorf("CalcAmount(1) = %v, want 300", got)
	}
	if got := a.CalcAmount(-1); got != 0 {
		t.Errorf("CalcAmount(-1) = %v, want 0", got)
	}
	if got := a.CalcAmount(5); got != 0 {
		t.Errorf("CalcAmount(5) = %v, want 0", got)
	}
}

type mockAuraHost struct {
	id           uint64
	ownedAuras   []*Aura
	appliedAuras []*Aura
}

func (h *mockAuraHost) GetID() uint64        { return h.id }
func (h *mockAuraHost) AddOwnedAura(a *Aura) { h.ownedAuras = append(h.ownedAuras, a) }
func (h *mockAuraHost) RemoveOwnedAura(idx int) {
	h.ownedAuras = append(h.ownedAuras[:idx], h.ownedAuras[idx+1:]...)
}
func (h *mockAuraHost) GetOwnedAuras() []*Aura { return h.ownedAuras }
func (h *mockAuraHost) AddAppliedAura(a *Aura) { h.appliedAuras = append(h.appliedAuras, a) }
func (h *mockAuraHost) RemoveAppliedAura(idx int) {
	h.appliedAuras = append(h.appliedAuras[:idx], h.appliedAuras[idx+1:]...)
}
func (h *mockAuraHost) GetAppliedAuras() []*Aura { return h.appliedAuras }
func (h *mockAuraHost) FindAppliedAura(spellID SpellID, casterID uint64) *Aura {
	for _, a := range h.appliedAuras {
		if a.SpellID == spellID && a.CasterID == casterID {
			return a
		}
	}
	return nil
}

func TestAuraManager_ApplyAura_New(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target := &mockAuraHost{id: 2}

	a := NewAura(SpellID(100), 1, 2, AuraModStat, 10*time.Second)
	mgr.ApplyAura(owner, target, a)

	if len(owner.GetOwnedAuras()) != 1 {
		t.Errorf("owner should have 1 owned aura, got %d", len(owner.GetOwnedAuras()))
	}
	if len(target.GetAppliedAuras()) != 1 {
		t.Errorf("target should have 1 applied aura, got %d", len(target.GetAppliedAuras()))
	}
	_ = a.ID // ID assigned by manager
}

func TestAuraManager_ApplyAura_StackRefresh(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target := &mockAuraHost{id: 2}

	a1 := NewAura(SpellID(100), 1, 2, AuraModStat, 10*time.Second)
	a1.StackRule = StackRefresh
	a1.ID = 1
	owner.AddOwnedAura(a1)
	target.AddAppliedAura(a1)

	a2 := NewAura(SpellID(100), 1, 2, AuraModStat, 10*time.Second)
	a2.StackRule = StackRefresh
	mgr.ApplyAura(owner, target, a2)

	// Refresh: should not add new aura
	if len(owner.GetOwnedAuras()) != 1 {
		t.Errorf("refresh should keep 1 aura, got %d", len(owner.GetOwnedAuras()))
	}
}

func TestAuraManager_ApplyAura_StackAddStack(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target := &mockAuraHost{id: 2}

	a1 := NewAura(SpellID(100), 1, 2, AuraModStat, 10*time.Second)
	a1.StackRule = StackAddStack
	a1.MaxStack = 5
	a1.ID = 1
	owner.AddOwnedAura(a1)
	target.AddAppliedAura(a1)

	a2 := NewAura(SpellID(100), 1, 2, AuraModStat, 10*time.Second)
	a2.StackRule = StackAddStack
	mgr.ApplyAura(owner, target, a2)

	if a1.StackAmount != 1 {
		t.Errorf("AddStack should increment to 1, got %d", a1.StackAmount)
	}
}

func TestAuraManager_RemoveAuraFromHosts(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target := &mockAuraHost{id: 2}

	a := NewAura(SpellID(100), 1, 2, AuraModStat, 10*time.Second)
	mgr.ApplyAura(owner, target, a)

	mgr.RemoveAuraFromHosts(a, owner, target, RemoveByCancel)

	if len(owner.GetOwnedAuras()) != 0 {
		t.Errorf("owner should have 0 auras after removal, got %d", len(owner.GetOwnedAuras()))
	}
	if len(target.GetAppliedAuras()) != 0 {
		t.Errorf("target should have 0 auras after removal, got %d", len(target.GetAppliedAuras()))
	}
}
