# 火球术 (Fireball)

> 来源: WoW 技能 (Spell ID: 25306, Rank 12) | 生成日期: 2026-04-23

## 概述

火球术是法师的基础火系伤害法术，具有 3.5 秒施法时间和 35 码射程。命中目标后造成直接火焰伤害，并附加一个持续 8 秒的周期性火焰伤害 DoT 效果。作为法术强度系数为 100% 的法术（3.5s / 3.5s = 1.0），火球术是法师火系天赋树的核心输出技能，在装备提升后收益显著。

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害 | SPELL_EFFECT_SCHOOL_DAMAGE + SPELL_AURA_PERIODIC_DAMAGE |
| 施法时间 | 3.5 秒 | SpellCastTimesEntry |
| 冷却 | 无 | 无冷却条目 |
| 充能 | 无 | — |
| 资源消耗 | 410 点法力 | SpellPowerEntry |
| 距离 | 35 码 | SpellRangeEntry |
| GCD | 触发 (1.5 秒) | GlobalCooldownMgr |
| 施法条件 | 需要站立、需要敌方目标、不可移动施法 | SPELL_INTERRUPT_FLAG_INTERRUPT |
| 弹道速度 | ~20 码/秒 | SpellMisc.Speed |
| 弹道最短时间 | 0 秒 | SpellMisc.MinDuration |

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 单体 | TARGET_UNIT_TARGET_ENEMY |
| 目标数量 | 1 | MaxAffectedTargets = 1 |
| 目标筛选 | 敌方 | SpellTargetCheckTypes = TARGET_CHECK_ENEMY |
| 半径 | 无 | — |
| 锥形角度 | 无 | — |
| 链式跳跃 | 无 | — |

## 效果拆解

### 效果 1: 火焰伤害 (School Damage)

- **类型**: 直接伤害 (TC: `SPELL_EFFECT_SCHOOL_DAMAGE`)
- **基础数值**: 596-760 (随机范围)
- **系数**: 法术强度 × 1.0 (100% 系数，基于 3.5s 施法时间的标准公式)
- **作用目标**: TargetA = TARGET_UNIT_TARGET_ENEMY
- **暴击**: 可暴击，暴击倍率 1.5x
- **WoW 映射**: 直接映射 → `EffectSchoolDamage`
- **备注**: 伤害类型为火焰 (Fire school)，受目标火焰抗性减免

### 效果 2: 周期性火焰伤害 (Periodic Damage DoT)

- **类型**: 施加 Aura (TC: `SPELL_EFFECT_APPLY_AURA` + `SPELL_AURA_PERIODIC_DAMAGE`)
- **基础数值**: 每次 19 点伤害
- **系数**: DoT 部分共享法术强度系数的约 12.5%（按 TC 的 DoT 系数分配规则）
- **作用目标**: TargetA = TARGET_UNIT_TARGET_ENEMY
- **暴击**: 可暴击（WotLK+ 机制）
- **WoW 映射**: 直接映射 → `EffectApplyAura` + `AuraPeriodicDamage`
- **备注**: 每 2 秒触发一次，共 4 次 (8s / 2s)，总计 76 点基础 DoT 伤害

## Aura

### Aura 1: 火球术 DoT

- **类型**: 周期性伤害 (TC: `SPELL_AURA_PERIODIC_DAMAGE`)
- **持续时间**: 8 秒
- **堆叠规则**: 不可堆叠，重复施法刷新持续时间 (StackRefresh)
- **最大堆叠时行为**: 刷新持续时间回到 8 秒
- **触发条件 (Proc)**: 无
- **周期效果**: 每 2 秒造成火焰伤害 (19 基础 + 法术强度系数)
- **中断条件**: 无特殊中断条件，正常到期移除或目标死亡
- **WoW 映射**: 直接映射
- **备注**: DoT 不与直接伤害暴击联动，独立暴击判定

## WoW 框架映射汇总

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果1: 直接伤害 | `SPELL_EFFECT_SCHOOL_DAMAGE` | 直接映射 | EffectSchoolDamage |
| 效果2: 施加 DoT | `SPELL_EFFECT_APPLY_AURA` | 直接映射 | EffectApplyAura |
| Aura: 周期伤害 | `SPELL_AURA_PERIODIC_DAMAGE` | 直接映射 | AuraPeriodicDamage |
| 目标选择 | `TARGET_UNIT_TARGET_ENEMY` | 直接映射 | 单体敌方 |
| 冷却 | 无 | 直接映射 | 无冷却 |
| 资源消耗 | 法力 (PowerType 0) | 直接映射 | 410 法力 |
| 施法流程 | 3.5s 施法 → 弹道飞行 → 命中 | 直接映射 | PREPARING → LAUNCHED → FINISHED |
| 弹道延迟 | SpellMisc.Speed 驱动 | 直接映射 | hitDelay = max(dist/Speed, MinDuration) |

## 实现建议

### 整体设计思路

火球术是一个**标准施法型弹道技能**，实现路径匹配 TC 框架：

```
SpellInfo (静态定义，含 Speed 字段)
    ↓
Spell (运行时实例)
    ↓ Prepare() → CheckCast() → 3.5s 倒计时 → Cast()
    ↓
StateLaunched (弹道飞行中)
    ↓ hitDelay = max(dist / Speed, MinDuration)
    ↓
TargetInfo (选定目标)
    ↓
Effect Pipeline (两个效果)
    ├── EffectSchoolDamage → 目标受到直接伤害
    └── EffectApplyAura → 目标获得 DoT Aura
    ↓
Aura Manager (管理 DoT 周期)
```

数据流：SpellInfo 定义 → NewSpell 创建实例 → 施法流程驱动 → **弹道延迟** → Effect 分发 → 各子系统执行。

数据流：SpellInfo 定义 → NewSpell 创建实例 → 施法流程驱动 → Effect 分发 → 各子系统执行。

火球术不需要任何自创机制，所有组件都有直接对应的 WoW 映射。

### 数据配置

所有参数均为数据驱动，通过 `SpellInfo` 和 `SpellEffectInfo` 结构体配置：

```go
// 配置项（数据驱动）
fireballInfo := &spell.SpellInfo{
    ID:          25306,
    Name:        "Fireball",
    CastTime:    3500,    // 3.5 秒 (毫秒)
    RangeMax:    35,
    PowerCost:   410,
    PowerType:   0,       // 法力
    Attributes:  spell.AttrBreakOnMove,
    Speed:       20.0,    // 弹道速度 20 码/秒 (SpellMisc.Speed)
    MinDuration: 0,       // 无最短飞行时间 (SpellMisc.MinDuration)
    Effects: []spell.SpellEffectInfo{
        {EffectIndex: 0, EffectType: spell.EffectSchoolDamage, BasePoints: 678, BonusCoeff: 1.0, TargetA: spell.TargetUnitTargetEnemy},
        {EffectIndex: 1, EffectType: spell.EffectApplyAura,   BasePoints: 19,  BonusCoeff: 0.125, TargetA: spell.TargetUnitTargetEnemy, AuraType: aura.AuraPeriodicDamage, AuraPeriod: 2000},
    },
}
```

不需要脚本扩展，纯数据配置即可实现。

### 施法流程

1. **触发**: 玩家主动施法
2. **前置检查** (CheckCast):
   - 施法者存活 (IsAlive)
   - 施法者可施法状态 (CanCast，非昏迷/沉默等)
   - 目标有效且存活
   - 距离检查 (35 码内)
   - 法力检查 (≥ 410)
   - 冷却检查 (火球术无冷却，但需检查 GCD)
3. **施法时间**: 3.5 秒倒计时 (StatePreparing)
   - 期间可被打断：移动、昏迷、沉默、目标死亡/超出范围
4. **施法完成**: Update(3500) 触发 cast()
5. **弹道飞行** (StateLaunched): cast() 检测到 Speed > 0，进入 StateLaunched
   - 计算 hitDelay = max(distance / Speed, MinDuration)，距离最小钳制 5 码
   - 施法者已自由，可以开始下一个施法
   - 效果尚未处理，伤害和 Aura 都未生效
6. **弹道命中**: Update(hitDelay) 倒计时到达，触发 HandleImmediate()
7. **目标选择**: 显式目标 — TargetUnitTargetEnemy (玩家已选中的敌方单位)
8. **效果分发**:
   - EffectSchoolDamage → handleSchoolDamage → 计算并应用火焰伤害
   - EffectApplyAura → handleApplyAura → 创建并挂载 DoT Aura
9. **后续处理**:
   - 扣除 410 法力 (在 cast() 开始时)
   - 触发 1.5s GCD
   - 触发 Proc 事件 (OnSpellCast, OnDamageDealt)

### 效果执行逻辑

**效果 1: School Damage**

执行时机：命中时 (HandleHit)
计算公式：`damage = basePoints + randomVariance + bonusCoeff * spellPower`
- basePoints = 678 (596 和 760 的平均值)
- randomVariance = rand(-82, +82) (约 ±12%)
- bonusCoeff = 1.0
- spellPower = 施法者法术强度
修正器链：天赋增伤 (如火焰强化 +10%) → 目标抗性减免
暴击判定：基于施法者 CritChance 属性，暴击倍率 1.5x
命中判定：法术通常不可被闪避/招架，但可被免疫 (HitImmune)

**效果 2: Apply Aura (DoT)**

执行时机：命中时 (HandleHit)，紧接效果 1 之后
创建 Aura：
- SpellID = 25306
- CasterID = 施法者 ID
- TargetID = 目标 ID
- AuraType = AuraPeriodicDamage
- Duration = 8s
- StackRule = StackRefresh (重复施法刷新)
周期效果：每 2 秒触发 TickPeriodic
Tick 计算：`tickDamage = 19 + 0.125 * spellPower`

效果间关系：效果 1 和效果 2 无依赖关系，可顺序执行。效果 1 的暴击不影响效果 2。

### Aura 生命周期

1. **创建时机**: 施法命中后，由 EffectApplyAura 处理器创建
2. **应用 (Apply)**: 首次挂载到目标，初始化 Tick 计时器
3. **刷新 (Refresh)**: 重复施法时，StackRefresh 规则 — 重置 Duration 为 8 秒，不重置 Tick 计时器（TC 原始行为）
4. **堆叠变化**: 不适用 (MaxStack = 1)
5. **周期效果**: 每 2 秒 TickPeriodic 触发
   - 计算 tick 伤害 (19 + coeff * SP)
   - 调用伤害回调
   - 递增 TicksDone
6. **移除 (Remove)**:
   - 正常: 8 秒到期 (TicksDone >= 4)
   - 异常: 目标死亡 → RemoveByDeath
   - 驱散: RemoveByDispel (火焰系可被驱散)
7. **中断响应**: 无特殊中断条件

### 特殊逻辑

无自创机制。火球术的所有行为均由现有框架直接支持。

### 跨表数据说明 (Phase 1.5 补充)

Wowhead 仅展示 Spell.dbc 数据。火球术的弹道数据来自 **SpellMisc.db2**（TC 源码 `DB2Structure.h` SpellMiscEntry），不在 Spell.dbc 中：

| 字段 | 来源 | 说明 |
|------|------|------|
| Speed | SpellMisc.db2 | 弹道速度（码/秒），服务端用于计算命中延迟 |
| LaunchDelay | SpellMisc.db2 | 施法完成后等待多久才发射弹道（火球为 0） |
| MinDuration | SpellMisc.db2 | 弹道最短飞行时间（秒），即使近距离也保证一定延迟 |

注意：`SpellVisualEffectNameEntry.BaseMissileSpeed` 是**客户端视觉速度**，服务端不使用。服务端的弹道延迟完全由 SpellMisc 的三个字段驱动。

**TC 命中延迟公式**（`Spell.cpp`）：
```
hitDelay = LaunchDelay + max(distance / Speed, MinDuration)
distance = max(casterToTargetDistance, 5.0)  // 最小 5 码
```
