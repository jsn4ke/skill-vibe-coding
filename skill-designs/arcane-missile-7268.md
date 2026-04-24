# 奥术飞弹·单发 (Arcane Missile) (Spell ID 7268)

> 来源: WoW 法术 (Triggered Spell, Uncategorized) | 生成日期: 2026-04-24
> 父法术: [Arcane Missiles (5143)](arcane-missiles.md)

## 概述

Arcane Missile (7268) 是奥术飞弹每 tick 触发的隐藏法术，玩家不直接施放。在 TC 中由父法术 5143 的 `SPELL_AURA_PERIODIC_TRIGGER_SPELL` Aura 每 1 秒触发一次，以 `TRIGGERED_FULL_MASK` 标记执行，走完整 CastSpell 流程。每次触发创建独立的 Spell 实例，有独立的暴击判定和伤害计算。

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害（即时·直接） | SPELL_EFFECT_SCHOOL_DAMAGE |
| 施法时间 | 即时 (0) | 无施法条 |
| 冷却 | 无 | — |
| 充能 | 无 | — |
| 资源消耗 | 无 | 触发法术本身 PowerCost=0；即使有消耗也会被 TRIGGERED_FULL_MASK 跳过 |
| 距离 | 30 码 | 继承父法术目标 |
| GCD | 不触发 | TRIGGERED_FULL_MASK 跳过 |
| 弹道速度 | 0 (SpellMisc.Speed=0) | 无弹道，即时命中 |
| 法术分类 | Uncategorized (隐藏) | 玩家不可直接施放 |
| 伤害类型 | 奥术 (Arcane) | SpellSchool |

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 单体 (继承父法术目标) | TARGET_UNIT_TARGET_ENEMY |
| 目标数量 | 1 | 继承父法术的 UnitTargetID |

## 效果拆解

### 效果 1: 奥术直接伤害

- **类型**: 直接伤害 (TC: `SPELL_EFFECT_SCHOOL_DAMAGE`)
- **基础数值**: 24 点 (BasePoints + 1 - 1 = 24，TC 中 BasePoints 存储的是 value-1)
- **随机方差**: 无 (DieSides = 0，无随机成分)
- **系数**: 法术强度 × 0.132 (13.2%)
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **暴击**: 可暴击，独立判定 (1.5x)
- **WoW 映射**: 直接映射 → 框架已有 `EffectSchoolDamage`
- **备注**: 每发飞弹独立暴击，独立计算伤害

**每发伤害公式**: `damage = 24 + 0.132 × spellPower`
- 无暴击: 24 + 0.132 × SP
- 暴击: (24 + 0.132 × SP) × 1.5

**3 发总计** (父法术 5143 引导 3 秒，每秒触发 1 发):
- 无暴击: 72 + 0.396 × SP
- 全暴击: 108 + 0.594 × SP
- SP=100 时: 111.6 (无暴击) ~ 167.4 (全暴击)

## 触发标记

触发法术使用 `TRIGGERED_FULL_MASK`，等价于以下所有 flag 的组合：

| Flag | 效果 | 对 7268 的影响 |
|------|------|----------------|
| TRIGGERED_IGNORE_GCD | 跳过 GCD | 不触发公共冷却 |
| TRIGGERED_IGNORE_SPELL_AND_CATEGORY_CD | 跳过冷却 | 不产生冷却记录 |
| TRIGGERED_IGNORE_POWER_AND_REAGENT_COST | 跳过资源消耗 | 即使有消耗也忽略 |
| TRIGGERED_IGNORE_CAST_ITEM | 跳过施法物品检查 | 不涉及 |
| TRIGGERED_IGNORE_AURA_SCALING | 跳过 Aura 缩放 | 不涉及 |
| TRIGGERED_IGNORE_CAST_IN_PROGRESS | 跳过"正在施法"检查 | 允许在引导中触发 |
| TRIGGERED_IGNORE_COMBO_POINTS_FROM_TARGET | 跳过连击点检查 | 不涉及 |
| TRIGGERED_IGNORE_SHAPESHIFT | 跳过形态检查 | 不涉及 |
| TRIGGERED_IGNORE_CASTER_AURASTATE | 跳过施法者 Aura 状态 | 不涉及 |
| TRIGGERED_IGNORE_CASTER_MOUNTED_OR_ON_VEHICLE | 跳过坐骑检查 | 不涉及 |

## 跨表数据

| 字段 | 来源 | 值 | 说明 |
|------|------|-----|------|
| Speed | SpellMisc.Speed | 0 | 无弹道，即时命中 |
| LaunchDelay | SpellMisc.LaunchDelay | 0 | 无发射延迟 |
| School | Spell.dbc | Arcane (64) | 奥术伤害类型 |
| BasePoints | SpellEffectInfo | 24 | 基础伤害 |
| BonusCoeff | SpellEffectInfo | 0.132 | 法术强度系数 |
| DieSides | SpellEffectInfo | 0 | 无随机方差 |

## 生命周期

触发法术 7268 的生命周期由父法术 5143 的 Aura tick 驱动：

```
父法术 Aura tick (每 1s)
  │
  ▼
HandlePeriodicTriggerSpellAuraTick
  │
  ▼
caster->CastSpell(target, 7268, TRIGGERED_FULL_MASK)
  │
  ▼
Spell(7268) 创建
  │ CastFlags = TriggeredFullMask
  │ Targets.UnitTargetID = 父法术目标
  │
  ▼
Prepare() ── CastTime=0 ──▶ Cast(true)
  │
  ▼
HandleImmediate()
  │ SelectTargets → 确认目标
  │ ProcessEffects → EffectSchoolDamage → 24 + 0.132×SP
  │ 暴击独立判定
  │ 发布 OnSpellHit 事件
  │
  ▼
Finish(CastOK)
```

**生命周期特点**:
- **无状态驻留**: 从 Prepare 到 Finish 是同步完成的，不存在中间状态（CastTime=0）
- **独立实例**: 每次触发创建新的 Spell 对象，互不干扰
- **事件独立**: 每发飞弹独立发布 OnSpellHit 事件
- **暴击独立**: 每发飞弹独立判定暴击

## WoW 框架映射

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果: 直接伤害 | SPELL_EFFECT_SCHOOL_DAMAGE | 直接映射 | 框架已有 EffectSchoolDamage |
| 触发方式 | TRIGGERED_FULL_MASK CastSpell | 需新增 | CastTriggeredSpell 辅助函数 |
| 目标继承 | 继承父法术 UnitTargetID | 直接映射 | 设置 Targets 即可 |
| 暴击判定 | 独立判定 | 直接映射 | 每发独立 |
| 命中检查 | TRIGGERED 跳过 | 初版省略 | 简化为必中 |

## 实现建议

### 数据配置

```go
// Arcane Missile (7268) - 触发法术定义（内部使用，不暴露给外部）
var MissileInfo = spell.SpellInfo{
    ID:        7268,
    Name:      "Arcane Missile",
    CastTime:  0,     // 即时
    RangeMax:  30,    // 继承父法术
    IsChanneled: false,
    Effects: []spell.SpellEffectInfo{
        {
            EffectIndex:  0,
            EffectType:   spell.EffectSchoolDamage,
            BasePoints:   24,
            BonusCoeff:   0.132,
            TargetA:      spell.TargetUnitTargetEnemy,
        },
    },
}
```

### 触发执行

```go
// onTick 回调中——由父法术的 PeriodicTriggerSpell Aura 每 1s 触发
func CastTriggeredSpell(caster Caster, targetID uint64, info *spell.SpellInfo, bus *event.Bus) {
    s := spell.NewSpell(info.ID, info, caster, spell.TriggeredFullMask)
    s.Targets.UnitTargetID = targetID
    s.Bus = bus
    s.Prepare()  // CastTime=0 → 立即 Cast(true) → HandleImmediate → Finish
}
```
