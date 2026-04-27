package targeting

import (
	"math"

	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
)

// SelectionCategory 表示目标选择方式的分类。
type SelectionCategory uint8

const (
	SelectDefault SelectionCategory = iota
	SelectNearby
	SelectCone
	SelectArea
	SelectChain
	SelectLine
)

// CheckType 表示目标检查类型（敌我关系）。
type CheckType uint8

const (
	CheckDefault CheckType = iota
	CheckEnemy
	CheckAlly
	CheckParty
	CheckRaid
	CheckSummoned
	CheckAny
)

// Descriptor 描述目标选择的参数。
type Descriptor struct {
	Selection SelectionCategory
	Check     CheckType
	ObjType   uint8
	Radius    float64
	MaxTargets int32
	ConeAngle  float64
}

// TargetSelector 根据描述符选择目标。
type TargetSelector struct {
	entities []Targetable
}

// Targetable 是可被选为目标的实体接口。
type Targetable interface {
	GetEntity() *entity.Entity
	IsAlive() bool
}

// NewSelector 创建一个目标选择器。
func NewSelector(entities []Targetable) *TargetSelector {
	return &TargetSelector{entities: entities}
}

// Select 根据施法者、描述符和排除 ID 选择目标。
func (ts *TargetSelector) Select(caster *entity.Entity, desc Descriptor, excludeID uint64) []*entity.Entity {
	var candidates []*entity.Entity

	for _, t := range ts.entities {
		e := t.GetEntity()
		if !t.IsAlive() {
			continue
		}
		if uint64(e.ID) == excludeID {
			continue
		}
		if !ts.passesCheck(caster, e, desc.Check) {
			continue
		}
		candidates = append(candidates, e)
	}

	switch desc.Selection {
	case SelectDefault:
		return ts.selectDefault(candidates, desc)
	case SelectNearby:
		return ts.selectNearby(caster, candidates, desc)
	case SelectArea:
		return ts.selectArea(caster, candidates, desc)
	case SelectCone:
		return ts.selectCone(caster, candidates, desc)
	case SelectChain:
		return ts.selectChain(caster, candidates, desc)
	default:
		return ts.selectDefault(candidates, desc)
	}
}

// passesCheck 检查目标是否通过敌我关系过滤。
func (ts *TargetSelector) passesCheck(caster, target *entity.Entity, check CheckType) bool {
	switch check {
	case CheckEnemy:
		return caster.Type != target.Type
	case CheckAlly:
		return caster.Type == target.Type
	case CheckAny:
		return true
	default:
		return true
	}
}

// selectDefault 选择默认目标（截取到 MaxTargets）。
func (ts *TargetSelector) selectDefault(candidates []*entity.Entity, desc Descriptor) []*entity.Entity {
	if desc.MaxTargets > 0 && int(desc.MaxTargets) < len(candidates) {
		return candidates[:desc.MaxTargets]
	}
	return candidates
}

// selectNearby 选择施法者附近的存活目标，按距离排序。
func (ts *TargetSelector) selectNearby(caster *entity.Entity, candidates []*entity.Entity, desc Descriptor) []*entity.Entity {
	var result []*entity.Entity
	for _, e := range candidates {
		dist := caster.Pos.DistanceTo(e.Pos)
		if dist <= desc.Radius {
			result = append(result, e)
		}
	}
	sortByDistance(result, caster)
	if desc.MaxTargets > 0 && int(desc.MaxTargets) < len(result) {
		result = result[:desc.MaxTargets]
	}
	return result
}

// selectArea 选择施法者区域内的目标（不排序）。
func (ts *TargetSelector) selectArea(caster *entity.Entity, candidates []*entity.Entity, desc Descriptor) []*entity.Entity {
	var result []*entity.Entity
	for _, e := range candidates {
		dist := caster.Pos.DistanceTo(e.Pos)
		if dist <= desc.Radius {
			result = append(result, e)
		}
	}
	if desc.MaxTargets > 0 && int(desc.MaxTargets) < len(result) {
		result = result[:desc.MaxTargets]
	}
	return result
}

// selectCone 选择施法者前方锥形区域内的目标。
func (ts *TargetSelector) selectCone(caster *entity.Entity, candidates []*entity.Entity, desc Descriptor) []*entity.Entity {
	var result []*entity.Entity
	halfAngle := desc.ConeAngle / 2.0
	if halfAngle == 0 {
		halfAngle = math.Pi / 4
	}
	for _, e := range candidates {
		dist := caster.Pos.DistanceTo(e.Pos)
		if dist > desc.Radius {
			continue
		}
		if isInCone(caster.Pos, e.Pos, halfAngle) {
			result = append(result, e)
		}
	}
	sortByDistance(result, caster)
	return result
}

// selectChain 选择链式目标，按距离排序后截取。
func (ts *TargetSelector) selectChain(caster *entity.Entity, candidates []*entity.Entity, desc Descriptor) []*entity.Entity {
	sortByDistance(candidates, caster)
	maxTargets := int(desc.MaxTargets)
	if maxTargets <= 0 {
		maxTargets = 1
	}
	if maxTargets > len(candidates) {
		maxTargets = len(candidates)
	}
	return candidates[:maxTargets]
}

// isInCone 判断目标是否在原点的锥形范围内。
func isInCone(origin entity.Position, target entity.Position, halfAngle float64) bool {
	dx := target.X - origin.X
	dy := target.Y - origin.Y
	angle := math.Atan2(dy, dx)
	diff := angle - origin.Facing
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	return math.Abs(diff) <= halfAngle
}

// sortByDistance 使用插入排序按到原点的二维距离排序。
func sortByDistance(entities []*entity.Entity, origin *entity.Entity) {
	for i := 1; i < len(entities); i++ {
		for j := i; j > 0; j-- {
			di := origin.Pos.DistanceTo2D(entities[j].Pos)
			dj := origin.Pos.DistanceTo2D(entities[j-1].Pos)
			if di < dj {
				entities[j], entities[j-1] = entities[j-1], entities[j]
			}
		}
	}
}

// SelectAroundPoint 选择指定中心点周围的目标，用于 AoE 法术的目标选择。
func (ts *TargetSelector) SelectAroundPoint(caster *entity.Entity, center entity.Position, candidates []Targetable, desc Descriptor, excludeID uint64) []*entity.Entity {
	var result []*entity.Entity
	for _, t := range candidates {
		e := t.GetEntity()
		if !t.IsAlive() {
			continue
		}
		if uint64(e.ID) == excludeID {
			continue
		}
		if caster != nil && !ts.passesCheck(caster, e, desc.Check) {
			continue
		}
		dist := center.DistanceTo(e.Pos)
		if dist <= desc.Radius {
			result = append(result, e)
		}
	}
	if desc.MaxTargets > 0 && int(desc.MaxTargets) < len(result) {
		result = result[:desc.MaxTargets]
	}
	return result
}

// DescriptorFromTarget 将隐式目标类型转换为目标选择描述符。
func DescriptorFromTarget(target spell.ImplicitTarget) Descriptor {
	switch target {
	case spell.TargetUnitTargetEnemy:
		return Descriptor{Selection: SelectDefault, Check: CheckEnemy, MaxTargets: 1}
	case spell.TargetUnitTargetAlly:
		return Descriptor{Selection: SelectDefault, Check: CheckAlly, MaxTargets: 1}
	case spell.TargetUnitTargetAny:
		return Descriptor{Selection: SelectDefault, Check: CheckAny, MaxTargets: 1}
	case spell.TargetUnitCaster:
		return Descriptor{Selection: SelectDefault, Check: CheckDefault, MaxTargets: 1}
	case spell.TargetUnitNearbyEnemy:
		return Descriptor{Selection: SelectNearby, Check: CheckEnemy, MaxTargets: 1}
	case spell.TargetUnitNearbyAlly:
		return Descriptor{Selection: SelectNearby, Check: CheckAlly, MaxTargets: 1}
	case spell.TargetUnitAreaEnemy:
		return Descriptor{Selection: SelectArea, Check: CheckEnemy}
	case spell.TargetUnitAreaAlly:
		return Descriptor{Selection: SelectArea, Check: CheckAlly}
	case spell.TargetUnitConeEnemy:
		return Descriptor{Selection: SelectCone, Check: CheckEnemy}
	case spell.TargetUnitChainEnemy:
		return Descriptor{Selection: SelectChain, Check: CheckEnemy, MaxTargets: 5}
	default:
		return Descriptor{Selection: SelectDefault, MaxTargets: 1}
	}
}
