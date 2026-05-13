package spellcore

import (
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
	HookBeforeHit
	HookAfterHit
	HookOnSpellLaunch
	HookOnSpellCancel
	HookOnTargetSelect
)

// AuraHook 表示光环脚本钩子的类型。
type AuraHook uint8

const (
	AuraHookOnPeriodic AuraHook = iota
	AuraHookAfterApply
	AuraHookAfterRemove
)

// SpellContext 是法术脚本钩子的执行上下文。
type SpellContext struct {
	Spell          *Spell
	PreventDefault bool
	EffectIndex    uint8
	TargetUnits    []targeting.TargetUnit
}

// HandlerFunc 是法术脚本钩子的处理函数类型。
type HandlerFunc func(ctx *SpellContext)

// AuraContext 是光环脚本钩子的执行上下文。
type AuraContext struct {
	SpellID        SpellID
	TargetID       uint64
	CasterID       uint64
	StackAmount    uint8
	EffectIndex    uint8
	Amount         float64
	PreventDefault bool
	RemoveMode     uint8
	Aura           *Aura
	// App 是光环应用实例（per-target），可能为 nil（旧代码路径）。
	App *AuraApplication
}

// AuraHandlerFunc 是光环脚本钩子的处理函数类型。
type AuraHandlerFunc func(ctx *AuraContext)

// Registry 是脚本注册中心，按 SpellID 精确匹配法术和光环钩子。
type Registry struct {
	spellHooks map[SpellID]map[Hook][]HandlerFunc
	auraHooks  map[SpellID]map[AuraHook][]AuraHandlerFunc
}

// NewRegistry 创建一个空的脚本注册中心。
func NewRegistry() *Registry {
	return &Registry{
		spellHooks: make(map[SpellID]map[Hook][]HandlerFunc),
		auraHooks:  make(map[SpellID]map[AuraHook][]AuraHandlerFunc),
	}
}

// RegisterSpellHook 为指定法术注册法术脚本钩子。
func (r *Registry) RegisterSpellHook(spellID SpellID, hook Hook, fn HandlerFunc) {
	if r.spellHooks[spellID] == nil {
		r.spellHooks[spellID] = make(map[Hook][]HandlerFunc)
	}
	r.spellHooks[spellID][hook] = append(r.spellHooks[spellID][hook], fn)
}

// RegisterAuraHook 为指定法术注册光环脚本钩子。
func (r *Registry) RegisterAuraHook(spellID SpellID, hook AuraHook, fn AuraHandlerFunc) {
	if r.auraHooks[spellID] == nil {
		r.auraHooks[spellID] = make(map[AuraHook][]AuraHandlerFunc)
	}
	r.auraHooks[spellID][hook] = append(r.auraHooks[spellID][hook], fn)
}

// CallSpellHook 调用指定法术的脚本钩子。
func (r *Registry) CallSpellHook(spellID SpellID, hook Hook, ctx *SpellContext) {
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

// CallAuraHook 调用指定法术的光环脚本钩子。
func (r *Registry) CallAuraHook(spellID SpellID, hook AuraHook, ctx *AuraContext) {
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
func (r *Registry) HasSpellHook(spellID SpellID, hook Hook) bool {
	hooks, ok := r.spellHooks[spellID]
	if !ok {
		return false
	}
	_, has := hooks[hook]
	return has
}

// HasAuraHook 检查指定法术是否注册了指定类型的光环脚本钩子。
func (r *Registry) HasAuraHook(spellID SpellID, hook AuraHook) bool {
	hooks, ok := r.auraHooks[spellID]
	if !ok {
		return false
	}
	_, has := hooks[hook]
	return has
}

// UnregisterAll 移除指定法术的所有脚本和光环钩子。
func (r *Registry) UnregisterAll(spellID SpellID) {
	delete(r.spellHooks, spellID)
	delete(r.auraHooks, spellID)
}
