package spellcore

import (
	"testing"
	"time"
)

// trackingMockHost extends mockAuraHost to record ForApp calls.
type trackingMockHost struct {
	mockAuraHost
	appliedForApp  int
	removedForApp  int
	lastAppliedApp *AuraApplication
	lastRemovedApp *AuraApplication
}

func newTrackingHost(id uint64) *trackingMockHost {
	return &trackingMockHost{
		mockAuraHost: mockAuraHost{id: id},
	}
}

func (h *trackingMockHost) ApplyAuraEffectsForApp(app *AuraApplication) {
	h.appliedForApp++
	h.lastAppliedApp = app
}

func (h *trackingMockHost) RemoveAuraEffectsForApp(app *AuraApplication) {
	h.removedForApp++
	h.lastRemovedApp = app
}

func TestNewAuraApplication(t *testing.T) {
	a := NewAura(SpellID(42), 1, 2, AuraModStat, 10*time.Second)
	app := NewAuraApplication(a, 2)

	if app.Base != a {
		t.Error("Base should point to the aura")
	}
	if app.TargetID != 2 {
		t.Errorf("TargetID = %d, want 2", app.TargetID)
	}
	if app.RemoveMode != RemoveNone {
		t.Errorf("RemoveMode = %d, want RemoveNone", app.RemoveMode)
	}
	if app.EffectsToApply != ^uint32(0) {
		t.Errorf("EffectsToApply = %d, want full mask", app.EffectsToApply)
	}
	if app.EffectMask != ^uint32(0) {
		t.Errorf("EffectMask = %d, want full mask", app.EffectMask)
	}
}

func TestAuraApplication_Getters(t *testing.T) {
	a := NewAura(SpellID(99), 10, 20, AuraModStun, 5*time.Second)
	app := NewAuraApplication(a, 20)

	if app.GetAura() != a {
		t.Error("GetAura should return the base aura")
	}
	if app.GetSpellID() != SpellID(99) {
		t.Errorf("GetSpellID = %d, want 99", app.GetSpellID())
	}
	if app.GetCasterID() != 10 {
		t.Errorf("GetCasterID = %d, want 10", app.GetCasterID())
	}
}

func TestAuraApplication_HasEffect(t *testing.T) {
	app := &AuraApplication{EffectMask: 0x05} // bits 0 and 2

	if !app.HasEffect(0) {
		t.Error("bit 0 should be set")
	}
	if app.HasEffect(1) {
		t.Error("bit 1 should not be set")
	}
	if !app.HasEffect(2) {
		t.Error("bit 2 should be set")
	}
	if app.HasEffect(3) {
		t.Error("bit 3 should not be set")
	}
}

func TestAuraApplication_UpdateApplyEffectMask_RemoveEffects(t *testing.T) {
	a := NewAura(SpellID(1), 1, 2, AuraModStat, 10*time.Second)
	app := NewAuraApplication(a, 2)
	target := newTrackingHost(2)

	// Remove effect 0: new mask has bits 1+ but not 0
	newMask := ^uint32(0) & ^uint32(1) // all bits except bit 0
	app.UpdateApplyEffectMask(newMask, target)

	if target.removedForApp != 1 {
		t.Errorf("RemoveAuraEffectsForApp called %d times, want 1", target.removedForApp)
	}
	if target.appliedForApp != 0 {
		t.Errorf("ApplyAuraEffectsForApp called %d times, want 0", target.appliedForApp)
	}
	// EffectMask should have bit 0 cleared
	if app.EffectMask&(1) != 0 {
		t.Error("EffectMask should have bit 0 cleared after removal")
	}
	// EffectsToApply should match new mask
	if app.EffectsToApply != newMask {
		t.Errorf("EffectsToApply = %d, want %d", app.EffectsToApply, newMask)
	}
}

func TestAuraApplication_UpdateApplyEffectMask_AddEffects(t *testing.T) {
	a := NewAura(SpellID(1), 1, 2, AuraModStat, 10*time.Second)
	app := &AuraApplication{
		Base:           a,
		TargetID:       2,
		EffectsToApply: 0x02, // only bit 1
		EffectMask:     0x02,
	}
	target := newTrackingHost(2)

	// Add bit 0
	newMask := uint32(0x03) // bits 0 and 1
	app.UpdateApplyEffectMask(newMask, target)

	if target.appliedForApp != 1 {
		t.Errorf("ApplyAuraEffectsForApp called %d times, want 1", target.appliedForApp)
	}
	if target.removedForApp != 0 {
		t.Errorf("RemoveAuraEffectsForApp called %d times, want 0", target.removedForApp)
	}
	if app.EffectMask&(1) == 0 {
		t.Error("EffectMask should have bit 0 set after addition")
	}
}

func TestAuraApplication_UpdateApplyEffectMask_NoChange(t *testing.T) {
	a := NewAura(SpellID(1), 1, 2, AuraModStat, 10*time.Second)
	app := NewAuraApplication(a, 2)
	target := newTrackingHost(2)

	// Same mask — should be a no-op
	app.UpdateApplyEffectMask(^uint32(0), target)

	if target.appliedForApp != 0 || target.removedForApp != 0 {
		t.Error("same mask should not trigger any ForApp calls")
	}
}

func TestAuraManager_RemoveAuraApplication_AreaAuraRetainsOwned(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target1 := &mockAuraHost{id: 2}
	target2 := &mockAuraHost{id: 3}

	// Create an area aura and apply to two targets
	a := NewAura(SpellID(100), 1, 0, AuraModStat, 10*time.Second)
	a.IsAreaAura = true

	app1 := NewAuraApplication(a, target1.id)
	app2 := NewAuraApplication(a, target2.id)

	owner.AddOwnedAura(a)
	target1.AddAppliedAuraApp(app1)
	target2.AddAppliedAuraApp(app2)
	a.Applications[target1.id] = app1
	a.Applications[target2.id] = app2

	// Remove one application — aura should still be owned
	mgr.RemoveAuraApplication(app1, owner, target1, RemoveByCancel)

	if len(owner.GetOwnedAuras()) != 1 {
		t.Error("aura should still be owned after removing one of two apps")
	}
	if len(target1.GetAppliedAuraApps()) != 0 {
		t.Error("target1 should have no apps")
	}
	if len(target2.GetAppliedAuraApps()) != 1 {
		t.Error("target2 should still have its app")
	}
	if len(a.Applications) != 1 {
		t.Errorf("expected 1 remaining application, got %d", len(a.Applications))
	}

	// Remove the last application — aura should be unowned
	mgr.RemoveAuraApplication(app2, owner, target2, RemoveByCancel)

	if len(owner.GetOwnedAuras()) != 0 {
		t.Error("aura should be removed from owner after last app removed")
	}
	if len(a.Applications) != 0 {
		t.Errorf("expected 0 applications, got %d", len(a.Applications))
	}
}

func TestAuraManager_RemoveAllApplications(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target1 := &mockAuraHost{id: 2}
	target2 := &mockAuraHost{id: 3}

	a := NewAura(SpellID(100), 1, 0, AuraModStat, 10*time.Second)
	a.IsAreaAura = true

	app1 := NewAuraApplication(a, target1.id)
	app2 := NewAuraApplication(a, target2.id)

	owner.AddOwnedAura(a)
	target1.AddAppliedAuraApp(app1)
	target2.AddAppliedAuraApp(app2)
	a.Applications[target1.id] = app1
	a.Applications[target2.id] = app2

	getTarget := func(id uint64) AuraHost {
		switch id {
		case 2:
			return target1
		case 3:
			return target2
		default:
			return nil
		}
	}
	mgr.RemoveAllApplications(a, owner, RemoveByCancel, getTarget)

	if len(owner.GetOwnedAuras()) != 0 {
		t.Error("owner should have no auras after RemoveAllApplications")
	}
	if len(target1.GetAppliedAuraApps()) != 0 {
		t.Error("target1 should have no apps")
	}
	if len(target2.GetAppliedAuraApps()) != 0 {
		t.Error("target2 should have no apps")
	}
	if len(a.Applications) != 0 {
		t.Error("all applications should be removed")
	}
}

func TestAuraManager_RemoveAllApplications_SkipsNilTarget(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}

	a := NewAura(SpellID(100), 1, 0, AuraModStat, 10*time.Second)
	app := NewAuraApplication(a, 999) // non-existent target
	owner.AddOwnedAura(a)
	a.Applications[999] = app

	getTarget := func(id uint64) AuraHost { return nil }
	mgr.RemoveAllApplications(a, owner, RemoveByCancel, getTarget)

	// Aura should be removed since the application was cleaned up
	// (even though target was nil, app is removed from map and ownedAuras)
	if len(a.Applications) != 0 {
		t.Error("application for nil target should be cleaned from map")
	}
}

func TestAuraManager_UpdateTargetMap_AddTargets(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target1 := &mockAuraHost{id: 2}
	target2 := &mockAuraHost{id: 3}

	a := NewAura(SpellID(100), 1, 0, AuraModStat, 10*time.Second)
	a.IsAreaAura = true
	owner.AddOwnedAura(a)

	getTarget := func(id uint64) AuraHost {
		switch id {
		case 2:
			return target1
		case 3:
			return target2
		default:
			return nil
		}
	}

	mgr.UpdateTargetMap(a, owner, []AuraHost{target1, target2}, getTarget)

	if len(a.Applications) != 2 {
		t.Errorf("expected 2 applications, got %d", len(a.Applications))
	}
	if len(target1.GetAppliedAuraApps()) != 1 {
		t.Error("target1 should have 1 app")
	}
	if len(target2.GetAppliedAuraApps()) != 1 {
		t.Error("target2 should have 1 app")
	}
}

func TestAuraManager_UpdateTargetMap_RemoveTarget(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target1 := &mockAuraHost{id: 2}
	target2 := &mockAuraHost{id: 3}

	a := NewAura(SpellID(100), 1, 0, AuraModStat, 10*time.Second)
	a.IsAreaAura = true
	owner.AddOwnedAura(a)

	getTarget := func(id uint64) AuraHost {
		switch id {
		case 2:
			return target1
		case 3:
			return target2
		default:
			return nil
		}
	}

	// Initially add both targets
	mgr.UpdateTargetMap(a, owner, []AuraHost{target1, target2}, getTarget)
	if len(a.Applications) != 2 {
		t.Fatalf("expected 2 apps after initial add, got %d", len(a.Applications))
	}

	// Remove target2 — only target1 in range
	mgr.UpdateTargetMap(a, owner, []AuraHost{target1}, getTarget)

	if len(a.Applications) != 1 {
		t.Errorf("expected 1 app after removal, got %d", len(a.Applications))
	}
	if len(target1.GetAppliedAuraApps()) != 1 {
		t.Error("target1 should still have its app")
	}
	if len(target2.GetAppliedAuraApps()) != 0 {
		t.Error("target2 should have no apps after removal")
	}
	// Aura should still be owned (1 app remaining)
	if len(owner.GetOwnedAuras()) != 1 {
		t.Error("aura should still be owned with 1 app remaining")
	}
}

func TestAuraManager_UpdateTargetMap_NoChange(t *testing.T) {
	mgr := NewAuraManager(nil)
	owner := &mockAuraHost{id: 1}
	target1 := &mockAuraHost{id: 2}

	a := NewAura(SpellID(100), 1, 0, AuraModStat, 10*time.Second)
	a.IsAreaAura = true
	owner.AddOwnedAura(a)

	getTarget := func(id uint64) AuraHost {
		if id == 2 {
			return target1
		}
		return nil
	}

	mgr.UpdateTargetMap(a, owner, []AuraHost{target1}, getTarget)

	// Call again with same targets — should be a no-op
	appBefore := a.Applications[2]
	mgr.UpdateTargetMap(a, owner, []AuraHost{target1}, getTarget)

	if len(a.Applications) != 1 {
		t.Error("should still have exactly 1 app")
	}
	if a.Applications[2] != appBefore {
		t.Error("app should be the same instance (not recreated)")
	}
}
