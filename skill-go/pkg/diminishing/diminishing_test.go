package diminishing

import (
	"testing"
	"time"
)

func TestApplyDiminishing_GroupNone(t *testing.T) {
	m := NewManager()
	dur, immune := m.ApplyDiminishing(1, 2, GroupNone, 5*time.Second)
	if immune {
		t.Error("GroupNone should not be immune")
	}
	if dur != 5*time.Second {
		t.Errorf("GroupNone duration = %v, want 5s", dur)
	}
}

func TestApplyDiminishing_UnregisteredGroup(t *testing.T) {
	m := NewManager()
	dur, immune := m.ApplyDiminishing(1, 2, GroupStun, 5*time.Second)
	if immune {
		t.Error("unregistered group should not be immune")
	}
	if dur != 5*time.Second {
		t.Errorf("unregistered group duration = %v, want 5s", dur)
	}
}

func TestApplyDiminishing_LevelProgression(t *testing.T) {
	m := NewManager()
	m.RegisterLevel(Level{Group: GroupStun, ReturnType: ReturnStandard, MaxLevel: 5, DurLimit: 0})

	// Level 0: full duration
	dur, immune := m.ApplyDiminishing(1, 2, GroupStun, 8*time.Second)
	if immune || dur != 8*time.Second {
		t.Errorf("level 0: dur=%v immune=%v, want 8s false", dur, immune)
	}

	// Level 1: half duration
	dur, immune = m.ApplyDiminishing(1, 2, GroupStun, 8*time.Second)
	if immune || dur != 4*time.Second {
		t.Errorf("level 1: dur=%v immune=%v, want 4s false", dur, immune)
	}

	// Level 2: quarter duration
	dur, immune = m.ApplyDiminishing(1, 2, GroupStun, 8*time.Second)
	if immune || dur != 2*time.Second {
		t.Errorf("level 2: dur=%v immune=%v, want 2s false", dur, immune)
	}

	// Level 3+: immune
	dur, immune = m.ApplyDiminishing(1, 2, GroupStun, 8*time.Second)
	if !immune || dur != 0 {
		t.Errorf("level 3: dur=%v immune=%v, want 0 true", dur, immune)
	}
}

func TestApplyDiminishing_MaxLevel(t *testing.T) {
	m := NewManager()
	m.RegisterLevel(Level{Group: GroupFear, ReturnType: ReturnStandard, MaxLevel: 2, DurLimit: 0})

	m.ApplyDiminishing(1, 2, GroupFear, 6*time.Second) // level 0
	m.ApplyDiminishing(1, 2, GroupFear, 6*time.Second) // level 1

	// level 2 >= MaxLevel(2) → immune
	dur, immune := m.ApplyDiminishing(1, 2, GroupFear, 6*time.Second)
	if !immune {
		t.Errorf("at MaxLevel should be immune, got dur=%v immune=%v", dur, immune)
	}
}

func TestApplyDiminishing_DurLimit(t *testing.T) {
	m := NewManager()
	m.RegisterLevel(Level{Group: GroupRoot, ReturnType: ReturnStandard, MaxLevel: 0, DurLimit: 3 * time.Second})

	dur, immune := m.ApplyDiminishing(1, 2, GroupRoot, 10*time.Second)
	if immune {
		t.Error("should not be immune")
	}
	if dur != 3*time.Second {
		t.Errorf("DurLimit should cap at 3s, got %v", dur)
	}
}

func TestApplyDiminishing_DifferentTargets(t *testing.T) {
	m := NewManager()
	m.RegisterLevel(Level{Group: GroupStun, ReturnType: ReturnStandard, MaxLevel: 0, DurLimit: 0})

	// Target A gets DR
	m.ApplyDiminishing(1, 2, GroupStun, 8*time.Second)
	dur, _ := m.ApplyDiminishing(1, 2, GroupStun, 8*time.Second)
	if dur != 4*time.Second {
		t.Errorf("target A level 1 should be 4s, got %v", dur)
	}

	// Target B is independent
	dur2, _ := m.ApplyDiminishing(10, 2, GroupStun, 8*time.Second)
	if dur2 != 8*time.Second {
		t.Errorf("target B level 0 should be 8s, got %v", dur2)
	}
}

func TestManager_GetLevel(t *testing.T) {
	m := NewManager()
	m.RegisterLevel(Level{Group: GroupStun, ReturnType: ReturnStandard, MaxLevel: 0, DurLimit: 0})

	if lvl := m.GetLevel(1, GroupStun); lvl != 0 {
		t.Errorf("initial level should be 0, got %d", lvl)
	}

	m.ApplyDiminishing(1, 2, GroupStun, 5*time.Second)
	if lvl := m.GetLevel(1, GroupStun); lvl != 1 {
		t.Errorf("after 1 apply, level should be 1, got %d", lvl)
	}
}

func TestManager_Clear(t *testing.T) {
	m := NewManager()
	m.RegisterLevel(Level{Group: GroupStun, ReturnType: ReturnStandard, MaxLevel: 0, DurLimit: 0})

	m.ApplyDiminishing(1, 2, GroupStun, 5*time.Second)
	m.ApplyDiminishing(1, 2, GroupStun, 5*time.Second)
	m.Clear(1)

	if lvl := m.GetLevel(1, GroupStun); lvl != 0 {
		t.Errorf("after Clear, level should be 0, got %d", lvl)
	}
}
