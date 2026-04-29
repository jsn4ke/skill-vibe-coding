package spellcore

import (
	"testing"
	"time"
)

func TestHistory_Cooldown(t *testing.T) {
	h := NewHistory()

	h.AddCooldown(SpellID(1), 100, 5*time.Second)

	if h.IsReady(SpellID(1), 100) {
		t.Error("spell on cooldown should not be ready")
	}
	if !h.IsReady(SpellID(2), 0) {
		t.Error("unrelated spell should be ready")
	}
	if h.IsReady(SpellID(1), 0) {
		t.Error("spell on cooldown by ID should not be ready even with categoryID=0")
	}
}

func TestHistory_CancelCooldown(t *testing.T) {
	h := NewHistory()
	h.AddCooldown(SpellID(1), 100, 5*time.Second)
	h.CancelCooldown(SpellID(1))

	// CancelCooldown only deletes spell-level cooldown; category cooldown persists
	if !h.IsReady(SpellID(1), 0) {
		t.Error("spell should be ready after cancel (no category check)")
	}
}

func TestHistory_GlobalCooldown(t *testing.T) {
	h := NewHistory()

	h.AddGlobalCooldown(100, 2*time.Second)
	if !h.HasGlobalCooldown(100) {
		t.Error("should have GCD")
	}

	h.CancelGlobalCooldown(100)
	if h.HasGlobalCooldown(100) {
		t.Error("GCD should be cancelled")
	}
}

func TestHistory_SchoolLockout(t *testing.T) {
	h := NewHistory()

	h.AddSchoolLockout(0, 3*time.Second)
	if !h.HasSchoolLockout(0) {
		t.Error("should have school lockout")
	}
	if h.HasSchoolLockout(7) {
		t.Error("school 7+ should not have lockout")
	}
}

func TestHistory_Charges(t *testing.T) {
	h := NewHistory()
	h.AddCharge(SpellID(10), 2, 5*time.Second)

	// Initially should have charges
	if !h.IsReady(SpellID(10), 0) {
		t.Error("should be ready with full charges")
	}

	// Use one charge
	if !h.UseCharge(SpellID(10)) {
		t.Error("UseCharge should succeed")
	}

	// Use second charge
	if !h.UseCharge(SpellID(10)) {
		t.Error("UseCharge should succeed for second charge")
	}
}

func TestHistory_IsReady_NoCooldown(t *testing.T) {
	h := NewHistory()
	if !h.IsReady(SpellID(999), 0) {
		t.Error("spell with no cooldown should be ready")
	}
}
