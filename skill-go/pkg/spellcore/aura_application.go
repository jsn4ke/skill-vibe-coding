package spellcore

// AuraApplication 表示一个光环在目标上的 per-target 应用实例，对齐 TC 的 AuraApplication。
// 一个 Aura（caster 拥有）可以在多个 target 上各有一个 AuraApplication。
// Aura 管理生命周期/duration/stack，AuraApplication 管理该 target 上的效果状态。
type AuraApplication struct {
	// Base 是所属的光环实例（caster 端拥有），对齐 TC 的 _base。
	Base *Aura

	// TargetID 是此应用挂载的目标 ID，对齐 TC 的 _target。
	// 单体光环与 Base.TargetID 相同；区域光环各 app 有不同 TargetID。
	TargetID uint64

	// RemoveMode 是此 target 上的移除原因，对齐 TC 的 _removeMode。
	RemoveMode RemoveMode

	// EffectsToApply 是此 target 应该应用的效果位掩码，对齐 TC 的 _effectsToApply。
	// 由 target 免疫、effect 条件等决定。默认全量 (^uint32(0))。
	EffectsToApply uint32

	// EffectMask 是当前已应用的效果位掩码，对齐 TC 的 _effectMask。
	// 与 EffectsToApply 的差集表示尚未应用的效果。
	EffectMask uint32
}

// GetAura 返回底层的光环实例。
func (app *AuraApplication) GetAura() *Aura { return app.Base }

// GetSpellID 返回光环的法术 ID。
func (app *AuraApplication) GetSpellID() SpellID { return app.Base.SpellID }

// GetCasterID 返回施法者 ID。
func (app *AuraApplication) GetCasterID() uint64 { return app.Base.CasterID }

// HasEffect 检查指定效果索引是否在已应用掩码中。
func (app *AuraApplication) HasEffect(effIndex uint8) bool {
	return app.EffectMask&(1<<effIndex) != 0
}

// NewAuraApplication 创建一个全量效果掩码的光环应用实例。
func NewAuraApplication(aura *Aura, targetID uint64) *AuraApplication {
	return &AuraApplication{
		Base:           aura,
		TargetID:       targetID,
		RemoveMode:     RemoveNone,
		EffectsToApply: ^uint32(0),
		EffectMask:     ^uint32(0),
	}
}

// UpdateApplyEffectMask 动态更新目标的效果掩码，对齐 TC 的 AuraApplication::UpdateApplyEffectMask。
// 差量 apply/remove effect。
func (app *AuraApplication) UpdateApplyEffectMask(newMask uint32, target AuraHost) {
	if app.EffectsToApply == newMask {
		return
	}

	removed := app.EffectsToApply & ^newMask
	added := newMask & ^app.EffectsToApply

	app.EffectsToApply = newMask

	if removed != 0 && app.EffectMask&removed != 0 {
		oldMask := app.EffectMask
		app.EffectMask = oldMask & removed
		target.RemoveAuraEffectsForApp(app)
		app.EffectMask = oldMask & ^removed
	}
	if added != 0 {
		oldMask := app.EffectMask
		app.EffectMask = added
		target.ApplyAuraEffectsForApp(app)
		app.EffectMask = oldMask | added
	}
}
