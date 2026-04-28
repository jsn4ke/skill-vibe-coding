package targeting

import (
	"math"
	"math/rand"
)

// ObjectTypes 对齐 TC 的 SpellTargetObjectTypes，表示隐式目标返回的对象类型。
type ObjectTypes uint8

const (
	ObjNone        ObjectTypes = iota // 无对象
	ObjSrc                            // 位置引用（源）
	ObjDest                           // 位置引用（目标）
	ObjUnit                           // 单位
	ObjUnitAndDest                    // 单位 + 位置
	ObjGobj                           // 游戏对象（暂不实现）
	ObjGobjItem                       // 游戏对象+物品（暂不实现）
	ObjItem                           // 物品（暂不实现）
	ObjCorpseEnemy                    // 敌方尸体（暂不实现）
	ObjCorpseAlly                     // 友方尸体（暂不实现）
	ObjCorpse                         // 尸体（暂不实现）
)

// ReferenceTypes 对齐 TC 的 SpellTargetReferenceTypes，决定目标选择的参考原点。
type ReferenceTypes uint8

const (
	RefNone   ReferenceTypes = iota // 无参考
	RefCaster                       // 施法者
	RefTarget                       // 当前选中的目标单位
	RefLast                         // 上一个加入 TargetInfo 的目标
	RefSrc                          // SpellTargets 的 SourcePos
	RefDest                         // SpellTargets 的 DestPos
)

// SelectionCategory 对齐 TC 的 SpellTargetSelectionCategories，驱动目标选择算法分发。
type SelectionCategory uint8

const (
	SelectNYI     SelectionCategory = iota // 未实现
	SelectDefault                          // 直接引用（caster/target/dest），不做空间搜索
	SelectChannel                          // 从当前 channel spell 取目标
	SelectNearby                           // 空间搜索：找最近的一个目标
	SelectCone                             // 空间搜索：扇形范围内所有目标
	SelectArea                             // 空间搜索：球形范围内所有目标
	SelectTraj                             // 抛物线碰撞（暂不实现）
	SelectLine                             // 空间搜索：线形范围内所有目标
)

// CheckTypes 对齐 TC 的 SpellTargetCheckTypes，控制友敌过滤条件。
type CheckTypes uint8

const (
	CheckDefault   CheckTypes = iota // 无特殊过滤
	CheckEnemy                       // 敌方
	CheckAlly                        // 友方
	CheckParty                       // 队友（当前 fallback 为 Ally）
	CheckRaid                        // 团队成员（当前 fallback 为 Ally）
	CheckRaidClass                   // 同职业团队成员（暂不实现）
	CheckPassenger                   // 乘客（暂不实现）
	CheckSummoned                    // 召唤物
	CheckEntry                       // 按 condition entry 过滤（暂不实现）
)

// DirectionTypes 对齐 TC 的 SpellTargetDirectionTypes，控制扇形和位置方向。
type DirectionTypes uint8

const (
	DirNone       DirectionTypes = iota // 无方向
	DirFront                            // 正前方 (0°)
	DirBack                             // 正后方 (π)
	DirRight                            // 右方 (-π/2)
	DirLeft                             // 左方 (π/2)
	DirFrontRight                       // 右前方 (-π/4)
	DirBackRight                        // 右后方 (-3π/4)
	DirBackLeft                         // 左后方 (3π/4)
	DirFrontLeft                        // 左前方 (π/4)
	DirRandom                           // 随机方向 [0, 2π)
	DirEntry                            // 由 condition entry 决定（暂不实现）
)

// StaticData 对齐 TC 的 SpellImplicitTargetInfo::StaticData，
// 每个 ImplicitTarget 值通过查表获得 5 维属性。
type StaticData struct {
	ObjectType    ObjectTypes
	ReferenceType ReferenceTypes
	Category      SelectionCategory
	CheckType     CheckTypes
	DirectionType DirectionTypes
}

// ImplicitTargetInfo 包装 ImplicitTarget 值，提供查表访问方法。
// 对齐 TC 的 SpellImplicitTargetInfo。
// 使用 uint16 而非 spell.ImplicitTarget 以避免循环导入。
type ImplicitTargetInfo struct {
	target uint16
}

// NewImplicitTargetInfo 创建 ImplicitTargetInfo。
func NewImplicitTargetInfo(target uint16) ImplicitTargetInfo {
	return ImplicitTargetInfo{target: target}
}

// GetTarget 返回原始 ImplicitTarget 值。
func (iti ImplicitTargetInfo) GetTarget() uint16 {
	return iti.target
}

// GetObjectType 返回目标返回的对象类型。
func (iti ImplicitTargetInfo) GetObjectType() ObjectTypes {
	idx := int(iti.target)
	if idx < 0 || idx >= len(targetData) {
		return ObjNone
	}
	return targetData[idx].ObjectType
}

// GetReferenceType 返回目标选择的参考原点类型。
func (iti ImplicitTargetInfo) GetReferenceType() ReferenceTypes {
	idx := int(iti.target)
	if idx < 0 || idx >= len(targetData) {
		return RefNone
	}
	return targetData[idx].ReferenceType
}

// GetSelectionCategory 返回目标选择的算法分类。
func (iti ImplicitTargetInfo) GetSelectionCategory() SelectionCategory {
	idx := int(iti.target)
	if idx < 0 || idx >= len(targetData) {
		return SelectNYI
	}
	return targetData[idx].Category
}

// GetCheckType 返回友敌过滤条件。
func (iti ImplicitTargetInfo) GetCheckType() CheckTypes {
	idx := int(iti.target)
	if idx < 0 || idx >= len(targetData) {
		return CheckDefault
	}
	return targetData[idx].CheckType
}

// GetDirectionType 返回方向类型。
func (iti ImplicitTargetInfo) GetDirectionType() DirectionTypes {
	idx := int(iti.target)
	if idx < 0 || idx >= len(targetData) {
		return DirNone
	}
	return targetData[idx].DirectionType
}

// CalcDirectionAngle 将 DirectionType 转换为弧度偏移，对齐 TC 的 CalcDirectionAngle。
func (iti ImplicitTargetInfo) CalcDirectionAngle() float64 {
	switch iti.GetDirectionType() {
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
	case DirRandom:
		return rand.Float64() * 2 * math.Pi
	default:
		return 0.0
	}
}

// IsArea 判断目标是否为区域效果（Area 或 Cone 分类），对齐 TC 的 IsArea。
func (iti ImplicitTargetInfo) IsArea() bool {
	cat := iti.GetSelectionCategory()
	return cat == SelectArea || cat == SelectCone
}

// MaxImplicitTarget 是 ImplicitTarget 查表数组的最大索引。
// 对齐 TC 的 TOTAL_SPELL_TARGETS (152)，预留空间便于后续扩展。
const MaxImplicitTarget = 153
