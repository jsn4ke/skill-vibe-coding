package livingbomb

import (
	"skill-go/pkg/engine"
	"skill-go/pkg/spellcore"
	"skill-go/pkg/unit"
)

// 法术 44457 — 活体炸弹（施放入口）
var Info = spellcore.SpellInfo{
	ID:        44457,
	Name:      "Living Bomb",
	CastTime:  0,
	RangeMax:  35,
	PowerCost: 470,
	PowerType: 0,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectDummy,
			TargetA:     spellcore.TargetUnitTargetEnemy,
		},
	},
}

// 法术 217694 — 活体炸弹周期（DoT）
var PeriodicInfo = spellcore.SpellInfo{
	ID:        217694,
	Name:      "Living Bomb Periodic",
	CastTime:  0,
	Duration:  4000,
	RangeMax:  40,
	PowerCost: 0,
	PowerType: 0,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectApplyAura,
			BonusCoeff:  0.06,
			AuraType:    uint16(spellcore.AuraPeriodicDamage),
			AuraPeriod:  1000,
			TargetA:     spellcore.TargetUnitTargetEnemy,
		},
		{
			EffectIndex: 1,
			EffectType:  spellcore.EffectDummy,
			TargetA:     spellcore.TargetUnitTargetEnemy,
		},
	},
}

// 法术 44461 — 活体炸弹爆炸（AoE）
// TargetA=TargetUnitSrcAreaEnemy + Radius=10.0：引擎自动在 SrcPos 半径内搜索敌方目标
var ExplosionInfo = spellcore.SpellInfo{
	ID:        44461,
	Name:      "Living Bomb Explode",
	CastTime:  0,
	RangeMax:  50000,
	PowerCost: 0,
	PowerType: 0,
	Effects: []spellcore.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spellcore.EffectDummy,
			TargetA:     spellcore.TargetUnitSrcAreaEnemy,
			Radius:      10.0,
		},
		{
			EffectIndex: 1,
			EffectType:  spellcore.EffectSchoolDamage,
			BonusCoeff:  0.14,
			TargetA:     spellcore.TargetUnitSrcAreaEnemy,
			Radius:      10.0,
		},
	},
}

// RegisterScripts 注册所有活体炸弹的脚本钩子。
// 所有触发法术使用 engine.CastSpell(WithTriggered)。
// 爆炸法术的 AoE 目标选择由引擎自动解析（TargetA+Radius），无需 WithAoE。
func RegisterScripts(registry *spellcore.Registry, caster *unit.Unit, eng *engine.Engine) {
	// 44457: OnEffectHitTarget — 拦截 Dummy，施放周期法术
	registry.RegisterSpellHook(44457, spellcore.HookOnEffectHitTarget, func(ctx *spellcore.SpellContext) {
		ctx.PreventDefault = true
		targetID := ctx.Spell.Targets.UnitTargetID
		eng.CastSpell(caster, &PeriodicInfo,
			engine.WithTarget(targetID),
			engine.WithTriggered(),
			engine.WithSpellValues(map[uint8]float64{2: 1}),
		)
	})

	// 217694: AfterRemove — 仅在过期时爆炸
	registry.RegisterAuraHook(217694, spellcore.AuraHookAfterRemove, func(ctx *spellcore.AuraContext) {
		if ctx.RemoveMode != uint8(spellcore.RemoveByExpire) {
			return
		}
		a := ctx.Aura
		if a == nil {
			return
		}
		canSpread := float64(0)
		if a.SpellValues != nil {
			canSpread = a.SpellValues[2]
		}
		castExplosion(eng, caster, ctx.TargetID, canSpread)
	})

	// 44461: OnEffectHitTarget EFFECT_1 — SchoolDamage 后传播到命中目标
	registry.RegisterSpellHook(44461, spellcore.HookOnEffectHitTarget, func(ctx *spellcore.SpellContext) {
		if ctx.EffectIndex != 1 {
			return
		}
		canSpread := ctx.Spell.SpellValues[2]
		if canSpread <= 0 {
			return
		}
		for _, ti := range ctx.Spell.TargetInfos {
			eng.CastSpell(caster, &PeriodicInfo,
				engine.WithTarget(ti.TargetID),
				engine.WithTriggered(),
				engine.WithSpellValues(map[uint8]float64{2: 0}),
			)
		}
		ctx.Spell.SpellValues[2] = 0
	})
}

// castExplosion 通过引擎创建并施放爆炸法术。
// 设置 SourcePos 为承载者位置，引擎自动在 SrcPos 半径内搜索敌方目标。
func castExplosion(eng *engine.Engine, caster *unit.Unit, carrierTargetID uint64, canSpread float64) {
	carrierPos := caster.GetTargetPosition(carrierTargetID)
	eng.CastSpell(caster, &ExplosionInfo,
		engine.WithTriggered(),
		engine.WithSrcPos(carrierPos.GetX(), carrierPos.GetY(), carrierPos.GetZ()),
		engine.WithSpellValues(map[uint8]float64{2: canSpread}),
	)
}

// RegisterSpells 将活体炸弹的所有法术注册到配置表中。
func RegisterSpells(store *spellcore.SpellStore) {
	store.Register(&Info)
	store.Register(&PeriodicInfo)
	store.Register(&ExplosionInfo)
}
