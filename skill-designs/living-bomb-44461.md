# 活动炸弹爆炸 (Living Bomb Explode) — 爆炸 AoE 法术

> 来源: WoW 技能 (Spell ID: 44461) | 生成日期: 2026-04-24

## 概述

活动炸弹爆炸 44461 是三体结构中的**输出终端**——当 217694 的周期 Aura 自然过期时触发，对爆炸载体周围 10 码内的所有敌人造成火焰 AoE 伤害，并可能将活动炸弹传染给被命中的目标。爆炸法术有两个关键脚本行为：(1) FilterTargets 排除爆炸载体本身不受 AoE 伤害，(2) OnEffectHit 中根据 SpellValue.BASE_POINT2 判断是否触发传染。这是三体结构中唯一涉及 AoE 目标选择的法术。

**父法术**: [living-bomb-217694.md](living-bomb-217694.md) — 周期 DoT (触发爆炸)
**根法术**: [living-bomb-44457.md](living-bomb-44457.md) — 施放入口

## 基本信息

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 分类 | 伤害（AoE + 传染） | SPELL_EFFECT_SCHOOL_DAMAGE + SPELL_EFFECT_DUMMY |
| 施法时间 | 即时 | 触发法术，CastTime = 0 |
| 冷却 | 无 | 被触发法术跳过 |
| 充能 | 无 | — |
| 资源消耗 | 无 | TRIGGERED_FULL_MASK 跳过消耗 |
| 距离 | 50000 码 | SpellRangeEntry (极大，确保不中断) |
| GCD | 不触发 | TRIGGERED_FULL_MASK 跳过 GCD |
| 弹道速度 | 0 (即时命中) | SpellMisc.Speed = 0 |
| 法术分类 | 隐藏 | 触发法术，不可直接施放 |

## 目标

| 属性 | 值 | WoW 参考 |
|------|-----|----------|
| 目标类型 | 区域 AoE (以爆炸载体为中心) | TARGET_UNIT_AREA_ENEMY |
| 目标数量 | 无限制 | MaxAffectedTargets = 0 |
| 目标筛选 | 敌方 | TARGET_CHECK_ENEMY |
| 半径 | 10 码 | SpellRadiusEntry |
| 锥形角度 | 无 | — |
| 链式跳跃 | 无 | — |

**特殊**: TC 脚本通过 FilterTargets 从目标列表中**移除爆炸载体本身**——载体不受到自己爆炸的伤害，但载体周围的其他敌人会受到伤害。

## 效果拆解

### 效果 0: Dummy (脚本 — 控制传染逻辑)

- **类型**: Dummy (TC: `SPELL_EFFECT_DUMMY`)
- **基础数值**: 不适用
- **系数**: 不适用
- **作用目标**: TARGET_UNIT_AREA_ENEMY
- **WoW 映射**: 脚本处理 — TC 中通过 `spell_gen_living_bomb_explosion` 脚本
- **脚本行为**:
  - FilterTargets(EFFECT_0): 从目标列表移除显式目标 (爆炸载体)
  - OnEffectHit(EFFECT_0): 无操作（逻辑在 EFFECT_1 的 OnEffectHit 中处理）
- **备注**: FilterTargets 是 TC SpellScript 的目标筛选 hook，在效果执行前修改目标列表

### 效果 1: 火焰 AoE 伤害 (School Damage)

- **类型**: 直接伤害 (TC: `SPELL_EFFECT_SCHOOL_DAMAGE`)
- **基础数值**: 基于 217694 的周期伤害值
- **系数**: 法术强度 × 0.14 (14%)
- **半径**: 10 码
- **作用目标**: TARGET_UNIT_AREA_ENEMY
- **暴击**: 可暴击 (独立判定)
- **WoW 映射**: 直接映射 → EffectSchoolDamage
- **脚本行为**:
  - OnEffectHit(EFFECT_1, SCHOOL_DAMAGE):
    - 读取 SpellValue.BASE_POINT2 (传染标记)
    - 如果 BASE_POINT2 > 0 → 对每个命中目标 CastSpell(217694, BASE_POINT2=0)
    - 传染的 217694 的 BASE_POINT2=0 → 爆炸后不再传染
- **备注**: 这是三体结构中唯一产生 AoE 伤害和传染行为的效果

## Aura

不适用 — 44461 自身不施加 Aura。传染通过重新施放 217694 实现。

## WoW 框架映射汇总

| 组件 | WoW 机制 | 映射方式 | 备注 |
|------|----------|----------|------|
| 效果0: Dummy | `SPELL_EFFECT_DUMMY` + SpellScript | 需新增 | FilterTargets hook |
| 效果1: SchoolDamage | `SPELL_EFFECT_SCHOOL_DAMAGE` | 直接映射 | AoE 火焰伤害 |
| AoE 目标选择 | `TARGET_UNIT_AREA_ENEMY` + Radius | 需新增 | 需 Spell.SelectTargets 支持 AoE |
| FilterTargets | SpellScript hook | 需新增 | 脚本修改目标列表 |
| 传染逻辑 | OnEffectHit + CastSpell(217694) | 需新增 | 每个 AoE 目标触发新 217694 |
| SpellValue | BASE_POINT2 (传染标记) | 需新增 | 判断是否可传染 |

## 实现建议

### 整体设计思路

44461 是三体结构的**末端输出**——它从 217694 的过期回调中创建，执行 AoE 伤害，并可能触发新一轮传染。它的特殊性在于：

1. **AoE 目标选择** — 需要从爆炸载体位置查找 10 码内的所有敌对目标
2. **FilterTargets** — 脚本可以在伤害生效前修改目标列表（排除载体）
3. **传染循环** — 命中目标后可能重新施放 217694，形成"炸弹传播"效果

```
44461 的架构角色:

触发: 217694 Aura 过期 → CastSpell(44461)
    │
    ▼
Spell(44461) 创建
    │
    ├── Prepare() → Cast()
    │
    ├── SelectTargets() → AoE 选择
    │   ├── 中心: 爆炸载体位置
    │   ├── 半径: 10 码
    │   ├── 筛选: 敌方
    │   └── Script: FilterTargets → 排除载体
    │
    ├── EFFECT_0 (Dummy) → 空操作
    │
    ├── EFFECT_1 (SchoolDamage) → AoE 火焰伤害
    │   └── Script: OnEffectHit(EFFECT_1)
    │       └── if BASE_POINT2 > 0:
    │           └── for each hitTarget:
    │               └── CastSpell(217694, BASE_POINT2=0)
    │
    └── Finish(CastOK)
```

关键框架需求：
1. **AoE 目标选择** — Spell.SelectTargets 支持 AreaEnemy 模式
2. **FilterTargets hook** — 脚本在目标选择后、效果执行前修改目标列表
3. **多目标效果处理** — EffectSchoolDamage 需要对每个选中的目标执行
4. **传染递归** — OnEffectHit 中施放新 217694

### 数据配置

```go
var ExplosionInfo = spell.SpellInfo{
    ID:        44461,
    Name:      "Living Bomb Explode",
    CastTime:  0,    // 触发法术
    RangeMax:  50000, // 极大，确保不中断
    PowerCost: 0,    // TRIGGERED 跳过
    Effects: []spell.SpellEffectInfo{
        {
            EffectIndex: 0,
            EffectType:  spell.EffectDummy, // 脚本 hook 点
            TargetA:     spell.TargetUnitAreaEnemy,
        },
        {
            EffectIndex: 1,
            EffectType:  spell.EffectSchoolDamage,
            BasePoints:  0,    // 从 217694 传递
            BonusCoeff:  0.14,
            Radius:      10,   // 10 码
            TargetA:     spell.TargetUnitAreaEnemy,
        },
    },
}
```

### 施法流程

44461 是触发法术，由 217694 的 Aura 过期回调施放：

1. **触发**: 217694 Aura 过期 (RemoveMode == EXPIRE)
   - AuraScript: AfterEffectRemove → CastSpell(44461, TRIGGERED_FULL_MASK)
   - SpellValues 传递: BASE_POINT0 = tickDamage, BASE_POINT2 = 传染标记
2. **前置检查**: TriggeredFullMask 跳过大部分检查
3. **Cast**: 即时完成
4. **目标选择**: AoE 选择
   - 中心点: 爆炸载体位置
   - 半径: 10 码
   - 筛选: 敌对目标
   - **FilterTargets hook**: 移除爆炸载体本身
5. **效果分发**:
   - EFFECT_0 (Dummy) → 空操作
   - EFFECT_1 (SchoolDamage) → 对每个 AoE 目标造成伤害
     - **OnEffectHit hook**: 对每个命中目标，如果 BASE_POINT2 > 0，施放 217694 (BASE_POINT2=0)
6. **Finish**: 完成施法

### 效果执行逻辑

**效果 1: School Damage (AoE)**

执行时机：HandleImmediate() 中处理 EFFECT_1

对每个 AoE 目标：
1. 计算伤害: `basePoints + 0.14 × spellPower`
2. 暴击判定 (独立)
3. 应用伤害
4. 发布 OnSpellHit 事件
5. **传染检查**: 如果 SpellValues[2] > 0 → CastSpell(target, 217694, SpellValues[2]=0)

**传染链终止条件**: 传染的 217694 的 BASE_POINT2=0，因此它爆炸后的 44461 的 BASE_POINT2 也是 0 → 不再传染

```
传染链示例:

玩家对 A 施放 44457
    └── A 获得 217694 (BASE_POINT2=1)
        └── 4s 后过期 → 爆炸 44461 (BASE_POINT2=1)
            ├── 命中 B → B 获得 217694 (BASE_POINT2=0)
            │   └── 4s 后过期 → 爆炸 44461 (BASE_POINT2=0)
            │       └── 命中 C/D → 不传染 (BASE_POINT2=0)
            ├── 命中 C → C 获得 217694 (BASE_POINT2=0)
            │   └── 4s 后过期 → 爆炸 → 不传染
            └── 命中 D → D 获得 217694 (BASE_POINT2=0)
                └── 4s 后过期 → 爆炸 → 不传染
```

### 特殊逻辑

#### AoE 目标选择

**机制目的**: 在爆炸载体周围 10 码内选择所有敌对目标

**为什么需要**:
1. 现有框架的 SelectTargets 只支持单体 (UnitTargetID)
2. 44461 需要以载体为中心、10 码半径选择多个目标
3. targeting 包已有 SelectAroundPoint 方法，但未接入 Spell

**设计方案**:
- Spell.SelectTargets 扩展支持 TargetUnitAreaEnemy
- 使用 targeting.TargetSelector.SelectAroundPoint(center, radius, exclude)
- 中心点从 Aura 的目标位置获取
- 传染脚本 hook 中排除爆炸载体 (FilterTargets)

#### FilterTargets Hook

**机制目的**: 脚本在效果执行前修改目标列表

**为什么需要**:
1. TC 中活动炸弹爆炸排除载体本身（载体不受到自己爆炸伤害）
2. 不同的法术可能需要不同的目标过滤逻辑
3. 过滤时机需要在 SelectTargets 之后、ProcessEffects 之前

**设计方案**:
- SpellScript 添加 FilterTargets hook
- 在 SelectTargets 后调用
- 脚本可以修改 Spell 的目标列表（删除/添加目标）

### 跨表数据说明

| 字段 | 来源 | 值 | 说明 |
|------|------|-----|------|
| CastTime | SpellCastTimesEntry | 0 (即时) | 触发法术 |
| Range | SpellRangeEntry | 50000 码 | 极大值，确保距离不中断 |
| Radius | SpellRadiusEntry | 10 码 | AoE 半径 |
| Speed | SpellMisc.Speed | 0 | 无弹道，即时命中 |
| BonusCoeff | TC SQL: spell_bonus_data | 0.4286 (direct_bonus) | SP 系数 |
| School | Spell.dbc | Fire (4) | 火焰伤害 |

**关于 BonusCoeff 的说明**: TC SQL 中 44461 的 `direct_bonus = 0.4286`，但 Wowhead 显示 `SP:0.14`。差异原因可能是版本不同（WotLK vs Legion）。实现时以 Wowhead 显示的 0.14 为准，因为我们的参考版本更接近 Legion+。
