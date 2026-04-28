package spell

import "skill-go/pkg/targeting"

// SpellAttribute 是法术属性的位掩码。
type SpellAttribute uint32

const (
	AttrNone    SpellAttribute = 0
	AttrPassive SpellAttribute = 1 << iota
	AttrAllowWhileDead
	_ // 已移除: AttrBreakOnMove — 使用 InterruptFlags 替代
	AttrChanneled
	AttrInstant
)

// SpellInterruptFlags 控制施法/引导期间的打断条件，对齐 TC 的 SpellInterruptFlags 位掩码。
type SpellInterruptFlags uint32

const (
	InterruptNone           SpellInterruptFlags = 0
	InterruptMovement       SpellInterruptFlags = 1 << iota // 施法者移动时取消
	InterruptDamageCancels                                  // 受到伤害时取消
	InterruptDamagePushback                                 // 受到伤害时推回施法时间（占位）
)

func (f SpellInterruptFlags) HasFlag(flag SpellInterruptFlags) bool {
	return f&flag != 0
}

// SpellAuraInterruptFlags 控制光环被外部事件移除的条件，对齐 TC 的 SpellAuraInterruptFlags 位掩码。
type SpellAuraInterruptFlags uint32

const (
	AuraInterruptNone       SpellAuraInterruptFlags = 0
	AuraInterruptOnMovement SpellAuraInterruptFlags = 1 << iota // 承载者移动时移除
	AuraInterruptOnDamage                                       // 承载者受到伤害时移除
	AuraInterruptOnAction                                       // 承载者执行动作时移除
)

func (f SpellAuraInterruptFlags) HasFlag(flag SpellAuraInterruptFlags) bool {
	return f&flag != 0
}

// SpellInfo 定义法术的静态信息。
type SpellInfo struct {
	ID             SpellID
	Name           string
	CastTime       uint32
	Duration       uint32
	CooldownTime   uint32
	CategoryID     uint32
	RangeMin       float64
	RangeMax       float64
	PowerCost      uint32
	PowerType      uint8
	LaunchDelay    uint32
	Speed          float64
	MinDuration    uint32
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
	EffectIndex        uint8
	EffectType         EffectType
	BasePoints         float64
	BaseDieSides       float64
	RealPtsPerLvl      float64
	BonusCoeff         float64
	MiscValue          int32
	TriggerSpellID     SpellID
	AuraType           uint16
	AuraPeriod         uint32
	AuraInterruptFlags SpellAuraInterruptFlags
	TargetA            ImplicitTarget
	TargetB            ImplicitTarget
	Radius             float64 // 区域效果半径，对齐 TC 的 SpellEffectInfo::Radius
	ChainTargets       int32   // Chain 跳跃次数，对齐 TC 的 SpellEffectInfo::ChainTarget
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

// ImplicitTarget 表示隐式目标类型的枚举，使用 TC 原始编号。
// 未实现的索引留空，后续按需添加。
type ImplicitTarget uint16

const (
	TargetNone ImplicitTarget = 0 // 无目标

	// Group A: Core Unit targets (SelectionCategory=DEFAULT, ObjectType=UNIT)
	TargetUnitCaster        ImplicitTarget = 1 // 施法者自身
	TargetUnitNearbyEnemy   ImplicitTarget = 2 // 最近的敌方单位
	TargetUnitNearbyAlly    ImplicitTarget = 3 // 最近的友方单位
	TargetUnitNearbyParty   ImplicitTarget = 4 // 最近的队友
	TargetUnitPet           ImplicitTarget = 5 // 施法者的宠物
	TargetUnitTargetEnemy   ImplicitTarget = 6 // 选中的敌方目标
	TargetUnitSrcAreaEntry  ImplicitTarget = 7 // SrcPos 区域内按 Entry 过滤（暂不实现）
	TargetUnitDestAreaEntry ImplicitTarget = 8 // DestPos 区域内按 Entry 过滤（暂不实现）
	TargetDestHome          ImplicitTarget = 9 // 施法者家园位置
	// 10: 未使用
	// 11: TARGET_UNIT_SRC_AREA_UNK_11 (NYI)
	// 12-14: 未使用
	TargetUnitSrcAreaEnemy  ImplicitTarget = 15 // SrcPos 区域内所有敌方单位
	TargetUnitDestAreaEnemy ImplicitTarget = 16 // DestPos 区域内所有敌方单位
	TargetDestDB            ImplicitTarget = 17 // 数据库配置位置
	TargetDestCaster        ImplicitTarget = 18 // 施法者位置作为目标位置
	// 19: 未使用
	TargetUnitCasterAreaParty ImplicitTarget = 20 // 施法者区域内所有队友
	TargetUnitTargetAlly      ImplicitTarget = 21 // 选中的友方目标
	TargetSrcCaster           ImplicitTarget = 22 // 施法者作为源位置
	// 23: TARGET_GAMEOBJECT_TARGET (暂不实现)
	TargetUnitConeEnemy24 ImplicitTarget = 24 // 施法者前方锥形敌方
	TargetUnitTargetAny   ImplicitTarget = 25 // 任意选中的目标
	// 26: TARGET_GAMEOBJECT_ITEM_TARGET (暂不实现)
	TargetUnitMaster        ImplicitTarget = 27 // 施法者的主人
	TargetDestDynobjEnemy   ImplicitTarget = 28 // 动态对象敌方位置
	TargetDestDynobjAlly    ImplicitTarget = 29 // 动态对象友方位置
	TargetUnitSrcAreaAlly   ImplicitTarget = 30 // SrcPos 区域内所有友方单位
	TargetUnitDestAreaAlly  ImplicitTarget = 31 // DestPos 区域内所有友方单位
	TargetDestCasterSummon  ImplicitTarget = 32 // 施法者前方召唤位置
	TargetUnitSrcAreaParty  ImplicitTarget = 33 // SrcPos 区域内所有队友
	TargetUnitDestAreaParty ImplicitTarget = 34 // DestPos 区域内所有队友
	TargetUnitTargetParty   ImplicitTarget = 35 // 选中的队友目标
	// 36: TARGET_DEST_CASTER_UNK_36 (NYI)
	TargetUnitLasttargetAreaParty ImplicitTarget = 37 // 上一个目标区域内队友
	TargetUnitNearbyEntry         ImplicitTarget = 38 // 最近的按 Entry 过滤单位（暂不实现）
	// 39: TARGET_DEST_CASTER_FISHING (暂不实现)
	// 40: TARGET_GAMEOBJECT_NEARBY_ENTRY (暂不实现)
	TargetDestCasterFrontRight    ImplicitTarget = 41 // 施法者右前方位置
	TargetDestCasterBackRight     ImplicitTarget = 42 // 施法者右后方位置
	TargetDestCasterBackLeft      ImplicitTarget = 43 // 施法者左后方位置
	TargetDestCasterFrontLeft     ImplicitTarget = 44 // 施法者左前方位置
	TargetUnitTargetChainhealAlly ImplicitTarget = 45 // ChainHeal 友方目标
	// 46: TARGET_DEST_NEARBY_ENTRY (暂不实现)
	TargetDestCasterFront ImplicitTarget = 47 // 施法者正前方位置
	TargetDestCasterBack  ImplicitTarget = 48 // 施法者正后方位置
	TargetDestCasterRight ImplicitTarget = 49 // 施法者右方位置
	TargetDestCasterLeft  ImplicitTarget = 50 // 施法者左方位置
	// 51-52: TARGET_GAMEOBJECT_SRC_AREA/DEST_AREA (暂不实现)
	TargetDestTargetEnemy         ImplicitTarget = 53 // 目标敌方位置
	TargetUnitCone180DegEnemy     ImplicitTarget = 54 // 施法者前方 180° 锥形敌方
	TargetDestCasterFrontLeap     ImplicitTarget = 55 // 施法者前方跳跃位置
	TargetUnitCasterAreaRaid      ImplicitTarget = 56 // 施法者区域内所有团队成员
	TargetUnitTargetRaid          ImplicitTarget = 57 // 选中的团队成员目标
	TargetUnitNearbyRaid          ImplicitTarget = 58 // 最近的团队成员
	TargetUnitConeAlly            ImplicitTarget = 59 // 施法者前方锥形友方
	TargetUnitConeEntry           ImplicitTarget = 60 // 施法者前方锥形按 Entry 过滤（暂不实现）
	TargetUnitTargetAreaRaidClass ImplicitTarget = 61 // 目标区域内同职业团队成员（暂不实现）
	TargetDestCasterGround        ImplicitTarget = 62 // 施法者地面位置（地面指向 AoE）
	TargetDestTargetAny           ImplicitTarget = 63 // 目标任意位置
	TargetDestTargetFront         ImplicitTarget = 64 // 目标正前方位置
	TargetDestTargetBack          ImplicitTarget = 65 // 目标正后方位置
	TargetDestTargetRight         ImplicitTarget = 66 // 目标右方位置
	TargetDestTargetLeft          ImplicitTarget = 67 // 目标左方位置
	TargetDestTargetFrontRight    ImplicitTarget = 68 // 目标右前方位置
	TargetDestTargetBackRight     ImplicitTarget = 69 // 目标右后方位置
	TargetDestTargetBackLeft      ImplicitTarget = 70 // 目标左后方位置
	TargetDestTargetFrontLeft     ImplicitTarget = 71 // 目标左前方位置
	TargetDestCasterRandom        ImplicitTarget = 72 // 施法者随机方向位置
	TargetDestCasterRadius        ImplicitTarget = 73 // 施法者半径内随机位置
	TargetDestTargetRandom        ImplicitTarget = 74 // 目标随机方向位置
	TargetDestTargetRadius        ImplicitTarget = 75 // 目标半径内随机位置
	TargetDestChannelTarget       ImplicitTarget = 76 // Channel spell 的目标位置
	TargetUnitChannelTarget       ImplicitTarget = 77 // Channel spell 的目标单位
	TargetDestDestFront           ImplicitTarget = 78 // DestPos 正前方位置
	TargetDestDestBack            ImplicitTarget = 79 // DestPos 正后方位置
	TargetDestDestRight           ImplicitTarget = 80 // DestPos 右方位置
	TargetDestDestLeft            ImplicitTarget = 81 // DestPos 左方位置
	TargetDestDestFrontRight      ImplicitTarget = 82 // DestPos 右前方位置
	TargetDestDestBackRight       ImplicitTarget = 83 // DestPos 右后方位置
	TargetDestDestBackLeft        ImplicitTarget = 84 // DestPos 左后方位置
	TargetDestDestFrontLeft       ImplicitTarget = 85 // DestPos 左前方位置
	TargetDestDestRandom          ImplicitTarget = 86 // DestPos 随机方向位置
	TargetDestDest                ImplicitTarget = 87 // DestPos 本身
	TargetDestDynobjNone          ImplicitTarget = 88 // 动态对象位置（无友敌标记）
	// 89: TARGET_DEST_TRAJ (暂不实现)
	// 90: TARGET_UNIT_TARGET_MINIPET (暂不实现)
	TargetDestDestRadius ImplicitTarget = 91 // DestPos 半径内随机位置
	TargetUnitSummoner   ImplicitTarget = 92 // 施法者的召唤者
	// 93: TARGET_CORPSE_SRC_AREA_ENEMY (暂不实现)
	// 94: TARGET_UNIT_VEHICLE (暂不实现)
	// 95: TARGET_UNIT_TARGET_PASSENGER (暂不实现)
	// 96-103: TARGET_UNIT_PASSENGER_0..7 (暂不实现)
	TargetUnitConeCasterToDestEnemy ImplicitTarget = 104 // 施法者→Dest 锥形敌方
	// 105: TARGET_UNIT_CASTER_AND_PASSENGERS (暂不实现)
	// 106-107: TARGET_DEST_NEARBY_DB/ENTRY_2 (暂不实现)
	// 108-110: GOBJ_CONE targets (暂不实现)
	// 111-114: 未使用
	TargetUnitSrcAreaFurthestEnemy ImplicitTarget = 115 // SrcPos 区域内最远敌方
	// 116: TARGET_UNIT_AND_DEST_LAST_ENEMY (暂不实现)
	// 117: 未使用
	TargetUnitTargetAllyOrRaid ImplicitTarget = 118 // 目标友方或团队成员
	// 119: TARGET_CORPSE_SRC_AREA_RAID (暂不实现)
	TargetUnitSelfAndSummons ImplicitTarget = 120 // 施法者 + 所有召唤物
	// 121: TARGET_CORPSE_TARGET_ALLY (暂不实现)
	TargetUnitAreaThreatList ImplicitTarget = 122 // 威胁列表中的单位（暂不实现）
	// 123-125: TAP_LIST/GROUND_2 (暂不实现)
	// 126-127: CLUMP targets (NYI)
	TargetUnitRectCasterAlly        ImplicitTarget = 128 // 施法者矩形区域友方
	TargetUnitRectCasterEnemy       ImplicitTarget = 129 // 施法者矩形区域敌方
	TargetUnitRectCaster            ImplicitTarget = 130 // 施法者矩形区域
	TargetDestSummoner              ImplicitTarget = 131 // 召唤者位置
	TargetDestTargetAlly            ImplicitTarget = 132 // 目标友方位置
	TargetUnitLineCasterToDestAlly  ImplicitTarget = 133 // 施法者→Dest 线形友方
	TargetUnitLineCasterToDestEnemy ImplicitTarget = 134 // 施法者→Dest 线形敌方
	TargetUnitLineCasterToDest      ImplicitTarget = 135 // 施法者→Dest 线形
	TargetUnitConeCasterToDestAlly  ImplicitTarget = 136 // 施法者→Dest 锥形友方
	// 137-152: 暂不实现
)

// IsAreaTarget 判断隐式目标类型是否为区域效果（Area、Cone 或 Line 分类）。
// 对齐 TC 的 SpellImplicitTargetInfo::IsArea，使用 targeting 包的查表数据。
func IsAreaTarget(t ImplicitTarget) bool {
	info := targeting.NewImplicitTargetInfo(uint16(t))
	return info.IsArea()
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
