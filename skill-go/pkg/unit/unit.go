package unit

import (
	"fmt"
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/stat"
	"time"
)

// Unit 是战斗模拟的核心实体，对齐 TC 的 Unit 类。
// 每个 Unit 持有自己的活跃法术、拥有的光环和施加的光环。
// 参考：tc-references/unit-update-architecture.md
type Unit struct {
	Entity  *entity.Entity
	Stats   *stat.StatSet
	History *spellcore.History
	engine  EngineRef

	// activeSpells — 当前正在施放、引导或飞行中的法术，对齐 TC 的 m_currentSpells。
	activeSpells []*spellcore.Spell

	// ownedAuras — 此 Unit 创建的光环。此 Unit 驱动其 tick 和过期，对齐 TC 的 m_ownedAuras。
	ownedAuras []*spellcore.Aura

	// appliedAuraApps — 当前影响此 Unit 的光环应用实例，对齐 TC 的 m_appliedAuras。
	appliedAuraApps []*spellcore.AuraApplication

	// 移动追踪 — 每帧检测位置变化，对齐 TC 的 _positionUpdateInfo.Relocated。
	prevPos  entity.Position
	isMoving bool
}

// EngineRef 是避免 Unit 和 Engine 循环依赖的接口。Engine 实现此接口。
type EngineRef interface {
	GetUnit(id uint64) *Unit
	GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []*Unit
	GetBus() *event.Bus
	GetSpellPower(casterID uint64) float64
	Tick() time.Duration
	AuraMgr() *spellcore.AuraManager
	ScriptRegistry() *spellcore.Registry
	// SettlePeriodicDamage 结算光环周期性伤害/治疗，对齐 TC 的 Aura::PeriodicTick → DealDamage
	SettlePeriodicDamage(sourceID, targetID uint64, spellID uint32, damage, healing float64, isCrit bool, spellName string)
	// TriggerPeriodicSpell 从配置表查找法术并触发施放，用于 AuraPeriodicTriggerSpell 自动触发
	TriggerPeriodicSpell(casterID, targetID uint64, spellID spellcore.SpellID)
}

// NewUnit 创建一个新 Unit，使用指定的实体和属性。
func NewUnit(ent *entity.Entity, stats *stat.StatSet, history *spellcore.History) *Unit {
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
		u.RemoveAurasWithInterruptFlags(spellcore.AuraInterruptOnMovement)
	}
	u.prevPos = curPos
}

// --- spellcore.Caster 接口实现 ---

func (u *Unit) GetID() uint64                  { return u.ID() }
func (u *Unit) IsAlive() bool                  { return u.Entity.IsAlive() }
func (u *Unit) CanCast() bool                  { return u.Entity.CanCast() }
func (u *Unit) IsMoving() bool                 { return u.isMoving }
func (u *Unit) GetHistory() *spellcore.History { return u.History }

// GetCCState 返回施法者的 CC 状态位掩码，将 entity.UnitState 映射为 spellcore.CasterCCState。
func (u *Unit) GetCCState() spellcore.CasterCCState {
	var state spellcore.CasterCCState
	if u.Entity.State.Has(entity.StateStunned) {
		state |= spellcore.CCStunned
	}
	if u.Entity.State.Has(entity.StateSilenced) {
		state |= spellcore.CCSilenced
	}
	if u.Entity.State.Has(entity.StatePacified) {
		state |= spellcore.CCPacified
	}
	if u.Entity.State.Has(entity.StateConfused) {
		state |= spellcore.CCConfused
	}
	if u.Entity.State.Has(entity.StateFeared) {
		state |= spellcore.CCFeared
	}
	return state
}

// ApplyAuraEffects 在光环施加时应用属性修改和 CC 状态，对齐 TC 的 Aura::Apply。
// CC 光环施加时立即中断匹配的活跃法术，对齐 TC 的 SetControlled → CastStop。
func (u *Unit) ApplyAuraEffects(a *spellcore.Aura) {
	// 检查光环主类型（直接通过 NewAura 创建的光环可能没有 Effects）
	if flag := ccStateFlag(a.AuraType); flag != 0 {
		u.Entity.State = u.Entity.State.Set(flag)
		u.interruptSpellsOnCCApply(a.AuraType)
	}

	for _, eff := range a.Effects {
		st := auraEffectStat(eff)
		if st != nil {
			source := fmt.Sprintf("aura:%d:%d", a.SpellID, a.ID)
			u.Stats.AddModifier(*st, stat.Modifier{Flat: eff.Amount, Source: source})
		}
		if flag := ccStateFlag(eff.AuraType); flag != 0 {
			u.Entity.State = u.Entity.State.Set(flag)
			u.interruptSpellsOnCCApply(eff.AuraType)
		}
	}
}

// RemoveAuraEffects 在光环移除时撤销属性修改和 CC 状态，对齐 TC 的 Aura::Remove。
// CC 状态清除前检查是否仍有同类型 aura 生效，对齐 TC 的 SetControlled(false) 重评估。
func (u *Unit) RemoveAuraEffects(a *spellcore.Aura) {
	if flag := ccStateFlag(a.AuraType); flag != 0 {
		if !u.hasAuraTypeActive(a.AuraType, a) {
			u.Entity.State = u.Entity.State.Clear(flag)
		}
	}

	for _, eff := range a.Effects {
		st := auraEffectStat(eff)
		if st != nil {
			source := fmt.Sprintf("aura:%d:%d", a.SpellID, a.ID)
			u.Stats.RemoveModifierBySource(*st, source)
		}
		if flag := ccStateFlag(eff.AuraType); flag != 0 {
			if !u.hasAuraTypeActive(eff.AuraType, a) {
				u.Entity.State = u.Entity.State.Clear(flag)
			}
		}
	}
}

// ApplyAuraEffectsForApp uses AuraApplication effect mask to apply aura effects.
func (u *Unit) ApplyAuraEffectsForApp(app *spellcore.AuraApplication) {
	a := app.Base
	for i := range a.Effects {
		if app.EffectMask&(1<<uint(i)) == 0 {
			continue
		}
		eff := &a.Effects[i]
		st := auraEffectStat(*eff)
		if st != nil {
			source := fmt.Sprintf("aura:%d:%d", a.SpellID, a.ID)
			u.Stats.AddModifier(*st, stat.Modifier{Flat: eff.Amount, Source: source})
		}
		if flag := ccStateFlag(eff.AuraType); flag != 0 {
			u.Entity.State = u.Entity.State.Set(flag)
			u.interruptSpellsOnCCApply(eff.AuraType)
		}
	}
}

// RemoveAuraEffectsForApp uses AuraApplication effect mask to remove aura effects.
// CC 状态清除前检查是否仍有同类型 aura 生效。
func (u *Unit) RemoveAuraEffectsForApp(app *spellcore.AuraApplication) {
	a := app.Base
	for i := range a.Effects {
		if app.EffectMask&(1<<uint(i)) == 0 {
			continue
		}
		eff := &a.Effects[i]
		st := auraEffectStat(*eff)
		if st != nil {
			source := fmt.Sprintf("aura:%d:%d", a.SpellID, a.ID)
			u.Stats.RemoveModifierBySource(*st, source)
		}
		if flag := ccStateFlag(eff.AuraType); flag != 0 {
			if !u.hasAuraTypeActive(eff.AuraType, a) {
				u.Entity.State = u.Entity.State.Clear(flag)
			}
		}
	}
}

// hasAuraTypeActive 检查目标是否仍有指定 AuraType 的光环生效中（排除指定 aura）。
// 用于 CC 移除时的状态重评估，对齐 TC 的 SetControlled(false) 检查剩余 aura。
func (u *Unit) hasAuraTypeActive(auraType spellcore.AuraType, exclude *spellcore.Aura) bool {
	for _, app := range u.appliedAuraApps {
		if app.Base == exclude {
			continue
		}
		if app.Base.AuraType == auraType {
			return true
		}
		for _, eff := range app.Base.Effects {
			if eff.AuraType == auraType {
				return true
			}
		}
	}
	return false
}

// interruptSpellsOnCCApply 在 CC 光环施加时中断匹配的活跃法术，
// 对齐 TC 的 SetControlled → CastStop 和 InterruptSpellsWithPreventionTypeOnAuraApply。
func (u *Unit) interruptSpellsOnCCApply(auraType spellcore.AuraType) {
	switch auraType {
	case spellcore.AuraModStun, spellcore.AuraModFear, spellcore.AuraModConfuse:
		// 完全丧失控制 — 中断所有活跃法术
		u.cancelAllActiveSpells()
	case spellcore.AuraModSilence:
		// 选择性中断 — 只中断 PreventionType & Silence 的法术
		u.cancelActiveSpellsByPrevention(spellcore.PreventSilence)
	case spellcore.AuraModPacify:
		// 选择性中断 — 只中断 PreventionType & Pacify 的法术
		u.cancelActiveSpellsByPrevention(spellcore.PreventPacify)
	}
}

// cancelAllActiveSpells 取消所有 Preparing/Channeling 法术，对齐 TC 的 CastStop()。
func (u *Unit) cancelAllActiveSpells() {
	for _, s := range u.activeSpells {
		if s.State == spellcore.StatePreparing || s.State == spellcore.StateChanneling {
			s.Cancel()
		}
	}
}

// cancelActiveSpellsByPrevention 取消 PreventionType 匹配的活跃法术，
// 对齐 TC 的 InterruptSpellsWithPreventionTypeOnAuraApply。
func (u *Unit) cancelActiveSpellsByPrevention(pt spellcore.SpellPreventionType) {
	for _, s := range u.activeSpells {
		if (s.State == spellcore.StatePreparing || s.State == spellcore.StateChanneling) &&
			s.Info.PreventionType&pt != 0 {
			s.Cancel()
		}
	}
}

// ccStateFlag 返回光环类型对应的 UnitState 标志，对齐 TC 的 AuraType → UnitState 映射。
func ccStateFlag(auraType spellcore.AuraType) entity.UnitState {
	switch auraType {
	case spellcore.AuraModStun:
		return entity.StateStunned
	case spellcore.AuraModRoot:
		return entity.StateRooted
	case spellcore.AuraModSilence:
		return entity.StateSilenced
	case spellcore.AuraModFear:
		return entity.StateFeared
	case spellcore.AuraModConfuse:
		return entity.StateConfused
	case spellcore.AuraModCharm:
		return entity.StateCharmed
	case spellcore.AuraModPacify:
		return entity.StatePacified
	case spellcore.AuraModStealth:
		return entity.StateStealthed
	default:
		return 0
	}
}

// auraEffectStat 返回光环效果对应的属性类型，对齐 TC 的 AuraEffect::GetModifier。
func auraEffectStat(eff spellcore.AuraEffect) *stat.StatType {
	switch eff.AuraType {
	case spellcore.AuraModStat:
		return nil // 需要更多信息，暂不支持通用属性修改
	case spellcore.AuraModAttackPower:
		st := stat.AttackPower
		return &st
	case spellcore.AuraModSpellPower:
		st := stat.SpellPower
		return &st
	case spellcore.AuraModResistance:
		st := stat.Resistance
		return &st
	case spellcore.AuraModCritChance:
		st := stat.CritChance
		return &st
	case spellcore.AuraModHaste:
		st := stat.Haste
		return &st
	default:
		return nil
	}
}

// SetPosition 更新单位位置。移动检测在 Update() 中进行。
func (u *Unit) SetPosition(pos entity.Position) {
	u.Entity.Pos = pos
}
func (u *Unit) GetStatValue(st uint8) float64 { return u.Stats.Get(stat.StatType(st)) }
func (u *Unit) GetPosition() spellcore.Position {
	return &entityPos{u.Entity.Pos}
}
func (u *Unit) GetTargetPosition(targetID uint64) spellcore.Position {
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
	// Our stat: 0=health, 1=maxhealth, 2=mana, ...
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

// ModifyHealth 修改单位生命值，对齐 TC 的 Unit::ModifyHealth。
// 生命值被钳制在 [0, MaxHealth] 范围内。返回实际变化的量。
// 调用方负责在血量归零时调用 Kill()。
func (u *Unit) ModifyHealth(delta float64) float64 {
	oldHealth := u.Stats.Get(stat.Health)
	newHealth := oldHealth + delta

	maxHealth := u.Stats.Get(stat.MaxHealth)
	if maxHealth > 0 && newHealth > maxHealth {
		newHealth = maxHealth
	}
	if newHealth < 0 {
		newHealth = 0
	}

	u.Stats.SetBase(stat.Health, newHealth)
	return newHealth - oldHealth
}

// Kill 将单位置入死亡状态，对齐 TC 的 Unit::Kill()。
func (u *Unit) Kill(attackerID uint64) {
	u.Entity.State = entity.StateDead
	u.Entity.State = u.Entity.State.Clear(entity.StateAlive)
	u.RemoveAllAurasOnDeath()

	if bus := u.getBus(); bus != nil {
		bus.Publish(event.Event{
			Type:     event.OnDeath,
			SourceID: attackerID,
			TargetID: u.ID(),
		})
	}
}

// RemoveAllAurasOnDeath 移除死亡单位的所有光环，对齐 TC 的 RemoveAllAurasOnDeath。
func (u *Unit) RemoveAllAurasOnDeath() {
	if u.engine == nil {
		return
	}
	mgr := u.engine.AuraMgr()

	// 移除拥有的光环（遍历所有 Aura 的 Applications）
	var ownedCopy []*spellcore.Aura
	for _, a := range u.ownedAuras {
		ownedCopy = append(ownedCopy, a)
	}
	removed := make(map[*spellcore.AuraApplication]bool)
	for _, a := range ownedCopy {
		// collect apps to avoid map iteration during modification
		var apps []*spellcore.AuraApplication
		for _, app := range a.Applications {
			apps = append(apps, app)
		}
		for _, app := range apps {
			target := u.engine.GetUnit(app.TargetID)
			if target == nil {
				target = u
			}
			mgr.RemoveAuraApplication(app, u, target, spellcore.RemoveByDeath)
			removed[app] = true
		}
	}
	u.ownedAuras = nil

	// remove applied aura apps, skip self-cast already handled above
	var appsCopy []*spellcore.AuraApplication
	for _, app := range u.appliedAuraApps {
		appsCopy = append(appsCopy, app)
	}
	for _, app := range appsCopy {
		if removed[app] {
			continue
		}
		owner := u.engine.GetUnit(app.Base.CasterID)
		if owner == nil {
			continue
		}
		mgr.RemoveAuraApplication(app, owner, u, spellcore.RemoveByDeath)
	}
	u.appliedAuraApps = nil
}

// --- 活跃法术管理 ---

// AddActiveSpell 将法术注册到此 Unit 的活跃列表。
func (u *Unit) AddActiveSpell(s *spellcore.Spell) {
	u.activeSpells = append(u.activeSpells, s)
}

// removeActiveSpell 从活跃列表中移除已完成的法术。
func (u *Unit) removeActiveSpell(idx int) {
	u.activeSpells = append(u.activeSpells[:idx], u.activeSpells[idx+1:]...)
}

// GetActiveSpells 返回当前活跃法术列表（只读）。
func (u *Unit) GetActiveSpells() []*spellcore.Spell {
	return u.activeSpells
}

// updateSpells 驱动所有活跃法术并清理已完成的。顺序：先清理 → 再驱动，对齐 TC 的 _UpdateSpells。
func (u *Unit) updateSpells(diff int32) {
	// 先清理已完成的法术（TC: m_currentSpells cleanup at Unit.cpp:2961）
	i := 0
	for i < len(u.activeSpells) {
		if u.activeSpells[i].State == spellcore.StateFinished {
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
func (u *Unit) AddOwnedAura(a *spellcore.Aura) {
	u.ownedAuras = append(u.ownedAuras, a)
}

// RemoveOwnedAura 按索引从拥有列表移除光环。
func (u *Unit) RemoveOwnedAura(idx int) {
	u.ownedAuras = append(u.ownedAuras[:idx], u.ownedAuras[idx+1:]...)
}

// GetOwnedAuras 返回此 Unit 拥有的光环列表。
func (u *Unit) GetOwnedAuras() []*spellcore.Aura {
	return u.ownedAuras
}

// --- 施加的光环管理 ---

// FindOwnedAura 按 spellID 和 targetID 查找此 Unit 拥有的光环。
func (u *Unit) FindOwnedAura(spellID spellcore.SpellID, targetID uint64) *spellcore.Aura {
	for _, a := range u.ownedAuras {
		if a.SpellID == spellID && a.TargetID == targetID {
			return a
		}
	}
	return nil
}

// FindAreaAura 按 spellID 查找此 Unit 拥有的区域光环。
func (u *Unit) FindAreaAura(spellID spellcore.SpellID) *spellcore.Aura {
	for _, a := range u.ownedAuras {
		if a.IsAreaAura && a.SpellID == spellID {
			return a
		}
	}
	return nil
}

// --- 光环应用实例管理（AuraApplication）---

// AddAppliedAuraApp 注册影响此 Unit 的光环应用实例。
func (u *Unit) AddAppliedAuraApp(app *spellcore.AuraApplication) {
	u.appliedAuraApps = append(u.appliedAuraApps, app)
}

// RemoveAppliedAuraApp 按索引从应用列表移除光环应用实例。
func (u *Unit) RemoveAppliedAuraApp(idx int) {
	u.appliedAuraApps = append(u.appliedAuraApps[:idx], u.appliedAuraApps[idx+1:]...)
}

// GetAppliedAuraApps 返回施加到此 Unit 的光环应用列表。
func (u *Unit) GetAppliedAuraApps() []*spellcore.AuraApplication {
	return u.appliedAuraApps
}

// FindAppliedAuraApp 按 spellID 和 casterID 查找光环应用实例。
func (u *Unit) FindAppliedAuraApp(spellID spellcore.SpellID, casterID uint64) *spellcore.AuraApplication {
	for _, app := range u.appliedAuraApps {
		if app.Base.SpellID == spellID && app.Base.CasterID == casterID {
			return app
		}
	}
	return nil
}

// InterruptSpellsOnDamage 在单位受到伤害时检查活跃法术是否需要取消，
// 对齐 TC 的 Spell::update 受伤检查（InterruptDamageCancels 标志）。
func (u *Unit) InterruptSpellsOnDamage() {
	for _, s := range u.activeSpells {
		if s.State == spellcore.StatePreparing || s.State == spellcore.StateChanneling {
			if s.Info.InterruptFlags.HasFlag(spellcore.InterruptDamageCancels) {
				s.Cancel()
			}
		}
	}
}

// RemoveAurasWithInterruptFlags 移除所有 InterruptFlags 匹配指定标志的拥有光环，
// 使用 RemoveByInterrupt 模式。同时检查当前引导法术是否匹配打断标志。
// 对齐 TC 的 Unit::RemoveAurasWithInterruptFlags。
func (u *Unit) RemoveAurasWithInterruptFlags(flag spellcore.SpellAuraInterruptFlags) {
	if flag == spellcore.AuraInterruptNone {
		return
	}

	// 先收集匹配的光环（避免遍历中修改）
	var toRemove []*spellcore.Aura
	for _, a := range u.ownedAuras {
		if a.InterruptFlags.HasFlag(flag) {
			toRemove = append(toRemove, a)
		}
	}

	// 通过光环管理器移除匹配的拥有光环
	if u.engine != nil {
		mgr := u.engine.AuraMgr()
		for _, a := range toRemove {
			// 遍历此 Aura 的所有 Application，逐个移除
			for _, app := range a.Applications {
				target := u.engine.GetUnit(app.TargetID)
				if target == nil {
					target = u
				}
				mgr.RemoveAuraApplication(app, u, target, spellcore.RemoveByInterrupt)
			}
		}
	}

	// 同时移除匹配的施加光环应用（对齐 TC：受伤打断检查目标身上的光环）
	var appsToRemove []*spellcore.AuraApplication
	for _, app := range u.appliedAuraApps {
		if app.Base.InterruptFlags.HasFlag(flag) {
			appsToRemove = append(appsToRemove, app)
		}
	}
	if u.engine != nil {
		mgr := u.engine.AuraMgr()
		for _, app := range appsToRemove {
			owner := u.engine.GetUnit(app.Base.CasterID)
			if owner == nil {
				continue
			}
			mgr.RemoveAuraApplication(app, owner, u, spellcore.RemoveByInterrupt)
		}
	}

	// 检查活跃法术是否匹配 ChannelInterruptFlags，对齐 TC 的 channel + preparing 中断。
	for _, s := range u.activeSpells {
		if (s.State == spellcore.StateChanneling || s.State == spellcore.StatePreparing) &&
			s.Info.ChannelInterruptFlags.HasFlag(flag) {
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
	var expired []*spellcore.Aura
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
			mgr := u.engine.AuraMgr()
			if a.IsAreaAura {
				mgr.RemoveAllApplications(a, u, spellcore.RemoveByExpire, func(id uint64) spellcore.AuraHost {
					return u.engine.GetUnit(id)
				})
			} else {
				target := u.engine.GetUnit(a.TargetID)
				if target == nil {
					target = u
				}
				mgr.RemoveAuraFromHosts(a, u, target, spellcore.RemoveByExpire)
			}
		}
	}
}

// tickSingleAura tick 单体光环的周期效果。
func (u *Unit) tickSingleAura(a *spellcore.Aura, elapsed time.Duration, sp float64, bus *event.Bus) {
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
				if eff.AuraType == spellcore.AuraPeriodicTriggerSpell && eff.TriggerSpellID != 0 {
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

			//// Registry hook for periodic ticks
			u.dispatchPeriodicTick(a, eff, amount, a.TargetID)
		}
	}
}

// tickAreaAura tick 区域光环的周期效果。
// 使用 UpdateTargetMap 增量更新 per-target AuraApplication，对齐 TC 的 Aura::UpdateTargetMap。
func (u *Unit) tickAreaAura(a *spellcore.Aura, elapsed time.Duration, sp float64, bus *event.Bus) {
	// 增量更新目标映射
	if u.engine != nil {
		areaTargets := u.engine.GetUnitsInRadius(a.AreaCenter, a.AreaRadius, 0)
		var hosts []spellcore.AuraHost
		for _, t := range areaTargets {
			hosts = append(hosts, t)
		}
		u.engine.AuraMgr().UpdateTargetMap(a, u, hosts, func(id uint64) spellcore.AuraHost { return u.engine.GetUnit(id) })
	}

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

			// 遍历所有已注册的 AuraApplication，对齐 TC 的 m_applications 遍历
			for _, app := range a.Applications {
				if bus != nil {
					bus.Publish(event.Event{
						Type:     event.OnAuraTick,
						SourceID: a.CasterID,
						TargetID: app.TargetID,
						SpellID:  uint32(a.SpellID),
						Value:    amount,
						Extra:    map[string]any{"spellName": a.SpellName},
					})
				}

				// 注册中心钩子：周期 tick（区域光环按目标）
				u.dispatchPeriodicTick(a, eff, amount, app.TargetID)
			}
		}
	}
}

// dispatchPeriodicTick 结算光环周期 tick 的伤害/治疗/触发，对齐 TC 的 Aura::PeriodicTick。
func (u *Unit) dispatchPeriodicTick(a *spellcore.Aura, eff *spellcore.AuraEffect, amount float64, targetID uint64) {
	if u.engine == nil {
		return
	}
	// Registry hook for periodic ticks
	if reg := u.engine.ScriptRegistry(); reg != nil {
		reg.CallAuraHook(a.SpellID, spellcore.AuraHookOnPeriodic, &spellcore.AuraContext{
			SpellID:  a.SpellID,
			TargetID: targetID,
			CasterID: a.CasterID,
			Amount:   amount,
		})
	}
	// 通过引擎结算周期性伤害/治疗，对齐 TC 的 Aura::PeriodicTick → DealDamage
	dmg := 0.0
	heal := 0.0
	if eff.AuraType == spellcore.AuraPeriodicDamage {
		dmg = amount
	} else if eff.AuraType == spellcore.AuraPeriodicHeal {
		heal = amount
	}
	if dmg > 0 || heal > 0 {
		u.engine.SettlePeriodicDamage(a.CasterID, targetID, uint32(a.SpellID), dmg, heal, false, a.SpellName)
	}
	// 对齐 TC 的 Aura::PeriodicTick 中 SPELL_AURA_PERIODIC_TRIGGER_SPELL 自动施放
	if eff.AuraType == spellcore.AuraPeriodicTriggerSpell && eff.TriggerSpellID != 0 {
		u.engine.TriggerPeriodicSpell(a.CasterID, targetID, eff.TriggerSpellID)
	}
}

func (u *Unit) getBus() *event.Bus {
	if u.engine == nil {
		return nil
	}
	return u.engine.GetBus()
}

// entityPos 将 entity.Position 适配为 spellcore.Position 接口。
type entityPos struct {
	p entity.Position
}

func (ep *entityPos) GetX() float64      { return ep.p.X }
func (ep *entityPos) GetY() float64      { return ep.p.Y }
func (ep *entityPos) GetZ() float64      { return ep.p.Z }
func (ep *entityPos) GetFacing() float64 { return ep.p.Facing }

// 确保 Unit 实现 spellcore.Caster 接口
var _ spellcore.Caster = (*Unit)(nil)
