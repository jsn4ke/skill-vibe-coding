package script

import (
	"skill-go/pkg/spell"
	"skill-go/pkg/targeting"
)

// Hook 表示法术脚本钩子的类型。
type Hook uint8

const (
	HookOnCast Hook = iota
	HookOnPrepare
	HookOnHit
	HookOnMiss
	HookOnEffectHit
	HookOnEffectHitTarget
	HookOnEffectLaunch
	HookOnEffectLaunchTarget
	HookBeforeCast
	HookAfterCast
	HookBeforeHit
	HookAfterHit
	HookOnSpellLaunch
	HookOnSpellCancel
	HookOnTargetSelect // 目标选择完成后、AddTarget 前调用
)

// AuraHook 表示光环脚本钩子的类型。
type AuraHook uint8

const (
	AuraHookOnApply AuraHook = iota
	AuraHookOnRemove
	AuraHookOnPeriodic
	AuraHookOnProc
	AuraHookOnStackChange
	AuraHookAfterApply
	AuraHookAfterRemove
)

// SpellContext 是法术脚本钩子的执行上下文。
type SpellContext struct {
	Spell          *spell.Spell
	PreventDefault bool
	EffectIndex    uint8
	// TargetUnits 是 HookOnTargetSelect 时已选中的目标列表，脚本可修改此切片干预目标选择
	TargetUnits []targeting.TargetUnit
}

// HandlerFunc 是法术脚本钩子的处理函数类型。
type HandlerFunc func(ctx *SpellContext)

// AuraContext 是光环脚本钩子的执行上下文。
type AuraContext struct {
	SpellID        spell.SpellID
	TargetID       uint64
	CasterID       uint64
	StackAmount    uint8
	EffectIndex    uint8
	Amount         float64
	PreventDefault bool
	RemoveMode     uint8
	Aura           interface{}
}

// AuraHandlerFunc 是光环脚本钩子的处理函数类型。
type AuraHandlerFunc func(ctx *AuraContext)

// Registry 是脚本注册中心，按 SpellID 精确匹配法术和光环钩子。
// 技能通过 RegisterSpellHook/RegisterAuraHook 注册脚本，
// 引擎通过 CallSpellHook/CallAuraHook 调用钩子。
type Registry struct {
	spellHooks map[spell.SpellID]map[Hook][]HandlerFunc
	auraHooks  map[spell.SpellID]map[AuraHook][]AuraHandlerFunc
}

// NewRegistry 创建一个空的脚本注册中心。
func NewRegistry() *Registry {
	return &Registry{
		spellHooks: make(map[spell.SpellID]map[Hook][]HandlerFunc),
		auraHooks:  make(map[spell.SpellID]map[AuraHook][]AuraHandlerFunc),
	}
}

// RegisterSpellHook 为指定法术注册法术脚本钩子。
func (r *Registry) RegisterSpellHook(spellID spell.SpellID, hook Hook, fn HandlerFunc) {
	if r.spellHooks[spellID] == nil {
		r.spellHooks[spellID] = make(map[Hook][]HandlerFunc)
	}
	r.spellHooks[spellID][hook] = append(r.spellHooks[spellID][hook], fn)
}

// RegisterAuraHook 为指定法术注册光环脚本钩子。
func (r *Registry) RegisterAuraHook(spellID spell.SpellID, hook AuraHook, fn AuraHandlerFunc) {
	if r.auraHooks[spellID] == nil {
		r.auraHooks[spellID] = make(map[AuraHook][]AuraHandlerFunc)
	}
	r.auraHooks[spellID][hook] = append(r.auraHooks[spellID][hook], fn)
}

// CallSpellHook 调用指定法术的脚本钩子。任一处理函数设置 PreventDefault 则停止后续调用。
func (r *Registry) CallSpellHook(spellID spell.SpellID, hook Hook, ctx *SpellContext) {
	hooks, ok := r.spellHooks[spellID]
	if !ok {
		return
	}
	handlers, ok := hooks[hook]
	if !ok {
		return
	}
	for _, fn := range handlers {
		fn(ctx)
		if ctx.PreventDefault {
			return
		}
	}
}

// CallAuraHook 调用指定法术的光环脚本钩子。任一处理函数设置 PreventDefault 则停止后续调用。
func (r *Registry) CallAuraHook(spellID spell.SpellID, hook AuraHook, ctx *AuraContext) {
	hooks, ok := r.auraHooks[spellID]
	if !ok {
		return
	}
	handlers, ok := hooks[hook]
	if !ok {
		return
	}
	for _, fn := range handlers {
		fn(ctx)
		if ctx.PreventDefault {
			return
		}
	}
}

// HasSpellHook 检查指定法术是否注册了指定类型的脚本钩子。
func (r *Registry) HasSpellHook(spellID spell.SpellID, hook Hook) bool {
	hooks, ok := r.spellHooks[spellID]
	if !ok {
		return false
	}
	_, has := hooks[hook]
	return has
}

// HasAuraHook 检查指定法术是否注册了指定类型的光环脚本钩子。
func (r *Registry) HasAuraHook(spellID spell.SpellID, hook AuraHook) bool {
	hooks, ok := r.auraHooks[spellID]
	if !ok {
		return false
	}
	_, has := hooks[hook]
	return has
}

// UnregisterAll 移除指定法术的所有脚本和光环钩子。
func (r *Registry) UnregisterAll(spellID spell.SpellID) {
	delete(r.spellHooks, spellID)
	delete(r.auraHooks, spellID)
}
