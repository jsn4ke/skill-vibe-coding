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
	AddOwnedAura(a *Aura)
	RemoveOwnedAura(idx int)
	GetOwnedAuras() []*Aura
	AddAppliedAura(a *Aura)
	RemoveAppliedAura(idx int)
	GetAppliedAuras() []*Aura
	FindAppliedAura(spellID SpellID, casterID uint64) *Aura
	// ApplyAuraEffects 在光环施加时应用属性修改，由 Unit 实现
	ApplyAuraEffects(a *Aura)
	// RemoveAuraEffects 在光环移除时撤销属性修改，由 Unit 实现
	RemoveAuraEffects(a *Aura)
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
		SpellID:     spellID,
		CasterID:    casterID,
		TargetID:    targetID,
		AuraType:    auraType,
		Duration:    duration,
		MaxDuration: duration,
		AppliedAt:   time.Now(),
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

// Tick 推进单体光环的周期效果。onTick 在每次 tick 时调用。
// 这是 Unit.updateAuras 调用的光环级 tick。
func (a *Aura) Tick(elapsed time.Duration, casterSpellPower float64, bus *event.Bus, onTick func(*Aura, *AuraEffect, float64)) {
	for i := range a.Effects {
		eff := &a.Effects[i]
		if eff.Period == 0 {
			continue
		}
		eff.PeriodTimer += elapsed
		for eff.PeriodTimer >= eff.Period {
			eff.PeriodTimer -= eff.Period
			eff.TicksDone++
			amount := eff.Amount + eff.BonusCoeff*casterSpellPower
			if bus != nil {
				tickExtra := map[string]any{"spellName": a.SpellName}
				if eff.AuraType == AuraPeriodicTriggerSpell {
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
			if onTick != nil {
				onTick(a, eff, amount)
			}
		}
	}
}

// TickArea 推进区域光环的周期效果，每次 tick 解析目标。
func (a *Aura) TickArea(elapsed time.Duration, casterSpellPower float64, bus *event.Bus, targets []uint64, onTick func(*Aura, *AuraEffect, float64, uint64)) {
	for i := range a.Effects {
		eff := &a.Effects[i]
		if eff.Period == 0 {
			continue
		}
		eff.PeriodTimer += elapsed
		for eff.PeriodTimer >= eff.Period {
			eff.PeriodTimer -= eff.Period
			eff.TicksDone++
			amount := eff.Amount + eff.BonusCoeff*casterSpellPower
			for _, tid := range targets {
				if bus != nil {
					bus.Publish(event.Event{
						Type:     event.OnAuraTick,
						SourceID: a.CasterID,
						TargetID: tid,
						SpellID:  uint32(a.SpellID),
						Value:    amount,
						Extra:    map[string]any{"spellName": a.SpellName},
					})
				}
				if onTick != nil {
					onTick(a, eff, amount, tid)
				}
			}
		}
	}
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
// 光环被添加到 owner.ownedAuras 和 target.appliedAuras。
func (m *AuraManager) ApplyAura(owner, target AuraHost, a *Aura) {
	a.ID = m.NextID()

	// Check for existing aura on target (stack/refresh/replace logic)
	existing := target.FindAppliedAura(a.SpellID, a.CasterID)
	if existing != nil {
		switch existing.StackRule {
		case StackRefresh:
			existing.Refresh()
			return
		case StackAddStack:
			existing.AddStack()
			return
		case StackReplace:
			m.RemoveAuraFromHosts(existing, owner, target, RemoveByStack)
		default:
			m.RemoveAuraFromHosts(existing, owner, target, RemoveByDefault)
		}
	}

	// 双重注册
	owner.AddOwnedAura(a)
	target.AddAppliedAura(a)

	// 目标端应用光环效果（属性修改等）
	target.ApplyAuraEffects(a)

	// 脚本钩子
	if m.registry != nil {
		m.registry.CallAuraHook(a.SpellID, AuraHookAfterApply, &AuraContext{
			SpellID:  a.SpellID,
			TargetID: a.TargetID,
			CasterID: a.CasterID,
			Aura:     a,
		})
	}

	// 事件
	if m.bus != nil {
		m.bus.Publish(event.Event{
			Type:     event.OnAuraApplied,
			SourceID: a.CasterID,
			TargetID: a.TargetID,
			SpellID:  uint32(a.SpellID),
			Extra:    map[string]any{"auraType": auraTypeName(a.AuraType), "duration": a.Duration, "spellName": a.SpellName},
		})
	}
}

// RemoveAuraFromHosts 从拥有者和目标两端移除光环。
func (m *AuraManager) RemoveAuraFromHosts(a *Aura, owner, target AuraHost, mode RemoveMode) {
	// 目标端撤销光环效果（属性修改等）
	target.RemoveAuraEffects(a)

	// 脚本钩子
	if m.registry != nil {
		m.registry.CallAuraHook(a.SpellID, AuraHookAfterRemove, &AuraContext{
			SpellID:    a.SpellID,
			TargetID:   a.TargetID,
			CasterID:   a.CasterID,
			RemoveMode: uint8(mode),
			Aura:       a,
		})
	}

	// 从拥有者的拥有列表移除
	for i, owned := range owner.GetOwnedAuras() {
		if owned.ID == a.ID {
			owner.RemoveOwnedAura(i)
			break
		}
	}

	// 从目标的施加列表移除
	for i, applied := range target.GetAppliedAuras() {
		if applied.ID == a.ID {
			target.RemoveAppliedAura(i)
			break
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
