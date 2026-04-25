package effect

import (
	"math/rand"
	"skill-go/pkg/aura"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"time"
)

type EffectHandler func(ctx *Context)

type Context struct {
	Spell            *spell.Spell
	EffectInfo       *spell.SpellEffectInfo
	CasterID         uint64
	TargetID         uint64
	Mode             spell.EffectHandleMode
	CasterSpellPower float64
	BaseDamage       float64
	FinalDamage      float64
	BaseHeal         float64
	FinalHeal        float64
	Crit             bool
	AppliedAura      *aura.Aura
}

var ScriptRegistry *script.Registry

var handlers = map[spell.EffectType]EffectHandler{
	spell.EffectSchoolDamage: handleSchoolDamage,
	spell.EffectHeal:         handleHeal,
	spell.EffectHealPct:      handleHealPct,
	spell.EffectApplyAura:    handleApplyAura,
	spell.EffectEnergize:     handleEnergize,
	spell.EffectEnergizePct:  handleEnergizePct,
	spell.EffectTriggerSpell: handleTriggerSpell,
	spell.EffectWeaponDamage: handleWeaponDamage,
	spell.EffectSummon:       handleSummon,
	spell.EffectDispel:       handleDispel,
	spell.EffectDummy:        handleDummy,
	spell.EffectTeleportUnits: handleTeleport,
	spell.EffectCharge:       handleCharge,
	spell.EffectKnockBack:    handleKnockBack,
	spell.EffectLeap:         handleLeap,
}

func Process(ctx *Context) {
	h, ok := handlers[ctx.EffectInfo.EffectType]
	if !ok {
		return
	}
	h(ctx)
}

func ProcessAll(s *spell.Spell, mode spell.EffectHandleMode) {
	if s.Info == nil {
		return
	}
	sp := s.Caster.GetStatValue(3) // stat.SpellPower = 3
	for i := range s.Info.Effects {
		ei := &s.Info.Effects[i]
		for j := range s.TargetInfos {
			ti := &s.TargetInfos[j]
			if ti.EfffectMask&(1<<ei.EffectIndex) == 0 {
				continue
			}
			ctx := &Context{
				Spell:            s,
				EffectInfo:       ei,
				CasterID:         s.Caster.GetID(),
				TargetID:         ti.TargetID,
				Mode:             mode,
				CasterSpellPower: sp,
				Crit:             ti.Crit,
			}
			if ScriptRegistry != nil && ScriptRegistry.HasSpellHook(s.ID, script.HookOnEffectHit) {
				spellCtx := &script.SpellContext{Spell: s, EffectIndex: ei.EffectIndex}
				ScriptRegistry.CallSpellHook(s.ID, script.HookOnEffectHit, spellCtx)
				if spellCtx.PreventDefault {
					continue
				}
			}
			Process(ctx)
			ti.Damage += ctx.FinalDamage
			ti.Healing += ctx.FinalHeal
			if ctx.AppliedAura != nil && s.OnAuraCreated != nil {
				s.OnAuraCreated(ctx.AppliedAura)
			}
		}
	}
}

func init() {
	spell.ProcessEffectsFn = ProcessAll
}

func handleSchoolDamage(ctx *Context) {
	ei := ctx.EffectInfo
	base := ei.BasePoints
	variance := 0.0
	if ei.BaseDieSides > 0 {
		variance = rand.Float64() * ei.BaseDieSides
	}
	scaling := ei.BonusCoeff * ctx.CasterSpellPower

	ctx.BaseDamage = base + variance
	ctx.FinalDamage = ctx.BaseDamage + scaling

	if ctx.Crit {
		ctx.FinalDamage *= 1.5
	}
}

func handleHeal(ctx *Context) {
	ctx.BaseHeal = ctx.EffectInfo.BasePoints
	ctx.FinalHeal = ctx.BaseHeal
}

func handleHealPct(ctx *Context) {
	ctx.BaseHeal = ctx.EffectInfo.BasePoints
	ctx.FinalHeal = ctx.BaseHeal
}

func handleApplyAura(ctx *Context) {
	ei := ctx.EffectInfo
	auraType := aura.AuraType(ei.AuraType)
	if auraType == aura.AuraNone {
		return
	}

	spellInfo := ctx.Spell.Info
	duration := time.Duration(spellInfo.Duration) * time.Millisecond
	if spellInfo.Duration == 0 {
		duration = 0
	}

	a := aura.NewAura(spellInfo.ID, ctx.CasterID, ctx.TargetID, auraType, duration)
	a.MaxStack = 1
	a.StackRule = aura.StackRefresh
	a.SpellName = spellInfo.Name

	// Copy SpellValues from spell to aura (used by Living Bomb etc.)
	if ctx.Spell.SpellValues != nil {
		a.SpellValues = ctx.Spell.SpellValues
	}

	if ei.AuraPeriod > 0 {
		a.Effects = append(a.Effects, aura.AuraEffect{
			EffectIndex:    ei.EffectIndex,
			AuraType:       auraType,
			Amount:         ei.BasePoints,
			BonusCoeff:     ei.BonusCoeff,
			Period:         time.Duration(ei.AuraPeriod) * time.Millisecond,
			TriggerSpellID: ei.TriggerSpellID,
		})
	}

	ctx.AppliedAura = a
}

func handleEnergize(ctx *Context) {
	ctx.BaseDamage = ctx.EffectInfo.BasePoints
}

func handleEnergizePct(ctx *Context) {
	ctx.BaseDamage = ctx.EffectInfo.BasePoints
}

func handleTriggerSpell(ctx *Context) {
	_ = ctx.EffectInfo.TriggerSpellID
}

func handleWeaponDamage(ctx *Context) {
	ctx.BaseDamage = ctx.EffectInfo.BasePoints
	ctx.FinalDamage = ctx.BaseDamage
}

func handleSummon(ctx *Context) {}

func handleDispel(ctx *Context) {}

func handleDummy(ctx *Context) {}

func handleTeleport(ctx *Context) {}

func handleCharge(ctx *Context) {}

func handleKnockBack(ctx *Context) {}

func handleLeap(ctx *Context) {}
