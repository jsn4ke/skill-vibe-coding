package diminishing

import (
	"time"
)

// Group 表示递减返回的分组类型，对齐 TC 的 DiminishingGroup。
type Group uint8

const (
	GroupNone      Group = iota // 无递减分组
	GroupStun                   // 昏迷
	GroupFear                   // 恐惧
	GroupRoot                   // 定身
	GroupSilence                // 沉默
	GroupCharm                  // 魅惑
	GroupConfuse                // 迷惑
	GroupDisarm                 // 缴械
	GroupBanish                 // 放逐
	GroupKnockback              // 击退
	GroupPolymorph              // 变形
	GroupSlow                   // 减速
)

// ReturnType 表示递减返回的结果类型。
type ReturnType uint8

const (
	ReturnNone     ReturnType = iota // 无递减效果
	ReturnStandard                   // 全时长（标准）
	ReturnHalf                       // 半时长
	ReturnQuarter                    // 四分之一时长
	ReturnImmune                     // 免疫
)

// Level 定义一个递减分组的规则，包含最大层数和持续时间上限。
type Level struct {
	Group      Group
	ReturnType ReturnType
	MaxLevel   uint8
	DurLimit   time.Duration
}

// Entry 记录一次递减效果的应用，用于追踪递减层数。
type Entry struct {
	Group    Group
	CasterID uint64
	TargetID uint64
	ApplyAt  time.Time
	Level    uint8
	MaxDur   time.Duration
}

// Manager 管理递减返回系统，对齐 TC 的 DiminishingReturnManager。
type Manager struct {
	levels  map[Group]Level
	entries map[uint64][]Entry
}

// NewManager 创建一个空的递减返回管理器。
func NewManager() *Manager {
	return &Manager{
		levels:  make(map[Group]Level),
		entries: make(map[uint64][]Entry),
	}
}

// RegisterLevel 注册一个递减分组的规则。
func (m *Manager) RegisterLevel(level Level) {
	m.levels[level.Group] = level
}

// ApplyDiminishing 对目标应用递减返回计算，返回修改后的持续时间和是否免疫。
// 递减规则：第 0 层全时长，第 1 层半时长，第 2 层四分之一时长，第 3 层及以上免疫。
// 18 秒内的同组应用计入递减层数，对齐 TC 的 DR 窗口。
func (m *Manager) ApplyDiminishing(targetID, casterID uint64, group Group, duration time.Duration) (time.Duration, bool) {
	if group == GroupNone {
		return duration, false
	}

	level, ok := m.levels[group]
	if !ok {
		return duration, false
	}

	entries := m.entries[targetID]
	now := time.Now()

	var active []Entry
	for _, e := range entries {
		if e.Group == group && now.Sub(e.ApplyAt) < 18*time.Second {
			active = append(active, e)
		}
	}

	currentLevel := uint8(len(active))
	if level.MaxLevel > 0 && currentLevel >= level.MaxLevel {
		return 0, true
	}

	var modifiedDuration time.Duration
	switch {
	case currentLevel == 0:
		modifiedDuration = duration
	case currentLevel == 1:
		modifiedDuration = duration / 2
	case currentLevel == 2:
		modifiedDuration = duration / 4
	default:
		return 0, true
	}

	if level.DurLimit > 0 && modifiedDuration > level.DurLimit {
		modifiedDuration = level.DurLimit
	}

	newEntry := Entry{
		Group:    group,
		CasterID: casterID,
		TargetID: targetID,
		ApplyAt:  now,
		Level:    currentLevel,
		MaxDur:   modifiedDuration,
	}
	m.entries[targetID] = append(entries, newEntry)

	return modifiedDuration, false
}

// GetLevel 获取目标在指定分组中的当前递减层数。
func (m *Manager) GetLevel(targetID uint64, group Group) uint8 {
	entries := m.entries[targetID]
	now := time.Now()
	var count uint8
	for _, e := range entries {
		if e.Group == group && now.Sub(e.ApplyAt) < 18*time.Second {
			count++
		}
	}
	return count
}

// Clear 清除目标的所有递减记录。
func (m *Manager) Clear(targetID uint64) {
	delete(m.entries, targetID)
}
