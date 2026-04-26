package engine

import (
	"fmt"
	"math"
	"skill-go/pkg/aura"
	"skill-go/pkg/cooldown"
	"skill-go/pkg/effect"
	"skill-go/pkg/entity"
	"skill-go/pkg/event"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/stat"
	"skill-go/pkg/timeline"
	"skill-go/pkg/unit"
	"time"
)

// Ensure effect handlers are registered (effect.init sets spell.ProcessEffectsFn)

// It is the sole driver of time progression via Tick(diff).
// Reference: tc-references/unit-update-architecture.md
type Engine struct {
	units    map[uint64]*unit.Unit
	bus      *event.Bus
	registry *script.Registry
	auraMgr  *aura.Manager
	renderer *timeline.TimelineRenderer

	// currentTime accumulates with each Tick
	currentTime time.Duration

	// nextUnitID for auto-generating unit IDs
	nextUnitID uint64
}

// New creates a new Engine with all subsystems initialized.
func New() *Engine {
	bus := event.NewBus()
	reg := script.NewRegistry()
	auraMgr := aura.NewManager(bus)
	auraMgr.SetRegistry(reg)

	// Wire effect pipeline to use the engine's script registry
	effect.ScriptRegistry = reg

	r := timeline.NewRenderer()
	r.SubscribeAll(bus)

	return &Engine{
		units:       make(map[uint64]*unit.Unit),
		bus:         bus,
		registry:    reg,
		auraMgr:     auraMgr,
		renderer:    r,
		nextUnitID:  1,
	}
}

// AddUnit creates and registers a new Unit in the engine.
func (e *Engine) AddUnit(ent *entity.Entity, stats *stat.StatSet) *unit.Unit {
	u := unit.NewUnit(ent, stats, cooldown.NewHistory())
	u.SetEngine(e)
	e.units[u.ID()] = u
	return u
}

// AddUnitWithID creates and registers a Unit with a specific entity ID.
func (e *Engine) AddUnitWithID(id uint64, ent *entity.Entity, stats *stat.StatSet) *unit.Unit {
	u := unit.NewUnit(ent, stats, cooldown.NewHistory())
	u.SetEngine(e)
	e.units[id] = u
	return u
}

// GetUnit returns a Unit by ID.
func (e *Engine) GetUnit(id uint64) *unit.Unit {
	return e.units[id]
}

// RemoveUnit removes a Unit from the engine.
func (e *Engine) RemoveUnit(id uint64) {
	delete(e.units, id)
}

// GetUnitsInRadius returns all units within radius of center, excluding excludeID.
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

// GetBus returns the event bus.
func (e *Engine) GetBus() interface{} { return e.bus }

// GetSpellPower returns the spell power stat for a given caster.
func (e *Engine) GetSpellPower(casterID uint64) float64 {
	u := e.units[casterID]
	if u == nil {
		return 0
	}
	return u.Stats.Get(stat.SpellPower)
}

// Tick returns current simulation time.
func (e *Engine) Tick() time.Duration { return e.currentTime }

// Registry returns the script registry.
func (e *Engine) Registry() *script.Registry { return e.registry }

// AuraMgr returns the aura manager.
func (e *Engine) AuraMgr() *aura.Manager { return e.auraMgr }

// Renderer returns the timeline renderer.
func (e *Engine) Renderer() *timeline.TimelineRenderer { return e.renderer }

// Bus returns the event bus.
func (e *Engine) Bus() *event.Bus { return e.bus }

// Simulate advances the simulation by totalMs milliseconds in steps of stepMs.
// This replaces the manual for-loop in tests.
func (e *Engine) Simulate(totalMs int32, stepMs int32) {
	for simMs := int32(0); simMs < totalMs; simMs += stepMs {
		e.Advance(stepMs)
	}
}

// Advance advances the simulation by diffMs.
// This is the core Tick — aligned with TC's Map::Update.
func (e *Engine) Advance(diffMs int32) {
	e.currentTime += time.Duration(diffMs) * time.Millisecond
	e.renderer.SetTime(e.currentTime)

	for _, u := range e.units {
		u.Update(diffMs)
	}
}

// --- CastSpell: Three execution paths (aligned with TC's Spell::prepare) ---

// CastOption configures a spell cast.
type CastOption func(*castConfig)

type castConfig struct {
	targetID    uint64
	destPos     [3]float64
	spellValues map[uint8]float64
	aoeSelector spell.AoESelector
	aoeCenter   [3]float64
	aoeExclude  uint64
	triggered   bool
}

// WithTarget sets the unit target for the spell.
func WithTarget(targetID uint64) CastOption {
	return func(c *castConfig) { c.targetID = targetID }
}

// WithDestPos sets the destination position for the spell.
func WithDestPos(x, y, z float64) CastOption {
	return func(c *castConfig) { c.destPos = [3]float64{x, y, z} }
}

// WithSpellValues sets spell values for script communication.
func WithSpellValues(sv map[uint8]float64) CastOption {
	return func(c *castConfig) { c.spellValues = sv }
}

// WithAoE sets AoE targeting parameters.
func WithAoE(selector spell.AoESelector, center [3]float64, excludeID uint64) CastOption {
	return func(c *castConfig) {
		c.aoeSelector = selector
		c.aoeCenter = center
		c.aoeExclude = excludeID
	}
}

// WithTriggered marks the spell as triggered (ignores GCD, cooldown, power).
func WithTriggered() CastOption {
	return func(c *castConfig) { c.triggered = true }
}

// CastSpell creates a spell and executes it following TC's three-path model.
//
// Path A (triggered instant): sync completion, no registration.
// Path B (normal instant): sync completion with registration.
// Path C (delayed): registration + timer, driven by Engine.Advance.
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
	s.Engine = e // Wire engine reference for interrupt checks

	// Wire OnAuraCreated so the effect pipeline can register auras automatically
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
	if cfg.spellValues != nil {
		s.SpellValues = cfg.spellValues
	}
	if cfg.aoeSelector != nil {
		s.AoESelector = cfg.aoeSelector
		s.AoECenter = cfg.aoeCenter
		s.AoEExcludeID = cfg.aoeExclude
	}

	// Prepare validates and sets state
	result := s.Prepare()
	if result != spell.CastOK {
		fmt.Printf("[engine] CastSpell %s failed: %v\n", info.Name, result)
		return s
	}

	isTriggeredInstant := cfg.triggered && !info.IsChanneled
	isNormalInstant := !isTriggeredInstant && s.CastTime == 0 && !info.IsChanneled && info.Speed == 0 && info.LaunchDelay == 0

	if isTriggeredInstant {
		// Path A: triggered instant — sync, no registration needed
		// Spell already completed inside Prepare() → Cast() → HandleImmediate() → Finish()
	} else if isNormalInstant {
		// Path B: normal instant — sync with registration
		caster.AddActiveSpell(s)
		// Spell already completed inside Prepare() for instant cast
	} else {
		// Path C: delayed — register and let Advance() drive it
		caster.AddActiveSpell(s)
	}

	return s
}

// Ensure Engine implements unit.EngineRef
var _ unit.EngineRef = (*Engine)(nil)

// ScriptRegistry returns the script registry, satisfying unit.EngineRef.
func (e *Engine) ScriptRegistry() *script.Registry { return e.registry }

// Ensure Engine implements spell.SpellEngineRef for interrupt checks
func (e *Engine) GetCasterUnit(id uint64) spell.CasterUnit {
	u := e.units[id]
	if u == nil {
		return nil
	}
	return u
}

func (e *Engine) AuraRemover() spell.AuraRemover {
	return &auraRemover{engine: e}
}

func (e *Engine) CallLaunchHook(spellID spell.SpellID, s *spell.Spell) {
	e.registry.CallSpellHook(spellID, script.HookOnSpellLaunch, &script.SpellContext{Spell: s})
}

func (e *Engine) CallCancelHook(spellID spell.SpellID, s *spell.Spell) {
	e.registry.CallSpellHook(spellID, script.HookOnSpellCancel, &script.SpellContext{Spell: s})
}

// auraRemover adapts engine to spell.AuraRemover for channel aura cleanup.
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

// RemoveOwnedAurasBySpellID removes all auras matching spellID from the caster's owned list.
// Used by Cancel() to clean up channel spell auras without depending on TargetInfos.
func (e *Engine) RemoveOwnedAurasBySpellID(casterID uint64, spellID spell.SpellID) {
	caster := e.GetUnit(casterID)
	if caster == nil {
		return
	}
	// Collect matching auras first (avoid mutating during iteration)
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
