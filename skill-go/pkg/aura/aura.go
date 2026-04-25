package aura

import (
	"fmt"
	"skill-go/pkg/event"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"time"
)

// AuraHost is an interface for entities that can own or receive auras.
// This breaks the import cycle between aura and unit packages.
// Both Unit.owner and Unit.target satisfy this interface.
type AuraHost interface {
	GetID() uint64
	AddOwnedAura(a *Aura)
	RemoveOwnedAura(idx int)
	GetOwnedAuras() []*Aura
	AddAppliedAura(a *Aura)
	RemoveAppliedAura(idx int)
	GetAppliedAuras() []*Aura
	FindAppliedAura(spellID spell.SpellID, casterID uint64) *Aura
}

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

type StackRule uint8

const (
	StackNone StackRule = iota
	StackRefresh
	StackAddStack
	StackReplace
)

type Aura struct {
	ID          uint64
	SpellID     spell.SpellID
	CasterID    uint64
	TargetID    uint64
	AuraType    AuraType
	Duration    time.Duration
	MaxDuration time.Duration
	StackAmount uint8
	MaxStack    uint8
	StackRule   StackRule
	Charges     int32
	ProcChance  float64
	PPM         float64
	Effects     []AuraEffect
	AppliedAt   time.Time
	Elapsed     time.Duration
	SpellName   string
	IsAreaAura  bool
	AreaCenter  [3]float64
	AreaRadius  float64
	SpellValues    map[uint8]float64
	InterruptFlags spell.SpellAuraInterruptFlags
}

type AuraEffect struct {
	EffectIndex    uint8
	AuraType       AuraType
	Amount         float64
	BonusCoeff     float64
	Period         time.Duration
	PeriodTimer    time.Duration
	TicksDone      uint32
	MiscValue      int32
	TriggerSpellID spell.SpellID
}

type AuraApplication struct {
	Aura       *Aura
	TargetID   uint64
	Flags      uint8
	EffectMask uint8
}

func NewAura(spellID spell.SpellID, casterID, targetID uint64, auraType AuraType, duration time.Duration) *Aura {
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

func (a *Aura) IsExpired() bool {
	if a.MaxDuration == 0 {
		return false
	}
	if a.Elapsed > 0 {
		return a.Elapsed >= a.MaxDuration
	}
	return time.Since(a.AppliedAt) >= a.MaxDuration
}

func (a *Aura) AddStack() {
	if a.MaxStack == 0 {
		return
	}
	if a.StackAmount < a.MaxStack {
		a.StackAmount++
	}
	a.AppliedAt = time.Now()
}

func (a *Aura) RemoveStack(count uint8) {
	if count >= a.StackAmount {
		a.StackAmount = 0
	} else {
		a.StackAmount -= count
	}
}

func (a *Aura) Refresh() {
	a.AppliedAt = time.Now()
}

func (a *Aura) CalcAmount(effIdx int) float64 {
	if effIdx < 0 || effIdx >= len(a.Effects) {
		return 0
	}
	base := a.Effects[effIdx].Amount
	return base * float64(a.StackAmount)
}

// Tick advances periodic effects on this aura by elapsed time.
// Returns the tick events that occurred. onTick is called for each tick.
// This is the aura-level tick that Unit.updateAuras calls.
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

// TickArea advances periodic effects on an area aura, resolving targets each tick.
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

// Manager manages aura lifecycle. In the new architecture, it acts as a factory
// and provides Unit-aware ApplyAura/RemoveAuraFromHosts. Legacy global-map
// methods (AddAura, RemoveAura, FindAura) are retained for skill code that
// hasn't migrated to engine.CastSpell yet.
type Manager struct {
	auras    map[uint64][]*Aura // legacy: keyed by targetID
	nextID   uint64
	bus      *event.Bus
	registry *script.Registry
}

func NewManager(bus *event.Bus) *Manager {
	return &Manager{
		auras: make(map[uint64][]*Aura),
		bus:   bus,
	}
}

func (m *Manager) SetRegistry(reg *script.Registry) {
	m.registry = reg
}

// NextID allocates the next aura ID (used by ApplyAura).
func (m *Manager) NextID() uint64 {
	id := m.nextID
	m.nextID++
	return id
}

// --- Unit-aware methods (new architecture) ---

// ApplyAura registers an aura on both owner and target AuraHosts.
// This is the new primary entry point for aura application.
// The aura is added to owner.ownedAuras and target.appliedAuras.
func (m *Manager) ApplyAura(owner, target AuraHost, a *Aura) {
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

	// Dual registration
	owner.AddOwnedAura(a)
	target.AddAppliedAura(a)

	// Script hook
	if m.registry != nil {
		m.registry.CallAuraHook(a.SpellID, script.AuraHookAfterApply, &script.AuraContext{
			SpellID:  a.SpellID,
			TargetID: a.TargetID,
			CasterID: a.CasterID,
			Aura:     a,
		})
	}

	// Event
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

// RemoveAuraFromHosts removes an aura from both owner and target AuraHosts.
func (m *Manager) RemoveAuraFromHosts(a *Aura, owner, target AuraHost, mode RemoveMode) {
	// Script hook
	if m.registry != nil {
		m.registry.CallAuraHook(a.SpellID, script.AuraHookAfterRemove, &script.AuraContext{
			SpellID:    a.SpellID,
			TargetID:   a.TargetID,
			CasterID:   a.CasterID,
			RemoveMode: uint8(mode),
			Aura:       a,
		})
	}

	// Remove from owner's owned list
	for i, owned := range owner.GetOwnedAuras() {
		if owned.ID == a.ID {
			owner.RemoveOwnedAura(i)
			break
		}
	}

	// Remove from target's applied list
	for i, applied := range target.GetAppliedAuras() {
		if applied.ID == a.ID {
			target.RemoveAppliedAura(i)
			break
		}
	}
}

// --- Legacy methods (backward compatibility) ---

// AddAura adds an aura to the legacy global map. Retained for existing skill code.
func (m *Manager) AddAura(aura *Aura) {
	aura.ID = m.nextID
	m.nextID++

	existing := m.FindAura(aura.TargetID, aura.SpellID, aura.CasterID)
	if existing != nil {
		switch existing.StackRule {
		case StackRefresh:
			existing.Refresh()
			return
		case StackAddStack:
			existing.AddStack()
			return
		case StackReplace:
			m.RemoveAura(existing, RemoveByStack)
		default:
			m.RemoveAura(existing, RemoveByDefault)
		}
	}

	m.auras[aura.TargetID] = append(m.auras[aura.TargetID], aura)

	if m.registry != nil {
		m.registry.CallAuraHook(aura.SpellID, script.AuraHookAfterApply, &script.AuraContext{
			SpellID:  aura.SpellID,
			TargetID: aura.TargetID,
			CasterID: aura.CasterID,
			Aura:     aura,
		})
	}

	if m.bus != nil {
		m.bus.Publish(event.Event{
			Type:     event.OnAuraApplied,
			SourceID: aura.CasterID,
			TargetID: aura.TargetID,
			SpellID:  uint32(aura.SpellID),
			Extra:    map[string]any{"auraType": auraTypeName(aura.AuraType), "duration": aura.Duration, "spellName": aura.SpellName},
		})
	}
}

// RemoveAura removes an aura from the legacy global map.
func (m *Manager) RemoveAura(aura *Aura, mode RemoveMode) {
	if m.registry != nil {
		m.registry.CallAuraHook(aura.SpellID, script.AuraHookAfterRemove, &script.AuraContext{
			SpellID:    aura.SpellID,
			TargetID:   aura.TargetID,
			CasterID:   aura.CasterID,
			RemoveMode: uint8(mode),
			Aura:       aura,
		})
	}
	list := m.auras[aura.TargetID]
	for i, a := range list {
		if a.ID == aura.ID {
			m.auras[aura.TargetID] = append(list[:i], list[i+1:]...)
			break
		}
	}
}

func (m *Manager) RemoveAurasBySpellID(targetID uint64, spellID spell.SpellID) {
	list := m.auras[targetID]
	var remaining []*Aura
	for _, a := range list {
		if a.SpellID != spellID {
			remaining = append(remaining, a)
		}
	}
	m.auras[targetID] = remaining
}

func (m *Manager) FindAura(targetID uint64, spellID spell.SpellID, casterID uint64) *Aura {
	for _, a := range m.auras[targetID] {
		if a.SpellID == spellID && a.CasterID == casterID {
			return a
		}
	}
	return nil
}

func (m *Manager) GetAuras(targetID uint64) []*Aura {
	return m.auras[targetID]
}

func (m *Manager) FindAreaAura(casterID uint64, spellID spell.SpellID) *Aura {
	for _, list := range m.auras {
		for _, a := range list {
			if a.IsAreaAura && a.CasterID == casterID && a.SpellID == spellID {
				return a
			}
		}
	}
	return nil
}

func auraTypeName(t AuraType) string {
	names := map[AuraType]string{
		AuraNone:             "None",
		AuraPeriodicDamage:   "PeriodicDamage",
		AuraPeriodicHeal:     "PeriodicHeal",
		AuraModStat:          "ModStat",
		AuraModStun:          "Stun",
		AuraModRoot:          "Root",
		AuraModSilence:       "Silence",
		AuraModFear:          "Fear",
		AuraModConfuse:       "Confuse",
		AuraModCharm:         "Charm",
		AuraModSpeed:         "ModSpeed",
		AuraModAttackPower:   "ModAttackPower",
		AuraModSpellPower:    "ModSpellPower",
		AuraModResistance:    "ModResistance",
		AuraModCritChance:    "ModCritChance",
		AuraModHaste:         "ModHaste",
		AuraModPacify:        "Pacify",
		AuraModStealth:             "Stealth",
		AuraPeriodicTriggerSpell:   "PeriodicTriggerSpell",
		AuraProcTriggerSpell:       "ProcTriggerSpell",
		AuraProcTriggerDamage: "ProcTriggerDamage",
		AuraMounted:          "Mounted",
		AuraSchoolImmunity:   "SchoolImmunity",
		AuraMechanicImmunity: "MechanicImmunity",
		AuraDamageImmunity:   "DamageImmunity",
		AuraModIncreaseSpeed: "ModIncreaseSpeed",
		AuraModDecreaseSpeed: "ModDecreaseSpeed",
	}
	if n, ok := names[t]; ok {
		return n
	}
	return fmt.Sprintf("AuraType(%d)", t)
}
