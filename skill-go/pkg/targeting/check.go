package targeting

// passesCheck 判断候选单位是否通过友敌过滤条件，对齐 TC 的 WorldObject::IsTargetMatching。
func passesCheck(check CheckTypes, caster TargetUnit, candidate TargetUnit) bool {
	if candidate == nil || !candidate.IsAlive() {
		return false
	}

	switch check {
	case CheckDefault:
		return true
	case CheckEnemy:
		// 不同 EntityType 视为敌方，对齐 TC 的 IsValidAttackTarget 逻辑
		return caster.GetEntityType() != candidate.GetEntityType()
	case CheckAlly:
		// 相同 EntityType 视为友方
		return caster.GetEntityType() == candidate.GetEntityType()
	case CheckParty:
		// 当前 fallback 为 Ally，后续实现队伍系统后替换
		return caster.GetEntityType() == candidate.GetEntityType()
	case CheckRaid:
		// 当前 fallback 为 Ally，后续实现团队系统后替换
		return caster.GetEntityType() == candidate.GetEntityType()
	case CheckSummoned:
		// 召唤物检查：候选单位的召唤者 ID 等于施法者 ID
		// 当前简化实现：同 EntityType 视为友方（召唤物系统待实现）
		return caster.GetEntityType() == candidate.GetEntityType()
	case CheckEntry:
		// 按 condition entry 过滤，当前 fallback 为 true
		return true
	default:
		return true
	}
}
