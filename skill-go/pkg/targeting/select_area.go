package targeting

// SearchAreaTargets 在指定中心点的球形范围内搜索目标，对齐 TC 的 SpellImplicitTargetInfo::SelectTypeArea。
func SearchAreaTargets(center [3]float64, radius float64, check CheckTypes, caster TargetUnit, spell SpellTargetRef, excludeID uint64) []TargetUnit {
	candidates := spell.GetUnitsInRadius(center, radius, excludeID)
	var result []TargetUnit
	for _, c := range candidates {
		if passesCheck(check, caster, c) {
			result = append(result, c)
		}
	}
	return result
}
