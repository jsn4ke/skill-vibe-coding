package targeting

import "math"

// SearchLineTargets 在线形范围内搜索目标，对齐 TC 的 SpellImplicitTargetInfo::SelectTypeLine。
// 计算候选到线段的距离，在 width/2 范围内的目标被选中。
func SearchLineTargets(from [3]float64, to [3]float64, width float64, check CheckTypes, caster TargetUnit, spell SpellTargetRef, excludeID uint64) []TargetUnit {
	// 线段长度作为搜索半径
	dx := to[0] - from[0]
	dy := to[1] - from[1]
	dz := to[2] - from[2]
	lineLen := math.Sqrt(dx*dx + dy*dy + dz*dz)

	// 先用线段长度作为搜索半径获取候选
	candidates := spell.GetUnitsInRadius(from, lineLen+width, excludeID)
	var result []TargetUnit
	halfWidth := width / 2

	for _, c := range candidates {
		if !passesCheck(check, caster, c) {
			continue
		}
		cp := c.GetPosition()
		dist := pointToSegmentDistance2D(
			cp.GetX(), cp.GetY(),
			from[0], from[1],
			to[0], to[1],
		)
		if dist <= halfWidth {
			result = append(result, c)
		}
	}
	return result
}

// pointToSegmentDistance2D 计算点到线段的二维距离。
func pointToSegmentDistance2D(px, py, ax, ay, bx, by float64) float64 {
	abx := bx - ax
	aby := by - ay
	apx := px - ax
	apy := py - ay

	abLenSq := abx*abx + aby*aby
	if abLenSq == 0 {
		// 线段退化为点
		return math.Sqrt(apx*apx + apy*apy)
	}

	// 投影参数 t，限制在 [0, 1]
	t := (apx*abx + apy*aby) / abLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	// 最近点
	closestX := ax + t*abx
	closestY := ay + t*aby
	dx := px - closestX
	dy := py - closestY
	return math.Sqrt(dx*dx + dy*dy)
}
