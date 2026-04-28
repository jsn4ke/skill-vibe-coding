package spellcore

import (
	"math/rand"
	"time"
)

// EffectHandler 是效果处理函数的类型。
type EffectHandler func(ctx *EffectContext)

// EffectContext 是效果处理的上下文。
type EffectContext struct {
	Spell            *Spell
	EffectInfo       *SpellEffectInfo
	CasterID         uint64
	TargetID         uint64
	Mode             EffectHandleMode
	CasterSpellPower float64
	BaseDamage       float64
	FinalDamage      float64
	BaseHeal         float64
	FinalHeal        float64
	Crit             bool
	AppliedAura      *Aura
}

// scriptRegistry 是全局脚本注册中心引用，由引擎初始化时设置。
var scriptRegistry *Registry

// SetScriptRegistry 设置全局脚本注册中心，由引擎初始化时调用。
func SetScriptRegistry(reg *Registry) {
	scriptRegistry = reg
}

var handlers = map[EffectType]EffectHandler{
	EffectSchoolDamage:  handleSchoolDamage,
	EffectHeal:          handleHeal,
	EffectHealPct:       handleHealPct,
	EffectApplyAura:     handleApplyAura,
	EffectEnergize:      handleEnergize,
	EffectEnergizePct:   handleEnergizePct,
	EffectTriggerSpell:  handleTriggerSpell,
	EffectWeaponDamage:  handleWeaponDamage,
	EffectSummon:        handleSummon,
	EffectDispel:        handleDispel,
	EffectDummy:         handleDummy,
	EffectTeleportUnits: handleTeleport,
	EffectCharge:        handleCharge,
	EffectKnockBack:     handleKnockBack,
	EffectLeap:          handleLeap,
}

// Process 按效果类型分发处理。
func Process(ctx *EffectContext) {
	h, ok := handlers[ctx.EffectInfo.EffectType]
	if !ok {
		return
	}
	h(ctx)
}

// ProcessLaunchPhase 处理法术的 Launch 阶段（Launch + LaunchTarget），在 Cast() 中调用。
// 对齐 TC HandleLaunchPhase()。
func ProcessLaunchPhase(s *Spell) {
	if s.Info == nil {
		return
	}
	sp := s.Caster.GetStatValue(uint8(4)) // stat.SpellPower = 4
	casterID := s.Caster.GetID()

	// 阶段 1: Launch（无目标），对齐 TC HandleLaunchPhase 的 LAUNCH 部分
	for i := range s.Info.Effects {
		ei := &s.Info.Effects[i]
		ctx := &EffectContext{
			Spell:            s,
			EffectInfo:       ei,
			CasterID:         casterID,
			TargetID:         casterID,
			Mode:             HandleLaunch,
			CasterSpellPower: sp,
		}
		processWithScript(ctx, HookOnEffectLaunch)
	}

	// 阶段 2: LaunchTarget（每目标），对齐 TC DoEffectOnLaunchTarget
	for i := range s.Info.Effects {
		ei := &s.Info.Effects[i]
		for j := range s.TargetInfos {
			ti := &s.TargetInfos[j]
			if ti.EfffectMask&(1<<ei.EffectIndex) == 0 {
				continue
			}
			ctx := &EffectContext{
				Spell:            s,
				EffectInfo:       ei,
				CasterID:         casterID,
				TargetID:         ti.TargetID,
				Mode:             HandleLaunchTarget,
				CasterSpellPower: sp,
				Crit:             ti.Crit,
			}
			processWithScript(ctx, HookOnEffectLaunchTarget)
			ti.Damage += ctx.FinalDamage
			ti.Healing += ctx.FinalHeal
		}
	}
}

// ProcessHitPhase 处理法术的 Hit 阶段（Hit + HitTarget），在 HandleImmediate() / 弹道命中时调用。
// 对齐 TC _handle_immediate_phase() + DoProcessTargetContainer()。
func ProcessHitPhase(s *Spell) {
	if s.Info == nil {
		return
	}
	sp := s.Caster.GetStatValue(uint8(4)) // stat.SpellPower = 4
	casterID := s.Caster.GetID()

	// 阶段 3: Hit（无目标），对齐 TC _handle_immediate_phase 的 HIT 部分
	for i := range s.Info.Effects {
		ei := &s.Info.Effects[i]
		ctx := &EffectContext{
			Spell:            s,
			EffectInfo:       ei,
			CasterID:         casterID,
			TargetID:         casterID,
			Mode:             HandleHit,
			CasterSpellPower: sp,
		}
		processWithScript(ctx, HookOnEffectHit)
	}

	// 阶段 4: HitTarget（每目标），对齐 TC DoTargetSpellHit 的 HIT_TARGET 部分
	for i := range s.Info.Effects {
		ei := &s.Info.Effects[i]
		for j := range s.TargetInfos {
			ti := &s.TargetInfos[j]
			if ti.EfffectMask&(1<<ei.EffectIndex) == 0 {
				continue
			}
			ctx := &EffectContext{
				Spell:            s,
				EffectInfo:       ei,
				CasterID:         casterID,
				TargetID:         ti.TargetID,
				Mode:             HandleHitTarget,
				CasterSpellPower: sp,
				Crit:             ti.Crit,
			}
			processWithScript(ctx, HookOnEffectHitTarget)
			ti.Damage += ctx.FinalDamage
			ti.Healing += ctx.FinalHeal
			if ctx.AppliedAura != nil && s.OnAuraCreated != nil {
				s.OnAuraCreated(ctx.AppliedAura)
			}
		}
	}
}

// processWithScript 调用脚本钩子后执行默认处理，对齐 TC HandleEffects 模板。
func processWithScript(ctx *EffectContext, hook Hook) {
	if scriptRegistry != nil && scriptRegistry.HasSpellHook(ctx.Spell.ID, hook) {
		spellCtx := &SpellContext{Spell: ctx.Spell, EffectIndex: ctx.EffectInfo.EffectIndex}
		scriptRegistry.CallSpellHook(ctx.Spell.ID, hook, spellCtx)
		if spellCtx.PreventDefault {
			return
		}
	}
	Process(ctx)
}

// handleSchoolDamage 处理法术伤害效果，对齐 TC 的 LAUNCH_TARGET 阶段。
func handleSchoolDamage(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

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

// handleHeal 处理治疗效果，对齐 TC 的 LAUNCH_TARGET 阶段。
func handleHeal(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

	ctx.BaseHeal = ctx.EffectInfo.BasePoints
	ctx.FinalHeal = ctx.BaseHeal
}

// handleHealPct 处理百分比治疗效果，对齐 TC 的 LAUNCH_TARGET 阶段。
func handleHealPct(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

	ctx.BaseHeal = ctx.EffectInfo.BasePoints
	ctx.FinalHeal = ctx.BaseHeal
}

// handleApplyAura 处理光环应用效果，对齐 TC 的 HIT_TARGET 阶段。
func handleApplyAura(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}

	ei := ctx.EffectInfo
	auraType := AuraType(ei.AuraType)
	if auraType == AuraNone {
		return
	}

	spellInfo := ctx.Spell.Info
	duration := time.Duration(spellInfo.Duration) * time.Millisecond
	if spellInfo.Duration == 0 {
		duration = 0
	}

	// For area effects, the aura target is the caster (not AoE enemies)
	targetID := ctx.TargetID
	isArea := IsAreaTarget(ei.TargetA) || IsAreaTarget(ei.TargetB)
	if isArea {
		targetID = ctx.CasterID
	}

	a := NewAura(spellInfo.ID, ctx.CasterID, targetID, auraType, duration)
	a.MaxStack = 1
	a.StackRule = StackRefresh
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
		a.Effects = append(a.Effects, AuraEffect{
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

// handleEnergize 处理能量恢复效果，对齐 TC 的 HIT_TARGET 阶段。
func handleEnergize(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}

	ctx.BaseDamage = ctx.EffectInfo.BasePoints
}

// handleEnergizePct 处理百分比能量恢复效果，对齐 TC 的 HIT_TARGET 阶段。
func handleEnergizePct(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}

	ctx.BaseDamage = ctx.EffectInfo.BasePoints
}

// handleTriggerSpell 处理触发法术效果，对齐 TC 的 LAUNCH_TARGET 阶段。
func handleTriggerSpell(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

	_ = ctx.EffectInfo.TriggerSpellID
}

// handleWeaponDamage 处理武器伤害效果，对齐 TC 的 LAUNCH_TARGET 阶段。
func handleWeaponDamage(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

	ctx.BaseDamage = ctx.EffectInfo.BasePoints
	ctx.FinalDamage = ctx.BaseDamage
}

// handleSummon 处理召唤效果，对齐 TC 的 LAUNCH 阶段。
func handleSummon(ctx *EffectContext) {
	if ctx.Mode != HandleLaunch {
		return
	}
}

// handleDispel 处理驱散效果，对齐 TC 的 HIT_TARGET 阶段。
func handleDispel(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
}

// handleDummy 处理 Dummy 效果（钩子挂载点），对齐 TC 的 HIT_TARGET 阶段。
func handleDummy(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
}

// handleTeleport 处理传送效果，对齐 TC 的 HIT_TARGET 阶段。
func handleTeleport(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
}

// handleCharge 处理冲锋效果，对齐 TC 的 LAUNCH_TARGET + HIT_TARGET 双阶段。
func handleCharge(ctx *EffectContext) {
	if ctx.Mode == HandleLaunchTarget {
		// 发射阶段：启动冲锋移动（占位）
		return
	}
	if ctx.Mode == HandleHitTarget {
		// 命中阶段：到达后发起攻击（占位）
		return
	}
}

// handleKnockBack 处理击退效果，对齐 TC 的 HIT_TARGET 阶段。
func handleKnockBack(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
}

// handleLeap 处理跳跃效果，对齐 TC 的 HIT_TARGET 阶段。
func handleLeap(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
}
