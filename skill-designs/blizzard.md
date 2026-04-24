# 暴风雪 (Blizzard)

> 来源: WoW 技能 (Spell ID: 10, Rank 1) | 生成日期: 2026-04-23

## 概述

暴风雪是法师的经典冰霜 AoE 技能，在目标区域生成持续 8 秒的暴风雪区域，每秒对范围内所有敌人造成冰霜伤害。作为**引导型持续区域光环 (Channeled Persistent Area Aura)**，它同时涉及两种重要机制：引导施法状态管理和区域周期伤害。在 WoW 中，暴风雪通过 DynamicObject + DynObjAura 实现——施法者在地面创建一个动态对象，该对象持有周期伤害光环并每秒 tick 一次。暴风雪是法师 AoE 农怪和升级的核心技能。

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害（AoE） | SPELL_EFFECT_PERSISTENT_AREA_AURA + SPELL_AURA_PERIODIC_DAMAGE |
| 施法时间 | 即时（引导 8 秒） | SpellCastTimesEntry = 0, IsChanneled = true |
| 冷却 | 无 | 无冷却条目 |
| 充能 | 无 | — |
| 资源消耗 | 320 点法力 | SpellPowerEntry |
| 距离 | 30 码 | SpellRangeEntry |
| GCD | 触发 (1.5 秒) | GlobalCooldownMgr |
| 施法条件 | 需要站立、需要目标地面位置、引导期间不可移动 | CHANNEL_INTERRUPT_FLAG |
| 引导时长 | 8 秒 | SpellDurationEntry |
| AoE 半径 | 8 码 | SpellRadiusEntry |

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 区域 (地面目标点为圆心) | TARGET_DEST_TARGET_ENEMY (A) + TARGET_UNIT_AREA_ENEMY (B) |
| 目标数量 | 无限 (范围内所有敌方单位) | MaxAffectedTargets = 0 (无限制) |
| 目标筛选 | 敌方 | TARGET_CHECK_ENEMY |
| 半径 | 8 码 | SpellRadiusEntry |
| 锥形角度 | 无 | — |
| 链式跳跃 | 无 | — |

**TC 目标选择细节**: TC 的 DynObjAura 通过 `FillTargetMap` 使用 `Cell::VisitAllObjects` 在每帧重新扫描范围内的单位，支持动态进出（单位走入/走出暴风雪区域）。我们的实现保留每次 tick 重新选目标的流程，用简单的列表遍历替代 Cell 空间分区。

## 效果拆解

### 效果 1: 持续区域周期伤害 (Persistent Area Aura - Periodic Damage)

- **类型**: 持续区域光环 (TC: `SPELL_EFFECT_PERSISTENT_AREA_AURA` + `SPELL_AURA_PERIODIC_DAMAGE`)
- **基础数值**: 每次 25 点冰霜伤害
- **系数**: 法术强度 × 0.042 (每 tick)
- **作用目标**: TargetA = TARGET_DEST_TARGET_ENEMY (地面目标点), TargetB = TARGET_UNIT_AREA_ENEMY (范围内敌方)
- **暴击**: 可暴击（TC 中 Consecrate/Blizzard tick 可暴击可命中失败）
- **WoW 映射**: 需适配 → 现有框架没有 `SPELL_EFFECT_PERSISTENT_AREA_AURA`，用 `EffectApplyAura` + `AuraPeriodicDamage` 组合替代
- **备注**: 每 1 秒触发一次，共 8 tick，总计 200 基础伤害 (25 × 8)

### 效果 2: 减速 (ModSpeed) — 天赋附加

- **类型**: 施加 Aura (TC: `SPELL_AURA_MOD_DECREASE_SPEED`)
- **基础数值**: 无（基础暴风雪**没有减速效果**）
- **天赋**: **Improved Blizzard** (冰霜天赋树第3层) 添加减速
  - Rank 1: -30% 移动速度，持续 1.5s
  - Rank 2: -50% 移动速度，持续 1.5s
  - Rank 3: -65% 移动速度，持续 1.5s
- **WoW 映射**: 后续扩展 → 天赋系统改造暴风雪法术，添加 AuraModDecreaseSpeed 效果
- **备注**: 减速由每个 tick 触发刷新（1.5s 持续 > 1s tick 间隔，持续覆盖）。初版不实现，设计需预留效果扩展能力

## Aura

### Aura 1: 暴风雪周期伤害

- **类型**: 周期性伤害 (TC: `SPELL_AURA_PERIODIC_DAMAGE`)
- **持续时间**: 8 秒（与引导同步）
- **堆叠规则**: 不可堆叠，重复施法替换 (StackReplace)
- **最大堆叠时行为**: 替换旧的暴风雪区域
- **触发条件 (Proc)**:
  - 触发事件: 无（非 proc 触发型）
  - 触发概率: 不适用
  - 内置冷却: 不适用
  - 充能次数: 不适用
- **周期效果**: 每 1 秒造成冰霜伤害 (25 基础 + 0.042 × 法术强度)
- **中断条件**: 引导取消 → 动态对象销毁 → Aura 被移除 (RemoveByDefault)
- **WoW 映射**: 直接映射（AuraType、Period、Duration），但创建方式不同（TC 用 DynObjAura 绑定 DynamicObject，我们用普通 Aura 绑定到目标）
- **备注**: TC 中 DynamicObject 持有 Aura 并负责每帧扫描目标，我们的简化版本将 Aura 直接应用于 Cast 时选中的目标

## WoW 框架映射汇总

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果: 持续区域伤害 | `SPELL_EFFECT_PERSISTENT_AREA_AURA` | 适配修改 | 用 `EffectApplyAura` 替代，缺少 DynamicObject 概念 |
| 效果: 减速 (天赋) | `SPELL_AURA_MOD_DECREASE_SPEED` | 后续扩展 | 基础法术无减速，需天赋系统支持 |
| Aura: 周期伤害 | `SPELL_AURA_PERIODIC_DAMAGE` | 直接映射 | AuraPeriodicDamage |
| 目标选择 (地面) | `TARGET_DEST_TARGET_ENEMY` | 直接映射 | TargetDestTargetEnemy |
| 目标选择 (范围) | `TARGET_UNIT_AREA_ENEMY` + FillTargetMap | 适配修改 | 保留每次 tick 重新选目标的流程，用列表遍历替代 Cell 扫描 |
| 引导状态 | `SPELL_STATE_CHANNELING` | 直接映射 | StateChanneling + Timer 倒计时 |
| 引导取消 | `cancel()` 移除 DynamicObject | 适配修改 | 引导取消时移除 aura |
| 资源消耗 | 法力 (PowerType 0) | 直接映射 | 320 法力 |
| DynamicObject | 地面动态对象，持有 Aura | 省略 | 无空间锚点，但保留 tick 时重新选目标的流程 |

## 实现建议

### 整体设计思路

暴风雪是一个**引导型持续区域技能**，与火球术的差异点：

```
Fireball:  SpellInfo → Spell → Preparing → Launched (弹道) → Hit → Aura
Blizzard:  SpellInfo → Spell → Preparing (0ms) → Channeling (8s) → Aura tick ×8
                                                    ↑ 同时创建
                                                 Persistent Area Aura
```

核心区别：暴风雪没有弹道延迟，Cast 后直接进入引导状态，同时创建 Aura 处理周期伤害。

```
TC 原始架构:
  Spell (引导倒计时)
    └── DynamicObject (地面锚点)
        └── DynObjAura (周期伤害)
            └── AuraEffect (每秒 tick)
                └── FillTargetMap (每帧扫描)

我们的架构 (保留流程，简化空间查询):
  Spell (引导倒计时, StateChanneling)
    └── CastBlizzard 创建区域伤害上下文 (center, radius)
        └── 每次 tick:
            1. SelectArea(center, radius) → 重新选目标 (列表遍历替代 Cell 扫描)
            2. 对选中目标计算伤害
            3. 发布 OnAuraTick 事件
```

**保留 tick 时重新选目标**：TC 的 DynObjAura 每 tick 都重新 FillTargetMap。我们保留这个流程——每次 tick 调用 SelectArea 重新获取范围内敌人，只是用简单的列表遍历替代 Cell 空间分区。这意味着：
- 单位走入暴风雪区域：下一 tick 被选中并受伤
- 单位走出暴风雪区域：下一 tick 不再被选中
- 流程与 TC 一致，空间查询的实现细节不同

### 数据配置

```go
var Info = spell.SpellInfo{
    ID:          10,
    Name:        "Blizzard",
    CastTime:    0,        // 即时施法
    Duration:    8000,     // 引导 8 秒
    RangeMax:    30,       // 30 码
    PowerCost:   320,      // 320 法力
    PowerType:   0,        // 法力
    IsChanneled: true,
    Attributes:  spell.AttrChanneled,
    Effects: []spell.SpellEffectInfo{
        {
            EffectIndex: 0,
            EffectType:  spell.EffectApplyAura,
            BasePoints:  25,
            BonusCoeff:  0.042,
            AuraType:    uint16(aura.AuraPeriodicDamage),
            AuraPeriod:  1000,  // 1 秒 tick
            TargetA:     spell.TargetDestTargetEnemy,
            TargetB:     spell.TargetUnitAreaEnemy,
            MiscValue:   8,     // 半径 8 码
        },
    },
}
```

### 施法流程

1. **触发**: 玩家选择地面目标位置，主动施法
2. **前置检查** (CheckCast):
   - 施法者存活 (IsAlive)
   - 施法者可施法状态 (CanCast)
   - 法力检查 (≥ 320)
   - 距离检查 (目标位置 ≤ 30 码)
3. **施法时间**: 0ms（即时），CastTime=0 导致 Prepare() 直接调 Cast(true)
4. **Cast 进入引导**:
   - `IsChanneled = true` → State 设为 `StateChanneling`
   - Timer = Duration = 8000ms
   - 无弹道（Speed = 0，不走 StateLaunched 路径）
5. **Aura 创建**: CastBlizzard 函数在 spell 进入引导后：
   - 通过 targeting.SelectArea 选中 8 码内敌方单位
   - 为每个目标创建 Aura (PeriodicDamage, 8s, 1s tick)
   - 添加到 auraMgr
6. **引导期间**: Update() 每步倒计时 Timer
   - auraMgr.TickPeriodic() 处理周期伤害 tick
   - 引导结束与 aura 过期解耦（aura 8s 自然过期）
7. **引导完成**: Timer 到 0 → Finish(CastOK)
8. **异常中断**: 移动/被打断 → Cancel() → Finish(CastFailedInterrupted)
   - Aura 独立过期（初版不主动移除）

**状态流转图**:
```
NULL ──Prepare()──▶ PREPARING (0ms)
                        │ cast(true) 即时完成
                        ▼
                   CHANNELING (8000ms)
                   │         │
                   │ tick ×8 │ cancel()
                   ▼         ▼
                   FINISHED  FINISHED
                   (CastOK)  (Interrupted)
```

### 效果执行逻辑

**效果 1: Persistent Area Aura → PeriodicDamage**

执行时机：Cast 时创建 Aura，之后每 1 秒由 auraMgr.TickPeriodic 触发

Tick 计算公式：`damage = 25 + 0.042 × spellPower`
- BasePoints = 25
- BonusCoeff = 0.042
- 无随机方差（WoW 的 Blizzard tick 不随机）
- 总伤害：8 tick × (25 + 0.042 × SP)

伤害修正器链（初版简化，不实现）：
- 冰霜天赋增伤 (如 Ice Shards +crit damage)
- 目标冰霜抗性减免
- 引导缩短（受 haste 影响）

暴击判定：基于施法者 CritChance，暴击倍率 1.5x

命中判定：TC 中 Persistent Area Aura tick 可命中失败 (Consecrate ticks can miss)，初版简化为必中

**AoE 目标选择 (每次 tick)**:
- 每 tick 调用 targeting.SelectArea(center, radius=8, CheckEnemy) 重新选目标
- 流程与 TC 一致：DynObjAura.UpdateOwner → FillTargetMap → 每帧扫描
- 简化点：用列表遍历（遍历所有已知单位检查距离）替代 Cell::VisitAllObjects 空间分区
- 效果：单位走入区域 → 下一 tick 受伤；单位走出区域 → 下一 tick 不再受伤

### Aura 生命周期

1. **创建时机**: Spell 进入 StateChanneling 后，由 CastBlizzard 函数创建
   - 创建一个区域级 Aura，持有 center 位置和 radius，不绑定到具体目标
   - 每次 tick 时重新选目标，对选中目标应用伤害
2. **应用 (Apply)**: auraMgr.AddAura()，触发 OnAuraApplied 事件
3. **刷新 (Refresh)**: StackReplace — 新暴风雪替换旧的（同一个 CasterID + SpellID 组合只应存在一个）
4. **堆叠变化**: 不适用 (MaxStack = 1)
5. **周期效果**: 每 1 秒触发 tick
   - 调用 SelectArea(center, radius) 重新获取范围内敌方单位
   - 对每个目标计算 tick 伤害 (25 + 0.042 × SP)
   - 发布 OnAuraTick 事件
   - 递增 TicksDone
   - 共 8 次 tick
6. **移除 (Remove)**:
   - 正常: 8 秒到期 (Elapsed >= MaxDuration) → OnAuraExpired
   - 引导取消: cancel() 时 auraMgr.RemoveAurasBySpellID() 立即移除（与 TC 行为一致）
7. **中断响应**:
   - 引导取消 → 移除区域 Aura（与 TC 的 DynamicObject 销毁 → Aura 移除一致）
   - 目标死亡 → 该目标不再被 SelectArea 选中（自然排除，无需额外处理）

### 特殊逻辑

#### Tick 时重新选目标 (与 TC FillTargetMap 流程一致)

- **TC 流程**: DynObjAura.UpdateOwner → FillTargetMap → Cell::VisitAllObjects → 每帧扫描
- **我们的流程**: Aura tick → SelectArea(center, radius) → 列表遍历 → 每 tick 重新选目标
- **简化点**: 空间查询用列表遍历（检查每个单位与 center 的距离 ≤ radius）替代 Cell 空间分区
- **保留的行为**:
  - 单位走入区域 → 下一 tick 受伤
  - 单位走出区域 → 下一 tick 不受伤
  - 新生成的单位 → 下一 tick 被选中
- **不保留的**: 实时帧级扫描（TC 是每帧，我们是每 tick = 1s，精度差异可接受）

#### 引导取消联动 Aura 移除

- **TC 行为**: cancel() → DynamicObject 销毁 → Aura 立即移除
- **我们的行为**: cancel() → auraMgr.RemoveAurasBySpellID() → Aura 立即移除
- **与 TC 一致**: 引导被打断/取消后，暴风雪区域伤害立即停止

### 跨表数据说明 (Phase 1.5 补充)

Wowhead 展示的数据来自 Spell.dbc，暴风雪的引导和周期信息需要跨表确认：

| 字段 | 来源 | 值 | 说明 |
|------|------|-----|------|
| Duration | SpellDurationEntry | 8000ms | 引导持续时间和 Aura 持续时间共用 |
| Period | SpellEffectInfo.ApplyAuraPeriod | 1000ms | Aura tick 周期 |
| Radius | SpellRadiusEntry | 8 码 | AoE 范围 |
| Range | SpellRangeEntry | 30 码 | 施法距离 |
| CastTime | SpellCastTimesEntry | 0 (即时) | 无施法条，直接进入引导 |
| Speed | SpellMisc.Speed | 0 | 无弹道 |
| IsChanneled | SpellMisc.Attributes | bit flag | 引导标记 |

**TC 关键注释** (SpellAuras.cpp:826):
```cpp
// used for example when triggered spell of spell:10 is modded
```
暴风雪 (spell:10) 是 TC 中引导光环需要实时应用 spell mod 的标准示例。

### 与火球术的对比

| 维度 | 火球术 | 暴风雪 |
|------|--------|--------|
| 施法类型 | 有施法条 (3.5s) | 引导 (8s) |
| 目标类型 | 单体 | 区域 AoE |
| 命中方式 | 弹道延迟 | 即时生效 |
| 伤害方式 | 直接 + DoT | 纯周期 (每秒) |
| 效果数 | 2 (SchoolDamage + ApplyAura) | 1 (PersistentAreaAura)，天赋可附加减速 |
| 目标选择时机 | Cast 时选一次 | 每 tick 重新选目标 |
| TC 空间机制 | 无 | DynamicObject (我们用列表遍历) |
| 移动中断 | 施法中移动打断 | 引导中移动打断 |
