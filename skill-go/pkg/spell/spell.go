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

func (s *Spell) Update(diffMs int32) {
	if s.State == StateFinished || s.State == StateNull {
		return
	}

	if !s.Caster.IsAlive() {
		s.Cancel()
		return
	}

	if s.State == StatePreparing && s.Caster.IsMoving() && s.Info.HasAttribute(AttrBreakOnMove) {
		s.Cancel()
		return
	}

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

	if s.Info.IsChanneled {
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

func (s *Spell) HandleImmediate() {
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
	s.Finish(CastOK)
}

func (s *Spell) Cancel() {
	if s.State == StateFinished {
		return
	}
	if s.OnCancel != nil {
		s.OnCancel()
	}
	if s.Bus != nil {
		s.Bus.Publish(event.Event{
			Type:     event.OnSpellCancel,
			SourceID: s.Caster.GetID(),
			SpellID:  uint32(s.ID),
			Extra:    map[string]any{"result": CastFailedInterrupted, "spellName": s.Info.Name},
		})
	}
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
