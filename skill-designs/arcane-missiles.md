# 奥术飞弹 (Arcane Missiles)

> 来源: WoW 技能 (Spell ID: 5143, Rank 1) | 生成日期: 2026-04-24

## 概述

奥术飞弹是法师的标志性引导技能，在 3 秒内每秒向目标发射一发奥术飞弹，每发造成奥术伤害。它的核心机制是**引导型周期触发法术 (Channeled Periodic Trigger Spell)**——与暴风雪的"引导+直接周期伤害"不同，奥术飞弹的每次 tick 不是直接造成伤害，而是**触发一个独立的法术实例**来处理伤害。这是 WoW 法术系统中"法术触发法术"模式的经典案例，也使得它与暴风雪形成了有趣的架构对比。

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害（引导·周期触发） | SPELL_EFFECT_APPLY_AURA + SPELL_AURA_PERIODIC_TRIGGER_SPELL |
| 施法时间 | 引导 3 秒 | SpellCastTimesEntry = 0 (即时进入引导), SpellDurationEntry = 3000 |
| 冷却 | 无 | 无冷却条目 |
| 充能 | 无 | — |
| 资源消耗 | 85 点法力 | SpellPowerEntry |
| 距离 | 30 码 | SpellRangeEntry |
| GCD | 触发 (1.5 秒) | GlobalCooldownMgr |
| 施法条件 | 需要站立、需要敌对目标、引导期间不可移动 | CHANNEL_INTERRUPT_FLAG |

**各等级数据:**

| Rank | Spell ID | 伤害/tick | 持续 | 触发法术 ID | 法力消耗 |
|------|----------|-----------|------|-------------|----------|
| 1 | 5143 | 26 | 3s | 7268 | 85 |
| 2 | 7270 | 58 | 5s | — | 165 |
| 3 | 8417 | 118 | 5s | — | 290 |
| 4 | 10212 | 196 | 5s | — | 420 |
| 5 | 25345 | 230 | 5s | — | 480 |

本文以 Rank 1 (5143) 为基准设计。

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 单体 (敌对) | TARGET_UNIT_TARGET_ENEMY |
| 目标数量 | 1 | MaxAffectedTargets = 1 |
| 目标筛选 | 敌方 | TARGET_CHECK_ENEMY |
| 半径 | 无 | — |
| 锥形角度 | 无 | — |
| 链式跳跃 | 无 | — |

**追踪目标 (Track Target in Channel)**: WoW 原版有一个特殊 flag `SPELL_ATTR1_TRACK_TARGET_IN_CHANNEL`，引导期间持续追踪目标位置。目标移动时飞弹仍然命中。初版简化为固定目标 ID。

## 效果拆解

### 效果 1: 周期触发法术 (Periodic Trigger Spell)

- **类型**: 施加 Aura (TC: `SPELL_EFFECT_APPLY_AURA` + `SPELL_AURA_PERIODIC_TRIGGER_SPELL`)
- **基础数值**: 1 (周期次数)
- **系数**: 不适用（伤害由触发法术计算）
- **作用目标**: TARGET_UNIT_TARGET_ENEMY（单体敌方目标）
- **周期**: 每 1 秒触发一次
- **触发法术**: Spell ID 7268 "Arcane Missile"（见下方）
- **暴击**: 触发的法术独立判定暴击
- **WoW 映射**: 需新增 Aura 类型 → 现有框架有 `AuraPeriodicDamage` 但无 `AuraPeriodicTriggerSpell`
- **备注**: 这是奥术飞弹的核心机制——不是直接周期伤害，而是周期触发另一个法术。TC 中通过 `HandlePeriodicTriggerSpellAuraTick` 实现

### 效果 2: Dummy Aura

- **类型**: 施加 Aura (TC: `SPELL_EFFECT_APPLY_AURA` + `SPELL_AURA_DUMMY`)
- **基础数值**: 1
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **WoW 映射**: 省略 → Dummy aura 在 WoW 中用于客户端视觉/服务端脚本，服务端逻辑由效果 1 完成
- **备注**: 不实现，仅记录

### 触发法术: Arcane Missile (Spell ID 7268)

> **完整设计文档**: [arcane-missile-7268.md](arcane-missile-7268.md)

- **类型**: 直接伤害 (TC: `SPELL_EFFECT_SCHOOL_DAMAGE`)
- **基础数值**: 24 点奥术伤害
- **系数**: 法术强度 × 0.132 (13.2%)
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **暴击**: 可暴击 (1.5x)，独立判定
- **成本**: 无 (由 TRIGGERED_FULL_MASK 跳过)
- **GCD**: 不触发
- **弹道**: 无 (SpellMisc.Speed=0，即时命中)
- **备注**: 隐藏法术，玩家不直接施放。TC 中以 `TRIGGERED_FULL_MASK` 触发，走完整 CastSpell 流程。每 tick 创建独立 Spell 实例。

**每发总伤害公式**: `damage = 24 + 0.132 × spellPower`
**3 秒总伤害**: `3 × (24 + 0.132 × SP)` = 72 + 0.396 × SP

## Aura

### Aura 1: 周期触发法术 (Periodic Trigger Spell)

- **类型**: 周期触发法术 (TC: `SPELL_AURA_PERIODIC_TRIGGER_SPELL`)
- **持续时间**: 3 秒（与引导同步）
- **堆叠规则**: 不可堆叠 (StackNone)
- **最大堆叠时行为**: 不适用
- **触发条件 (Proc)**: 不适用（这是周期触发，非 proc 触发）
- **周期效果**: 每 1 秒触发 Spell 7268 (Arcane Missile)
  - 触发者: 施法者 (caster, 不是 target)
  - 触发标记: TRIGGERED_FULL_MASK（忽略 GCD/冷却/大部分检查）
  - 每次触发创建独立法术实例处理伤害
- **中断条件**: 引导取消 → Aura 被移除 → 后续 tick 不触发
- **WoW 映射**: 需新增 → 框架无 `AuraPeriodicTriggerSpell` 类型
- **备注**: TC 的 `HandlePeriodicTriggerSpellAuraTick` 逻辑——读取 `TriggerSpellID`，以 caster 身份 `CastSpell` 触发法术

## WoW 框架映射汇总

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果: 周期触发法术 | `SPELL_EFFECT_APPLY_AURA` + `SPELL_AURA_PERIODIC_TRIGGER_SPELL` | 需新增 | 需添加 AuraPeriodicTriggerSpell 类型 |
| 效果: Dummy Aura | `SPELL_AURA_DUMMY` | 省略 | 仅客户端视觉 |
| 触发法术: 直接伤害 | `SPELL_EFFECT_SCHOOL_DAMAGE` | 直接映射 | EffectSchoolDamage 已存在 |
| Aura: 周期触发 | `SPELL_AURA_PERIODIC_TRIGGER_SPELL` | 需新增 | 核心——每 tick 触发独立法术 |
| 引导状态 | `SPELL_STATE_CHANNELING` | 直接映射 | StateChanneling 已存在 |
| 引导取消 | `cancel()` 移除 Aura | 直接映射 | 与暴风雪一致的联动模式 |
| 目标追踪 | `SPELL_ATTR1_TRACK_TARGET_IN_CHANNEL` | 初版省略 | 固定目标 ID |
| 资源消耗 | 法力 | 直接映射 | 85 法力 |
| 触发法术执行 | `TRIGGERED_FULL_MASK` CastSpell | 需新增 | 需在 tick 时创建并执行法术 |

## 实现建议

### 整体设计思路

奥术飞弹是**引导型单体周期触发技能**，与现有两个技能的关系：

```
              ┌──────────────┬──────────────┬──────────────┐
              │ Fireball     │ Blizzard     │ Arcane Missiles│
              ├──────────────┼──────────────┼──────────────┤
 施法类型      │ 有施法条 3.5s│ 引导 8s      │ 引导 3s       │
 目标类型      │ 单体         │ 区域 AoE     │ 单体          │
 命中方式      │ 弹道延迟     │ 即时         │ 即时/tick     │
 伤害方式      │ 直接+DoT    │ 直接周期伤害  │ 周期触发法术   │
 tick 绑定对象 │ 无 tick      │ 地面区域      │ 施法者→目标    │
```

核心架构差异：
- **暴风雪**: Aura tick → 直接在目标上计算伤害 → `TickPeriodicArea`
- **奥术飞弹**: Aura tick → 创建触发法术实例 → 触发法术计算伤害 → `TickPeriodic`

```
TC 原始架构:
  Spell (引导倒计时, StateChanneling)
    └── Aura (PeriodicTriggerSpell, 绑定在目标上)
        └── AuraEffect (每 1s tick)
            └── HandlePeriodicTriggerSpellAuraTick
                └── caster->CastSpell(target, 7268, TRIGGERED_FULL_MASK)
                    └── Spell(7268) → EffectSchoolDamage → 伤害

我们的架构 (完整 Spell 实例，复刻 TC):
  Spell (引导倒计时, StateChanneling)
    └── Aura (PeriodicTriggerSpell, 绑定在目标上)
        └── AuraEffect (每 1s tick)
            └── TickPeriodic → onTick 回调
                └── CastTriggeredSpell(caster, target, MissileInfo, bus)
                    └── Spell(7268, TriggeredFullMask)
                        └── Prepare → Cast → HandleImmediate → 伤害 → Finish
```

与 TC 一致：每次 tick 创建完整 Spell 实例走 CastSpell 流程，使用 TriggeredFullMask 跳过 GCD/冷却/法力。触发法术独立走 Prepare → Cast → HandleImmediate → Finish 生命周期，有独立的 TargetInfo、Crit 判定、伤害计算。

### 数据配置

```go
// Arcane Missiles (5143) - 父法术定义
var Info = spell.SpellInfo{
    ID:          5143,
    Name:        "Arcane Missiles",
    CastTime:    0,        // 即时施法
    Duration:    3000,     // 引导 3 秒
    RangeMax:    30,       // 30 码
    PowerCost:   85,       // 85 法力
    PowerType:   0,        // 法力
    IsChanneled: true,
    Attributes:  spell.AttrChanneled | spell.AttrBreakOnMove,
    Effects: []spell.SpellEffectInfo{
        {
            EffectIndex:    0,
            EffectType:     spell.EffectApplyAura,
            BasePoints:     1,      // 触发次数 (逻辑值)
            AuraType:       uint16(AuraPeriodicTriggerSpell), // 新增 Aura 类型
            AuraPeriod:     1000,   // 每 1 秒触发一次
            TriggerSpellID: 7268,   // 触发的法术 ID
            TargetA:        spell.TargetUnitTargetEnemy,
        },
        // 效果 2 (Dummy) 省略
    },
}

// Arcane Missile (7268) - 触发法术定义（内部使用，不暴露给外部）
var missileInfo = spell.SpellEffectInfo{
    EffectIndex:  0,
    EffectType:   spell.EffectSchoolDamage,
    BasePoints:   24,
    BonusCoeff:   0.132,
    TargetA:      spell.TargetUnitTargetEnemy,
}
```

### 施法流程

1. **触发**: 玩家选择敌对目标，主动施法
2. **前置检查** (CheckCast):
   - 施法者存活 (IsAlive)
   - 施法者可施法状态 (CanCast)
   - 法力检查 (≥ 85)
   - 距离检查 (目标 ≤ 30 码)
   - 目标存活 (IsAlive)
3. **施法时间**: 0ms（即时），CastTime=0 导致 Prepare() 直接调 Cast(true)
4. **Cast 进入引导**:
   - `IsChanneled = true` → State 设为 `StateChanneling`
   - Timer = Duration = 3000ms
   - 扣除 85 法力
   - 发布 OnSpellLaunch 事件
5. **Aura 创建**: CastArcaneMissiles 函数在 spell 进入引导后：
   - 创建 Aura (PeriodicTriggerSpell, 3s, 1s period)
   - Aura 的 TriggerSpellID = 7268
   - 添加到 auraMgr，绑定到目标
6. **引导期间**: Update() 每步倒计时 Timer
   - auraMgr.TickPeriodic() 检测到 PeriodicTriggerSpell 类型
   - 每 1 秒触发一次 onTick 回调
   - onTick 回调执行触发法术的伤害计算
   - 发布 OnSpellHit 事件（每发飞弹独立）
7. **引导完成**: Timer 到 0 → Aura 自然过期 → OnAuraExpired → Finish(CastOK)
8. **异常中断**: 移动/被打断/目标死亡 → Cancel()
   - 引导取消联动 Aura 移除（与暴风雪一致）
   - 已发射的飞弹不回溯（伤害已生效）

**状态流转图**:
```
NULL ──Prepare()──▶ PREPARING (0ms)
                        │ cast(true) 即时完成
                        ▼
                   CHANNELING (3000ms)
                   │              │
                   │ tick ×3      │ cancel()
                   │ (每 1s       ▼
                   │  触发一发)  FINISHED
                   ▼             (Interrupted)
                   FINISHED
                   (CastOK)
```

**对比暴风雪的引导流程**:
```
暴风雪:     CHANNELING → TickPeriodicArea (每 tick 重新选目标)
奥术飞弹:   CHANNELING → TickPeriodic     (目标固定，每 tick 触发法术)
```

### 效果执行逻辑

**效果 1: 周期触发 → 触发法术伤害**

执行时机：Aura 每 1 秒 tick 时，通过 onTick 回调执行触发法术效果

每发飞弹伤害公式：`damage = 24 + 0.132 × spellPower`
- BasePoints = 24
- BonusCoeff = 0.132
- 无随机方差（触发法术的 BaseDieSides = 0）
- 暴击判定: 独立判定（每发飞弹独立暴击）
- 暴击倍率: 1.5x

**3 发总计**:
- 无暴击: 72 + 0.396 × SP
- 全暴击: 108 + 0.594 × SP
- SP=100 时: 72 + 39.6 = 111.6 (无暴击) ~ 167.4 (全暴击)

命中判定: 初版简化为必中（TC 中 Arcane Missile 触发法术使用 `TRIGGERED_FULL_MASK` 跳过命中检查）

### Aura 生命周期

1. **创建时机**: Spell 进入 StateChanneling 后，由 CastArcaneMissiles 函数创建
   - 绑定到目标 (targetID)
   - 持续时间 3 秒，与引导同步
   - Aura 类型: PeriodicTriggerSpell
2. **应用 (Apply)**: auraMgr.AddAura()，触发 OnAuraApplied 事件
3. **刷新 (Refresh)**: StackNone — 不允许叠加（同一施法者对同一目标只能有一个奥术飞弹 Aura）
4. **堆叠变化**: 不适用 (MaxStack = 1)
5. **周期效果**: 每 1 秒触发一次
   - TickPeriodic 检测到 AuraPeriodicTriggerSpell 类型
   - 调用 onTick 回调
   - 回调中：
     a. 读取 AuraEffect 的 TriggerSpellID (7268)
     b. 获取触发法术的效果定义 (BasePoints=24, BonusCoeff=0.132)
     c. 计算伤害: `24 + 0.132 × spellPower`
     d. 暴击判定（独立）
     e. 发布 OnSpellHit 事件（每发飞弹）
   - 共 3 次 tick (1000ms/2000ms/3000ms)
6. **移除 (Remove)**:
   - 正常: 3 秒到期 → OnAuraExpired → Finish(CastOK)
   - 引导取消: cancel() → 移除 Aura → 后续 tick 不触发
7. **中断响应**:
   - 引导取消 → 立即移除 Aura（与暴风雪一致）
   - 目标死亡 → Aura 被移除（TickPeriodic 检查目标存活）

### 特殊逻辑

#### 周期触发法术机制 (Periodic Trigger Spell)

**机制目的**: 每 tick 不是直接计算伤害，而是触发一个独立的法术来处理效果

**为什么需要**:
1. WoW 中很多技能使用这种模式（奥术飞弹、腐蚀术种子等）
2. 触发法术可以有自己的暴击/命中判定，独立于父法术
3. 触发法术可以触发 proc（如每发飞弹独立触发 Clearcasting）
4. 框架需要支持"法术触发法术"这一通用模式

**设计方案**:

在 `aura.AuraType` 中新增 `AuraPeriodicTriggerSpell`，在 `TickPeriodic` 中对这种 Aura 类型走特殊处理路径：

```go
// TickPeriodic 中新增分支
case AuraPeriodicTriggerSpell:
    // 1. 读取 TriggerSpellID
    // 2. 查找触发法术的效果定义（从本地 missileInfo 或全局注册表）
    // 3. 构建效果上下文
    // 4. 调用 effect.Process() 处理效果
    // 5. 发布 OnSpellHit 事件
```

**完整 Spell 实例方案**: 每次 tick 创建完整 Spell 实例，走完整 CastSpell 流程（复刻 TC 原版）

```go
// onTick 回调中
func onTick(caster Caster, targetID uint64, info *spell.SpellInfo, bus *event.Bus) {
    s := spell.NewSpell(info.ID, info, caster, spell.TriggeredFullMask)
    s.Targets.UnitTargetID = targetID
    s.Bus = bus
    s.Prepare()  // CastTime=0 → 立即 Cast(true) → HandleImmediate → Finish
}
```

**设计理由**:
- TC 原版 `HandlePeriodicTriggerSpellAuraTick` 调用 `caster->CastSpell(target, triggerSpellId, TRIGGERED_FULL_MASK)`
- 完整 Spell 实例保证统一的 Prepare → Cast → HandleImmediate 流程
- 触发法术有独立的 SelectTargets、ProcessEffects、OnSpellHit 发布，完全复用现有管线
- 为未来 proc 触发和嵌套施法预留正确的基础

#### 与暴风雪的对比

| 维度 | 暴风雪 | 奥术飞弹 |
|------|--------|----------|
| 目标类型 | 区域 AoE | 单体 |
| tick 函数 | TickPeriodicArea | TickPeriodic |
| tick 伤害方式 | 直接计算伤害 | 触发法术→计算伤害 |
| 目标选择 | 每 tick 重新选 | 固定目标 |
| Aura 绑定 | 区域级（无特定目标）| 目标级 |
| 暴击判定 | 每个 tick 独立 | 每发飞弹独立 |
| 引导取消联动 | 移除 Aura | 移除 Aura（一致）|

### 跨表数据说明

| 字段 | 来源 | 值 | 说明 |
|------|------|-----|------|
| Duration | SpellDurationEntry | 3000ms | 引导持续时间和 Aura 持续时间共用 |
| Period | SpellEffectInfo.ApplyAuraPeriod | 1000ms | 触发周期 |
| Range | SpellRangeEntry | 30 码 | 施法距离 |
| CastTime | SpellCastTimesEntry | 0 (即时) | 无施法条，直接进入引导 |
| Speed | SpellMisc.Speed | 0 | 无弹道（飞弹是视觉，服务端即时命中） |
| IsChanneled | SpellMisc.Attributes | bit flag | 引导标记 |
| TriggerSpellID | SpellEffectInfo.TriggerSpell | 7268 | 触发的飞弹法术 |

**触发法术 7268 跨表数据**: 见 [arcane-missile-7268.md](arcane-missile-7268.md) 跨表数据段落。关键值: Speed=0 (即时命中), BasePoints=24, BonusCoeff=0.132, DieSides=0。
