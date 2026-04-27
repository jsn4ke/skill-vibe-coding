package spell

import (
	"skill-go/pkg/event"

	"math"
)

// SpellID 是法术的唯一标识符
type SpellID uint32

// SpellState 表示法术的状态
type SpellState uint8

const (
	StateNull SpellState = iota
	StatePreparing
	StateLaunched
	StateChanneling
	StateFinished
)

// SpellCastResult 表示施法结果的枚举
type SpellCastResult uint16

const (
	CastOK SpellCastResult = iota
	CastFailedCasterDead
	CastFailedNotReady
	CastFailedNoPower
	CastFailedOutOfRange
	CastFailedInterrupted
	CastFailedStunned
	CastFailedSilenced
	CastFailedRooted
	CastFailedTargetDead
	CastFailedTargetInvalid
	CastFailedAuraBounced
	CastFailedCantDoRightNow
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

// EffectProcessorFunc 是效果处理函数的类型
type EffectProcessorFunc func(s *Spell, mode EffectHandleMode)

// ProcessEffectsFn 是全局效果处理函数，由 effect 包的 init() 设置
var ProcessEffectsFn EffectProcessorFunc

const (
	TriggeredNone        CastFlags = 0
	TriggeredIgnoreGCD   CastFlags = 1 << iota
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

// CancelHook 是取消回调的类型
type CancelHook func()

// AuraCreatedFunc 在效果管线创建光环时调用。使用 interface{} 避免导入 aura 包
type AuraCreatedFunc func(a interface{})

// AoESelector 是 AoE 目标选择器接口
type AoESelector interface {
	SelectAoETargets(center [3]float64, excludeID uint64) []uint64
}

// TargetInfo 包含法术目标的命中信息
type TargetInfo struct {
	TargetID     uint64
	MissReason   HitResult
	EfffectMask  uint8
	Damage       float64
	Healing      float64
	Crit         bool
}

// Spell 表示一个法术实例，包含完整的施法状态和数据
type Spell struct {
	ID          SpellID
	Info        *SpellInfo
	State       SpellState
	Caster      Caster
	CastFlags   CastFlags
	Targets     TargetData
	CastTime    uint32
	Timer       int32
	HitTimer    int32
	Result      SpellCastResult
	TargetInfos []TargetInfo
	Bus            *event.Bus
	OnCancel       CancelHook
	OnAuraCreated  AuraCreatedFunc
	SpellValues    map[uint8]float64
	AoESelector    AoESelector
	AoECenter      [3]float64
	AoEExcludeID   uint64
	Engine         SpellEngineRef
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
}

// Position 是位置接口
type Position interface {
	GetX() float64
	GetY() float64
	GetZ() float64
	GetFacing() float64
}

// SpellEngineRef 提供法术对引擎的访问，用于打断检查和脚本钩子。由 engine.Engine 实现以避免循环导入
type SpellEngineRef interface {
	GetCasterUnit(id uint64) CasterUnit
	AuraRemover() AuraRemover
	CallLaunchHook(spellID SpellID, s *Spell)
	CallCancelHook(spellID SpellID, s *Spell)
	RemoveOwnedAurasBySpellID(casterID uint64, spellID SpellID)
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

// Prepare 准备施法，验证条件并设置状态
func (s *Spell) Prepare() SpellCastResult {
	if result := s.CheckCast(true); result != CastOK {
		return result
	}

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
		if s.Timer > 0 {
			s.Timer -= diffMs
			if s.Timer <= 0 {
				s.Timer = 0
				s.HandleImmediate()
			}
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

	s.SelectTargets()

	if s.State == StateFinished {
		return
	}

	if !(s.CastFlags&TriggeredIgnorePower != 0) {
		s.TakePower()
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
		// 对引导法术执行效果管线
	// 对齐 TC: 引导法术在进入引导状态前处理效果
		s.ProcessEffects()
		s.State = StateChanneling
		s.Timer = int32(s.Info.Duration)
	} else if s.Info.Speed > 0 {
		s.HitTimer = s.calculateHitDelay()
		s.Timer = s.HitTimer
	} else if s.Info.LaunchDelay == 0 {
		s.HandleImmediate()
	} else {
		s.Timer = int32(s.Info.LaunchDelay)
	}
}

// ProcessEffects 处理效果管线并发布 SpellHit 事件。不调用 Finish()，由调用者决定何时完成
func (s *Spell) ProcessEffects() {
	s.HandleEffects(HandleLaunch)
	if ProcessEffectsFn != nil {
		ProcessEffectsFn(s, HandleHit)
	}
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

// HandleImmediate 处理即时法术的效果并完成
func (s *Spell) HandleImmediate() {
	s.ProcessEffects()
	s.Finish(CastOK)
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

	// 用户钩子
	if s.OnCancel != nil {
		s.OnCancel()
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
	if strict && s.CastFlags&TriggeredIgnoreGCD == 0 {
	}
	if s.CastFlags&TriggeredIgnoreCooldown == 0 {
	}
	if s.CastFlags&TriggeredIgnorePower == 0 {
	}
	if s.CastFlags&TriggeredIgnoreCasterAuraState == 0 {
		if !s.Caster.CanCast() {
			return CastFailedSilenced
		}
	}
	return CastOK
}

func (s *Spell) SelectTargets() {
	s.TargetInfos = nil

	// 检查是否有效果使用区域目标
	hasAreaTarget := false
	if s.Info != nil {
		for i := range s.Info.Effects {
			t := s.Info.Effects[i].TargetA
			if t == TargetUnitAreaEnemy || t == TargetUnitAreaAlly || t == TargetUnitConeEnemy || t == TargetUnitChainEnemy || t == TargetUnitChainAlly {
				hasAreaTarget = true
				break
			}
		}
	}

	if hasAreaTarget && s.AoESelector != nil {
		targets := s.AoESelector.SelectAoETargets(s.AoECenter, s.AoEExcludeID)
		for _, tid := range targets {
			s.TargetInfos = append(s.TargetInfos, TargetInfo{
				TargetID:    tid,
				MissReason:  HitNormal,
				EfffectMask: 0xFF,
			})
		}
	}

	if s.Targets.UnitTargetID != 0 && !hasAreaTarget {
		s.TargetInfos = append(s.TargetInfos, TargetInfo{
			TargetID:    s.Targets.UnitTargetID,
			MissReason:  HitNormal,
			EfffectMask: 0xFF,
		})
	}
}

func (s *Spell) TakePower() {
	if s.Info.PowerCost > 0 {
		s.Caster.ModifyPower(s.Info.PowerType, -float64(s.Info.PowerCost))
	}
}

func (s *Spell) HandleEffects(mode EffectHandleMode) {
	_ = mode
}

func (s *Spell) calculateHitDelay() int32 {
	if len(s.TargetInfos) == 0 {
		return 0
	}
	casterPos := s.Caster.GetPosition()
	targetPos := s.Caster.GetTargetPosition(s.TargetInfos[0].TargetID)
	dist := positionDistance(casterPos, targetPos)
	dist = math.Max(dist, 5.0)
	hitDelaySec := dist / s.Info.Speed
	minDelayMs := float64(s.Info.MinDuration)
	hitDelayMs := math.Max(hitDelaySec*1000.0, minDelayMs)
	return int32(hitDelayMs)
}

func positionDistance(a, b Position) float64 {
	dx := a.GetX() - b.GetX()
	dy := a.GetY() - b.GetY()
	dz := a.GetZ() - b.GetZ()
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
