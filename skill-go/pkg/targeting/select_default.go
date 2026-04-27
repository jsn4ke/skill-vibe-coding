package targeting

import "math"

// ResolveDefaultTargets 处理 SelectionCategory=DEFAULT 的目标选择，对齐 TC 的 SpellImplicitTargetInfo::SelectTypeDefault。
// DEFAULT 分支不做空间搜索，直接根据 ObjectType 和 ReferenceType 解析目标。
func ResolveDefaultTargets(info ImplicitTargetInfo, spell SpellTargetRef) []TargetUnit {
	objType := info.GetObjectType()
	refType := info.GetReferenceType()
	dirType := info.GetDirectionType()

	switch objType {
	case ObjUnit:
		// 单位类型：直接解析参考单位
		u := ResolveReferer(refType, spell)
		if u != nil {
			return []TargetUnit{u}
		}
		return nil

	case ObjDest, ObjSrc:
		// 位置类型：根据方向偏移生成 DestPos
		pos := ResolveCenter(refType, spell)
		if dirType != DirNone {
			pos = ApplyDirectionOffset(pos, dirType, spell)
		}
		// DEFAULT + DEST/SRC 类型设置目标位置，不返回单位
		// 位置设置由调用者（spell 包）处理
		return nil

	case ObjUnitAndDest:
		// 单位 + 位置：解析参考单位，同时设置位置
		u := ResolveReferer(refType, spell)
		if u != nil {
			return []TargetUnit{u}
		}
		return nil

	default:
		return nil
	}
}

// ApplyDirectionOffset 在参考位置上应用方向偏移，对齐 TC 的 Spell::CalculateDestOffset。
func ApplyDirectionOffset(pos [3]float64, dirType DirectionTypes, spell SpellTargetRef) [3]float64 {
	angle := CalcDirectionAngle(dirType)
	if angle == 0.0 && dirType != DirFront {
		return pos
	}

	// 使用施法者朝向作为基础方向，对齐 TC 的 m_caster->GetOrientation()
	var facing float64
	if caster := spell.GetCaster(); caster != nil {
		facing = caster.GetPosition().GetFacing()
	}

	// 计算世界坐标方向角 = 施法者朝向 + 方向偏移
	worldAngle := facing + angle

	// 默认偏移距离，对齐 TC 的 DEFAULT_RADIUS (0.0 — 方向仅用于 DestPos 生成)
	// 实际偏移距离由 SpellInfo 的 Effects[].Radius 或其他字段决定
	offsetDist := 0.0

	return [3]float64{
		pos[0] + offsetDist*math.Cos(worldAngle),
		pos[1] + offsetDist*math.Sin(worldAngle),
		pos[2],
	}
}

// CalcDirectionAngle 将 DirectionTypes 转换为弧度偏移，公开供外部使用。
func CalcDirectionAngle(dir DirectionTypes) float64 {
	switch dir {
	case DirFront:
		return 0.0
	case DirBack:
		return math.Pi
	case DirRight:
		return -math.Pi / 2
	case DirLeft:
		return math.Pi / 2
	case DirFrontRight:
		return -math.Pi / 4
	case DirBackRight:
		return -3 * math.Pi / 4
	case DirBackLeft:
		return 3 * math.Pi / 4
	case DirFrontLeft:
		return math.Pi / 4
	default:
		return 0.0
	}
}
