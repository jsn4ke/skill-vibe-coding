package engine

import (
	"fmt"
	"math"
	"skill-go/pkg/aura"
	"skill-go/pkg/combat"
	"skill-go/pkg/cooldown"
	"skill-go/pkg/effect"
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/proc"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"skill-go/pkg/targeting"
	"skill-go/pkg/timeline"
	"skill-go/pkg/unit"
	"time"
)

// Ensure effect handlers are registered (effect.init sets spell.ProcessEffectsFn)

// Engine 是模拟引擎，是时间推进的唯一驱动者。
// 参考：tc-references/unit-update-architecture.md
type Engine struct {
	units    map[uint64]*unit.Unit
	bus      *event.Bus
	registry *script.Registry
	auraMgr  *aura.Manager
	procMgr  *proc.Manager
	renderer *timeline.TimelineRenderer

	// currentTime 随每次 Tick 累加的模拟时间
	currentTime time.Duration

	// nextUnitID 自动生成的单位 ID
	nextUnitID uint64
}

// New 创建一个新的引擎，初始化所有子系统。
func New() *Engine {
	bus := event.NewBus()
	reg := script.NewRegistry()
	auraMgr := aura.NewManager(bus)
	auraMgr.SetRegistry(reg)

	// 将效果管线连接到引擎的脚本注册中心
	effect.ScriptRegistry = reg

	r := timeline.NewRenderer()
	r.SubscribeAll(bus)

	return &Engine{
		units:      make(map[uint64]*unit.Unit),
		bus:        bus,
		registry:   reg,
		auraMgr:    auraMgr,
		procMgr:    proc.NewManager(),
		renderer:   r,
		nextUnitID: 1,
	}
}

// AddUnit 创建并注册一个新单位到引擎中。
func (e *Engine) AddUnit(ent *entity.Entity, stats *stat.StatSet) *unit.Unit {
	u := unit.NewUnit(ent, stats, cooldown.NewHistory())
	u.SetEngine(e)
	e.units[u.ID()] = u
	return u
}

// AddUnitWithID 创建并注册一个指定实体 ID 的单位。
func (e *Engine) AddUnitWithID(id uint64, ent *entity.Entity, stats *stat.StatSet) *unit.Unit {
	u := unit.NewUnit(ent, stats, cooldown.NewHistory())
	u.SetEngine(e)
	e.units[id] = u
	return u
}

// GetUnit 按 ID 获取单位。
func (e *Engine) GetUnit(id uint64) *unit.Unit {
	return e.units[id]
}

// RemoveUnit 从引擎中移除一个单位。
func (e *Engine) RemoveUnit(id uint64) {
	delete(e.units, id)
}

// GetUnitsInRadius 返回中心点指定半径内的所有单位，排除 excludeID。
func (e *Engine) GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []*unit.Unit {
	var result []*unit.Unit
	for _, u := range e.units {
		if u.ID() == excludeID {
			continue
		}
		pos := u.Entity.Pos
		dx := pos.X - center[0]
		dy := pos.Y - center[1]
		dz := pos.Z - center[2]
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist <= radius {
			result = append(result, u)
		}
	}
	return result
}

// GetBus 返回事件总线。
func (e *Engine) GetBus() interface{} { return e.bus }

// GetSpellPower 返回指定施法者的法术强度属性。
func (e *Engine) GetSpellPower(casterID uint64) float64 {
	u := e.units[casterID]
	if u == nil {
		return 0
	}
	return u.Stats.Get(stat.SpellPower)
}

// Tick 返回当前模拟时间。
func (e *Engine) Tick() time.Duration { return e.currentTime }

// Registry 返回脚本注册中心。
func (e *Engine) Registry() *script.Registry { return e.registry }

// AuraMgr 返回光环管理器。
func (e *Engine) AuraMgr() *aura.Manager { return e.auraMgr }

// ProcMgr 返回触发器管理器。
func (e *Engine) ProcMgr() *proc.Manager { return e.procMgr }

// Renderer 返回时间线渲染器。
func (e *Engine) Renderer() *timeline.TimelineRenderer { return e.renderer }

// Bus 返回事件总线。
func (e *Engine) Bus() *event.Bus { return e.bus }

// Simulate 以指定步长推进模拟指定总时长。替代测试中的手动循环。
func (e *Engine) Simulate(totalMs int32, stepMs int32) {
	for simMs := int32(0); simMs < totalMs; simMs += stepMs {
		e.Advance(stepMs)
	}
}

// Advance 推进模拟 diffMs 毫秒。这是核心 Tick，对齐 TC 的 Map::Update。
func (e *Engine) Advance(diffMs int32) {
	e.currentTime += time.Duration(diffMs) * time.Millisecond
	e.renderer.SetTime(e.currentTime)

	for _, u := range e.units {
		u.Update(diffMs)
	}
}

// --- CastSpell：三条执行路径（对齐 TC 的 Spell::prepare）---

// CastOption 配置法术施放的选项。
type CastOption func(*castConfig)

type castConfig struct {
	targetID    uint64
	destPos     [3]float64
	srcPos      [3]float64
	spellValues map[uint8]float64
	triggered   bool
}

// WithTarget 设置法术的单位目标。
func WithTarget(targetID uint64) CastOption {
	return func(c *castConfig) { c.targetID = targetID }
}

// WithDestPos 设置法术的目标位置。
func WithDestPos(x, y, z float64) CastOption {
	return func(c *castConfig) { c.destPos = [3]float64{x, y, z} }
}

// WithSrcPos 设置法术的源位置，用于 SrcPos 区域目标选择。
func WithSrcPos(x, y, z float64) CastOption {
	return func(c *castConfig) { c.srcPos = [3]float64{x, y, z} }
}

// WithSpellValues 设置法术值，用于脚本间通信。
func WithSpellValues(sv map[uint8]float64) CastOption {
	return func(c *castConfig) { c.spellValues = sv }
}

// WithTriggered 标记法术为触发的（忽略 GCD、冷却、资源消耗）。
func WithTriggered() CastOption {
	return func(c *castConfig) { c.triggered = true }
}

// CastSpell 创建法术并按 TC 三路径模型执行。
//
// 路径 A（触发即时）：同步完成，无需注册。
// 路径 B（普通即时）：同步完成并注册。
// 路径 C（延迟）：注册 + 定时器，由 Engine.Advance 驱动。
func (e *Engine) CastSpell(caster *unit.Unit, info *spell.SpellInfo, opts ...CastOption) *spell.Spell {
	cfg := &castConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	flags := spell.TriggeredNone
	if cfg.triggered {
		flags = spell.TriggeredFullMask
	}

	s := spell.NewSpell(spell.SpellID(info.ID), info, caster, flags)
	s.Bus = e.bus
	s.Engine = e                                    // 连接引擎引用，用于打断检查
	s.DoDamageAndTriggersFn = e.doDamageAndTriggers // 连接伤害结算

	// 连接 OnAuraCreated，使效果管线能自动注册光环
	s.OnAuraCreated = func(a interface{}) {
		aura := a.(*aura.Aura)
		owner := e.GetUnit(aura.CasterID)
		target := e.GetUnit(aura.TargetID)
		if owner != nil && target != nil {
			e.auraMgr.ApplyAura(owner, target, aura)
		}
	}

	if cfg.targetID != 0 {
		s.Targets.UnitTargetID = cfg.targetID
	}
	if cfg.destPos != [3]float64{} {
		s.Targets.DestPos = cfg.destPos
	}
	if cfg.srcPos != [3]float64{} {
		s.Targets.SourcePos = cfg.srcPos
	}
	if cfg.spellValues != nil {
		s.SpellValues = cfg.spellValues
	}

	// Prepare 验证并设置状态
	result := s.Prepare()
	if result != spell.CastOK {
		fmt.Printf("[engine] CastSpell %s failed: %v\n", info.Name, result)
		return s
	}

	isTriggeredInstant := cfg.triggered && !info.IsChanneled
	isNormalInstant := !isTriggeredInstant && s.CastTime == 0 && !info.IsChanneled && !info.HasHitDelay()

	if isTriggeredInstant {
		// 路径 A：触发即时 — 同步完成，无需注册
	} else if isNormalInstant {
		// 路径 B：普通即时 — 同步完成并注册
		caster.AddActiveSpell(s)
	} else {
		// 路径 C：延迟 — 注册并由 Advance() 驱动
		caster.AddActiveSpell(s)
	}

	return s
}

// 确保 Engine 实现 unit.EngineRef 接口
var _ unit.EngineRef = (*Engine)(nil)

// ScriptRegistry 返回脚本注册中心，满足 unit.EngineRef 接口。
func (e *Engine) ScriptRegistry() *script.Registry { return e.registry }

// GetCasterUnit 按 ID 获取施法者单位，满足 spell.SpellEngineRef 接口。
func (e *Engine) GetCasterUnit(id uint64) spell.CasterUnit {
	u := e.units[id]
	if u == nil {
		return nil
	}
	return u
}

// GetTargetUnit 按 ID 获取单位作为 TargetUnit，满足 spell.SpellEngineRef 接口。
func (e *Engine) GetTargetUnit(id uint64) targeting.TargetUnit {
	u := e.units[id]
	if u == nil {
		return nil
	}
	return &unitTargetAdapter{unit: u}
}

// GetTargetUnitsInRadius 获取指定半径内的所有单位作为 TargetUnit，满足 spell.SpellEngineRef 接口。
func (e *Engine) GetTargetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []targeting.TargetUnit {
	units := e.GetUnitsInRadius(center, radius, excludeID)
	result := make([]targeting.TargetUnit, len(units))
	for i, u := range units {
		result[i] = &unitTargetAdapter{unit: u}
	}
	return result
}

// unitTargetAdapter 将 *unit.Unit 适配为 targeting.TargetUnit。
type unitTargetAdapter struct {
	unit *unit.Unit
}

func (a *unitTargetAdapter) GetID() uint64 { return a.unit.ID() }
func (a *unitTargetAdapter) GetPosition() targeting.TargetPosition {
	p := a.unit.Entity.Pos
	return &entityPosAdapter{pos: p}
}
func (a *unitTargetAdapter) GetEntityType() uint8 { return uint8(a.unit.Entity.Type) }
func (a *unitTargetAdapter) IsAlive() bool        { return a.unit.IsAlive() }

// entityPosAdapter 将 entity.Position 适配为 targeting.TargetPosition。
type entityPosAdapter struct {
	pos entity.Position
}

func (a *entityPosAdapter) GetX() float64      { return a.pos.X }
func (a *entityPosAdapter) GetY() float64      { return a.pos.Y }
func (a *entityPosAdapter) GetZ() float64      { return a.pos.Z }
func (a *entityPosAdapter) GetFacing() float64 { return a.pos.Facing }

func (e *Engine) AuraRemover() spell.AuraRemover {
	return &auraRemover{engine: e}
}

func (e *Engine) CallLaunchHook(spellID spell.SpellID, s *spell.Spell) {
	e.registry.CallSpellHook(spellID, script.HookOnSpellLaunch, &script.SpellContext{Spell: s})
}

func (e *Engine) CallCancelHook(spellID spell.SpellID, s *spell.Spell) {
	e.registry.CallSpellHook(spellID, script.HookOnSpellCancel, &script.SpellContext{Spell: s})
}

// CallTargetSelectHook
func (e *Engine) CallTargetSelectHook(spellID spell.SpellID, s *spell.Spell, units []targeting.TargetUnit) {
	e.registry.CallSpellHook(spellID, script.HookOnTargetSelect, &script.SpellContext{Spell: s, TargetUnits: units})
}

// auraRemover 将引擎适配为 spell.AuraRemover，用于引导法术的光环清理。
type auraRemover struct {
	engine *Engine
}

func (r *auraRemover) RemoveAuraFromChannel(ownerID uint64, targetID uint64, spellID spell.SpellID) {
	owner := r.engine.GetUnit(ownerID)
	target := r.engine.GetUnit(targetID)
	if owner == nil || target == nil {
		return
	}
	for _, a := range owner.GetOwnedAuras() {
		if a.SpellID == spellID && a.TargetID == targetID {
			r.engine.auraMgr.RemoveAuraFromHosts(a, owner, target, aura.RemoveByCancel)
			return
		}
	}
}

// RemoveOwnedAurasBySpellID 移除施法者拥有的所有匹配 SpellID 的光环。
// 用于 Cancel() 清理引导法术光环，不依赖 TargetInfos。
func (e *Engine) RemoveOwnedAurasBySpellID(casterID uint64, spellID spell.SpellID) {
	caster := e.GetUnit(casterID)
	if caster == nil {
		return
	}
	// 先收集匹配的光环（避免遍历中修改）
	var toRemove []*aura.Aura
	for _, a := range caster.GetOwnedAuras() {
		if a.SpellID == spellID {
			toRemove = append(toRemove, a)
		}
	}
	for _, a := range toRemove {
		target := e.GetUnit(a.TargetID)
		if target == nil {
			target = caster // area aura fallback
		}
		e.auraMgr.RemoveAuraFromHosts(a, caster, target, aura.RemoveByCancel)
	}
}

// doDamageAndTriggers 执行法术的伤害/治疗结算，对齐 TC 的 DoDamageAndTriggers。
func (e *Engine) doDamageAndTriggers(s *spell.Spell) {
	for i := range s.TargetInfos {
		ti := &s.TargetInfos[i]
		if ti.Damage == 0 && ti.Healing == 0 {
			continue
		}
		ctx := combat.SettlementContext{
			SourceID:   s.Caster.GetID(),
			TargetID:   ti.TargetID,
			SpellID:    uint32(s.ID),
			Damage:     ti.Damage,
			Healing:    ti.Healing,
			IsPeriodic: false,
			IsCrit:     ti.Crit,
			SpellName:  s.Info.Name,
		}
		e.settleOneTarget(ctx)
	}
}

// settleOneTarget 对单个目标执行伤害/治疗结算，对齐 TC 的 DoDamageAndTriggers 单目标流程。
func (e *Engine) settleOneTarget(ctx combat.SettlementContext) {
	target := e.GetUnit(ctx.TargetID)
	if target == nil || !target.IsAlive() {
		return
	}

	// 伤害结算，对齐 TC 的 DealSpellDamage → DealDamage
	if ctx.Damage > 0 {
		actualDelta := target.ModifyHealth(-ctx.Damage)

		// 受伤打断光环，对齐 TC 的 RemoveAurasWithInterruptFlags(Damage)
		if actualDelta < 0 {
			target.RemoveAurasWithInterruptFlags(spell.AuraInterruptOnDamage)
		}

		// 发布伤害事件
		e.bus.Publish(event.Event{
			Type:     event.OnDamageDealt,
			SourceID: ctx.SourceID,
			TargetID: ctx.TargetID,
			SpellID:  ctx.SpellID,
			Value:    -actualDelta,
			Extra:    map[string]any{"crit": ctx.IsCrit, "spellName": ctx.SpellName},
		})
		e.bus.Publish(event.Event{
			Type:     event.OnDamageTaken,
			SourceID: ctx.TargetID,
			TargetID: ctx.SourceID,
			SpellID:  ctx.SpellID,
			Value:    -actualDelta,
			Extra:    map[string]any{"crit": ctx.IsCrit, "spellName": ctx.SpellName},
		})

		// 死亡处理，对齐 TC 的 DealDamage 中 health <= 0 判定
		if actualDelta < 0 && target.Stats.Get(stat.Health) <= 0 && target.IsAlive() {
			target.Kill(ctx.SourceID)
		}
	}

	// 治疗结算，对齐 TC 的 HealBySpell
	if ctx.Healing > 0 {
		actualDelta := target.ModifyHealth(ctx.Healing)

		e.bus.Publish(event.Event{
			Type:     event.OnHealDealt,
			SourceID: ctx.SourceID,
			TargetID: ctx.TargetID,
			SpellID:  ctx.SpellID,
			Value:    actualDelta,
			Extra:    map[string]any{"crit": ctx.IsCrit, "spellName": ctx.SpellName},
		})
		e.bus.Publish(event.Event{
			Type:     event.OnHealTaken,
			SourceID: ctx.TargetID,
			TargetID: ctx.SourceID,
			SpellID:  ctx.SpellID,
			Value:    actualDelta,
			Extra:    map[string]any{"crit": ctx.IsCrit, "spellName": ctx.SpellName},
		})
	}

	// Proc 触发，对齐 TC 的 ProcSkillsAndAuras
	attackerProcEvent := combat.BuildProcEvent(ctx)
	victimProcEvent := combat.BuildVictimProcEvent(ctx)
	e.procMgr.Check(attackerProcEvent)
	e.procMgr.Check(victimProcEvent)
}

// SettlePeriodicDamage 结算光环周期性伤害/治疗，对齐 TC 的 Aura::PeriodicTick → DealDamage。
func (e *Engine) SettlePeriodicDamage(sourceID, targetID uint64, spellID uint32, damage, healing float64, isCrit bool, spellName string) {
	ctx := combat.SettlementContext{
		SourceID:   sourceID,
		TargetID:   targetID,
		SpellID:    spellID,
		Damage:     damage,
		Healing:    healing,
		IsPeriodic: true,
		IsCrit:     isCrit,
		SpellName:  spellName,
	}
	e.settleOneTarget(ctx)
}
