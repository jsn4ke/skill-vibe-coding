package spellcore

// SpellStore 是法术配置表，按 SpellID 索引 SpellInfo。
// 对齐 TC 的 SpellMgr（sSpellMgr），提供全局法术数据查找。
type SpellStore struct {
	spells map[SpellID]*SpellInfo
}

// NewSpellStore 创建空的法术配置表。
func NewSpellStore() *SpellStore {
	return &SpellStore{spells: make(map[SpellID]*SpellInfo)}
}

// Register 将 SpellInfo 注册到配置表中，以 ID 为键。
func (s *SpellStore) Register(info *SpellInfo) {
	s.spells[info.ID] = info
}

// Get 按 SpellID 查找 SpellInfo，未找到返回 nil。
func (s *SpellStore) Get(id SpellID) *SpellInfo {
	return s.spells[id]
}
