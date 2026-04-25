package spell

type SpellAttribute uint32

const (
	AttrNone          SpellAttribute = 0
	AttrPassive       SpellAttribute = 1 << iota
	AttrAllowWhileDead
	AttrBreakOnMove
	AttrChanneled
	AttrInstant
)

// SpellInterruptFlags controls interrupt conditions during cast/channel.
// Aligned with TC's SpellInterruptFlags bitmask.
type SpellInterruptFlags uint32

const (
	InterruptNone            SpellInterruptFlags = 0
	InterruptMovement        SpellInterruptFlags = 1 << iota // cancel on caster movement
	InterruptDamageCancels                                    // cancel on damage taken
	InterruptDamagePushback                                   // pushback cast time on damage (placeholder)
)

func (f SpellInterruptFlags) HasFlag(flag SpellInterruptFlags) bool {
	return f&flag != 0
}

// SpellAuraInterruptFlags controls when auras are removed by external events.
// Aligned with TC's SpellAuraInterruptFlags bitmask.
type SpellAuraInterruptFlags uint32

const (
	AuraInterruptNone        SpellAuraInterruptFlags = 0
	AuraInterruptOnMovement  SpellAuraInterruptFlags = 1 << iota // remove on carrier movement
	AuraInterruptOnDamage                                        // remove on carrier taking damage
	AuraInterruptOnAction                                        // remove on carrier performing action
)

func (f SpellAuraInterruptFlags) HasFlag(flag SpellAuraInterruptFlags) bool {
	return f&flag != 0
}

type SpellInfo struct {
	ID           SpellID
	Name         string
	CastTime     uint32
	Duration     uint32
	CooldownTime uint32
	CategoryID   uint32
	RangeMin     float64
	RangeMax     float64
	PowerCost    uint32
	PowerType    uint8
	LaunchDelay  uint32
	Speed        float64
	MinDuration  uint32
	IsChanneled    bool
	Attributes     SpellAttribute
	InterruptFlags SpellInterruptFlags
	Effects        []SpellEffectInfo
}

func (si *SpellInfo) HasAttribute(attr SpellAttribute) bool {
	return si.Attributes&attr != 0
}

type SpellEffectInfo struct {
	EffectIndex    uint8
	EffectType     EffectType
	BasePoints     float64
	BaseDieSides   float64
	RealPtsPerLvl  float64
	BonusCoeff     float64
	MiscValue      int32
	TriggerSpellID SpellID
	AuraType       uint16
	AuraPeriod          uint32
	AuraInterruptFlags SpellAuraInterruptFlags
	TargetA             ImplicitTarget
	TargetB             ImplicitTarget
}

type EffectType uint16

const (
	EffectNone EffectType = iota
	EffectSchoolDamage
	EffectHeal
	EffectApplyAura
	EffectEnergize
	EffectSummon
	EffectTeleportUnits
	EffectWeaponDamage
	EffectTriggerSpell
	EffectDummy
	EffectHealPct
	EffectEnergizePct
	EffectLeap
	EffectKnockBack
	EffectCharge
	EffectDispel
)

type ImplicitTarget uint16

const (
	TargetNone ImplicitTarget = iota
	TargetUnitCaster
	TargetUnitTargetEnemy
	TargetUnitTargetAlly
	TargetUnitTargetAny
	TargetUnitNearbyEnemy
	TargetUnitNearbyAlly
	TargetUnitNearbyAny
	TargetDestCaster
	TargetDestTargetEnemy
	TargetDestTargetAlly
	TargetDestDest
	TargetUnitConeEnemy
	TargetUnitAreaEnemy
	TargetUnitAreaAlly
	TargetUnitChainEnemy
	TargetUnitChainAlly
)

type HitResult uint8

const (
	HitNormal HitResult = iota
	HitCrit
	HitMiss
	HitImmune
	HitDodge
	HitEvade
)
