package targeting

// TargetUnit 提供目标选择所需的最小单位接口。
// 避免导入 unit 包导致循环依赖。
type TargetUnit interface {
	GetID() uint64
	GetPosition() TargetPosition
	GetEntityType() uint8
	IsAlive() bool
}

// TargetPosition 提供位置信息接口。
type TargetPosition interface {
	GetX() float64
	GetY() float64
	GetZ() float64
	GetFacing() float64
}

// SpellTargetRef 提供目标解析所需的最小法术数据接口。
// 由 spell.Spell 实现，避免 targeting → spell 循环导入。
type SpellTargetRef interface {
	// GetCaster 返回施法者（作为 TargetUnit）
	GetCaster() TargetUnit
	// GetUnitTargetID 返回当前选中的单位目标 ID
	GetUnitTargetID() uint64
	// GetSourcePos 返回法术源位置
	GetSourcePos() [3]float64
	// GetDestPos 返回法术目标位置
	GetDestPos() [3]float64
	// GetLastTargetID 返回上一个加入 TargetInfos 的目标 ID
	GetLastTargetID() uint64
	// GetUnitByID 按 ID 获取单位，委托给引擎
	GetUnitByID(id uint64) TargetUnit
	// GetUnitsInRadius 获取指定半径内的所有单位，委托给引擎
	GetUnitsInRadius(center [3]float64, radius float64, excludeID uint64) []TargetUnit
}

// ResolveCenter 根据 ReferenceType 解析搜索中心位置，对齐 TC 的 Spell::CalculateSearcherData。
func ResolveCenter(ref ReferenceTypes, spell SpellTargetRef) [3]float64 {
	switch ref {
	case RefCaster:
		if u := spell.GetCaster(); u != nil {
			p := u.GetPosition()
			return [3]float64{p.GetX(), p.GetY(), p.GetZ()}
		}
	case RefTarget:
		if id := spell.GetUnitTargetID(); id != 0 {
			if u := spell.GetUnitByID(id); u != nil {
				p := u.GetPosition()
				return [3]float64{p.GetX(), p.GetY(), p.GetZ()}
			}
		}
	case RefSrc:
		return spell.GetSourcePos()
	case RefDest:
		return spell.GetDestPos()
	case RefLast:
		if id := spell.GetLastTargetID(); id != 0 {
			if u := spell.GetUnitByID(id); u != nil {
				p := u.GetPosition()
				return [3]float64{p.GetX(), p.GetY(), p.GetZ()}
			}
		}
	}
	return [3]float64{}
}

// ResolveReferer 根据 ReferenceType 解析参考单位，对齐 TC 的 Spell::CalculateSearcherData。
func ResolveReferer(ref ReferenceTypes, spell SpellTargetRef) TargetUnit {
	switch ref {
	case RefCaster:
		return spell.GetCaster()
	case RefTarget:
		if id := spell.GetUnitTargetID(); id != 0 {
			return spell.GetUnitByID(id)
		}
	case RefLast:
		if id := spell.GetLastTargetID(); id != 0 {
			return spell.GetUnitByID(id)
		}
	}
	return nil
}
