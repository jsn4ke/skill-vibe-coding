package effect

import (
	"math"
	"testing"
	"time"

	"skill-go/pkg/aura"
	"skill-go/pkg/spell"
)

func TestHandleSchoolDamage_WithSpellPower(t *testing.T) {
	ctx := &Context{
		EffectInfo: &spell.SpellEffectInfo{
			EffectType: spell.EffectSchoolDamage,
			BasePoints: 678,
			BaseDieSides: 0,
			BonusCoeff: 1.0,
		},
		CasterSpellPower: 100,
		Crit:             false,
	}
	handleSchoolDamage(ctx)
	if ctx.BaseDamage != 678 {
		t.Errorf("expected base damage 678, got %.0f", ctx.BaseDamage)
	}
	if math.Abs(ctx.FinalDamage-778) > 0.01 {
		t.Errorf("expected final damage 778 (678+100), got %.2f", ctx.FinalDamage)
	}
}

func TestHandleSchoolDamage_ZeroSpellPower(t *testing.T) {
	ctx := &Context{
		EffectInfo: &spell.SpellEffectInfo{
			EffectType: spell.EffectSchoolDamage,
			BasePoints: 678,
			BaseDieSides: 0,
			BonusCoeff: 1.0,
		},
		CasterSpellPower: 0,
		Crit:             false,
	}
	handleSchoolDamage(ctx)
	if ctx.FinalDamage != 678 {
		t.Errorf("expected final damage 678, got %.0f", ctx.FinalDamage)
	}
}

func TestHandleSchoolDamage_WithCrit(t *testing.T) {
	ctx := &Context{
		EffectInfo: &spell.SpellEffectInfo{
			EffectType: spell.EffectSchoolDamage,
			BasePoints: 678,
			BaseDieSides: 0,
			BonusCoeff: 1.0,
		},
		CasterSpellPower: 100,
		Crit:             true,
	}
	handleSchoolDamage(ctx)
	expected := (678 + 100) * 1.5
	if math.Abs(ctx.FinalDamage-expected) > 0.01 {
		t.Errorf("expected crit damage %.2f, got %.2f", expected, ctx.FinalDamage)
	}
}

func TestHandleSchoolDamage_WithVariance(t *testing.T) {
	ctx := &Context{
		EffectInfo: &spell.SpellEffectInfo{
			EffectType: spell.EffectSchoolDamage,
			BasePoints: 678,
			BaseDieSides: 164,
			BonusCoeff: 1.0,
		},
		CasterSpellPower: 0,
		Crit:             false,
	}
	handleSchoolDamage(ctx)
	if ctx.BaseDamage < 678 || ctx.BaseDamage > 678+164 {
		t.Errorf("expected base damage between 678 and 842, got %.0f", ctx.BaseDamage)
	}
}

func TestHandleApplyAura_CreatesPeriodicDamageAura(t *testing.T) {
	ctx := &Context{
		Spell: &spell.Spell{
			Info: &spell.SpellInfo{
				ID:       25306,
				Duration: 8000,
			},
		},
		EffectInfo: &spell.SpellEffectInfo{
			EffectIndex: 1,
			EffectType:  spell.EffectApplyAura,
			BasePoints:  19,
			BonusCoeff:  0.125,
			AuraType:    uint16(aura.AuraPeriodicDamage),
			AuraPeriod:  2000,
		},
		CasterID: 1,
		TargetID: 2,
	}
	handleApplyAura(ctx)

	if ctx.AppliedAura == nil {
		t.Fatal("expected aura to be created")
	}
	a := ctx.AppliedAura
	if a.AuraType != aura.AuraPeriodicDamage {
		t.Errorf("expected AuraPeriodicDamage, got %v", a.AuraType)
	}
	if a.SpellID != 25306 {
		t.Errorf("expected spell ID 25306, got %d", a.SpellID)
	}
	if a.Duration != 8*time.Second {
		t.Errorf("expected duration 8s, got %v", a.Duration)
	}
	if a.StackRule != aura.StackRefresh {
		t.Errorf("expected StackRefresh, got %v", a.StackRule)
	}
	if len(a.Effects) != 1 {
		t.Fatalf("expected 1 aura effect, got %d", len(a.Effects))
	}
	eff := a.Effects[0]
	if eff.Amount != 19 {
		t.Errorf("expected amount 19, got %.0f", eff.Amount)
	}
	if eff.BonusCoeff != 0.125 {
		t.Errorf("expected coeff 0.125, got %f", eff.BonusCoeff)
	}
	if eff.Period != 2*time.Second {
		t.Errorf("expected period 2s, got %v", eff.Period)
	}
}

func TestHandleApplyAura_SkipsNoneAuraType(t *testing.T) {
	ctx := &Context{
		Spell: &spell.Spell{
			Info: &spell.SpellInfo{ID: 1},
		},
		EffectInfo: &spell.SpellEffectInfo{
			EffectType: spell.EffectApplyAura,
			AuraType:   0,
		},
	}
	handleApplyAura(ctx)
	if ctx.AppliedAura != nil {
		t.Error("expected no aura for AuraNone type")
	}
}
