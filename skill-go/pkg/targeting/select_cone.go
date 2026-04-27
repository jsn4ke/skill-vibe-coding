package targeting

import "math"

// SearchConeTargets 在扇形范围内搜索目标，对齐 TC 的 SpellImplicitTargetInfo::SelectTypeCone。
// 在 Area 基础上增加角度过滤：候选相对施法者朝向的角度必须在 [direction - arcAngle/2, direction + arcAngle/2] 范围内。
func SearchConeTargets(center [3]float64, direction float64, arcAngle float64, radius float64, check CheckTypes, caster TargetUnit, spell SpellTargetRef, excludeID uint64) []TargetUnit {
	candidates := spell.GetUnitsInRadius(center, radius, excludeID)
	var result []TargetUnit
	for _, c := range candidates {
		if !passesCheck(check, caster, c) {
			continue
		}
		// 计算候选相对中心的角度
		cp := c.GetPosition()
		dx := cp.GetX() - center[0]
		dy := cp.GetY() - center[1]
		angle := math.Atan2(dy, dx)

		// 判断角度是否在扇形范围内
		if isAngleInCone(angle, direction, arcAngle) {
			result = append(result, c)
		}
	}
	return result
}

// isAngleInCone 判断角度是否在扇形范围内，对齐 TC 的 IsInArc 检查。
func isAngleInCone(angle, direction, arcAngle float64) bool {
	halfArc := arcAngle / 2
	diff := angle - direction
	// 归一化到 [-π, π]
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	return math.Abs(diff) <= halfArc
}
