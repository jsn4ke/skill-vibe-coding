package spell

// SpellAttribute 是法术属性的位掩码。
type SpellAttribute uint32

const (
	AttrNone          SpellAttribute = 0
	AttrPassive       SpellAttribute = 1 << iota
	AttrAllowWhileDead
	_ // 已移除: AttrBreakOnMove — 使用 InterruptFlags 替代
	AttrChanneled
	AttrInstant
)

// SpellInterruptFlags 控制施法/引导期间的打断条件，对齐 TC 的 SpellInterruptFlags 位掩码。
type SpellInterruptFlags uint32

const (
	InterruptNone            SpellInterruptFlags = 0
	InterruptMovement        SpellInterruptFlags = 1 << iota // 施法者移动时取消
	InterruptDamageCancels                                    // 受到伤害时取消
	InterruptDamagePushback                                   // 受到伤害时推回施法时间（占位）
)

func (f SpellInterruptFlags) HasFlag(flag SpellInterruptFlags) bool {
	return f&flag != 0
}

// SpellAuraInterruptFlags 控制光环被外部事件移除的条件，对齐 TC 的 SpellAuraInterruptFlags 位掩码。
type SpellAuraInterruptFlags uint32

const (
	AuraInterruptNone        SpellAuraInterruptFlags = 0
	AuraInterruptOnMovement  SpellAuraInterruptFlags = 1 << iota // 承载者移动时移除
	AuraInterruptOnDamage                                        // 承载者受到伤害时移除
	AuraInterruptOnAction                                        // 承载者执行动作时移除
)

func (f SpellAuraInterruptFlags) HasFlag(flag SpellAuraInterruptFlags) bool {
	return f&flag != 0
}

// SpellInfo 定义法术的静态信息。
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

// SpellEffectInfo 定义法术效果的静态信息。
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

// EffectType 表示效果类型的枚举。
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

// ImplicitTarget 表示隐式目标类型的枚举。
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

// IsAreaTarget 判断隐式目标类型是否为区域效果（AoE、锥形或链式），需要区域光环处理。
func IsAreaTarget(t ImplicitTarget) bool {
	switch t {
	case TargetUnitAreaEnemy, TargetUnitAreaAlly, TargetUnitConeEnemy, TargetUnitChainEnemy, TargetUnitChainAlly:
		return true
	}
	return false
}

// HitResult 表示命中结果的枚举。
type HitResult uint8

const (
	HitNormal HitResult = iota
	HitCrit
	HitMiss
	HitImmune
	HitDodge
	HitEvade
)
