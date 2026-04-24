# 活动炸弹 (Living Bomb) — 施放法术

> 来源: WoW 技能 (Spell ID: 44457) | 生成日期: 2026-04-24

## 概述

活动炸弹是火焰法师的标志性 DoT + AoE 技能，采用**三体法术结构**：玩家施放的活动炸弹本身 (44457) 是一个入口法术，其效果被 TC 脚本拦截（EffectDummy → PreventDefault），转而向目标施放周期法术 217694。周期法术到期后触发爆炸法术 44461，对周围敌人造成 AoE 伤害并将活动炸弹传染给附近目标。这种"施放→周期→爆炸→传染"的链式结构是 WoW 法术脚本系统的经典范式。

**三体关系图**：

```
玩家施放 44457 (Living Bomb)
    │
    │ TC Script: OnEffectHit EFFECT_0 DUMMY
    │ → PreventDefault → CastSpell(217694, BASE_POINT2=1)
    ▼
目标获得 217694 (Periodic DoT, 4s)
    │ 每 1s tick: 周期火焰伤害
    │
    │ TC Script: AfterEffectRemove EFFECT_1 DUMMY
    │ → if RemoveMode == EXPIRE → CastSpell(44461)
    ▼
爆炸触发 44461 (Living Bomb Explode)
    │ AoE 10yd SchoolDamage + 传染
    │
    │ TC Script: OnEffectHit EFFECT_1 SCHOOL_DAMAGE
    │ → if BASE_POINT0 > 0 → CastSpell(217694, BASE_POINT2=0) 每个命中目标
    ▼
传染目标获得 217694 (Periodic DoT, 不可再传染)
    │ BASE_POINT2=0 → 传染副本不会触发二次传染
    ▼
到期 → 再次爆炸 (44461) → 但不传染 (BASE_POINT2=0)
```

**关联设计文档**：
- [living-bomb-217694.md](living-bomb-217694.md) — 周期 DoT 法术
- [living-bomb-44461.md](living-bomb-44461.md) — 爆炸法术

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害（DoT + AoE + 传染） | SPELL_EFFECT_DUMMY (脚本拦截) |
| 施法时间 | 即时 | SpellCastTimesEntry |
| 冷却 | 无 | 无冷却条目 |
| 充能 | 无 | — |
| 资源消耗 | 基础法力 22% | SpellPowerEntry |
| 距离 | 35 码 | SpellRangeEntry |
| GCD | 触发 (1.5 秒) | GlobalCooldownMgr |
| 施法条件 | 需要站立、需要敌对目标 | — |

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 单体 (敌对) | TARGET_UNIT_TARGET_ENEMY |
| 目标数量 | 1 | MaxAffectedTargets = 1 |
| 目标筛选 | 敌方 | TARGET_CHECK_ENEMY |
| 半径 | 无 | — |
| 锥形角度 | 无 | — |
| 链式跳跃 | 无 | — |

## 效果拆解

### 效果 0: Dummy (脚本拦截)

- **类型**: Dummy (TC: `SPELL_EFFECT_DUMMY`)
- **基础数值**: 不适用
- **系数**: 不适用
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **WoW 映射**: 自定义脚本处理 — TC 中通过 `spell_gen_living_bomb` 脚本的 `OnEffectHit` hook 拦截 EFFECT_0
- **脚本行为**:
  1. `PreventDefault()` — 阻止默认效果处理
  2. 以施法者身份向目标施放法术 217694
  3. 传递 `BASE_POINT2 = 1` (标记为"原始活动炸弹"，可传染)
- **备注**: 这就是三体结构的入口——44457 的 SpellInfo 效果是 Dummy，真正的逻辑由脚本接管

### 效果 1: 周期性火焰伤害 (Periodic Damage)

- **类型**: 施加 Aura (TC: `SPELL_EFFECT_APPLY_AURA` + `SPELL_AURA_PERIODIC_DAMAGE`)
- **基础数值**: 153
- **系数**: 法术强度 × 0.2 (20%)
- **周期**: 每 3 秒
- **作用目标**: TARGET_UNIT_TARGET_ENEMY
- **WoW 映射**: 适配修改 — TC 中此效果由脚本替代，实际 DoT 由 217694 承担
- **备注**: WoW 原始数据中存在此效果，但 TC 脚本通过 `PreventDefault()` 跳过了默认处理，由 217694 的周期效果替代。在我们的实现中，可以省略此效果，完全由 217694 承担 DoT 职责

## Aura

不适用 — 44457 自身不施加 Aura。周期 DoT 由触发法术 217694 处理。

## WoW 框架映射汇总

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果0: Dummy | `SPELL_EFFECT_DUMMY` + SpellScript | 需新增 | 需脚本系统 hook + PreventDefault |
| 效果1: PeriodicDamage | `SPELL_AURA_PERIODIC_DAMAGE` | 由 217694 替代 | 脚本拦截，不由 44457 执行 |
| 目标选择 | `TARGET_UNIT_TARGET_ENEMY` | 直接映射 | 单体敌方 |
| 触发法术 | `CastSpell(217694, TRIGGERED_FULL_MASK)` | 需新增 | 脚本中触发独立法术 |
| SpellValue | `BASE_POINT2 = 1` | 需新增 | 需 SpellValue 传递机制 |
| GCD | 触发 1.5s | 直接映射 | |
| 资源消耗 | 基础法力 22% | 直接映射 | |

## 实现建议

### 整体设计思路

活动炸弹 44457 是一个**脚本驱动的触发器法术**——它自身的 SpellInfo 效果不直接产生结果，而是通过脚本系统拦截 Dummy 效果，转而施放真正的周期法术 217694。

```
44457 的架构角色:

SpellInfo (静态定义, Dummy 效果)
    ↓
Spell (运行时实例, 即时施法)
    ↓ Prepare() → Cast() → HandleImmediate()
    ↓
Effect Pipeline → EFFECT_0: Dummy
    ↓
Script Hook: OnEffectHit(EFFECT_0)
    ↓ PreventDefault()
    ↓ CastSpell(217694, SpellValues{BASE_POINT2: 1})
    ↓
217694 接管后续所有逻辑
```

关键框架需求：
1. **脚本 Hook 系统** — Effect Pipeline 中调用注册的 OnEffectHit hook
2. **PreventDefault** — 脚本可阻止默认效果处理
3. **SpellValue 传递** — 施放触发法术时传递 BASE_POINT2 等值
4. **触发施法** — 从脚本中调用 CastSpell 创建新法术实例

### 数据配置

```go
var Info = spell.SpellInfo{
    ID:        44457,
    Name:      "Living Bomb",
    CastTime:  0,    // 即时
    RangeMax:  35,
    PowerCost: 0,    // TODO: 基础法力百分比消耗
    PowerType: 0,    // 法力
    Effects: []spell.SpellEffectInfo{
        {
            EffectIndex: 0,
            EffectType:  spell.EffectDummy,
            TargetA:     spell.TargetUnitTargetEnemy,
        },
        // Effect#1 (PeriodicDamage) 省略 — 由 217694 替代
    },
}
```

### 施法流程

1. **触发**: 玩家选择敌对目标，主动施法
2. **前置检查** (CheckCast):
   - 施法者存活
   - 施法者可施法状态
   - 法力检查 (≥ 22% 基础法力)
   - 距离检查 (目标 ≤ 35 码)
   - 目标存活
3. **施法时间**: 0ms (即时) → Prepare() 直接进入 Cast
4. **Cast**: 即时完成 → HandleImmediate()
5. **目标选择**: 显式目标 — TargetUnitTargetEnemy
6. **效果分发**:
   - EFFECT_0 (Dummy) → 脚本拦截
     - Script: OnEffectHit(EFFECT_0) → PreventDefault
     - CastSpell(target, 217694, TRIGGERED_FULL_MASK, SpellValues{2: 1})
7. **后续处理**: 扣除法力，触发 GCD

**状态流转**:
```
NULL ──Prepare()──▶ PREPARING (0ms)
                        │ 即时
                        ▼
                   CAST → HandleImmediate()
                        │
                        │ Script intercepts EFFECT_0
                        │ → CastSpell(217694)
                        ▼
                   FINISHED (CastOK)
```

### 效果执行逻辑

**效果 0: Dummy (脚本拦截)**

执行时机：HandleImmediate() 中 Effect Pipeline 处理 EFFECT_0 时

脚本行为：
1. 检查是否注册了 OnEffectHit(EFFECT_0) hook
2. 调用 hook → 脚本执行 PreventDefault
3. 脚本中创建 217694 法术实例：
   - caster = 原施法者
   - target = 原目标
   - TriggeredFullMask (跳过 GCD/冷却/法力)
   - SpellValues[2] = 1 (标记为原始活动炸弹，可传染)
4. 由于 PreventDefault，不执行默认 Dummy 处理（handleDummy 本身也是空操作）

### 特殊逻辑

#### 脚本拦截机制 (Script Hook + PreventDefault)

**机制目的**: 允许脚本替换法术的默认效果处理逻辑

**为什么需要**:
1. TC 中大量法术使用 Dummy + Script 模式——SpellInfo 只声明"这里有自定义逻辑"
2. 脚本可以在运行时决定实际行为（如施放另一个法术）
3. PreventDefault 允许脚本完全接管，阻止默认管线继续

**设计方案**:
- `effect.ProcessAll` 中，每个效果处理前后调用注册的 hook
- `SpellContext` 包含 `Prevented` 标志
- 脚本设置 `Prevented = true` 后，默认处理器跳过

### 跨表数据说明

| 字段 | 来源 | 值 | 说明 |
|------|------|-----|------|
| CastTime | SpellCastTimesEntry | 0 (即时) | 无施法条 |
| Range | SpellRangeEntry | 35 码 | 施法距离 |
| PowerCost | SpellPowerEntry | 22% 基础法力 | 百分比消耗 |
| Speed | SpellMisc.Speed | 0 | 无弹道 |
| GCD | GlobalCooldownMgr | 1500ms | 标准 1.5s GCD |
