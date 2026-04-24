package cooldown

import (
	"sync"
	"time"

	"skill-go/pkg/spell"
)

type CooldownEntry struct {
	SpellID    spell.SpellID
	CategoryID uint32
	EndTime    time.Time
	OnHold     bool
}

type ChargeEntry struct {
	RechargeStart time.Time
	RechargeEnd   time.Time
}

type History struct {
	mu              sync.RWMutex
	cooldowns       map[spell.SpellID]*CooldownEntry
	categoryCooldowns map[uint32]*CooldownEntry
	charges         map[spell.SpellID][]ChargeEntry
	maxCharges      map[spell.SpellID]int32
	chargeRecharge  map[spell.SpellID]time.Duration
	gcdCategory     map[uint32]time.Time
	schoolLockouts  [7]time.Time
}

func NewHistory() *History {
	return &History{
		cooldowns:       make(map[spell.SpellID]*CooldownEntry),
		categoryCooldowns: make(map[uint32]*CooldownEntry),
		charges:         make(map[spell.SpellID][]ChargeEntry),
		maxCharges:      make(map[spell.SpellID]int32),
		chargeRecharge:  make(map[spell.SpellID]time.Duration),
		gcdCategory:     make(map[uint32]time.Time),
	}
}

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

	var oldestIdx int
	oldestTime := charges[0].RechargeEnd
	for i, c := range charges {
		if c.RechargeEnd.Before(oldestTime) {
			oldestTime = c.RechargeEnd
			oldestIdx = i
		}
	}

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

func (h *History) CancelCooldown(spellID spell.SpellID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.cooldowns, spellID)
}

func (h *History) AddGlobalCooldown(categoryID uint32, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.gcdCategory[categoryID] = time.Now().Add(duration)
}

func (h *History) HasGlobalCooldown(categoryID uint32) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	end, ok := h.gcdCategory[categoryID]
	return ok && end.After(time.Now())
}

func (h *History) CancelGlobalCooldown(categoryID uint32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.gcdCategory, categoryID)
}

func (h *History) AddSchoolLockout(school uint8, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if school < 7 {
		h.schoolLockouts[school] = time.Now().Add(duration)
	}
}

func (h *History) HasSchoolLockout(school uint8) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if school >= 7 {
		return false
	}
	return h.schoolLockouts[school].After(time.Now())
}
