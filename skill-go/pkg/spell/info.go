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
	IsChanneled  bool
	Attributes   SpellAttribute
	Effects      []SpellEffectInfo
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
	AuraPeriod     uint32
	TargetA        ImplicitTarget
	TargetB        ImplicitTarget
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
