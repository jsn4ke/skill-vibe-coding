package spellcore

import (
	"fmt"
	"skill-go/pkg/event"
	"time"
)

// AuraHost 是可以拥有或接受光环的实体接口。
// 打破 aura 和 unit 包之间的导入循环。
// Unit.owner 和 Unit.target 都满足此接口。
type AuraHost interface {
	GetID() uint64

	// --- 拥有端（caster 侧）---
	AddOwnedAura(a *Aura)
	RemoveOwnedAura(idx int)
	GetOwnedAuras() []*Aura

	// --- 施加端（target 侧，使用 AuraApplication）---
	// AddAppliedAuraApp 注册影响此 Unit 的光环应用实例。
	AddAppliedAuraApp(app *AuraApplication)
	// RemoveAppliedAuraApp 按索引从应用列表移除光环应用实例。
	RemoveAppliedAuraApp(idx int)
	// GetAppliedAuraApps 返回施加到此 Unit 的光环应用列表。
	GetAppliedAuraApps() []*AuraApplication
	// FindAppliedAuraApp 按 spellID 和 casterID 查找光环应用实例。
	FindAppliedAuraApp(spellID SpellID, casterID uint64) *AuraApplication

	// ApplyAuraEffects 在光环施加时应用属性修改，由 Unit 实现
	ApplyAuraEffects(a *Aura)
	RemoveAuraEffects(a *Aura)
	// ApplyAuraEffectsForApp 使用 AuraApplication 的效果掩码应用光环效果。
	ApplyAuraEffectsForApp(app *AuraApplication)
	// RemoveAuraEffectsForApp 使用 AuraApplication 的效果掩码撤销光环效果。
	RemoveAuraEffectsForApp(app *AuraApplication)
}

// AuraType 表示光环类型的枚举。
type AuraType uint16

const (
	AuraNone AuraType = iota
	AuraPeriodicDamage
	AuraPeriodicHeal
	AuraModStat
	AuraModStun
	AuraModRoot
	AuraModSilence
	AuraModFear
	AuraModConfuse
	AuraModCharm
	AuraModSpeed
	AuraModAttackPower
	AuraModSpellPower
	AuraModResistance
	AuraModCritChance
	AuraModHaste
	AuraModPacify
	AuraModStealth
	AuraPeriodicTriggerSpell
	AuraProcTriggerSpell
	AuraProcTriggerDamage
	AuraMounted
	AuraSchoolImmunity
	AuraMechanicImmunity
	AuraDamageImmunity
	AuraModIncreaseSpeed
	AuraModDecreaseSpeed
)

// RemoveMode 表示光环移除的原因。
type RemoveMode uint8

const (
	RemoveNone RemoveMode = iota
	RemoveByDefault
	RemoveByCancel
	RemoveByExpire
	RemoveByDeath
	RemoveByDispel
	RemoveByStack
	RemoveByInterrupt
)

// StackRule 表示光环叠加规则。
type StackRule uint8

const (
	StackNone StackRule = iota
	StackRefresh
	StackAddStack
	StackReplace
)

// Aura 表示一个光环实例。
type Aura struct {
	ID             uint64
	SpellID        SpellID
	CasterID       uint64
	TargetID       uint64
	AuraType       AuraType
	Duration       time.Duration
	MaxDuration    time.Duration
	StackAmount    uint8
	MaxStack       uint8
	StackRule      StackRule
	Effects        []AuraEffect
	AppliedAt      time.Time
	Elapsed        time.Duration
	SpellName      string
	IsAreaAura     bool
	AreaCenter     [3]float64
	AreaRadius     float64
	SpellValues    map[uint8]float64
	InterruptFlags SpellAuraInterruptFlags
	// Applications 是此光环的所有目标应用映射，对齐 TC 的 m_applications。
	// key 为 targetID。区域光环可以有多个 application。
	Applications map[uint64]*AuraApplication
}

// AuraEffect 表示光环的周期效果。
type AuraEffect struct {
	EffectIndex    uint8
	AuraType       AuraType
	Amount         float64
	BonusCoeff     float64
	Period         time.Duration
	PeriodTimer    time.Duration
	TicksDone      uint32
	MiscValue      int32
	TriggerSpellID SpellID
}

// NewAura 创建一个新的光环。
func NewAura(spellID SpellID, casterID, targetID uint64, auraType AuraType, duration time.Duration) *Aura {
	return &Aura{
		SpellID:      spellID,
		CasterID:     casterID,
		TargetID:     targetID,
		AuraType:     auraType,
		Duration:     duration,
		MaxDuration:  duration,
		AppliedAt:    time.Now(),
		Applications: make(map[uint64]*AuraApplication),
	}
}

// IsExpired 判断光环是否已过期。
func (a *Aura) IsExpired() bool {
	if a.MaxDuration == 0 {
		return false
	}
	if a.Elapsed > 0 {
		return a.Elapsed >= a.MaxDuration
	}
	return time.Since(a.AppliedAt) >= a.MaxDuration
}

// AddStack 增加一层叠加。
func (a *Aura) AddStack() {
	if a.MaxStack == 0 {
		return
	}
	if a.StackAmount < a.MaxStack {
		a.StackAmount++
	}
	a.AppliedAt = time.Now()
}

// RemoveStack 移除指定层数。
func (a *Aura) RemoveStack(count uint8) {
	if count >= a.StackAmount {
		a.StackAmount = 0
	} else {
		a.StackAmount -= count
	}
}

// Refresh 刷新光环的施加时间。
func (a *Aura) Refresh() {
	a.AppliedAt = time.Now()
}

// CalcAmount 计算指定效果的数值。
func (a *Aura) CalcAmount(effIdx int) float64 {
	if effIdx < 0 || effIdx >= len(a.Effects) {
		return 0
	}
	base := a.Effects[effIdx].Amount
	return base * float64(a.StackAmount)
}

// AuraManager 管理光环生命周期，提供 Unit 级别的 ApplyAura/RemoveAuraFromHosts。
type AuraManager struct {
	nextID   uint64
	bus      *event.Bus
	registry *Registry
}

// NewAuraManager 创建一个新的光环管理器。
func NewAuraManager(bus *event.Bus) *AuraManager {
	return &AuraManager{
		bus: bus,
	}
}

// SetRegistry 设置脚本注册中心。
func (m *AuraManager) SetRegistry(reg *Registry) {
	m.registry = reg
}

// NextID 分配下一个光环 ID。
func (m *AuraManager) NextID() uint64 {
	id := m.nextID
	m.nextID++
	return id
}

// --- Unit 级别方法（新架构）---

// ApplyAura 在拥有者和目标两端注册光环。这是光环应用的主入口。
// 光环被添加到 owner.ownedAuras，同时在 target 上创建 AuraApplication。
func (m *AuraManager) ApplyAura(owner, target AuraHost, a *Aura) {
	a.ID = m.NextID()

	// 检查目标上是否已有同法术同施法者的光环应用（叠加/刷新/替换逻辑）
	existing := target.FindAppliedAuraApp(a.SpellID, a.CasterID)
	if existing != nil {
		switch existing.Base.StackRule {
		case StackRefresh:
			existing.Base.Refresh()
			return
		case StackAddStack:
			existing.Base.AddStack()
			return
		case StackReplace:
			m.RemoveAuraApplication(existing, owner, target, RemoveByStack)
		default:
			m.RemoveAuraApplication(existing, owner, target, RemoveByDefault)
		}
	}

	// 创建 AuraApplication（per-target 应用实例），对齐 TC 的 _CreateAuraApplication
	app := NewAuraApplication(a, target.GetID())

	// 双重注册：owner 持有 Aura，target 持有 AuraApplication
	owner.AddOwnedAura(a)
	target.AddAppliedAuraApp(app)
	a.Applications[target.GetID()] = app

	// 目标端应用光环效果（属性修改等）
	target.ApplyAuraEffects(a)

	// 脚本钩子
	if m.registry != nil {
		m.registry.CallAuraHook(a.SpellID, AuraHookAfterApply, &AuraContext{
			SpellID:  a.SpellID,
			TargetID: target.GetID(),
			CasterID: a.CasterID,
			Aura:     a,
			App:      app,
		})
	}

	// 事件
	if m.bus != nil {
		m.bus.Publish(event.Event{
			Type:     event.OnAuraApplied,
			SourceID: a.CasterID,
			TargetID: target.GetID(),
			SpellID:  uint32(a.SpellID),
			Extra:    map[string]any{"auraType": auraTypeName(a.AuraType), "duration": a.Duration, "spellName": a.SpellName},
		})
	}
}

// RemoveAuraApplication 是核心移除方法，从 owner 和 target 移除光环应用。
func (m *AuraManager) RemoveAuraApplication(app *AuraApplication, owner, target AuraHost, mode RemoveMode) {
	a := app.Base
	app.RemoveMode = mode

	// 目标端撤销光环效果（属性修改等）
	target.RemoveAuraEffects(a)

	// 脚本钩子
	if m.registry != nil {
		m.registry.CallAuraHook(a.SpellID, AuraHookAfterRemove, &AuraContext{
			SpellID:    a.SpellID,
			TargetID:   app.TargetID,
			CasterID:   a.CasterID,
			RemoveMode: uint8(mode),
			Aura:       a,
			App:        app,
		})
	}

	// target AuraApplication list removal
	for i, a2 := range target.GetAppliedAuraApps() {
		if a2 == app {
			target.RemoveAppliedAuraApp(i)
			break
		}
	}

	delete(a.Applications, target.GetID())

	if len(a.Applications) == 0 {
		for i, owned := range owner.GetOwnedAuras() {
			if owned.ID == a.ID {
				owner.RemoveOwnedAura(i)
				break
			}
		}
	}
}

// RemoveAuraFromHosts 是便捷方法，通过 Aura 查找对应 target 的 AuraApplication 并移除。
// 向后兼容：Living Bomb 等脚本钩子仍然持有 *Aura 并调用此方法。
func (m *AuraManager) RemoveAuraFromHosts(a *Aura, owner, target AuraHost, mode RemoveMode) {
	app := a.Applications[target.GetID()]
	if app == nil {
		return
	}
	m.RemoveAuraApplication(app, owner, target, mode)
}

// RemoveAllApplications 移除 Aura 的所有 AuraApplication，用于区域光环过期/取消。
func (m *AuraManager) RemoveAllApplications(a *Aura, owner AuraHost, mode RemoveMode, getTarget func(uint64) AuraHost) {
	// 收集所有 app，避免遍历中修改 map
	var apps []*AuraApplication
	for _, app := range a.Applications {
		apps = append(apps, app)
	}
	for _, app := range apps {
		target := getTarget(app.TargetID)
		if target == nil {
			delete(a.Applications, app.TargetID)
			continue
		}
		m.RemoveAuraApplication(app, owner, target, mode)
	}
}

// UpdateTargetMap 增量更新区域光环的目标映射，对齐 TC 的 Aura::UpdateTargetMap。
// 比对当前 application map 与新目标列表，只对差集操作。
func (m *AuraManager) UpdateTargetMap(a *Aura, owner AuraHost, targets []AuraHost, getTarget func(uint64) AuraHost) {
	newTargets := make(map[uint64]AuraHost)
	for _, t := range targets {
		newTargets[t.GetID()] = t
	}

	// 移除不在新目标集合中的现有应用
	var toRemove []*AuraApplication
	for targetID, app := range a.Applications {
		if _, exists := newTargets[targetID]; !exists {
			toRemove = append(toRemove, app)
		}
	}
	for _, app := range toRemove {
		t := getTarget(app.TargetID)
		if t != nil {
			t.RemoveAuraEffects(a)
			for i, a2 := range t.GetAppliedAuraApps() {
				if a2 == app {
					t.RemoveAppliedAuraApp(i)
					break
				}
			}
		}
		delete(a.Applications, app.TargetID)
	}

	// 为新目标创建应用
	for targetID, target := range newTargets {
		if _, exists := a.Applications[targetID]; exists {
			continue
		}
		app := NewAuraApplication(a, targetID)
		target.AddAppliedAuraApp(app)
		a.Applications[targetID] = app
		target.ApplyAuraEffects(a)

		if m.registry != nil {
			m.registry.CallAuraHook(a.SpellID, AuraHookAfterApply, &AuraContext{
				SpellID:  a.SpellID,
				TargetID: targetID,
				CasterID: a.CasterID,
				Aura:     a,
				App:      app,
			})
		}

		if m.bus != nil {
			m.bus.Publish(event.Event{
				Type:     event.OnAuraApplied,
				SourceID: a.CasterID,
				TargetID: targetID,
				SpellID:  uint32(a.SpellID),
				Extra:    map[string]any{"auraType": auraTypeName(a.AuraType), "duration": a.Duration, "spellName": a.SpellName},
			})
		}
	}
}

func auraTypeName(t AuraType) string {
	names := map[AuraType]string{
		AuraNone:                 "None",
		AuraPeriodicDamage:       "PeriodicDamage",
		AuraPeriodicHeal:         "PeriodicHeal",
		AuraModStat:              "ModStat",
		AuraModStun:              "Stun",
		AuraModRoot:              "Root",
		AuraModSilence:           "Silence",
		AuraModFear:              "Fear",
		AuraModConfuse:           "Confuse",
		AuraModCharm:             "Charm",
		AuraModSpeed:             "ModSpeed",
		AuraModAttackPower:       "ModAttackPower",
		AuraModSpellPower:        "ModSpellPower",
		AuraModResistance:        "ModResistance",
		AuraModCritChance:        "ModCritChance",
		AuraModHaste:             "ModHaste",
		AuraModPacify:            "Pacify",
		AuraModStealth:           "Stealth",
		AuraPeriodicTriggerSpell: "PeriodicTriggerSpell",
		AuraProcTriggerSpell:     "ProcTriggerSpell",
		AuraProcTriggerDamage:    "ProcTriggerDamage",
		AuraMounted:              "Mounted",
		AuraSchoolImmunity:       "SchoolImmunity",
		AuraMechanicImmunity:     "MechanicImmunity",
		AuraDamageImmunity:       "DamageImmunity",
		AuraModIncreaseSpeed:     "ModIncreaseSpeed",
		AuraModDecreaseSpeed:     "ModDecreaseSpeed",
	}
	if n, ok := names[t]; ok {
		return n
	}
	return fmt.Sprintf("AuraType(%d)", t)
}
