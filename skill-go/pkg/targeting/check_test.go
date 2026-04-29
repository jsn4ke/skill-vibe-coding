package targeting

import "testing"

func TestPassesCheck(t *testing.T) {
	caster := &mockTargetUnit{id: 1, entityType: 1, alive: true, pos: &mockTargetPosition{}}

	tests := []struct {
		name      string
		check     CheckTypes
		candidate *mockTargetUnit
		want      bool
	}{
		{"nil candidate returns false", CheckDefault, nil, false},
		{"dead candidate returns false", CheckDefault, &mockTargetUnit{id: 2, alive: false, pos: &mockTargetPosition{}}, false},
		{"CheckDefault alive passes", CheckDefault, &mockTargetUnit{id: 2, entityType: 1, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckDefault different type passes", CheckDefault, &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckEnemy different type passes", CheckEnemy, &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckEnemy same type fails", CheckEnemy, &mockTargetUnit{id: 2, entityType: 1, alive: true, pos: &mockTargetPosition{}}, false},
		{"CheckAlly same type passes", CheckAlly, &mockTargetUnit{id: 2, entityType: 1, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckAlly different type fails", CheckAlly, &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, false},
		{"CheckParty same type passes (fallback to ally)", CheckParty, &mockTargetUnit{id: 2, entityType: 1, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckParty different type fails", CheckParty, &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, false},
		{"CheckRaid same type passes (fallback to ally)", CheckRaid, &mockTargetUnit{id: 2, entityType: 1, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckRaid different type fails", CheckRaid, &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, false},
		{"CheckSummoned same type passes (fallback to ally)", CheckSummoned, &mockTargetUnit{id: 2, entityType: 1, alive: true, pos: &mockTargetPosition{}}, true},
		{"CheckEntry always passes", CheckEntry, &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, true},
		{"unknown check type passes", CheckTypes(99), &mockTargetUnit{id: 2, entityType: 2, alive: true, pos: &mockTargetPosition{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var candidate TargetUnit
			if tt.candidate != nil {
				candidate = tt.candidate
			}
			if got := passesCheck(tt.check, caster, candidate); got != tt.want {
				t.Errorf("passesCheck(%v, ...) = %v, want %v", tt.check, got, tt.want)
			}
		})
	}
}
