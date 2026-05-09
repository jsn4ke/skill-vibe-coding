package spellcore

import (
	"math"
	"math/rand"
	"time"
)

// EffectHandler 是效果处理函数的类型。
type EffectHandler func(ctx *EffectContext)

// EffectContext 是效果处理的上下文。
type EffectContext struct {
	Spell             *Spell
	EffectInfo        *SpellEffectInfo
	CasterID          uint64
	TargetID          uint64
	Mode              EffectHandleMode
	CasterSpellPower  float64
	BaseDamage        float64
	FinalDamage       float64
	BaseHeal          float64
	FinalHeal         float64
	Crit              bool
	AppliedAura       *Aura
	EnergizeAmount    float64
	EnergizePowerType uint8
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
			ti.Energize += ctx.EnergizeAmount
			if ctx.EnergizeAmount != 0 {
				ti.EnergizePowerType = ctx.EnergizePowerType
			}
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
			ti.Energize += ctx.EnergizeAmount
			if ctx.EnergizeAmount != 0 {
				ti.EnergizePowerType = ctx.EnergizePowerType
			}
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

// --- 效果处理函数 ---

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
// 包含法术强度缩放和暴击倍率，对齐 TC Spell::EffectHeal。
func handleHeal(ctx *EffectContext) {
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

	ctx.BaseHeal = base + variance
	ctx.FinalHeal = ctx.BaseHeal + scaling

	if ctx.Crit {
		ctx.FinalHeal *= 1.5
	}
}

// handleHealPct 处理百分比治疗效果，对齐 TC 的 LAUNCH_TARGET 阶段。
// BasePoints 为百分比数值（如 20 表示 20%），治疗量为目标最大生命值的百分比。
func handleHealPct(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

	pct := ctx.EffectInfo.BasePoints / 100.0
	maxHealth := 0.0
	if ctx.Spell.Engine != nil {
		maxHealth = ctx.Spell.Engine.GetUnitStatValue(ctx.TargetID, 1) // stat.MaxHealth = 1
	}

	if maxHealth > 0 {
		ctx.BaseHeal = maxHealth * pct
	} else {
		// 无最大生命值数据时回退到固定值
		ctx.BaseHeal = ctx.EffectInfo.BasePoints
	}
	ctx.FinalHeal = ctx.BaseHeal

	if ctx.Crit {
		ctx.FinalHeal *= 1.5
	}
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

	// 区域效果时光环目标为施法者（非 AoE 敌方）
	targetID := ctx.TargetID
	isArea := IsAreaTarget(ei.TargetA) || IsAreaTarget(ei.TargetB)
	if isArea {
		targetID = ctx.CasterID
	}

	a := NewAura(spellInfo.ID, ctx.CasterID, targetID, auraType, duration)
	a.MaxStack = 1
	a.StackRule = StackRefresh
	a.SpellName = spellInfo.Name

	// 将法术值复制到光环（Living Bomb 等技能使用）
	if ctx.Spell.SpellValues != nil {
		a.SpellValues = ctx.Spell.SpellValues
	}

	// 从效果复制光环打断标志
	a.InterruptFlags = ei.AuraInterruptFlags

	// 区域光环：设置 IsAreaAura、AreaCenter、AreaRadius
	if isArea {
		a.IsAreaAura = true
		destPos := ctx.Spell.Targets.DestPos
		if destPos != [3]float64{} {
			a.AreaCenter = destPos
		} else {
			// 回退：使用施法者位置
			pos := ctx.Spell.Caster.GetPosition()
			a.AreaCenter = [3]float64{pos.GetX(), pos.GetY(), pos.GetZ()}
		}
		a.AreaRadius = float64(ei.MiscValue)
	}

	// 所有光环都需要 AuraEffect，不仅限于周期性光环
	ae := AuraEffect{
		EffectIndex:    ei.EffectIndex,
		AuraType:       auraType,
		Amount:         ei.BasePoints,
		BonusCoeff:     ei.BonusCoeff,
		TriggerSpellID: ei.TriggerSpellID,
	}
	if ei.AuraPeriod > 0 {
		ae.Period = time.Duration(ei.AuraPeriod) * time.Millisecond
	}
	a.Effects = append(a.Effects, ae)

	ctx.AppliedAura = a
}

// handleEnergize 处理能量恢复效果，对齐 TC 的 HIT_TARGET 阶段。
// MiscValue 指定能量类型（0=法力，3=能量等），BasePoints 为恢复量。
func handleEnergize(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}

	ctx.EnergizeAmount = ctx.EffectInfo.BasePoints
	ctx.EnergizePowerType = uint8(ctx.EffectInfo.MiscValue)
}

// handleEnergizePct 处理百分比能量恢复效果，对齐 TC 的 HIT_TARGET 阶段。
// BasePoints 为百分比值，MiscValue 指定能量类型。
func handleEnergizePct(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}

	ctx.EnergizeAmount = ctx.EffectInfo.BasePoints
	ctx.EnergizePowerType = uint8(ctx.EffectInfo.MiscValue)
}

// handleTriggerSpell 处理触发法术效果，对齐 TC 的 LAUNCH_TARGET/LAUNCH 阶段。
// 从 SpellStore 查找 TriggerSpellID 对应的法术并通过引擎触发施放，对齐 TC EffectTriggerSpell。
func handleTriggerSpell(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget && ctx.Mode != HandleLaunch {
		return
	}

	triggerSpellID := ctx.EffectInfo.TriggerSpellID
	if triggerSpellID == 0 {
		return
	}
	if ctx.Spell.Engine == nil {
		return
	}

	ctx.Spell.Engine.TriggerSpell(ctx.CasterID, ctx.TargetID, triggerSpellID)
}

// handleWeaponDamage 处理武器伤害效果，对齐 TC 的 LAUNCH_TARGET 阶段。
// 使用 BasePoints + 随机方差作为基础伤害，BonusCoeff 缩放攻击强度。
func handleWeaponDamage(ctx *EffectContext) {
	if ctx.Mode != HandleLaunchTarget {
		return
	}

	ei := ctx.EffectInfo
	base := ei.BasePoints
	variance := 0.0
	if ei.BaseDieSides > 0 {
		variance = rand.Float64() * ei.BaseDieSides
	}
	// BonusCoeff 缩放攻击强度，对齐 TC 的 AP normalization
	apScaling := ei.BonusCoeff * ctx.Spell.Caster.GetStatValue(uint8(3)) // stat.AttackPower = 3

	ctx.BaseDamage = base + variance
	ctx.FinalDamage = ctx.BaseDamage + apScaling

	if ctx.Crit {
		ctx.FinalDamage *= 1.5
	}
}

// handleSummon 处理召唤效果，对齐 TC 的 LAUNCH 阶段。
// MiscValue 为生物 Entry ID，在施法者位置或 DestPos 创建新单位。
func handleSummon(ctx *EffectContext) {
	if ctx.Mode != HandleLaunch {
		return
	}
	if ctx.Spell.Engine == nil {
		return
	}

	// 确定召唤位置：优先使用 DestPos，否则使用施法者位置
	pos := ctx.Spell.Targets.DestPos
	if pos == [3]float64{} {
		casterPos := ctx.Spell.Caster.GetPosition()
		pos = [3]float64{casterPos.GetX(), casterPos.GetY(), casterPos.GetZ()}
	}

	ctx.Spell.Engine.SummonUnit(ctx.CasterID, ctx.EffectInfo.MiscValue, pos)
}

// handleDispel 处理驱散效果，对齐 TC 的 HIT_TARGET 阶段。
// MiscValue 为最大驱散数量，从目标身上移除光环。
func handleDispel(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
	if ctx.Spell.Engine == nil {
		return
	}

	count := ctx.EffectInfo.MiscValue
	if count <= 0 {
		count = 1
	}
	ctx.Spell.Engine.DispelAuras(ctx.TargetID, count)
}

// handleDummy 处理 Dummy 效果（钩子挂载点），对齐 TC 的 HIT_TARGET 阶段。
// 无默认行为，由脚本钩子（HookOnEffectHit 等）提供自定义逻辑。
func handleDummy(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
}

// handleTeleport 处理传送效果，对齐 TC 的 HIT_TARGET 阶段。
// 将目标传送到法术的 DestPos 指定位置。
func handleTeleport(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
	if ctx.Spell.Engine == nil {
		return
	}

	destPos := ctx.Spell.Targets.DestPos
	if destPos != [3]float64{} {
		ctx.Spell.Engine.SetUnitPosition(ctx.TargetID, destPos[0], destPos[1], destPos[2])
	}
}

// handleCharge 处理冲锋效果，对齐 TC 的 LAUNCH_TARGET + HIT_TARGET 双阶段。
// LAUNCH_TARGET 阶段将施法者移动到目标附近，HIT_TARGET 阶段由其他 Effect 处理后续效果。
func handleCharge(ctx *EffectContext) {
	if ctx.Mode == HandleLaunchTarget {
		if ctx.Spell.Engine == nil {
			return
		}
		// 将施法者传送到目标位置
		targetPos := ctx.Spell.Caster.GetTargetPosition(ctx.TargetID)
		ctx.Spell.Engine.SetUnitPosition(ctx.CasterID, targetPos.GetX(), targetPos.GetY(), targetPos.GetZ())
		return
	}
	if ctx.Mode == HandleHitTarget {
		// 到达后效果由同一法术的其他 Effect 处理
		return
	}
}

// handleKnockBack 处理击退效果，对齐 TC 的 HIT_TARGET 阶段。
// MiscValue 为击退距离，方向为从施法者指向目标的延伸方向。
func handleKnockBack(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
	if ctx.Spell.Engine == nil {
		return
	}

	// 计算击退方向（施法者→目标方向延伸）
	casterPos := ctx.Spell.Caster.GetPosition()
	targetPos := ctx.Spell.Caster.GetTargetPosition(ctx.TargetID)
	dx := targetPos.GetX() - casterPos.GetX()
	dy := targetPos.GetY() - casterPos.GetY()
	dz := targetPos.GetZ() - casterPos.GetZ()
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if dist == 0 {
		return
	}

	distance := float64(ctx.EffectInfo.MiscValue)
	if distance <= 0 {
		distance = 10.0 // 默认击退距离
	}
	newX := targetPos.GetX() + dx/dist*distance
	newY := targetPos.GetY() + dy/dist*distance
	newZ := targetPos.GetZ() + dz/dist*distance

	ctx.Spell.Engine.SetUnitPosition(ctx.TargetID, newX, newY, newZ)
}

// handleLeap 处理跳跃效果，对齐 TC 的 HIT_TARGET 阶段。
// 将施法者移动到 DestPos 指定位置。
func handleLeap(ctx *EffectContext) {
	if ctx.Mode != HandleHitTarget {
		return
	}
	if ctx.Spell.Engine == nil {
		return
	}

	destPos := ctx.Spell.Targets.DestPos
	if destPos == [3]float64{} {
		// 回退：使用目标位置
		targetPos := ctx.Spell.Caster.GetTargetPosition(ctx.TargetID)
		destPos = [3]float64{targetPos.GetX(), targetPos.GetY(), targetPos.GetZ()}
	}

	ctx.Spell.Engine.SetUnitPosition(ctx.CasterID, destPos[0], destPos[1], destPos[2])
}
