package aura

import (
	"math"
	"testing"
	"time"

	"skill-go/pkg/spell"
)

func TestTickPeriodic_DamageWithSpellPower(t *testing.T) {
	mgr := NewManager(nil)
	a := NewAura(spell.SpellID(25306), 1, 2, AuraPeriodicDamage, 8*time.Second)
	a.Effects = []AuraEffect{
		{EffectIndex: 0, AuraType: AuraPeriodicDamage, Amount: 19, BonusCoeff: 0.125, Period: 2 * time.Second},
	}
	mgr.AddAura(a)

	var tickAmount float64
	mgr.TickPeriodic(2, 2*time.Second, 100, func(a *Aura, eff *AuraEffect, amount float64) {
		tickAmount = amount
	})

	expected := 19 + 0.125*100
	if math.Abs(tickAmount-expected) > 0.01 {
		t.Errorf("expected tick damage %.2f, got %.2f", expected, tickAmount)
	}
}

func TestTickPeriodic_ZeroSpellPower(t *testing.T) {
	mgr := NewManager(nil)
	a := NewAura(spell.SpellID(25306), 1, 2, AuraPeriodicDamage, 8*time.Second)
	a.Effects = []AuraEffect{
		{EffectIndex: 0, AuraType: AuraPeriodicDamage, Amount: 19, BonusCoeff: 0.125, Period: 2 * time.Second},
	}
	mgr.AddAura(a)

	var tickAmount float64
	mgr.TickPeriodic(2, 2*time.Second, 0, func(a *Aura, eff *AuraEffect, amount float64) {
		tickAmount = amount
	})

	if tickAmount != 19 {
		t.Errorf("expected tick damage 19 with 0 SP, got %.2f", tickAmount)
	}
}

func TestTickPeriodic_MultipleTicks(t *testing.T) {
	mgr := NewManager(nil)
	a := NewAura(spell.SpellID(25306), 1, 2, AuraPeriodicDamage, 8*time.Second)
	a.Effects = []AuraEffect{
		{EffectIndex: 0, AuraType: AuraPeriodicDamage, Amount: 19, BonusCoeff: 0.125, Period: 2 * time.Second},
	}
	mgr.AddAura(a)

	tickCount := 0
	mgr.TickPeriodic(2, 8*time.Second, 100, func(a *Aura, eff *AuraEffect, amount float64) {
		tickCount++
	})

	if tickCount != 4 {
		t.Errorf("expected 4 ticks over 8s with 2s period, got %d", tickCount)
	}
}
