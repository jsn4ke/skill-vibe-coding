package cooldown

import (
	"sync"
	"time"

	"skill-go/pkg/spell"
)

// CooldownEntry 表示一个冷却记录，对齐 TC 的 CooldownEntry。
type CooldownEntry struct {
	SpellID    spell.SpellID
	CategoryID uint32
	EndTime    time.Time
	OnHold     bool
}

// ChargeEntry 表示一个充能条目，记录充能的开始和结束时间。
type ChargeEntry struct {
	RechargeStart time.Time
	RechargeEnd   time.Time
}

// History 管理法术冷却、充能、公共冷却和魔法学校锁定，对齐 TC 的 SpellHistory。
type History struct {
	mu                sync.RWMutex
	cooldowns         map[spell.SpellID]*CooldownEntry
	categoryCooldowns map[uint32]*CooldownEntry
	charges           map[spell.SpellID][]ChargeEntry
	maxCharges        map[spell.SpellID]int32
	chargeRecharge    map[spell.SpellID]time.Duration
	gcdCategory       map[uint32]time.Time
	schoolLockouts    [7]time.Time
}

// NewHistory 创建一个空的冷却历史记录。
func NewHistory() *History {
	return &History{
		cooldowns:         make(map[spell.SpellID]*CooldownEntry),
		categoryCooldowns: make(map[uint32]*CooldownEntry),
		charges:           make(map[spell.SpellID][]ChargeEntry),
		maxCharges:        make(map[spell.SpellID]int32),
		chargeRecharge:    make(map[spell.SpellID]time.Duration),
		gcdCategory:       make(map[uint32]time.Time),
	}
}

// AddCooldown 为法术添加冷却记录，同时记录分类冷却。
func (h *History) AddCooldown(spellID spell.SpellID, categoryID uint32, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	endTime := time.Now().Add(duration)
	h.cooldowns[spellID] = &CooldownEntry{
		SpellID:    spellID,
		CategoryID: categoryID,
		EndTime:    endTime,
	}
	if categoryID != 0 {
		h.categoryCooldowns[categoryID] = &CooldownEntry{
			SpellID:    spellID,
			CategoryID: categoryID,
			EndTime:    endTime,
		}
	}
}

// AddCharge 初始化法术的充能系统。首次调用时创建所有充能条目。
func (h *History) AddCharge(spellID spell.SpellID, maxCharges int32, rechargeTime time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.maxCharges[spellID] = maxCharges
	h.chargeRecharge[spellID] = rechargeTime

	if len(h.charges[spellID]) == 0 {
		for i := int32(0); i < maxCharges; i++ {
			h.charges[spellID] = append(h.charges[spellID], ChargeEntry{
				RechargeEnd: time.Now(),
			})
		}
	}
}

// UseCharge 消耗一个充能。选择最早可用的充能条目开始充能计时。
func (h *History) UseCharge(spellID spell.SpellID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	available := h.availableCharges(spellID)
	if available == 0 {
		return false
	}

	now := time.Now()
	charges := h.charges[spellID]
	recharge := h.chargeRecharge[spellID]

	// 找到最早完成充能的条目
	var oldestIdx int
	oldestTime := charges[0].RechargeEnd
	for i, c := range charges {
		if c.RechargeEnd.Before(oldestTime) {
			oldestTime = c.RechargeEnd
			oldestIdx = i
		}
	}

	// 充能开始时间取当前时间和最早完成时间的较晚者
	startTime := now
	if oldestTime.After(now) {
		startTime = oldestTime
	}
	charges[oldestIdx] = ChargeEntry{
		RechargeStart: startTime,
		RechargeEnd:   startTime.Add(recharge),
	}
	return true
}

// availableCharges 计算当前可用的充能数量。
func (h *History) availableCharges(spellID spell.SpellID) int32 {
	charges := h.charges[spellID]
	if len(charges) == 0 {
		return h.maxCharges[spellID]
	}
	now := time.Now()
	var count int32
	for _, c := range charges {
		if !c.RechargeEnd.After(now) {
			count++
		}
	}
	return count
}

// IsReady 检查法术是否就绪（冷却结束、分类冷却结束、充能可用）。
func (h *History) IsReady(spellID spell.SpellID, categoryID uint32) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()

	if cd, ok := h.cooldowns[spellID]; ok && cd.EndTime.After(now) && !cd.OnHold {
		return false
	}
	if categoryID != 0 {
		if cd, ok := h.categoryCooldowns[categoryID]; ok && cd.EndTime.After(now) {
			return false
		}
	}
	if h.maxCharges[spellID] > 0 {
		return h.availableCharges(spellID) > 0
	}
	return true
}

// CancelCooldown 取消法术的冷却记录。
func (h *History) CancelCooldown(spellID spell.SpellID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.cooldowns, spellID)
}

// AddGlobalCooldown 为分类添加公共冷却。
func (h *History) AddGlobalCooldown(categoryID uint32, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.gcdCategory[categoryID] = time.Now().Add(duration)
}

// HasGlobalCooldown 检查分类是否处于公共冷却中。
func (h *History) HasGlobalCooldown(categoryID uint32) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	end, ok := h.gcdCategory[categoryID]
	return ok && end.After(time.Now())
}

// CancelGlobalCooldown 取消分类的公共冷却。
func (h *History) CancelGlobalCooldown(categoryID uint32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.gcdCategory, categoryID)
}

// AddSchoolLockout 为魔法学校添加施法锁定，对齐 TC 的 school lockout 机制。
func (h *History) AddSchoolLockout(school uint8, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if school < 7 {
		h.schoolLockouts[school] = time.Now().Add(duration)
	}
}

// HasSchoolLockout 检查魔法学校是否处于施法锁定中。
func (h *History) HasSchoolLockout(school uint8) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if school >= 7 {
		return false
	}
	return h.schoolLockouts[school].After(time.Now())
}
