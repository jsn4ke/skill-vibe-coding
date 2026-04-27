package unit

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/cooldown"
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"time"
)

// Unit 是战斗模拟的核心实体，对齐 TC 的 Unit 类。
// 每个 Unit 持有自己的活跃法术、拥有的光环和施加的光环。
// 参考：tc-references/unit-update-architecture.md
type Unit struct {
	Entity   *entity.Entity
	Stats    *stat.StatSet
	History  *cooldown.History
	engine   EngineRef

	// activeSpells — 当前正在施放、引导或飞行中的法术，对齐 TC 的 m_currentSpells。
	activeSpells []*spell.Spell

	// ownedAuras — 此 Unit 创建的光环。此 Unit 驱动其 tick 和过期，对齐 TC 的 m_ownedAuras。
	ownedAuras []*aura.Aura

	// appliedAuras — 当前影响此 Unit 的光环，对齐 TC 的 m_appliedAuras。
	appliedAuras []*aura.Aura

	// 移动追踪 — 每帧检测位置变化，对齐 TC 的 _positionUpdateInfo.Relocated。
	prevPos  entity.Position
	isMoving bool
}

// EngineRef 是避免 Unit 和 Engine 循环依赖的接口。Engine 实现此接口。
type EngineRef interface {
	GetUnit(id uint64) *Unit
	GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []*Unit
	GetBus() interface{}
	GetSpellPower(casterID uint64) float64
	Tick() time.Duration
	AuraMgr() *aura.Manager
	ScriptRegistry() *script.Registry
}

// NewUnit 创建一个新 Unit，使用指定的实体和属性。
func NewUnit(ent *entity.Entity, stats *stat.StatSet, history *cooldown.History) *Unit {
	return &Unit{
		Entity:  ent,
		Stats:   stats,
		History: history,
	}
}

// SetEngine 设置引擎反向引用。由 Engine 在 AddUnit 时调用。
func (u *Unit) SetEngine(e EngineRef) {
	u.engine = e
}

// ID 返回单位的实体 ID。
func (u *Unit) ID() uint64 {
	return uint64(u.Entity.ID)
}

// Update 驱动此 Unit 的法术和光环更新，顺序对齐 TC 的 Unit::_UpdateSpells：
//  1. 更新活跃法术（先清理已完成的，再驱动剩余的）
//  2. 更新拥有的光环（tick 周期效果，清理已过期的）
//  3. 检测移动并触发光环打断
//
// 参考：TC Unit.cpp:2952-2986, Unit::InterruptMovementBasedAuras
func (u *Unit) Update(diff int32) {
	u.updateSpells(diff)
	u.updateAuras(diff)

	// 移动检测 — 比较当前帧和前一帧的位置，对齐 TC 的 _positionUpdateInfo.Relocated 检查。
	curPos := u.Entity.Pos
	u.isMoving = curPos != u.prevPos
	if u.isMoving {
		u.RemoveAurasWithInterruptFlags(spell.AuraInterruptOnMovement)
	}
	u.prevPos = curPos
}

// --- spell.Caster 接口实现 ---

func (u *Unit) GetID() uint64          { return u.ID() }
func (u *Unit) IsAlive() bool          { return u.Entity.IsAlive() }
func (u *Unit) CanCast() bool          { return u.Entity.CanCast() }
func (u *Unit) IsMoving() bool { return u.isMoving }

// SetPosition 更新单位位置。移动检测在 Update() 中进行。
func (u *Unit) SetPosition(pos entity.Position) {
	u.Entity.Pos = pos
}
func (u *Unit) GetStatValue(st uint8) float64 { return u.Stats.Get(stat.StatType(st)) }
func (u *Unit) GetPosition() spell.Position {
	return &entityPos{u.Entity.Pos}
}
func (u *Unit) GetTargetPosition(targetID uint64) spell.Position {
	if u.engine != nil {
		if target := u.engine.GetUnit(targetID); target != nil {
			return &entityPos{target.Entity.Pos}
		}
	}
	return &entityPos{u.Entity.Pos}
}
func (u *Unit) ModifyPower(pt uint8, amount float64) bool {
	// Map TC power types to stat types.
	// TC: 0=mana, 1=rage, 2=focus, 3=energy, etc.
	// Our stat: 0=health, 1=mana, ...
	var st stat.StatType
	switch pt {
	case 0: // PowerMana
		st = stat.Mana
	default:
		st = stat.StatType(pt)
	}
	cur := u.Stats.Get(st)
	u.Stats.SetBase(st, cur+amount)
	return true
}

// --- 活跃法术管理 ---

// AddActiveSpell 将法术注册到此 Unit 的活跃列表。
func (u *Unit) AddActiveSpell(s *spell.Spell) {
	u.activeSpells = append(u.activeSpells, s)
}

// removeActiveSpell 从活跃列表中移除已完成的法术。
func (u *Unit) removeActiveSpell(idx int) {
	u.activeSpells = append(u.activeSpells[:idx], u.activeSpells[idx+1:]...)
}

// GetActiveSpells 返回当前活跃法术列表（只读）。
func (u *Unit) GetActiveSpells() []*spell.Spell {
	return u.activeSpells
}

// updateSpells 驱动所有活跃法术并清理已完成的。顺序：先清理 → 再驱动，对齐 TC 的 _UpdateSpells。
func (u *Unit) updateSpells(diff int32) {
	// 先清理已完成的法术（TC: m_currentSpells cleanup at Unit.cpp:2961）
	i := 0
	for i < len(u.activeSpells) {
		if u.activeSpells[i].State == spell.StateFinished {
			u.removeActiveSpell(i)
			continue
		}
		i++
	}

	// 驱动剩余的活跃法术（TC: SpellEvent::Execute → Spell::update）
	for _, s := range u.activeSpells {
		s.Update(diff)
	}
}

// --- 拥有的光环管理 ---

// AddOwnedAura 注册此 Unit 创建的光环。
func (u *Unit) AddOwnedAura(a *aura.Aura) {
	u.ownedAuras = append(u.ownedAuras, a)
}

// RemoveOwnedAura 按索引从拥有列表移除光环。
func (u *Unit) RemoveOwnedAura(idx int) {
	u.ownedAuras = append(u.ownedAuras[:idx], u.ownedAuras[idx+1:]...)
}

// GetOwnedAuras 返回此 Unit 拥有的光环列表。
func (u *Unit) GetOwnedAuras() []*aura.Aura {
	return u.ownedAuras
}

// --- 施加的光环管理 ---

// AddAppliedAura 注册影响此 Unit 的光环。
func (u *Unit) AddAppliedAura(a *aura.Aura) {
	u.appliedAuras = append(u.appliedAuras, a)
}

// RemoveAppliedAura 按索引从施加列表移除光环。
func (u *Unit) RemoveAppliedAura(idx int) {
	u.appliedAuras = append(u.appliedAuras[:idx], u.appliedAuras[idx+1:]...)
}

// GetAppliedAuras 返回施加到此 Unit 的光环列表。
func (u *Unit) GetAppliedAuras() []*aura.Aura {
	return u.appliedAuras
}

// FindAppliedAura 按 spellID 和 casterID 查找施加到此 Unit 的光环。
func (u *Unit) FindAppliedAura(spellID spell.SpellID, casterID uint64) *aura.Aura {
	for _, a := range u.appliedAuras {
		if a.SpellID == spellID && a.CasterID == casterID {
			return a
		}
	}
	return nil
}

// FindOwnedAura 按 spellID 和 targetID 查找此 Unit 拥有的光环。
func (u *Unit) FindOwnedAura(spellID spell.SpellID, targetID uint64) *aura.Aura {
	for _, a := range u.ownedAuras {
		if a.SpellID == spellID && a.TargetID == targetID {
			return a
		}
	}
	return nil
}

// FindAreaAura 按 spellID 查找此 Unit 拥有的区域光环。
func (u *Unit) FindAreaAura(spellID spell.SpellID) *aura.Aura {
	for _, a := range u.ownedAuras {
		if a.IsAreaAura && a.SpellID == spellID {
			return a
		}
	}
	return nil
}

// RemoveAurasWithInterruptFlags 移除所有 InterruptFlags 匹配指定标志的拥有光环，
// 使用 RemoveByInterrupt 模式。同时检查当前引导法术是否匹配打断标志。
// 对齐 TC 的 Unit::RemoveAurasWithInterruptFlags。
func (u *Unit) RemoveAurasWithInterruptFlags(flag spell.SpellAuraInterruptFlags) {
	if flag == spell.AuraInterruptNone {
		return
	}

	// 先收集匹配的光环（避免遍历中修改）
	var toRemove []*aura.Aura
	for _, a := range u.ownedAuras {
		if a.InterruptFlags.HasFlag(flag) {
			toRemove = append(toRemove, a)
		}
	}

	// 通过光环管理器移除匹配的光环
	if u.engine != nil {
		mgr := u.engine.AuraMgr()
		for _, a := range toRemove {
			target := u.engine.GetUnit(a.TargetID)
			if target == nil {
				target = u // area aura fallback
			}
			mgr.RemoveAuraFromHosts(a, u, target, aura.RemoveByInterrupt)
		}
	}

	// 检查引导法术是否匹配引导打断标志
	for _, s := range u.activeSpells {
		if s.State == spell.StateChanneling && s.Info.InterruptFlags.HasFlag(spell.InterruptMovement) {
			s.Cancel()
			break
		}
	}
}

// updateAuras tick 所有拥有的周期光环并移除已过期的。
// 对齐 TC 的拥有光环更新循环（Unit.cpp:2971-2986）。
func (u *Unit) updateAuras(diff int32) {
	elapsed := time.Duration(diff) * time.Millisecond
	sp := u.Stats.Get(stat.SpellPower)
	bus := u.getBus()

	// 防御性清理迭代（TC: m_auraUpdateIterator 模式）
	// 第一遍：tick 所有拥有的光环
	var expired []*aura.Aura
	for _, a := range u.ownedAuras {
		a.Elapsed += elapsed

		if a.IsAreaAura {
			// Area aura: resolve targets each tick
			u.tickAreaAura(a, elapsed, sp, bus)
		} else {
			// Single-target aura: tick normally
			u.tickSingleAura(a, elapsed, sp, bus)
		}

		if a.IsExpired() {
			expired = append(expired, a)
		}
	}

	// 第二遍：移除已过期的光环（TC: expired aura cleanup Unit.cpp:2979）
	for _, a := range expired {
		if bus != nil {
			bus.Publish(event.Event{
				Type:     event.OnAuraExpired,
				SourceID: a.CasterID,
				TargetID: a.TargetID,
				SpellID:  uint32(a.SpellID),
				Extra:    map[string]any{"spellName": a.SpellName},
			})
		}
		// 使用光环管理器进行完整移除，包括脚本钩子（AfterRemove 等）
		if u.engine != nil {
			target := u.engine.GetUnit(a.TargetID)
			if target == nil {
				target = u // area aura fallback: owner is also the "target"
			}
			u.engine.AuraMgr().RemoveAuraFromHosts(a, u, target, aura.RemoveByExpire)
		}
	}
}

// tickSingleAura tick 单体光环的周期效果。
func (u *Unit) tickSingleAura(a *aura.Aura, elapsed time.Duration, sp float64, bus *event.Bus) {
	for i := range a.Effects {
		eff := &a.Effects[i]
		if eff.Period == 0 {
			continue
		}
		eff.PeriodTimer += elapsed
		for eff.PeriodTimer >= eff.Period {
			eff.PeriodTimer -= eff.Period
			eff.TicksDone++
			amount := eff.Amount + eff.BonusCoeff*sp
			if bus != nil {
				tickExtra := map[string]any{"spellName": a.SpellName}
				if eff.AuraType == aura.AuraPeriodicTriggerSpell && eff.TriggerSpellID != 0 {
					tickExtra["triggerSpellID"] = uint32(eff.TriggerSpellID)
				}
				bus.Publish(event.Event{
					Type:     event.OnAuraTick,
					SourceID: a.CasterID,
					TargetID: a.TargetID,
					SpellID:  uint32(a.SpellID),
					Value:    amount,
					Extra:    tickExtra,
				})
			}

			// Registry hook for periodic ticks
			if u.engine != nil {
				if reg := u.engine.ScriptRegistry(); reg != nil {
					reg.CallAuraHook(a.SpellID, script.AuraHookOnPeriodic, &script.AuraContext{
						SpellID:  a.SpellID,
						TargetID: a.TargetID,
						CasterID: a.CasterID,
						Amount:   amount,
					})
				}
			}
			_ = amount // TODO: apply damage/heal via combat system
		}
	}
}

// tickAreaAura tick 区域光环的周期效果，每次 tick 解析目标。
func (u *Unit) tickAreaAura(a *aura.Aura, elapsed time.Duration, sp float64, bus *event.Bus) {
	for i := range a.Effects {
		eff := &a.Effects[i]
		if eff.Period == 0 {
			continue
		}
		eff.PeriodTimer += elapsed
		for eff.PeriodTimer >= eff.Period {
			eff.PeriodTimer -= eff.Period
			eff.TicksDone++
			amount := eff.Amount + eff.BonusCoeff*sp

			// 每次 tick 解析区域目标
			if u.engine != nil {
				targets := u.engine.GetUnitsInRadius(a.AreaCenter, a.AreaRadius, 0)
				for _, t := range targets {
					if bus != nil {
						bus.Publish(event.Event{
							Type:     event.OnAuraTick,
							SourceID: a.CasterID,
							TargetID: t.ID(),
							SpellID:  uint32(a.SpellID),
							Value:    amount,
							Extra:    map[string]any{"spellName": a.SpellName},
						})
					}

					// 注册中心钩子：周期 tick（区域光环按目标）
					if reg := u.engine.ScriptRegistry(); reg != nil {
						reg.CallAuraHook(a.SpellID, script.AuraHookOnPeriodic, &script.AuraContext{
							SpellID:  a.SpellID,
							TargetID: t.ID(),
							CasterID: a.CasterID,
							Amount:   amount,
						})
					}
				}
			}
		}
	}
}

// removeAuraFromBoth 从拥有者和目标两端移除光环。
// 这是集中化的移除路径，确保一致性。
func (u *Unit) removeAuraFromBoth(a *aura.Aura, mode aura.RemoveMode) {
	// 从此单位的拥有列表移除
	for i, owned := range u.ownedAuras {
		if owned.ID == a.ID {
			u.ownedAuras = append(u.ownedAuras[:i], u.ownedAuras[i+1:]...)
			break
		}
	}

	// 从目标的施加列表移除
	if u.engine != nil {
		target := u.engine.GetUnit(a.TargetID)
		if target != nil {
			for i, applied := range target.appliedAuras {
				if applied.ID == a.ID {
					target.appliedAuras = append(target.appliedAuras[:i], target.appliedAuras[i+1:]...)
					break
				}
			}
		}
	}
}

func (u *Unit) getBus() *event.Bus {
	if u.engine == nil {
		return nil
	}
	if bus, ok := u.engine.GetBus().(*event.Bus); ok {
		return bus
	}
	return nil
}

// entityPos 将 entity.Position 适配为 spell.Position 接口。
type entityPos struct {
	p entity.Position
}

func (ep *entityPos) GetX() float64      { return ep.p.X }
func (ep *entityPos) GetY() float64      { return ep.p.Y }
func (ep *entityPos) GetZ() float64      { return ep.p.Z }
func (ep *entityPos) GetFacing() float64 { return ep.p.Facing }

// 确保 Unit 实现 spell.Caster 接口
var _ spell.Caster = (*Unit)(nil)
