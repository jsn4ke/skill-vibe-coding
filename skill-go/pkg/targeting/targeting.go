package targeting

import (
	"math"

	"skill-go/pkg/entity"
	"skill-go/pkg/spell"
)

type SelectionCategory uint8

const (
	SelectDefault SelectionCategory = iota
	SelectNearby
	SelectCone
	SelectArea
	SelectChain
	SelectLine
)

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

type Descriptor struct {
	Selection SelectionCategory
	Check     CheckType
	ObjType   uint8
	Radius    float64
	MaxTargets int32
	ConeAngle  float64
}

type TargetSelector struct {
	entities []Targetable
}

type Targetable interface {
	GetEntity() *entity.Entity
	IsAlive() bool
}

func NewSelector(entities []Targetable) *TargetSelector {
	return &TargetSelector{entities: entities}
}

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

func (ts *TargetSelector) selectDefault(candidates []*entity.Entity, desc Descriptor) []*entity.Entity {
	if desc.MaxTargets > 0 && int(desc.MaxTargets) < len(candidates) {
		return candidates[:desc.MaxTargets]
	}
	return candidates
}

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
