package targeting

import "math"

// SearchNearbyTarget 在指定中心点的球形范围内找最近的一个合法目标，对齐 TC 的 SpellImplicitTargetInfo::SelectTypeNearby。
func SearchNearbyTarget(center [3]float64, radius float64, check CheckTypes, caster TargetUnit, spell SpellTargetRef, excludeID uint64) TargetUnit {
	candidates := spell.GetUnitsInRadius(center, radius, excludeID)

	var best TargetUnit
	bestDist := math.MaxFloat64

	for _, c := range candidates {
		if !passesCheck(check, caster, c) {
			continue
		}
		cp := c.GetPosition()
		dx := cp.GetX() - center[0]
		dy := cp.GetY() - center[1]
		dz := cp.GetZ() - center[2]
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist < bestDist {
			bestDist = dist
			best = c
		}
	}
	return best
}
