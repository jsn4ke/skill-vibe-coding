package spellcore

import (
	"math"
	"skill-go/pkg/event"
	"skill-go/pkg/targeting"
	"time"
)

// SpellID 是法术的唯一标识符
type SpellID uint32

// SpellState 表示法术的状态
type SpellState uint8

const (
	StateNull       SpellState = iota // 初始状态
	StatePreparing                    // 施法准备（读条中）
	StateLaunched                     // 已发射（弹道飞行中）
	StateChanneling                   // 引导中
	StateFinished                     // 已完成
)

// SpellCastResult 表示施法结果的枚举
type SpellCastResult uint16

const (
	CastOK                   SpellCastResult = iota // 施法成功
	CastFailedCasterDead                            // 施法者已死亡
	CastFailedNotReady                              // 冷却未就绪
	CastFailedNoPower                               // 能量不足
	CastFailedOutOfRange                            // 超出范围
	CastFailedInterrupted                           // 被打断
	CastFailedStunned                               // 昏迷中无法施法
	CastFailedSilenced                              // 沉默中无法施法
	CastFailedRooted                                // 定身中无法施法
	CastFailedTargetDead                            // 目标已死亡
	CastFailedTargetInvalid                         // 目标无效
	CastFailedAuraBounced                           // 光环被弹回
	CastFailedCantDoRightNow                        // 当前无法执行
)

// EffectHandleMode 表示效果处理的阶段
type EffectHandleMode uint8

const (
	HandleLaunch EffectHandleMode = iota
	HandleLaunchTarget
	HandleHit
	HandleHitTarget
)

// CastFlags 是施法标志的位掩码
type CastFlags uint32

const (
	TriggeredNone      CastFlags = 0
	TriggeredIgnoreGCD CastFlags = 1 << iota
	TriggeredIgnoreCooldown
	TriggeredIgnorePower
	TriggeredIgnoreCastItems
	TriggeredIgnoreCasterAuraState
	TriggeredIgnoreCastInProgress
	TriggeredFullMask CastFlags = TriggeredIgnoreGCD |
		TriggeredIgnoreCooldown | TriggeredIgnorePower |
		TriggeredIgnoreCastItems | TriggeredIgnoreCasterAuraState |
		TriggeredIgnoreCastInProgress
)

// AuraCreatedFunc 在效果管线创建光环时调用。
type AuraCreatedFunc func(a *Aura)

// TargetInfo 包含法术目标的命中信息
type TargetInfo struct {
	TargetID          uint64
	MissReason        HitResult
	EfffectMask       uint8
	Damage            float64
	Healing           float64
	Crit              bool
	TimeDelay         int32 // 命中延迟 (ms), 对齐 TC 的 TargetInfo::TimeDelay
	Energize          float64
	EnergizePowerType uint8
}

// Spell 表示一个法术实例，包含完整的施法状态和数据
type Spell struct {
	ID                    SpellID
	Info                  *SpellInfo
	State                 SpellState
	Caster                Caster
	CastFlags             CastFlags
	Targets               TargetData
	CastTime              uint32
	Timer                 int32
	Result                SpellCastResult
	TargetInfos           []TargetInfo
	Bus                   *event.Bus
	OnAuraCreated         AuraCreatedFunc
	DoDamageAndTriggersFn func(s *Spell) // 伤害结算函数，由 Engine 在 CastSpell 时设置
	SpellValues           map[uint8]float64
	Engine                SpellEngineRef
	launchHandled         bool // LaunchPhase 是否已执行，对齐 TC 的 m_launchHandled
}

// Caster 是施法者接口，对齐 TC 的 Unit 施法相关方法
type Caster interface {
	GetID() uint64
	IsAlive() bool
	CanCast() bool
	GetPosition() Position
	GetTargetPosition(targetID uint64) Position
	GetStatValue(st uint8) float64
	ModifyPower(pt uint8, amount float64) bool
	IsMoving() bool
	GetHistory() *History
}

// Position 是位置接口
type Position interface {
	GetX() float64
	GetY() float64
	GetZ() float64
	GetFacing() float64
}

// SpellEngineRef 提供法术对引擎的访问，用于打断检查、脚本钩子和目标选择。由 engine.Engine 实现以避免循环导入
type SpellEngineRef interface {
	GetCasterUnit(id uint64) CasterUnit
	AuraRemover() AuraRemover
	CallLaunchHook(spellID SpellID, s *Spell)
	CallCancelHook(spellID SpellID, s *Spell)
	RemoveOwnedAurasBySpellID(casterID uint64, spellID SpellID)
	// 目标选择所需的引擎方法
	GetTargetUnit(id uint64) targeting.TargetUnit
	GetTargetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []targeting.TargetUnit
	// 调用目标选择脚本钩子
	CallTargetSelectHook(spellID SpellID, s *Spell, units []targeting.TargetUnit)
	// 触发法术效果，用于 EffectTriggerSpell，对齐 TC 的 EffectTriggerSpell
	TriggerSpell(casterID, targetID uint64, spellID SpellID)
	// 召唤新单位，用于 EffectSummon
	SummonUnit(casterID uint64, entry int32, pos [3]float64) uint64
	// 设置单位位置，用于移动效果（Teleport/Leap/Charge/KnockBack）
	SetUnitPosition(id uint64, x, y, z float64)
	// 获取单位属性值，用于 HealPct
	GetUnitStatValue(id uint64, statType uint8) float64
	// 驱散光环，用于 EffectDispel
	DispelAuras(targetID uint64, count int32) int
	// 脚本注册中心，用于法术生命周期钩子
	ScriptRegistry() *Registry
}

// CasterUnit 是法术需要的最小目标单位接口
type CasterUnit interface {
	IsAlive() bool
}

// AuraRemover 按 SpellID 和目标 ID 移除光环
type AuraRemover interface {
	RemoveAuraFromChannel(ownerID uint64, targetID uint64, spellID SpellID)
}

// TargetData 包含法术的目标数据
type TargetData struct {
	UnitTargetID uint64
	SourcePos    [3]float64
	DestPos      [3]float64
	TargetMask   uint32
}

// NewSpell 创建一个新的法术实例
func NewSpell(id SpellID, info *SpellInfo, caster Caster, flags CastFlags) *Spell {
	return &Spell{
		ID:        id,
		Info:      info,
		Caster:    caster,
		CastFlags: flags,
		State:     StateNull,
	}
}

// callHook 调用指定类型的脚本钩子。
func (s *Spell) callHook(hook Hook) {
	if s.Engine != nil {
		if reg := s.Engine.ScriptRegistry(); reg != nil {
			reg.CallSpellHook(s.ID, hook, &SpellContext{Spell: s})
		}
	}
}

// Prepare 准备施法，验证条件并设置状态
func (s *Spell) Prepare() SpellCastResult {
	if result := s.CheckCast(true); result != CastOK {
		return result
	}

	s.callHook(HookOnPrepare)

	s.CastTime = s.Info.CastTime
	s.Timer = int32(s.CastTime)
	s.State = StatePreparing

	if s.Bus != nil {
		s.Bus.Publish(event.Event{
			Type:     event.OnSpellCastStart,
			SourceID: s.Caster.GetID(),
			SpellID:  uint32(s.ID),
			Extra:    map[string]any{"castTime": s.CastTime, "spellName": s.Info.Name},
		})
	}

	if s.CastTime == 0 {
		s.Cast(true)
	}
	return CastOK
}

// MaxRangeTolerance 是法术范围检查的最大容差，对齐 TC 的 MAX_SPELL_RANGE_TOLERANCE
const MaxRangeTolerance = 3.0

// Update 驱动法术的状态机，包含打断检查和状态转换
func (s *Spell) Update(diffMs int32) {
	if s.State == StateFinished || s.State == StateNull {
		return
	}

	// --- 打断检查（对齐 TC Spell::update） ---

	// 1. 施法者死亡
	if !s.Caster.IsAlive() {
		s.Cancel()
		return
	}

	// 2. 目标消失
	if s.Targets.UnitTargetID != 0 && s.Engine != nil {
		if s.Engine.GetCasterUnit(s.Targets.UnitTargetID) == nil {
			s.Cancel()
			return
		}
	}

	// 3. 移动打断
	// 触发的法术免疫移动打断
	if s.CastFlags&TriggeredFullMask == 0 && s.Caster.IsMoving() {
		shouldInterrupt := false
		if s.State == StatePreparing {
			shouldInterrupt = s.Info.InterruptFlags.HasFlag(InterruptMovement)
		} else if s.State == StateChanneling {
			shouldInterrupt = s.Info.InterruptFlags.HasFlag(InterruptMovement)
		}
		if shouldInterrupt {
			s.Cancel()
			return
		}
	}

	// 4. PREPARING 状态下的范围检查
	if s.State == StatePreparing && s.Targets.UnitTargetID != 0 && s.Info.RangeMax > 0 {
		casterPos := s.Caster.GetPosition()
		targetPos := s.Caster.GetTargetPosition(s.Targets.UnitTargetID)
		dist := positionDistance(casterPos, targetPos)
		tolerance := math.Min(MaxRangeTolerance, s.Info.RangeMax*0.1)
		if dist > s.Info.RangeMax+tolerance {
			s.Cancel()
			return
		}
	}

	// --- 状态机 ---
	switch s.State {
	case StatePreparing:
		if s.Timer > 0 {
			s.Timer -= diffMs
			if s.Timer <= 0 {
				s.Timer = 0
				s.Cast(false)
			}
		}
	case StateChanneling:
		// 每 tick 验证引导目标
		s.validateChannelTargets()

		if s.Timer > 0 {
			s.Timer -= diffMs
			if s.Timer <= 0 {
				s.Timer = 0
				s.Finish(CastOK)
			}
		}
	case StateLaunched:
		// 对齐 TC handle_delayed(): Timer 正计累计经过时间 (ms)
		s.Timer += diffMs

		// Phase 1: LaunchPhase 延迟执行，对齐 TC 的 m_launchHandled 逻辑
		if !s.launchHandled {
			if s.Info.LaunchDelay == 0 || s.Timer >= int32(s.Info.LaunchDelay) {
				s.HandleLaunchPhase()
				s.launchHandled = true
			}
			return
		}

		// Phase 2: 等待所有目标的 TimeDelay 到期
		if s.Timer >= s.maxTargetDelay() {
			s.HandleImmediate()
		}
	}
}

// validateChannelTargets 验证引导法术的每个目标是否仍然合法（存在、存活、范围内），对齐 TC 的 Spell::UpdateChanneledTargetList
// 目标在以下情况下被移除：单位消失、死亡或超出范围。
// 如果所有目标都被移除，则取消法术。
func (s *Spell) validateChannelTargets() {
	if s.Engine == nil || len(s.TargetInfos) == 0 {
		return
	}

	tolerance := math.Min(MaxRangeTolerance, s.Info.RangeMax*0.1)
	var remaining []TargetInfo

	for _, ti := range s.TargetInfos {
		// 检查目标是否存在
		targetUnit := s.Engine.GetCasterUnit(ti.TargetID)
		if targetUnit == nil {
			s.removeChannelAura(ti.TargetID)
			continue
		}

		// 检查目标是否存活
		if !targetUnit.IsAlive() {
			s.removeChannelAura(ti.TargetID)
			continue
		}

		// 检查范围（仅当法术有范围限制时）
		if s.Info.RangeMax > 0 {
			casterPos := s.Caster.GetPosition()
			targetPos := s.Caster.GetTargetPosition(ti.TargetID)
			dist := positionDistance(casterPos, targetPos)
			if dist > s.Info.RangeMax+tolerance {
				s.removeChannelAura(ti.TargetID)
				continue
			}
		}

		remaining = append(remaining, ti)
	}

	s.TargetInfos = remaining

	// 所有目标消失 — 取消引导 (TC: "Channeled spell removed due to lack of targets")
	if len(s.TargetInfos) == 0 {
		s.Cancel()
	}
}

// removeChannelAura 当目标离开范围、死亡或消失时移除其光环
func (s *Spell) removeChannelAura(targetID uint64) {
	if s.Engine == nil {
		return
	}
	remover := s.Engine.AuraRemover()
	if remover != nil {
		remover.RemoveAuraFromChannel(s.Caster.GetID(), targetID, s.ID)
	}
}

func (s *Spell) Cast(skipCheck bool) {
	if !skipCheck {
		if result := s.CheckCast(false); result != CastOK {
			s.Finish(result)
			return
		}
	}

	s.callHook(HookBeforeCast)

	s.SelectEffectTargets()

	if s.State == StateFinished {
		return
	}

	s.callHook(HookOnCast)

	if s.State == StateFinished {
		return
	}

	if !(s.CastFlags&TriggeredIgnorePower != 0) {
		s.TakePower()
	}

	// 施放后应用冷却和 GCD，对齐 TC 的 Spell::cast → AddCooldown + AddGlobalCooldown
	if s.CastFlags&TriggeredIgnoreCooldown == 0 {
		s.ApplyCooldown()
	}
	if s.CastFlags&TriggeredIgnoreGCD == 0 {
		s.ApplyGlobalCooldown()
	}

	// 对齐 TC: LaunchDelay == 0 时立即执行 LaunchPhase，否则推迟到 Update() 中
	if s.Info.LaunchDelay == 0 {
		s.HandleLaunchPhase()
		s.launchHandled = true
	}

	s.State = StateLaunched

	if s.Bus != nil {
		targetID := uint64(0)
		if len(s.TargetInfos) > 0 {
			targetID = s.TargetInfos[0].TargetID
		}
		launchExtra := map[string]any{"speed": s.Info.Speed, "spellName": s.Info.Name}
		if s.Targets.DestPos != [3]float64{} {
			launchExtra["destX"] = s.Targets.DestPos[0]
			launchExtra["destY"] = s.Targets.DestPos[1]
			launchExtra["destZ"] = s.Targets.DestPos[2]
		}
		s.Bus.Publish(event.Event{
			Type:     event.OnSpellLaunch,
			SourceID: s.Caster.GetID(),
			TargetID: targetID,
			SpellID:  uint32(s.ID),
			Extra:    launchExtra,
		})
	}

	if s.Engine != nil {
		s.Engine.CallLaunchHook(s.ID, s)
	}

	if s.Info.IsChanneled {
		// 对齐 TC: 引导法术在进入引导状态前处理 Hit 阶段
		s.HandleHitPhase()
		s.DoDamageAndTriggers()
		s.State = StateChanneling
		s.Timer = int32(s.Info.Duration)
	} else if s.Info.HasHitDelay() {
		// 对齐 TC: 有命中延迟时计算每个目标的 TimeDelay, Timer 从 0 开始正计
		s.calculateTargetDelays()
		s.Timer = 0
	} else {
		s.HandleImmediate()
	}
}

// HandleLaunchPhase 处理 Launch 阶段（Launch + LaunchTarget），在 Cast() 中调用。
// 对齐 TC HandleLaunchPhase()。
func (s *Spell) HandleLaunchPhase() {
	ProcessLaunchPhase(s)
}

// HandleHitPhase 处理 Hit 阶段（Hit + HitTarget），在 HandleImmediate() / 弹道命中时调用。
// 对齐 TC _handle_immediate_phase() + DoProcessTargetContainer()。
func (s *Spell) HandleHitPhase() {
	s.callHook(HookBeforeHit)
	ProcessHitPhase(s)
	// OnHit/OnMiss 按目标分派
	for i := range s.TargetInfos {
		ti := &s.TargetInfos[i]
		if ti.MissReason == HitNormal || ti.MissReason == HitCrit {
			s.callHook(HookOnHit)
		} else if ti.MissReason != HitNormal {
			s.callHook(HookOnMiss)
		}
	}
	s.callHook(HookAfterHit)
	if s.Bus != nil {
		for i := range s.TargetInfos {
			ti := &s.TargetInfos[i]
			s.Bus.Publish(event.Event{
				Type:     event.OnSpellHit,
				SourceID: s.Caster.GetID(),
				TargetID: ti.TargetID,
				SpellID:  uint32(s.ID),
				Value:    ti.Damage,
				Extra:    map[string]any{"crit": ti.Crit, "spellName": s.Info.Name},
			})
		}
	}
}

// HandleImmediate 处理即时法术的 Hit 阶段并完成，对齐 TC handle_immediate()
func (s *Spell) HandleImmediate() {
	s.HandleHitPhase()
	s.DoDamageAndTriggers()
	s.Finish(CastOK)
}

// DoDamageAndTriggers 执行伤害/治疗结算，对齐 TC 的 DoDamageAndTriggers。
func (s *Spell) DoDamageAndTriggers() {
	if s.DoDamageAndTriggersFn != nil {
		s.DoDamageAndTriggersFn(s)
	}
}

func (s *Spell) Cancel() {
	if s.State == StateFinished {
		return
	}

	// 记录旧状态用于状态特定的清理（TC 模式）
	oldState := s.State
	s.State = StateFinished

	// 状态特定的清理
	switch oldState {
	case StatePreparing:
		// TC: CancelGlobalCooldown() — 未来 GCD 系统的占位
	case StateChanneling:
		// Remove all owned auras matching this spell's SpellID from the caster.
		// Aligned with TC: cancel() iterates and removes auras by spell ID.
		if s.Engine != nil {
			s.Engine.RemoveOwnedAurasBySpellID(s.Caster.GetID(), s.ID)
		}
	}

	// 注册中心钩子（在 Bus 事件之前）
	if s.Engine != nil {
		s.Engine.CallCancelHook(s.ID, s)
	}

	// Bus 事件
	if s.Bus != nil {
		s.Bus.Publish(event.Event{
			Type:     event.OnSpellCancel,
			SourceID: s.Caster.GetID(),
			SpellID:  uint32(s.ID),
			Extra:    map[string]any{"result": CastFailedInterrupted, "spellName": s.Info.Name},
		})
	}

	// 恢复旧状态供 finish() 判断原始状态（TC 模式）
	s.State = oldState
	s.Finish(CastFailedInterrupted)
}

func (s *Spell) Finish(result SpellCastResult) {
	if s.State == StateFinished {
		return
	}
	s.State = StateFinished
	s.Result = result
	if s.Bus != nil {
		s.Bus.Publish(event.Event{
			Type:     event.OnSpellFinish,
			SourceID: s.Caster.GetID(),
			SpellID:  uint32(s.ID),
			Extra:    map[string]any{"result": result, "spellName": s.Info.Name},
		})
	}
}

func (s *Spell) CheckCast(strict bool) SpellCastResult {
	if !s.Caster.IsAlive() && !s.Info.HasAttribute(AttrAllowWhileDead) {
		return CastFailedCasterDead
	}
	if !s.Caster.CanCast() {
		return CastFailedStunned
	}

	// GCD 检查，对齐 TC 的 Spell::CheckCast → CheckGlobalCooldown
	if strict && s.CastFlags&TriggeredIgnoreGCD == 0 {
		if h := s.Caster.GetHistory(); h != nil {
			if h.HasGlobalCooldown(s.Info.CategoryID) {
				return CastFailedCantDoRightNow
			}
		}
	}

	// 冷却就绪检查，对齐 TC 的 Spell::CheckCast → CheckCooldown
	if s.CastFlags&TriggeredIgnoreCooldown == 0 {
		if h := s.Caster.GetHistory(); h != nil {
			if !h.IsReady(s.ID, s.Info.CategoryID) {
				return CastFailedNotReady
			}
		}
	}

	// 能量检查，对齐 TC 的 Spell::CheckCast → CheckPower
	if s.CastFlags&TriggeredIgnorePower == 0 && s.Info.PowerCost > 0 {
		// PowerType=0 对应 Mana，但 GetStatValue 使用 stat.StatType 映射
		// Unit.ModifyPower 做了 PowerType→StatType 映射，这里用相同的映射
		statType := s.Info.PowerType
		if statType == 0 {
			statType = 2 // Mana 对应 StatType=2
		}
		current := s.Caster.GetStatValue(statType)
		if current < float64(s.Info.PowerCost) {
			return CastFailedNoPower
		}
	}

	if s.CastFlags&TriggeredIgnoreCasterAuraState == 0 {
		if !s.Caster.CanCast() {
			return CastFailedSilenced
		}
	}
	return CastOK
}

// SelectEffectTargets 对每个 effect 的 TargetA 和 TargetB 分别解析目标，对齐 TC 的 Spell::SelectImplicitTargets。
// 替代旧的 SelectTargets()，使用 targeting 包的数据驱动查表。
func (s *Spell) SelectEffectTargets() {
	s.TargetInfos = nil
	if s.Info == nil || len(s.Info.Effects) == 0 {
		return
	}

	// processedEffectMask 记录已处理的效果索引，避免重复搜索
	processedMask := uint8(0)

	for i := range s.Info.Effects {
		eff := &s.Info.Effects[i]
		effectBit := uint8(1) << eff.EffectIndex

		// TargetA 解析
		if eff.TargetA != TargetNone {
			s.selectEffectImplicitTargets(eff.TargetA, eff, effectBit, &processedMask)
		}

		// TargetB 解析（使用 TargetA 解析后的参考位置）
		if eff.TargetB != TargetNone {
			s.selectEffectImplicitTargets(eff.TargetB, eff, effectBit, &processedMask)
		}
	}
}

// selectEffectImplicitTargets 根据 ImplicitTarget 的 5 维属性分发目标选择，对齐 TC 的 SpellImplicitTargetInfo::SelectTarget。
func (s *Spell) selectEffectImplicitTargets(target ImplicitTarget, eff *SpellEffectInfo, effectBit uint8, processedMask *uint8) {
	info := targeting.NewImplicitTargetInfo(uint16(target))
	category := info.GetSelectionCategory()

	if category == targeting.SelectNYI {
		return
	}

	var units []targeting.TargetUnit

	switch category {
	case targeting.SelectDefault:
		units = targeting.ResolveDefaultTargets(info, s)

	case targeting.SelectArea:
		center := targeting.ResolveCenter(info.GetReferenceType(), s)
		radius := eff.Radius
		if radius <= 0 {
			radius = 5.0 // 默认半径，对齐 TC 的 DEFAULT_RADIUS
		}
		units = targeting.SearchAreaTargets(center, radius, info.GetCheckType(), s.GetCaster(), s, 0)

	case targeting.SelectCone:
		center := targeting.ResolveCenter(info.GetReferenceType(), s)
		radius := eff.Radius
		if radius <= 0 {
			radius = 5.0
		}
		// 锥形方向：施法者朝向 + DirectionType 偏移
		var direction float64
		if caster := s.GetCaster(); caster != nil {
			direction = caster.GetPosition().GetFacing()
		}
		dirAngle := info.CalcDirectionAngle()
		direction += dirAngle
		// 默认弧度 90°，对齐 TC 的 DEFAULT_CONE_ANGLE
		arcAngle := math.Pi / 2
		units = targeting.SearchConeTargets(center, direction, arcAngle, radius, info.GetCheckType(), s.GetCaster(), s, 0)

	case targeting.SelectNearby:
		center := targeting.ResolveCenter(info.GetReferenceType(), s)
		radius := eff.Radius
		if radius <= 0 {
			radius = 30.0 // Nearby 默认搜索半径
		}
		if u := targeting.SearchNearbyTarget(center, radius, info.GetCheckType(), s.GetCaster(), s, 0); u != nil {
			units = []targeting.TargetUnit{u}
		}

	case targeting.SelectLine:
		from := targeting.ResolveCenter(targeting.RefCaster, s)
		to := targeting.ResolveCenter(info.GetReferenceType(), s)
		width := eff.Radius
		if width <= 0 {
			width = 5.0
		}
		units = targeting.SearchLineTargets(from, to, width, info.GetCheckType(), s.GetCaster(), s, 0)

	case targeting.SelectChannel:
		units = targeting.SelectChannelTargets(info, s)
	}

	// Chain 跳跃处理
	if eff.ChainTargets > 0 && len(units) > 0 {
		excludeIDs := make([]uint64, 0, len(s.TargetInfos)+1)
		for _, ti := range s.TargetInfos {
			excludeIDs = append(excludeIDs, ti.TargetID)
		}
		chainResult := targeting.SearchChainTargets(units[0], int(eff.ChainTargets), eff.Radius, info.GetCheckType(), s.GetCaster(), s, excludeIDs)
		units = chainResult
	}

	// 将搜索结果添加到 TargetInfos
	for _, u := range units {
		uid := u.GetID()
		// 查找是否已存在，若存在则合并 EffectMask
		found := false
		for i := range s.TargetInfos {
			if s.TargetInfos[i].TargetID == uid {
				s.TargetInfos[i].EfffectMask |= effectBit
				found = true
				break
			}
		}
		if !found {
			s.TargetInfos = append(s.TargetInfos, TargetInfo{
				TargetID:    uid,
				MissReason:  HitNormal,
				EfffectMask: effectBit,
			})
		}
	}

	// Fallback: 当引擎不可用时，如果 UnitTargetID 已设置且 TargetInfos 为空，直接添加
	if len(units) == 0 && s.Engine == nil && s.Targets.UnitTargetID != 0 {
		exists := false
		for _, ti := range s.TargetInfos {
			if ti.TargetID == s.Targets.UnitTargetID {
				exists = true
				break
			}
		}
		if !exists {
			s.TargetInfos = append(s.TargetInfos, TargetInfo{
				TargetID:    s.Targets.UnitTargetID,
				MissReason:  HitNormal,
				EfffectMask: effectBit,
			})
		}
	}

	// 处理 DEST/SRC 类型：设置位置
	objType := info.GetObjectType()
	if objType == targeting.ObjDest || objType == targeting.ObjSrc || objType == targeting.ObjUnitAndDest {
		center := targeting.ResolveCenter(info.GetReferenceType(), s)
		dirType := info.GetDirectionType()
		if dirType != targeting.DirNone {
			center = targeting.ApplyDirectionOffset(center, dirType, s)
		}
		// 设置 DestPos（如果尚未设置）
		if s.Targets.DestPos == [3]float64{} {
			s.Targets.DestPos = center
		}
		// 设置 SourcePos（SRC 类型）
		if objType == targeting.ObjSrc {
			if s.Targets.SourcePos == [3]float64{} {
				s.Targets.SourcePos = center
			}
		}
	}
}

func (s *Spell) TakePower() {
	if s.Info.PowerCost > 0 {
		s.Caster.ModifyPower(s.Info.PowerType, -float64(s.Info.PowerCost))
	}
}

// ApplyCooldown 为法术添加冷却记录，对齐 TC 的 Spell::cast → AddCooldown。
func (s *Spell) ApplyCooldown() {
	if s.Info.CooldownTime == 0 {
		return
	}
	if h := s.Caster.GetHistory(); h != nil {
		h.AddCooldown(s.ID, s.Info.CategoryID, time.Duration(s.Info.CooldownTime)*time.Millisecond)
	}
}

// ApplyGlobalCooldown 为法术分类添加 GCD，对齐 TC 的 Spell::cast → AddGlobalCooldown。
// CategoryID=0 的法术不参与 GCD 系统。
func (s *Spell) ApplyGlobalCooldown() {
	if s.Info.CategoryID == 0 {
		return
	}
	if h := s.Caster.GetHistory(); h != nil {
		gcdMs := int32(1500)
		if s.Info.CastTime > 0 {
			gcdMs = int32(s.Info.CastTime)
		}
		h.AddGlobalCooldown(s.Info.CategoryID, time.Duration(gcdMs)*time.Millisecond)
	}
}

// calculateTargetDelays 计算每个目标的命中延迟 (ms)，对齐 TC AddUnitTarget() 中的 TimeDelay 计算。
// 公式: TimeDelay = LaunchDelay + max(dist/Speed*1000, MinDuration)
func (s *Spell) calculateTargetDelays() {
	launchDelayMs := int32(s.Info.LaunchDelay)
	for i := range s.TargetInfos {
		hitDelayMs := launchDelayMs
		if s.Info.Speed > 0 {
			dist := s.getTargetDistance(s.TargetInfos[i].TargetID)
			if dist < 5.0 {
				dist = 5.0
			}
			flightMs := int32(dist / s.Info.Speed * 1000.0)
			minDurMs := int32(s.Info.MinDuration)
			if flightMs < minDurMs {
				flightMs = minDurMs
			}
			hitDelayMs += flightMs
		}
		s.TargetInfos[i].TimeDelay = hitDelayMs
	}
}

// getTargetDistance 获取施法者到指定目标的距离，对齐 TC 的 GetDistance（最小 5 码）。
func (s *Spell) getTargetDistance(targetID uint64) float64 {
	casterPos := s.Caster.GetPosition()
	targetPos := s.Caster.GetTargetPosition(targetID)
	return positionDistance(casterPos, targetPos)
}

// maxTargetDelay 返回所有目标中的最大 TimeDelay。
func (s *Spell) maxTargetDelay() int32 {
	var max int32
	for _, ti := range s.TargetInfos {
		if ti.TimeDelay > max {
			max = ti.TimeDelay
		}
	}
	return max
}

// --- targeting.SpellTargetRef 接口实现 ---

// GetCaster 返回施法者作为 TargetUnit。
func (s *Spell) GetCaster() targeting.TargetUnit {
	return &casterAdapter{caster: s.Caster}
}

// GetUnitTargetID 返回当前选中的单位目标 ID。
func (s *Spell) GetUnitTargetID() uint64 {
	return s.Targets.UnitTargetID
}

// GetSourcePos 返回法术源位置。
func (s *Spell) GetSourcePos() [3]float64 {
	return s.Targets.SourcePos
}

// GetDestPos 返回法术目标位置。
func (s *Spell) GetDestPos() [3]float64 {
	return s.Targets.DestPos
}

// GetLastTargetID 返回上一个加入 TargetInfos 的目标 ID。
func (s *Spell) GetLastTargetID() uint64 {
	if len(s.TargetInfos) == 0 {
		return 0
	}
	return s.TargetInfos[len(s.TargetInfos)-1].TargetID
}

// GetUnitByID 按 ID 获取单位，委托给引擎。
func (s *Spell) GetUnitByID(id uint64) targeting.TargetUnit {
	if s.Engine == nil {
		return nil
	}
	return s.Engine.GetTargetUnit(id)
}

// GetUnitsInRadius 获取指定半径内的所有单位，委托给引擎。
func (s *Spell) GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []targeting.TargetUnit {
	if s.Engine == nil {
		return nil
	}
	return s.Engine.GetTargetUnitsInRadius(center, radius, excludeID)
}

// casterAdapter 将 Caster 适配为 targeting.TargetUnit。
type casterAdapter struct {
	caster Caster
}

func (ca *casterAdapter) GetID() uint64 { return ca.caster.GetID() }
func (ca *casterAdapter) GetPosition() targeting.TargetPosition {
	return &posAdapter{pos: ca.caster.GetPosition()}
}
func (ca *casterAdapter) GetEntityType() uint8 { return 0 } // Caster 接口无 EntityType，默认 0
func (ca *casterAdapter) IsAlive() bool        { return ca.caster.IsAlive() }

// posAdapter 将 Position 适配为 targeting.TargetPosition。
type posAdapter struct {
	pos Position
}

func (pa *posAdapter) GetX() float64      { return pa.pos.GetX() }
func (pa *posAdapter) GetY() float64      { return pa.pos.GetY() }
func (pa *posAdapter) GetZ() float64      { return pa.pos.GetZ() }
func (pa *posAdapter) GetFacing() float64 { return pa.pos.GetFacing() }

// 确保 Spell 实现 targeting.SpellTargetRef 接口
var _ targeting.SpellTargetRef = (*Spell)(nil)

func positionDistance(a, b Position) float64 {
	dx := a.GetX() - b.GetX()
	dy := a.GetY() - b.GetY()
	dz := a.GetZ() - b.GetZ()
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
