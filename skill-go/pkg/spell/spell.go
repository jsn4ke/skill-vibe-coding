package spell

import (
	"skill-go/pkg/event"

	"math"
)

type SpellID uint32

type SpellState uint8

const (
	StateNull SpellState = iota
	StatePreparing
	StateLaunched
	StateChanneling
	StateFinished
)

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

type EffectHandleMode uint8

const (
	HandleLaunch EffectHandleMode = iota
	HandleLaunchTarget
	HandleHit
	HandleHitTarget
)

type CastFlags uint32

type EffectProcessorFunc func(s *Spell, mode EffectHandleMode)

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

type CancelHook func()

// AuraCreatedFunc is called by the effect pipeline when an aura is created.
// Uses interface{} to avoid importing aura package.
type AuraCreatedFunc func(a interface{})

type AoESelector interface {
	SelectAoETargets(center [3]float64, excludeID uint64) []uint64
}

type TargetInfo struct {
	TargetID     uint64
	MissReason   HitResult
	EfffectMask  uint8
	Damage       float64
	Healing      float64
	Crit         bool
}

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

type Position interface {
	GetX() float64
	GetY() float64
	GetZ() float64
	GetFacing() float64
}

// SpellEngineRef provides spell with engine access for interrupt checks and script hooks.
// Implemented by engine.Engine to avoid circular imports.
type SpellEngineRef interface {
	GetCasterUnit(id uint64) CasterUnit
	AuraRemover() AuraRemover
	CallLaunchHook(spellID SpellID, s *Spell)
	CallCancelHook(spellID SpellID, s *Spell)
	RemoveOwnedAurasBySpellID(casterID uint64, spellID SpellID)
}

// CasterUnit is a minimal interface for what Spell needs from a target unit.
type CasterUnit interface {
	IsAlive() bool
}

// AuraRemover removes an aura from owner/target by spell ID and target ID.
type AuraRemover interface {
	RemoveAuraFromChannel(ownerID uint64, targetID uint64, spellID SpellID)
}

type TargetData struct {
	UnitTargetID uint64
	SourcePos    [3]float64
	DestPos      [3]float64
	TargetMask   uint32
}

func NewSpell(id SpellID, info *SpellInfo, caster Caster, flags CastFlags) *Spell {
	return &Spell{
		ID:        id,
		Info:      info,
		Caster:    caster,
		CastFlags: flags,
		State:     StateNull,
	}
}

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

// MaxRangeTolerance is the maximum range tolerance for spell range checks.
// Aligned with TC's MAX_SPELL_RANGE_TOLERANCE.
const MaxRangeTolerance = 3.0

func (s *Spell) Update(diffMs int32) {
	if s.State == StateFinished || s.State == StateNull {
		return
	}

	// --- Interrupt checks (aligned with TC Spell::update) ---

	// 1. Caster death
	if !s.Caster.IsAlive() {
		s.Cancel()
		return
	}

	// 2. Target disappeared (TC: UpdatePointers + target GUID check)
	if s.Targets.UnitTargetID != 0 && s.Engine != nil {
		if s.Engine.GetCasterUnit(s.Targets.UnitTargetID) == nil {
			s.Cancel()
			return
		}
	}

	// 3. Movement interrupt (TC: CheckMovement)
	// Triggered spells are immune to movement interrupt
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

	// 4. Range check during PREPARING (TC: CheckRange in cast)
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

	// --- State machine ---
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
		// Validate channel targets each tick (TC: UpdateChanneledTargetList)
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

// validateChannelTargets checks each target in TargetInfos for validity.
// Targets are removed if: unit gone, dead, or out of range.
// If all targets are removed, the spell is cancelled.
// Aligned with TC's Spell::UpdateChanneledTargetList.
func (s *Spell) validateChannelTargets() {
	if s.Engine == nil || len(s.TargetInfos) == 0 {
		return
	}

	tolerance := math.Min(MaxRangeTolerance, s.Info.RangeMax*0.1)
	var remaining []TargetInfo

	for _, ti := range s.TargetInfos {
		// Check target exists
		targetUnit := s.Engine.GetCasterUnit(ti.TargetID)
		if targetUnit == nil {
			s.removeChannelAura(ti.TargetID)
			continue
		}

		// Check target alive
		if !targetUnit.IsAlive() {
			s.removeChannelAura(ti.TargetID)
			continue
		}

		// Check range (only if spell has range)
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

	// All targets gone — cancel channel (TC: "Channeled spell removed due to lack of targets")
	if len(s.TargetInfos) == 0 {
		s.Cancel()
	}
}

// removeChannelAura removes the aura from a channel target when the target
// leaves range, dies, or disappears.
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
		// Execute effect pipeline for channel spells (aura creation, etc.)
		// Aligned with TC: channel spells process effects before entering channel state.
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

// ProcessEffects handles the effect pipeline and publishes SpellHit events.
// Does NOT call Finish() — the caller decides when to finish.
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

func (s *Spell) HandleImmediate() {
	s.ProcessEffects()
	s.Finish(CastOK)
}

func (s *Spell) Cancel() {
	if s.State == StateFinished {
		return
	}

	// Record old state for state-specific cleanup (TC pattern)
	oldState := s.State
	s.State = StateFinished

	// State-specific cleanup
	switch oldState {
	case StatePreparing:
		// TC: CancelGlobalCooldown() — placeholder for future GCD system
	case StateChanneling:
		// Remove all owned auras matching this spell's SpellID from the caster.
		// Aligned with TC: cancel() iterates and removes auras by spell ID.
		if s.Engine != nil {
			s.Engine.RemoveOwnedAurasBySpellID(s.Caster.GetID(), s.ID)
		}
	}

	// User hook
	if s.OnCancel != nil {
		s.OnCancel()
	}

	// Registry hook (before Bus event)
	if s.Engine != nil {
		s.Engine.CallCancelHook(s.ID, s)
	}

	// Bus event
	if s.Bus != nil {
		s.Bus.Publish(event.Event{
			Type:     event.OnSpellCancel,
			SourceID: s.Caster.GetID(),
			SpellID:  uint32(s.ID),
			Extra:    map[string]any{"result": CastFailedInterrupted, "spellName": s.Info.Name},
		})
	}

	// Restore old state for finish() to know original state (TC pattern)
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

	// Check if any effect uses area targeting
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
