package unit

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/cooldown"
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"time"
)

// Unit is the central entity for combat simulation, aligned with TC's Unit class.
// Each Unit holds its own active spells, owned auras, and applied auras.
// Reference: tc-references/unit-update-architecture.md
type Unit struct {
	Entity   *entity.Entity
	Stats    *stat.StatSet
	History  *cooldown.History
	engine   EngineRef

	// activeSpells — spells currently being cast, channeled, or in flight by this Unit.
	// Aligned with TC's m_currentSpells.
	activeSpells []*spell.Spell

	// ownedAuras — auras created by this Unit. This Unit drives their tick and expiry.
	// Aligned with TC's m_ownedAuras.
	ownedAuras []*aura.Aura

	// appliedAuras — auras currently affecting this Unit.
	// Aligned with TC's m_appliedAuras.
	appliedAuras []*aura.Aura
}

// EngineRef is an interface to avoid circular dependency between Unit and Engine.
// Engine implements this interface.
type EngineRef interface {
	GetUnit(id uint64) *Unit
	GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []*Unit
	GetBus() interface{}
	GetSpellPower(casterID uint64) float64
	Tick() time.Duration
	AuraMgr() *aura.Manager
}

// NewUnit creates a new Unit with the given entity and stats.
func NewUnit(ent *entity.Entity, stats *stat.StatSet, history *cooldown.History) *Unit {
	return &Unit{
		Entity:  ent,
		Stats:   stats,
		History: history,
	}
}

// SetEngine sets the engine back-reference. Called by Engine during AddUnit.
func (u *Unit) SetEngine(e EngineRef) {
	u.engine = e
}

// ID returns the unit's entity ID.
func (u *Unit) ID() uint64 {
	return uint64(u.Entity.ID)
}

// Update drives this Unit's spell and aura updates for one tick.
// Order matches TC's Unit::_UpdateSpells:
//  1. Update active spells (clean finished first, then drive remaining)
//  2. Update owned auras (tick periodic, clean expired)
//
// Reference: TC Unit.cpp:2952-2986
func (u *Unit) Update(diff int32) {
	u.updateSpells(diff)
	u.updateAuras(diff)
}

// --- spell.Caster interface implementation ---

func (u *Unit) GetID() uint64          { return u.ID() }
func (u *Unit) IsAlive() bool          { return u.Entity.IsAlive() }
func (u *Unit) CanCast() bool          { return u.Entity.CanCast() }
func (u *Unit) IsMoving() bool         { return false } // TODO: movement system
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
	// TODO: implement power modification through stat system
	return true
}

// --- Active Spell management ---

// AddActiveSpell registers a spell to this Unit's active list.
func (u *Unit) AddActiveSpell(s *spell.Spell) {
	u.activeSpells = append(u.activeSpells, s)
}

// RemoveActiveSpell removes a finished spell from the active list.
func (u *Unit) removeActiveSpell(idx int) {
	u.activeSpells = append(u.activeSpells[:idx], u.activeSpells[idx+1:]...)
}

// GetActiveSpells returns the list of currently active spells (read-only).
func (u *Unit) GetActiveSpells() []*spell.Spell {
	return u.activeSpells
}

// updateSpells drives all active spells and cleans up finished ones.
// Order: clean finished → update remaining. Matches TC's _UpdateSpells.
func (u *Unit) updateSpells(diff int32) {
	// Clean finished spells first (TC: m_currentSpells cleanup at Unit.cpp:2961)
	i := 0
	for i < len(u.activeSpells) {
		if u.activeSpells[i].State == spell.StateFinished {
			u.removeActiveSpell(i)
			continue
		}
		i++
	}

	// Drive remaining active spells (TC: SpellEvent::Execute → Spell::update)
	for _, s := range u.activeSpells {
		s.Update(diff)
	}
}

// --- Owned Aura management ---

// AddOwnedAura registers an aura created by this Unit.
func (u *Unit) AddOwnedAura(a *aura.Aura) {
	u.ownedAuras = append(u.ownedAuras, a)
}

// RemoveOwnedAura removes an aura from the owned list by index.
func (u *Unit) RemoveOwnedAura(idx int) {
	u.ownedAuras = append(u.ownedAuras[:idx], u.ownedAuras[idx+1:]...)
}

// GetOwnedAuras returns the list of auras owned by this Unit.
func (u *Unit) GetOwnedAuras() []*aura.Aura {
	return u.ownedAuras
}

// --- Applied Aura management ---

// AddAppliedAura registers an aura affecting this Unit.
func (u *Unit) AddAppliedAura(a *aura.Aura) {
	u.appliedAuras = append(u.appliedAuras, a)
}

// RemoveAppliedAura removes an aura from the applied list by index.
func (u *Unit) RemoveAppliedAura(idx int) {
	u.appliedAuras = append(u.appliedAuras[:idx], u.appliedAuras[idx+1:]...)
}

// GetAppliedAuras returns the list of auras applied to this Unit.
func (u *Unit) GetAppliedAuras() []*aura.Aura {
	return u.appliedAuras
}

// FindAppliedAura finds an aura on this Unit by spellID and casterID.
func (u *Unit) FindAppliedAura(spellID spell.SpellID, casterID uint64) *aura.Aura {
	for _, a := range u.appliedAuras {
		if a.SpellID == spellID && a.CasterID == casterID {
			return a
		}
	}
	return nil
}

// FindOwnedAura finds an aura owned by this Unit by spellID and targetID.
func (u *Unit) FindOwnedAura(spellID spell.SpellID, targetID uint64) *aura.Aura {
	for _, a := range u.ownedAuras {
		if a.SpellID == spellID && a.TargetID == targetID {
			return a
		}
	}
	return nil
}

// FindAreaAura finds an area aura owned by this Unit by spellID.
func (u *Unit) FindAreaAura(spellID spell.SpellID) *aura.Aura {
	for _, a := range u.ownedAuras {
		if a.IsAreaAura && a.SpellID == spellID {
			return a
		}
	}
	return nil
}

// updateAuras ticks all owned periodic auras and removes expired ones.
// Matches TC's owned aura update loop (Unit.cpp:2971-2986).
func (u *Unit) updateAuras(diff int32) {
	elapsed := time.Duration(diff) * time.Millisecond
	sp := u.Stats.Get(stat.SpellPower)
	bus := u.getBus()

	// Iterate with defensive cleanup (TC: m_auraUpdateIterator pattern)
	// First pass: tick all owned auras
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

	// Second pass: remove expired auras (TC: expired aura cleanup Unit.cpp:2979)
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
		// Use aura manager for proper removal including script hooks (AfterRemove etc.)
		if u.engine != nil {
			target := u.engine.GetUnit(a.TargetID)
			if target == nil {
				target = u // area aura fallback: owner is also the "target"
			}
			u.engine.AuraMgr().RemoveAuraFromHosts(a, u, target, aura.RemoveByExpire)
		}
	}
}

// tickSingleAura ticks periodic effects on a single-target aura.
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
				bus.Publish(event.Event{
					Type:     event.OnAuraTick,
					SourceID: a.CasterID,
					TargetID: a.TargetID,
					SpellID:  uint32(a.SpellID),
					Value:    amount,
					Extra:    map[string]any{"spellName": a.SpellName},
				})
			}
			_ = amount // TODO: apply damage/heal via combat system
		}
	}
}

// tickAreaAura ticks periodic effects on an area aura, resolving targets each tick.
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

			// Resolve area targets each tick
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
				}
			}
		}
	}
}

// removeAuraFromBoth removes an aura from both owner and target.
// This is the centralized removal path, ensuring consistency.
func (u *Unit) removeAuraFromBoth(a *aura.Aura, mode aura.RemoveMode) {
	// Remove from this unit's owned list
	for i, owned := range u.ownedAuras {
		if owned.ID == a.ID {
			u.ownedAuras = append(u.ownedAuras[:i], u.ownedAuras[i+1:]...)
			break
		}
	}

	// Remove from target's applied list
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

// entityPos adapts entity.Position to spell.Position interface.
type entityPos struct {
	p entity.Position
}

func (ep *entityPos) GetX() float64      { return ep.p.X }
func (ep *entityPos) GetY() float64      { return ep.p.Y }
func (ep *entityPos) GetZ() float64      { return ep.p.Z }
func (ep *entityPos) GetFacing() float64 { return ep.p.Facing }

// Ensure Unit implements spell.Caster
var _ spell.Caster = (*Unit)(nil)
