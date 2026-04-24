# 活动炸弹周期 (Living Bomb Periodic) — 周期 DoT 法术

> 来源: WoW 技能 (Spell ID: 217694, Legion+) | 生成日期: 2026-04-24

## 概述

活动炸弹周期法术 217694 是三体结构中的**核心载体**——它挂载在目标身上作为周期性伤害 Aura (DoT)，每秒造成一次火焰伤害，持续 4 秒。当 Aura 自然过期时（仅 RemoveMode == EXPIRE），触发爆炸法术 44461。这是三体结构的"中间层"，连接施放入口 (44457) 和爆炸输出 (44461)。

**关键机制**：217694 的 Aura 过期回调通过 TC 的 `AfterEffectRemove` 脚本 hook 实现，并且**严格过滤 RemoveMode**——只有自然过期才触发爆炸，目标死亡、被驱散、施法者死亡等情况都不触发爆炸。

**父法术**: [living-bomb-44457.md](living-bomb-44457.md) — 施放入口
**子法术**: [living-bomb-44461.md](living-bomb-44461.md) — 爆炸

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害（周期 DoT） | SPELL_AURA_PERIODIC_DAMAGE + SPELL_EFFECT_DUMMY |
| 施法时间 | 即时 | 触发法术，CastTime = 0 |
| 冷却 | 无 | 被触发法术跳过 |
| 充能 | 无 | — |
| 资源消耗 | 无 | TRIGGERED_FULL_MASK 跳过消耗 |
| 距离 | 40 码 | SpellRangeEntry |
| GCD | 不触发 | TRIGGERED_FULL_MASK 跳过 GCD |
| 法术分类 | 隐藏 | 触发法术，不可直接施放 |

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 单体 (继承父法术目标) | TARGET_UNIT_TARGET_ENEMY |
| 目标数量 | 1 | MaxAffectedTargets = 1 |
| 目标筛选 | 敌方 | TARGET_CHECK_ENEMY |
| 半径 | 无 | — |

## 效果拆解

### 效果 0: 周期性火焰伤害 (Periodic Damage)

- **类型**: 施加 Aura (TC: `SPELL_EFFECT_APPLY_AURA` + `SPELL_AURA_PERIODIC_DAMAGE`)
- **基础数值**: 基于 44457 效果 1 的 BasePoints
- **系数**: 法术强度 × 0.06 (6%)
- **周期**: 每 1 秒
- **持续时间**: 4 秒 (4 tick)
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **暴击**: 可暴击 (独立判定)
- **WoW 映射**: 直接映射 → AuraPeriodicDamage
- **备注**: 核心伤害来源，每秒 tick 一次

### 效果 1: Dummy (脚本 — 过期触发爆炸)

- **类型**: Dummy (TC: `SPELL_EFFECT_DUMMY`)
- **基础数值**: 不适用
- **系数**: 不适用
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **WoW 映射**: 脚本处理 — TC 中通过 `spell_gen_living_bomb_periodic` 脚本
- **脚本行为**:
  - Hook: `AfterEffectRemove(EFFECT_1, AURA_EFFECT_DUMMY)`
  - 条件: `if (aura.GetRemoveMode() == AURA_REMOVE_BY_EXPIRE)`
  - 动作: `caster->CastSpell(target, 44461, TRIGGERED_FULL_MASK, &basePoints)`
- **备注**: 这个 Dummy 效果的唯一目的是提供一个 hook 点，让脚本在 Aura 移除时触发爆炸。`AURA_REMOVE_BY_EXPIRE` 过滤确保只有自然过期才爆炸

### SpellValue 传递

217694 从 44457 继承 `BASE_POINT2`：
- **BASE_POINT2 = 1**: 原始活动炸弹（由玩家直接施放的 44457 触发），爆炸后**可以传染**
- **BASE_POINT2 = 0**: 传染副本（由 44461 爆炸传染触发），爆炸后**不可再传染**

这个值存储在 Aura 的 SpellValue 中，在触发爆炸时传递给 44461。

## Aura

### Aura 1: 活动炸弹 DoT

- **类型**: 周期性伤害 (TC: `SPELL_AURA_PERIODIC_DAMAGE`)
- **持续时间**: 4 秒
- **堆叠规则**: 不可堆叠，重复施法刷新 (StackRefresh)
- **最大堆叠时行为**: 刷新持续时间回到 4 秒
- **触发条件 (Proc)**: 无
- **周期效果**: 每 1 秒造成火焰伤害
  - tick 伤害 = basePoints + 0.06 × spellPower
  - 共 4 次 tick (1000ms / 2000ms / 3000ms / 4000ms)
  - **首次 tick**: t = 1s（无 SPELL_ATTR5_EXTRA_INITIAL_PERIOD 标志）
- **中断条件**: 无特殊中断条件
- **过期行为**: 自然过期 → 脚本触发爆炸 44461
- **WoW 映射**: 适配修改 — 需要在 Aura 过期时调用脚本 hook 并传递 RemoveMode
- **备注**: 这是三体结构的核心 Aura

### Tick 时间线

| Tick # | 时间 | 事件 |
|--------|------|------|
| 1 | 1000ms | 第一次周期伤害 |
| 2 | 2000ms | 第二次周期伤害 |
| 3 | 3000ms | 第三次周期伤害 |
| 4 | 4000ms | 第四次周期伤害 → Aura 过期 → 触发 44461 爆炸 |

**注意**: 首次 tick 在 t=1s 而非 t=0s。TC 中 `_periodicTimer` 初始为 0，第一个 period (1s) 后才触发第一次 tick。

## WoW 框架映射汇总

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果0: PeriodicDamage | `SPELL_AURA_PERIODIC_DAMAGE` | 直接映射 | AuraPeriodicDamage |
| 效果1: Dummy (过期 hook) | `SPELL_EFFECT_DUMMY` + AuraScript | 需新增 | 需 Aura 过期回调 hook |
| Aura: 周期伤害 | `SPELL_AURA_PERIODIC_DAMAGE` | 直接映射 | 每 1s tick |
| 过期触发 | `AfterEffectRemove` + RemoveMode 过滤 | 需新增 | 仅 EXPIRE 触发 |
| SpellValue | BASE_POINT2 (传染标记) | 需新增 | 需 SpellValue 存储在 Aura 中 |
| 触发施法 | `CastSpell(44461, TRIGGERED_FULL_MASK)` | 需新增 | 过期时创建爆炸法术 |

## 实现建议

### 整体设计思路

217694 是三体结构的**枢纽**——它接收来自 44457 的"施放请求"，以 Aura 形式在目标上持续存在，在过期时"引爆"44461。

```
217694 的架构角色:

Aura 挂载在目标上
    │
    ├── 每 1s: TickPeriodic → 周期火焰伤害
    │
    │   t=1000ms: tick 1 (伤害)
    │   t=2000ms: tick 2 (伤害)
    │   t=3000ms: tick 3 (伤害)
    │   t=4000ms: tick 4 (伤害)
    │
    └── Aura 过期:
        │
        │ AuraScript: AfterEffectRemove(EFFECT_1)
        │ if RemoveMode == EXPIRE:
        │     CastSpell(44461, SpellValues{BASE_POINT0: storedDamage})
        │
        ▼
    44461 爆炸法术
```

关键框架需求：
1. **Aura 过期回调** — `AuraScript.AfterEffectRemove` hook
2. **RemoveMode 传递** — AuraContext 必须包含 RemoveMode
3. **SpellValue 存储在 Aura** — 传染标记 (BASE_POINT2) 需要随 Aura 传递
4. **过期触发施法** — 在 Aura 过期回调中创建新法术实例

### 数据配置

```go
var PeriodicInfo = spell.SpellInfo{
    ID:        217694,
    Name:      "Living Bomb Periodic",
    CastTime:  0,    // 触发法术
    Duration:  4000, // 4 秒
    RangeMax:  40,
    PowerCost: 0,    // 由 TRIGGERED 跳过
    Effects: []spell.SpellEffectInfo{
        {
            EffectIndex: 0,
            EffectType:  spell.EffectApplyAura,
            BasePoints:  0,    // 从父法术的 BasePoints 传递
            BonusCoeff:  0.06,
            AuraType:    uint16(aura.AuraPeriodicDamage),
            AuraPeriod:  1000, // 每 1 秒
            TargetA:     spell.TargetUnitTargetEnemy,
        },
        {
            EffectIndex: 1,
            EffectType:  spell.EffectDummy, // 脚本 hook 点
            TargetA:     spell.TargetUnitTargetEnemy,
        },
    },
}
```

### 施法流程

217694 是触发法术，由 44457 的脚本施放：

1. **触发**: 44457 脚本中 CastSpell(217694, TRIGGERED_FULL_MASK)
2. **前置检查**: 由 TriggeredFullMask 跳过大部分检查
3. **Cast**: 即时完成 → HandleImmediate()
4. **效果分发**:
   - EFFECT_0 (ApplyAura) → 创建 DoT Aura (4s, 1s period)
   - EFFECT_1 (Dummy) → 空操作（仅作为脚本 hook 点）
5. **Aura 挂载**: 目标获得 DoT Aura，开始周期 tick

### Aura 生命周期

1. **创建时机**: 44457 脚本施放 217694，HandleImmediate 中 EFFECT_0 创建 Aura
2. **应用 (Apply)**: AddAura → 目标获得活动炸弹 Debuff
   - 存储 SpellValues: BASE_POINT2 (传染标记)
3. **刷新 (Refresh)**: StackRefresh — 重复施放时重置 4s 持续时间
4. **堆叠变化**: 不适用 (MaxStack = 1)
5. **周期效果**: 每 1 秒 TickPeriodic
   - 计算 tick 伤害: `basePoints + 0.06 × spellPower`
   - 发布 OnAuraTick 事件
   - 共 4 次 tick
6. **移除 (Remove)**:
   - **自然过期** (RemoveByExpire):
     - Tick 完成 4 次 → Aura 持续时间归零
     - 调用 `AfterEffectRemove` hook
     - RemoveMode == RemoveByExpire → CastSpell(44461)
     - 传递 BASE_POINT0 = tickDamage, BASE_POINT2 = 传染标记
   - **目标死亡** (RemoveByDeath):
     - RemoveMode == RemoveByDeath → **不触发爆炸**
   - **驱散** (RemoveByDispel):
     - RemoveMode == RemoveByDispel → **不触发爆炸**
   - **施法者死亡** (RemoveByCasterDeath):
     - RemoveMode == RemoveByCasterDeath → **不触发爆炸**
7. **中断响应**: 无特殊中断条件（不因移动/攻击中断）

### 特殊逻辑

#### RemoveMode 过滤

**机制目的**: 只有活动炸弹自然过期才触发爆炸，防止死亡/驱散等场景触发

**为什么需要**:
1. TC 中活动炸弹明确只在 `AURA_REMOVE_BY_EXPIRE` 时爆炸
2. 目标死亡时不应该爆炸（尸体爆炸不符合游戏逻辑）
3. 驱散活动炸弹是反制手段，不应触发爆炸（否则驱散反而帮了对方）

**设计方案**:
- `AuraContext` 包含 `RemoveMode aura.RemoveMode`
- 脚本 hook 检查 `ctx.RemoveMode == aura.RemoveByExpire`
- 只有匹配时才执行爆炸逻辑

#### SpellValue 在 Aura 中的传递

**机制目的**: 传染标记需要在整个三体链中传递

**为什么需要**:
1. 原始活动炸弹 (BASE_POINT2=1) 爆炸后传染
2. 传染副本 (BASE_POINT2=0) 爆炸后不传染
3. 这个标记从 44457 → 217694 Aura → 44461 → 新的 217694 Aura

**设计方案**:
- Aura 结构体添加 `SpellValues map[uint8]float64`
- 44457 脚本创建 217694 时设置 SpellValues[2] = 1
- 44461 爆炸传染时创建 217694 设置 SpellValues[2] = 0
- Aura 过期时将 SpellValues 传递给 44461

### 跨表数据说明

| 字段 | 来源 | 值 | 说明 |
|------|------|-----|------|
| Duration | SpellDurationEntry | 4000ms | DoT 持续时间 |
| Period | SpellEffectInfo.ApplyAuraPeriod | 1000ms | tick 周期 |
| Range | SpellRangeEntry | 40 码 | 触发法术距离（较大以确保不中断） |
| CastTime | SpellCastTimesEntry | 0 (即时) | 触发法术 |
| Speed | SpellMisc.Speed | 0 | 无弹道 |
