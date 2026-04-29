package combat

import (
	"testing"

	"skill-go/pkg/spellcore"
)

func TestBuildProcEvent(t *testing.T) {
	tests := []struct {
		name     string
		ctx      SettlementContext
		wantFlag spellcore.ProcFlag
		wantType spellcore.SpellTypeMask
		wantHit  spellcore.ProcHitMask
	}{
		{
			"non-periodic damage",
			SettlementContext{Damage: 100, IsPeriodic: false, IsCrit: false},
			spellcore.FlagSpellDamageDealt, spellcore.TypeMaskDamage, spellcore.ProcHitNormal,
		},
		{
			"periodic damage",
			SettlementContext{Damage: 50, IsPeriodic: true, IsCrit: false},
			spellcore.FlagPeriodicDamageDealt, spellcore.TypeMaskDamage, spellcore.ProcHitNormal,
		},
		{
			"non-periodic heal",
			SettlementContext{Healing: 80, IsPeriodic: false, IsCrit: false},
			spellcore.FlagSpellHealDealt, spellcore.TypeMaskHeal, spellcore.ProcHitNormal,
		},
		{
			"periodic heal",
			SettlementContext{Healing: 30, IsPeriodic: true, IsCrit: false},
			spellcore.FlagPeriodicHealDealt, spellcore.TypeMaskHeal, spellcore.ProcHitNormal,
		},
		{
			"no damage no heal",
			SettlementContext{Damage: 0, Healing: 0, IsCrit: false},
			spellcore.FlagSpellHit, spellcore.TypeMaskNonDmgHeal, spellcore.ProcHitNormal,
		},
		{
			"crit adds ProcHitCrit",
			SettlementContext{Damage: 100, IsCrit: true},
			spellcore.FlagSpellDamageDealt, spellcore.TypeMaskDamage, spellcore.ProcHitNormal | spellcore.ProcHitCrit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildProcEvent(tt.ctx)
			if got.Flag != tt.wantFlag {
				t.Errorf("Flag = %v, want %v", got.Flag, tt.wantFlag)
			}
			if got.TypeMask != tt.wantType {
				t.Errorf("TypeMask = %v, want %v", got.TypeMask, tt.wantType)
			}
			if got.HitMask != tt.wantHit {
				t.Errorf("HitMask = %v, want %v", got.HitMask, tt.wantHit)
			}
			if got.PhaseMask != spellcore.PhaseHit {
				t.Errorf("PhaseMask = %v, want PhaseHit", got.PhaseMask)
			}
		})
	}
}

func TestBuildVictimProcEvent(t *testing.T) {
	tests := []struct {
		name     string
		ctx      SettlementContext
		wantFlag spellcore.ProcFlag
	}{
		{"non-periodic damage taken", SettlementContext{Damage: 100, IsPeriodic: false}, spellcore.FlagSpellDamageTaken},
		{"periodic damage taken", SettlementContext{Damage: 50, IsPeriodic: true}, spellcore.FlagPeriodicDamageTaken},
		{"non-periodic heal taken", SettlementContext{Healing: 80, IsPeriodic: false}, spellcore.FlagSpellHealTaken},
		{"periodic heal taken", SettlementContext{Healing: 30, IsPeriodic: true}, spellcore.FlagPeriodicHealTaken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildVictimProcEvent(tt.ctx)
			if got.Flag != tt.wantFlag {
				t.Errorf("Flag = %v, want %v", got.Flag, tt.wantFlag)
			}
		})
	}
}
