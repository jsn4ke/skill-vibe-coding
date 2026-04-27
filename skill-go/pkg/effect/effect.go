package effect

import (
	"math/rand"
	"skill-go/pkg/aura"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"time"
)

// EffectHandler 是效果处理函数的类型。
type EffectHandler func(ctx *Context)

// Context 是效果处理的上下文。
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

// ScriptRegistry 是全局脚本注册中心引用，由引擎初始化时设置。
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

// Process 按效果类型分发处理。
func Process(ctx *Context) {
	h, ok := handlers[ctx.EffectInfo.EffectType]
	if !ok {
		return
	}
	h(ctx)
}

// ProcessAll 处理法术的所有效果，包括脚本钩子拦截和光环自动注册。
func ProcessAll(s *spell.Spell, mode spell.EffectHandleMode) {
	if s.Info == nil {
		return
	}
	sp := s.Caster.GetStatValue(3) // stat.SpellPower = 3
	casterID := s.Caster.GetID()
	for i := range s.Info.Effects {
		ei := &s.Info.Effects[i]

		// For area effects with no TargetInfos, process once with caster as target
		// so handleApplyAura can create the area aura.
		if len(s.TargetInfos) == 0 && (spell.IsAreaTarget(ei.TargetA) || spell.IsAreaTarget(ei.TargetB)) {
			ctx := &Context{
				Spell:            s,
				EffectInfo:       ei,
				CasterID:         casterID,
				TargetID:         casterID,
				Mode:             mode,
				CasterSpellPower: sp,
			}
			if ScriptRegistry != nil && ScriptRegistry.HasSpellHook(s.ID, script.HookOnEffectHit) {
				spellCtx := &script.SpellContext{Spell: s, EffectIndex: ei.EffectIndex}
				ScriptRegistry.CallSpellHook(s.ID, script.HookOnEffectHit, spellCtx)
				if spellCtx.PreventDefault {
					continue
				}
			}
			Process(ctx)
			if ctx.AppliedAura != nil && s.OnAuraCreated != nil {
				s.OnAuraCreated(ctx.AppliedAura)
			}
			continue
		}

		for j := range s.TargetInfos {
			ti := &s.TargetInfos[j]
			if ti.EfffectMask&(1<<ei.EffectIndex) == 0 {
				continue
			}
			ctx := &Context{
				Spell:            s,
				EffectInfo:       ei,
				CasterID:         casterID,
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

// init 注册效果管线到 spell 包。
func init() {
	spell.ProcessEffectsFn = ProcessAll
}

// handleSchoolDamage 处理法术伤害效果。
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

// handleHeal 处理治疗效果。
func handleHeal(ctx *Context) {
	ctx.BaseHeal = ctx.EffectInfo.BasePoints
	ctx.FinalHeal = ctx.BaseHeal
}

// handleHealPct 处理百分比治疗效果。
func handleHealPct(ctx *Context) {
	ctx.BaseHeal = ctx.EffectInfo.BasePoints
	ctx.FinalHeal = ctx.BaseHeal
}

// handleApplyAura 处理光环应用效果，创建光环实例并通过 OnAuraCreated 回调注册。
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

	// For area effects, the aura target is the caster (not AoE enemies)
	targetID := ctx.TargetID
	isArea := spell.IsAreaTarget(ei.TargetA) || spell.IsAreaTarget(ei.TargetB)
	if isArea {
		targetID = ctx.CasterID
	}

	a := aura.NewAura(spellInfo.ID, ctx.CasterID, targetID, auraType, duration)
	a.MaxStack = 1
	a.StackRule = aura.StackRefresh
	a.SpellName = spellInfo.Name

	// Copy SpellValues from spell to aura (used by Living Bomb etc.)
	if ctx.Spell.SpellValues != nil {
		a.SpellValues = ctx.Spell.SpellValues
	}

	// Copy AuraInterruptFlags from effect to aura
	a.InterruptFlags = ei.AuraInterruptFlags

	// Area aura: set IsAreaAura, AreaCenter, AreaRadius
	if isArea {
		a.IsAreaAura = true
		destPos := ctx.Spell.Targets.DestPos
		if destPos != [3]float64{} {
			a.AreaCenter = destPos
		} else {
			// Fallback: use caster position
			pos := ctx.Spell.Caster.GetPosition()
			a.AreaCenter = [3]float64{pos.GetX(), pos.GetY(), pos.GetZ()}
		}
		a.AreaRadius = float64(ei.MiscValue)
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

// handleEnergize 处理能量恢复效果。
func handleEnergize(ctx *Context) {
	ctx.BaseDamage = ctx.EffectInfo.BasePoints
}

// handleEnergizePct 处理百分比能量恢复效果。
func handleEnergizePct(ctx *Context) {
	ctx.BaseDamage = ctx.EffectInfo.BasePoints
}

// handleTriggerSpell 处理触发法术效果。
func handleTriggerSpell(ctx *Context) {
	_ = ctx.EffectInfo.TriggerSpellID
}

// handleWeaponDamage 处理武器伤害效果。
func handleWeaponDamage(ctx *Context) {
	ctx.BaseDamage = ctx.EffectInfo.BasePoints
	ctx.FinalDamage = ctx.BaseDamage
}

// handleSummon 处理召唤效果（占位）。
func handleSummon(ctx *Context) {}

// handleDispel 处理驱散效果（占位）。
func handleDispel(ctx *Context) {}

// handleDummy 处理 Dummy 效果（钩子挂载点，无内置行为）。
func handleDummy(ctx *Context) {}

// handleTeleport 处理传送效果（占位）。
func handleTeleport(ctx *Context) {}

// handleCharge 处理冲锋效果（占位）。
func handleCharge(ctx *Context) {}

// handleKnockBack 处理击退效果（占位）。
func handleKnockBack(ctx *Context) {}

// handleLeap 处理跳跃效果（占位）。
func handleLeap(ctx *Context) {}
