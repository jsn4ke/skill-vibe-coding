package targeting

// SelectChannelTargets 从当前 channel spell 获取目标，对齐 TC 的 SpellImplicitTargetInfo::SelectTypeChannel。
// Channel 目标来自正在引导的法术的目标列表。
func SelectChannelTargets(info ImplicitTargetInfo, spell SpellTargetRef) []TargetUnit {
	objType := info.GetObjectType()

	switch objType {
	case ObjDest:
		// TARGET_DEST_CHANNEL_TARGET: 返回引导法术的目标位置
		// 当前简化实现：使用 UnitTargetID 的位置
		if id := spell.GetUnitTargetID(); id != 0 {
			if u := spell.GetUnitByID(id); u != nil {
				return []TargetUnit{u}
			}
		}
		return nil

	case ObjUnit:
		// TARGET_UNIT_CHANNEL_TARGET: 返回引导法术的目标单位
		if id := spell.GetUnitTargetID(); id != 0 {
			if u := spell.GetUnitByID(id); u != nil {
				return []TargetUnit{u}
			}
		}
		return nil

	default:
		return nil
	}
}
