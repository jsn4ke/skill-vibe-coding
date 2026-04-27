package targeting

import "math"

// SearchChainTargets 实现 Chain 跳跃搜索，对齐 TC 的 ChainHeal/ChainLightning 跳跃模型。
// 从 initial 开始，每次在 jumpRadius 内找最近/最优目标，更新 chainSource 继续跳跃。
func SearchChainTargets(initial TargetUnit, maxJumps int, jumpRadius float64, check CheckTypes, caster TargetUnit, spell SpellTargetRef, excludeIDs []uint64) []TargetUnit {
	if initial == nil || maxJumps <= 0 {
		return nil
	}

	result := []TargetUnit{initial}
	excluded := make(map[uint64]bool)
	for _, id := range excludeIDs {
		excluded[id] = true
	}
	excluded[initial.GetID()] = true

	chainSource := initial

	for i := 0; i < maxJumps; i++ {
		// 获取 chainSource 位置作为搜索中心
		srcPos := chainSource.GetPosition()
		center := [3]float64{srcPos.GetX(), srcPos.GetY(), srcPos.GetZ()}

		// 在 jumpRadius 内搜索候选
		candidates := spell.GetUnitsInRadius(center, jumpRadius, 0)

		var best TargetUnit
		bestDist := math.MaxFloat64

		for _, c := range candidates {
			if excluded[c.GetID()] {
				continue
			}
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

		if best == nil {
			break
		}

		result = append(result, best)
		excluded[best.GetID()] = true
		chainSource = best
	}

	return result
}
