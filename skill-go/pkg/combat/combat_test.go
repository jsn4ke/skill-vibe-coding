package combat

import (
	"math/rand"
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

func TestAttackRoll(t *testing.T) {
	tests := []struct {
		name        string
		critChance  float64
		dodgeChance float64
		parryChance float64
		blockChance float64
		wantHit     spellcore.HitResult
		wantCrit    bool
		wantMult    float64
	}{
		{"all zero → normal", 0, 0, 0, 0, spellcore.HitNormal, false, 1.0},
		{"100% crit", 1.0, 0, 0, 0, spellcore.HitCrit, true, 1.5},
		{"100% dodge", 0, 1.0, 0, 0, spellcore.HitDodge, false, 0},
		{"100% parry", 0, 0, 1.0, 0, spellcore.HitEvade, false, 0},
		{"100% block", 0, 0, 0, 1.0, spellcore.HitNormal, false, 1.0},
		{"dodge beats crit", 1.0, 1.0, 0, 0, spellcore.HitDodge, false, 0},
		{"parry beats crit", 1.0, 0, 1.0, 0, spellcore.HitEvade, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(0))
			got := AttackRoll(tt.critChance, tt.dodgeChance, tt.parryChance, tt.blockChance, rng)
			if got.HitResult != tt.wantHit {
				t.Errorf("HitResult = %v, want %v", got.HitResult, tt.wantHit)
			}
			if got.IsCrit != tt.wantCrit {
				t.Errorf("IsCrit = %v, want %v", got.IsCrit, tt.wantCrit)
			}
			if got.DamageMult != tt.wantMult {
				t.Errorf("DamageMult = %v, want %v", got.DamageMult, tt.wantMult)
			}
		})
	}
}

func TestMitigateDamage(t *testing.T) {
	tests := []struct {
		name       string
		damage     float64
		armor      float64
		resistance float64
		school     DamageSchool
		wantMin    float64
		wantMax    float64
	}{
		{"zero armor full physical", 100, 0, 0, SchoolPhysical, 100, 100},
		{"armor reduces physical", 1000, 1000, 0, SchoolPhysical, 300, 325},
		{"zero resistance full magic", 100, 0, 0, SchoolFire, 100, 100},
		{"resistance reduces magic", 1000, 0, 300, SchoolFire, 330, 340},
		{"negative damage returns 0", -10, 0, 0, SchoolPhysical, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MitigateDamage(tt.damage, tt.armor, tt.resistance, tt.school)
			if got < tt.wantMin-1 || got > tt.wantMax+1 {
				t.Errorf("MitigateDamage = %v, want [%v, %v]", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestApplyAbsorb(t *testing.T) {
	tests := []struct {
		name          string
		damage        float64
		absorb        float64
		wantRemaining float64
		wantAbsorbed  float64
	}{
		{"damage exceeds absorb", 100, 30, 70, 30},
		{"absorb fully blocks", 50, 100, 0, 50},
		{"exact match", 100, 100, 0, 100},
		{"zero absorb", 100, 0, 100, 0},
		{"zero damage", 0, 100, 0, 0},
		{"both zero", 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyAbsorb(tt.damage, tt.absorb)
			if got.RemainingDamage != tt.wantRemaining {
				t.Errorf("RemainingDamage = %v, want %v", got.RemainingDamage, tt.wantRemaining)
			}
			if got.Absorbed != tt.wantAbsorbed {
				t.Errorf("Absorbed = %v, want %v", got.Absorbed, tt.wantAbsorbed)
			}
		})
	}
}
