package livingbomb

import (
	"skill-go/pkg/aura"
	"skill-go/pkg/engine"
	"skill-go/pkg/script"
	"skill-go/pkg/spell"
	"skill-go/pkg/unit"
)

// 法术 44457 — 活体炸弹（施放入口）
var Info = spell.SpellInfo{
	ID:        44457,
	Name:      "Living Bomb",
	CastTime:  0,
	RangeMax:  35,
	PowerCost: 470,
	PowerType: 0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectDummy,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
	},
}

// 法术 217694 — 活体炸弹周期（DoT）
var PeriodicInfo = spell.SpellInfo{
	ID:        217694,
	Name:      "Living Bomb Periodic",
	CastTime:  0,
	Duration:  4000,
	RangeMax:  40,
	PowerCost: 0,
	PowerType: 0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectApplyAura,
			BonusCoeff:  0.06,
			AuraType:    uint16(aura.AuraPeriodicDamage),
			AuraPeriod:  1000,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
		{
			EffectIndex: 1,
			EffectType:  spell.EffectDummy,
			TargetA:     spell.TargetUnitTargetEnemy,
		},
	},
}

// 法术 44461 — 活体炸弹爆炸（AoE）
// TargetA=TargetUnitSrcAreaEnemy + Radius=10.0：引擎自动在 SrcPos 半径内搜索敌方目标
var ExplosionInfo = spell.SpellInfo{
	ID:        44461,
	Name:      "Living Bomb Explode",
	CastTime:  0,
	RangeMax:  50000,
	PowerCost: 0,
	PowerType: 0,
	Effects: []spell.SpellEffectInfo{
		{
			EffectIndex: 0,
			EffectType:  spell.EffectDummy,
			TargetA:     spell.TargetUnitSrcAreaEnemy,
			Radius:      10.0,
		},
		{
			EffectIndex: 1,
			EffectType:  spell.EffectSchoolDamage,
			BonusCoeff:  0.14,
			TargetA:     spell.TargetUnitSrcAreaEnemy,
			Radius:      10.0,
		},
	},
}

// RegisterScripts 注册所有活体炸弹的脚本钩子。
// 所有触发法术使用 engine.CastSpell(WithTriggered)。
// 爆炸法术的 AoE 目标选择由引擎自动解析（TargetA+Radius），无需 WithAoE。
func RegisterScripts(registry *script.Registry, caster *unit.Unit, eng *engine.Engine) {
	// 44457: OnEffectHitTarget — 拦截 Dummy，施放周期法术
	registry.RegisterSpellHook(44457, script.HookOnEffectHitTarget, func(ctx *script.SpellContext) {
		ctx.PreventDefault = true
		targetID := ctx.Spell.Targets.UnitTargetID
		eng.CastSpell(caster, &PeriodicInfo,
			engine.WithTarget(targetID),
			engine.WithTriggered(),
			engine.WithSpellValues(map[uint8]float64{2: 1}),
		)
	})

	// 217694: AfterRemove — 仅在过期时爆炸
	registry.RegisterAuraHook(217694, script.AuraHookAfterRemove, func(ctx *script.AuraContext) {
		if ctx.RemoveMode != uint8(aura.RemoveByExpire) {
			return
		}
		a, ok := ctx.Aura.(*aura.Aura)
		if !ok || a == nil {
			return
		}
		canSpread := float64(0)
		if a.SpellValues != nil {
			canSpread = a.SpellValues[2]
		}
		castExplosion(eng, caster, ctx.TargetID, canSpread)
	})

	// 44461: OnEffectHitTarget EFFECT_1 — SchoolDamage 后传播到命中目标
	registry.RegisterSpellHook(44461, script.HookOnEffectHitTarget, func(ctx *script.SpellContext) {
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
