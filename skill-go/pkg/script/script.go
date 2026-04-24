package script

import (
	"skill-go/pkg/spell"
)

type Hook uint8

const (
	HookOnCast Hook = iota
	HookOnPrepare
	HookOnHit
	HookOnMiss
	HookOnEffectHit
	HookOnEffectLaunch
	HookBeforeCast
	HookAfterCast
	HookBeforeHit
	HookAfterHit
)

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

type SpellContext struct {
	Spell        *spell.Spell
	PreventDefault bool
}

type HandlerFunc func(ctx *SpellContext)

type AuraContext struct {
	SpellID      spell.SpellID
	TargetID     uint64
	CasterID     uint64
	StackAmount  uint8
	EffectIndex  uint8
	Amount       float64
	PreventDefault bool
}

type AuraHandlerFunc func(ctx *AuraContext)

type Registry struct {
	spellHooks map[spell.SpellID]map[Hook][]HandlerFunc
	auraHooks  map[spell.SpellID]map[AuraHook][]AuraHandlerFunc
}

func NewRegistry() *Registry {
	return &Registry{
		spellHooks: make(map[spell.SpellID]map[Hook][]HandlerFunc),
		auraHooks:  make(map[spell.SpellID]map[AuraHook][]AuraHandlerFunc),
	}
}

func (r *Registry) RegisterSpellHook(spellID spell.SpellID, hook Hook, fn HandlerFunc) {
	if r.spellHooks[spellID] == nil {
		r.spellHooks[spellID] = make(map[Hook][]HandlerFunc)
	}
	r.spellHooks[spellID][hook] = append(r.spellHooks[spellID][hook], fn)
}

func (r *Registry) RegisterAuraHook(spellID spell.SpellID, hook AuraHook, fn AuraHandlerFunc) {
	if r.auraHooks[spellID] == nil {
		r.auraHooks[spellID] = make(map[AuraHook][]AuraHandlerFunc)
	}
	r.auraHooks[spellID][hook] = append(r.auraHooks[spellID][hook], fn)
}

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

func (r *Registry) HasSpellHook(spellID spell.SpellID, hook Hook) bool {
	hooks, ok := r.spellHooks[spellID]
	if !ok {
		return false
	}
	_, has := hooks[hook]
	return has
}

func (r *Registry) HasAuraHook(spellID spell.SpellID, hook AuraHook) bool {
	hooks, ok := r.auraHooks[spellID]
	if !ok {
		return false
	}
	_, has := hooks[hook]
	return has
}

func (r *Registry) UnregisterAll(spellID spell.SpellID) {
	delete(r.spellHooks, spellID)
	delete(r.auraHooks, spellID)
}
