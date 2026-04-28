package targeting

// nyi 是未实现目标的默认 StaticData。
var nyi = StaticData{
	ObjectType:    ObjNone,
	ReferenceType: RefNone,
	Category:      SelectNYI,
	CheckType:     CheckDefault,
	DirectionType: DirNone,
}

// targetData 对齐 TC 的 SpellImplicitTargetInfo::_data，
// 每个 ImplicitTarget 值映射到 StaticData（5 维属性）。
// 未实现的索引填充 NYI 默认值。
var targetData [MaxImplicitTarget]StaticData

func init() {
	// 先全部填充 NYI
	for i := range targetData {
		targetData[i] = nyi
	}

	// Group A: Core Unit targets (SelectionCategory=DEFAULT, ObjectType=UNIT)
	set(1, ObjUnit, RefCaster, SelectDefault, CheckDefault, DirNone)       // TARGET_UNIT_CASTER
	set(2, ObjUnit, RefCaster, SelectNearby, CheckEnemy, DirNone)          // TARGET_UNIT_NEARBY_ENEMY
	set(3, ObjUnit, RefCaster, SelectNearby, CheckAlly, DirNone)           // TARGET_UNIT_NEARBY_ALLY
	set(4, ObjUnit, RefCaster, SelectNearby, CheckParty, DirNone)          // TARGET_UNIT_NEARBY_PARTY
	set(5, ObjUnit, RefCaster, SelectDefault, CheckDefault, DirNone)       // TARGET_UNIT_PET
	set(6, ObjUnit, RefTarget, SelectDefault, CheckEnemy, DirNone)         // TARGET_UNIT_TARGET_ENEMY
	set(7, ObjUnit, RefSrc, SelectArea, CheckEntry, DirNone)               // TARGET_UNIT_SRC_AREA_ENTRY
	set(8, ObjUnit, RefDest, SelectArea, CheckEntry, DirNone)              // TARGET_UNIT_DEST_AREA_ENTRY
	set(9, ObjDest, RefCaster, SelectDefault, CheckDefault, DirNone)       // TARGET_DEST_HOME
	set(15, ObjUnit, RefSrc, SelectArea, CheckEnemy, DirNone)              // TARGET_UNIT_SRC_AREA_ENEMY
	set(16, ObjUnit, RefDest, SelectArea, CheckEnemy, DirNone)             // TARGET_UNIT_DEST_AREA_ENEMY
	set(17, ObjDest, RefCaster, SelectDefault, CheckDefault, DirNone)      // TARGET_DEST_DB
	set(18, ObjDest, RefCaster, SelectDefault, CheckDefault, DirNone)      // TARGET_DEST_CASTER
	set(20, ObjUnit, RefCaster, SelectArea, CheckParty, DirNone)           // TARGET_UNIT_CASTER_AREA_PARTY
	set(21, ObjUnit, RefTarget, SelectDefault, CheckAlly, DirNone)         // TARGET_UNIT_TARGET_ALLY
	set(22, ObjSrc, RefCaster, SelectDefault, CheckDefault, DirNone)       // TARGET_SRC_CASTER
	set(24, ObjUnit, RefCaster, SelectCone, CheckEnemy, DirFront)          // TARGET_UNIT_CONE_ENEMY_24
	set(25, ObjUnit, RefTarget, SelectDefault, CheckDefault, DirNone)      // TARGET_UNIT_TARGET_ANY
	set(27, ObjUnit, RefCaster, SelectDefault, CheckDefault, DirNone)      // TARGET_UNIT_MASTER
	set(28, ObjDest, RefDest, SelectDefault, CheckEnemy, DirNone)          // TARGET_DEST_DYNOBJ_ENEMY
	set(29, ObjDest, RefDest, SelectDefault, CheckAlly, DirNone)           // TARGET_DEST_DYNOBJ_ALLY
	set(30, ObjUnit, RefSrc, SelectArea, CheckAlly, DirNone)               // TARGET_UNIT_SRC_AREA_ALLY
	set(31, ObjUnit, RefDest, SelectArea, CheckAlly, DirNone)              // TARGET_UNIT_DEST_AREA_ALLY
	set(32, ObjDest, RefCaster, SelectDefault, CheckDefault, DirFrontLeft) // TARGET_DEST_CASTER_SUMMON
	set(33, ObjUnit, RefSrc, SelectArea, CheckParty, DirNone)              // TARGET_UNIT_SRC_AREA_PARTY
	set(34, ObjUnit, RefDest, SelectArea, CheckParty, DirNone)             // TARGET_UNIT_DEST_AREA_PARTY
	set(35, ObjUnit, RefTarget, SelectDefault, CheckParty, DirNone)        // TARGET_UNIT_TARGET_PARTY
	set(37, ObjUnit, RefLast, SelectArea, CheckParty, DirNone)             // TARGET_UNIT_LASTTARGET_AREA_PARTY
	set(38, ObjUnit, RefCaster, SelectNearby, CheckEntry, DirNone)         // TARGET_UNIT_NEARBY_ENTRY

	// Group H: Destination targets (ObjectType=DEST/SRC, Category=DEFAULT)
	set(41, ObjDest, RefCaster, SelectDefault, CheckDefault, DirFrontRight) // TARGET_DEST_CASTER_FRONT_RIGHT
	set(42, ObjDest, RefCaster, SelectDefault, CheckDefault, DirBackRight)  // TARGET_DEST_CASTER_BACK_RIGHT
	set(43, ObjDest, RefCaster, SelectDefault, CheckDefault, DirBackLeft)   // TARGET_DEST_CASTER_BACK_LEFT
	set(44, ObjDest, RefCaster, SelectDefault, CheckDefault, DirFrontLeft)  // TARGET_DEST_CASTER_FRONT_LEFT
	set(45, ObjUnit, RefTarget, SelectDefault, CheckAlly, DirNone)          // TARGET_UNIT_TARGET_CHAINHEAL_ALLY
	set(47, ObjDest, RefCaster, SelectDefault, CheckDefault, DirFront)      // TARGET_DEST_CASTER_FRONT
	set(48, ObjDest, RefCaster, SelectDefault, CheckDefault, DirBack)       // TARGET_DEST_CASTER_BACK
	set(49, ObjDest, RefCaster, SelectDefault, CheckDefault, DirRight)      // TARGET_DEST_CASTER_RIGHT
	set(50, ObjDest, RefCaster, SelectDefault, CheckDefault, DirLeft)       // TARGET_DEST_CASTER_LEFT
	set(53, ObjDest, RefTarget, SelectDefault, CheckEnemy, DirNone)         // TARGET_DEST_TARGET_ENEMY
	set(54, ObjUnit, RefCaster, SelectCone, CheckEnemy, DirFront)           // TARGET_UNIT_CONE_180_DEG_ENEMY
	set(55, ObjDest, RefCaster, SelectDefault, CheckDefault, DirNone)       // TARGET_DEST_CASTER_FRONT_LEAP
	set(56, ObjUnit, RefCaster, SelectArea, CheckRaid, DirNone)             // TARGET_UNIT_CASTER_AREA_RAID
	set(57, ObjUnit, RefTarget, SelectDefault, CheckRaid, DirNone)          // TARGET_UNIT_TARGET_RAID
	set(58, ObjUnit, RefCaster, SelectNearby, CheckRaid, DirNone)           // TARGET_UNIT_NEARBY_RAID
	set(59, ObjUnit, RefCaster, SelectCone, CheckAlly, DirFront)            // TARGET_UNIT_CONE_ALLY
	set(60, ObjUnit, RefCaster, SelectCone, CheckEntry, DirFront)           // TARGET_UNIT_CONE_ENTRY
	set(62, ObjDest, RefCaster, SelectDefault, CheckDefault, DirNone)       // TARGET_DEST_CASTER_GROUND
	set(63, ObjDest, RefTarget, SelectDefault, CheckDefault, DirNone)       // TARGET_DEST_TARGET_ANY
	set(64, ObjDest, RefTarget, SelectDefault, CheckDefault, DirFront)      // TARGET_DEST_TARGET_FRONT
	set(65, ObjDest, RefTarget, SelectDefault, CheckDefault, DirBack)       // TARGET_DEST_TARGET_BACK
	set(66, ObjDest, RefTarget, SelectDefault, CheckDefault, DirRight)      // TARGET_DEST_TARGET_RIGHT
	set(67, ObjDest, RefTarget, SelectDefault, CheckDefault, DirLeft)       // TARGET_DEST_TARGET_LEFT
	set(68, ObjDest, RefTarget, SelectDefault, CheckDefault, DirFrontRight) // TARGET_DEST_TARGET_FRONT_RIGHT
	set(69, ObjDest, RefTarget, SelectDefault, CheckDefault, DirBackRight)  // TARGET_DEST_TARGET_BACK_RIGHT
	set(70, ObjDest, RefTarget, SelectDefault, CheckDefault, DirBackLeft)   // TARGET_DEST_TARGET_BACK_LEFT
	set(71, ObjDest, RefTarget, SelectDefault, CheckDefault, DirFrontLeft)  // TARGET_DEST_TARGET_FRONT_LEFT
	set(72, ObjDest, RefCaster, SelectDefault, CheckDefault, DirRandom)     // TARGET_DEST_CASTER_RANDOM
	set(73, ObjDest, RefCaster, SelectDefault, CheckDefault, DirRandom)     // TARGET_DEST_CASTER_RADIUS
	set(74, ObjDest, RefTarget, SelectDefault, CheckDefault, DirRandom)     // TARGET_DEST_TARGET_RANDOM
	set(75, ObjDest, RefTarget, SelectDefault, CheckDefault, DirRandom)     // TARGET_DEST_TARGET_RADIUS

	// Group F: Channel targets
	set(76, ObjDest, RefCaster, SelectChannel, CheckDefault, DirNone) // TARGET_DEST_CHANNEL_TARGET
	set(77, ObjUnit, RefCaster, SelectChannel, CheckDefault, DirNone) // TARGET_UNIT_CHANNEL_TARGET

	// DestDest 方向变体
	set(78, ObjDest, RefDest, SelectDefault, CheckDefault, DirFront)      // TARGET_DEST_DEST_FRONT
	set(79, ObjDest, RefDest, SelectDefault, CheckDefault, DirBack)       // TARGET_DEST_DEST_BACK
	set(80, ObjDest, RefDest, SelectDefault, CheckDefault, DirRight)      // TARGET_DEST_DEST_RIGHT
	set(81, ObjDest, RefDest, SelectDefault, CheckDefault, DirLeft)       // TARGET_DEST_DEST_LEFT
	set(82, ObjDest, RefDest, SelectDefault, CheckDefault, DirFrontRight) // TARGET_DEST_DEST_FRONT_RIGHT
	set(83, ObjDest, RefDest, SelectDefault, CheckDefault, DirBackRight)  // TARGET_DEST_DEST_BACK_RIGHT
	set(84, ObjDest, RefDest, SelectDefault, CheckDefault, DirBackLeft)   // TARGET_DEST_DEST_BACK_LEFT
	set(85, ObjDest, RefDest, SelectDefault, CheckDefault, DirFrontLeft)  // TARGET_DEST_DEST_FRONT_LEFT
	set(86, ObjDest, RefDest, SelectDefault, CheckDefault, DirRandom)     // TARGET_DEST_DEST_RANDOM
	set(87, ObjDest, RefDest, SelectDefault, CheckDefault, DirNone)       // TARGET_DEST_DEST
	set(88, ObjDest, RefDest, SelectDefault, CheckDefault, DirNone)       // TARGET_DEST_DYNOBJ_NONE
	set(91, ObjDest, RefDest, SelectDefault, CheckDefault, DirRandom)     // TARGET_DEST_DEST_RADIUS
	set(92, ObjUnit, RefCaster, SelectDefault, CheckDefault, DirNone)     // TARGET_UNIT_SUMMONER

	// Cone/Rect 变体
	set(104, ObjUnit, RefCaster, SelectCone, CheckEnemy, DirFront)     // TARGET_UNIT_CONE_CASTER_TO_DEST_ENEMY
	set(115, ObjUnit, RefSrc, SelectArea, CheckEnemy, DirNone)         // TARGET_UNIT_SRC_AREA_FURTHEST_ENEMY
	set(118, ObjUnit, RefTarget, SelectArea, CheckRaid, DirNone)       // TARGET_UNIT_TARGET_ALLY_OR_RAID
	set(120, ObjUnit, RefCaster, SelectArea, CheckSummoned, DirNone)   // TARGET_UNIT_SELF_AND_SUMMONS
	set(128, ObjUnit, RefCaster, SelectCone, CheckAlly, DirFront)      // TARGET_UNIT_RECT_CASTER_ALLY
	set(129, ObjUnit, RefCaster, SelectCone, CheckEnemy, DirFront)     // TARGET_UNIT_RECT_CASTER_ENEMY
	set(130, ObjUnit, RefCaster, SelectCone, CheckDefault, DirFront)   // TARGET_UNIT_RECT_CASTER
	set(131, ObjDest, RefCaster, SelectDefault, CheckDefault, DirNone) // TARGET_DEST_SUMMONER
	set(132, ObjDest, RefTarget, SelectDefault, CheckAlly, DirNone)    // TARGET_DEST_TARGET_ALLY

	// Group E: Line targets
	set(133, ObjUnit, RefDest, SelectLine, CheckAlly, DirNone)    // TARGET_UNIT_LINE_CASTER_TO_DEST_ALLY
	set(134, ObjUnit, RefDest, SelectLine, CheckEnemy, DirNone)   // TARGET_UNIT_LINE_CASTER_TO_DEST_ENEMY
	set(135, ObjUnit, RefDest, SelectLine, CheckDefault, DirNone) // TARGET_UNIT_LINE_CASTER_TO_DEST
	set(136, ObjUnit, RefCaster, SelectCone, CheckAlly, DirFront) // TARGET_UNIT_CONE_CASTER_TO_DEST_ALLY
}

// set 设置指定索引的 StaticData。
func set(idx int, objType ObjectTypes, refType ReferenceTypes, category SelectionCategory, checkType CheckTypes, dirType DirectionTypes) {
	if idx < 0 || idx >= MaxImplicitTarget {
		return
	}
	targetData[idx] = StaticData{
		ObjectType:    objType,
		ReferenceType: refType,
		Category:      category,
		CheckType:     checkType,
		DirectionType: dirType,
	}
}
