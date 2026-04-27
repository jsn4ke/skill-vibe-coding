# Target Selection

> Source: TrinityCore | Generated: 2026-04-27 | Topic: target-selection, implicit-target, area-target, chain-target, cone-target

## 1. Core Data Structures

### SpellTargetSelectionCategories (SpellInfo.h:41-51)
每个 ImplicitTarget 值映射到一个 SelectionCategory，决定选择算法：

| Category | 算法 |
|----------|------|
| NYI | 未实现 |
| DEFAULT | 直接引用（caster/target/dest），不做空间搜索 |
| CHANNEL | 从当前 channel spell 取目标 |
| NEARBY | 空间搜索：从 reference 点找最近的一个目标 |
| CONE | 空间搜索：扇形范围内所有目标 |
| AREA | 空间搜索：球形范围内所有目标 |
| TRAJ | 抛物线轨迹碰撞检测 |
| LINE | 线形范围（caster→dest 方向的矩形条带） |

### SpellTargetReferenceTypes (SpellInfo.h:53-61)
决定搜索的参考原点：

| Reference | 含义 |
|-----------|------|
| NONE | 无参考 |
| CASTER | 施法者位置 |
| TARGET | 当前选中的目标 |
| LAST | 上一个加入 TargetInfo 的目标 |
| SRC | SpellTargets 的 SrcPos |
| DEST | SpellTargets 的 DestPos |

### SpellTargetObjectTypes (SpellInfo.h:63-77)
返回的对象类型：

| ObjectType | 含义 |
|------------|------|
| NONE | 无 |
| SRC/DEST | 位置引用 |
| UNIT | 单位 |
| UNIT_AND_DEST | 单位 + 位置 |
| GOBJ | 游戏对象 |
| GOBJ_ITEM | 游戏对象+物品 |
| ITEM | 物品 |
| CORPSE | 尸体 |

### SpellTargetCheckTypes (SpellInfo.h:79-90)
友敌过滤条件：

| Check | 含义 |
|-------|------|
| DEFAULT | 无特殊过滤 |
| ENTRY | 按 condition entry 过滤 |
| ENEMY | 敌方 |
| ALLY | 友方 |
| PARTY | 队友 |
| RAID | 团队成员 |
| RAID_CLASS | 同职业团队成员 |
| PASSENGER | 乘客 |
| SUMMONED | 召唤物 |

### SpellTargetDirectionTypes (SpellInfo.h:92-105)
扇形/位置方向：

| Direction | 角度 |
|-----------|------|
| NONE | 无方向 |
| FRONT | 0° |
| BACK | 180° |
| RIGHT | -90° |
| LEFT | 90° |
| FRONT_RIGHT | -45° |
| BACK_RIGHT | -135° |
| BACK_LEFT | 135° |
| FRONT_LEFT | 45° |
| RANDOM | 随机 [0, 2π) |
| ENTRY | 由 condition entry 决定 |

### SpellImplicitTargetInfo (SpellInfo.h:174-207)
每个 ImplicitTarget 值（1-152）通过 `_data` 静态数组映射到 StaticData：
```
StaticData { ObjectType, ReferenceType, SelectionCategory, CheckType, DirectionType }
```
这是 TC 目标选择的核心数据驱动设计——ImplicitTarget 枚举值本身只是索引，真正的行为由 StaticData 决定。

### SpellEffectInfo 中的目标字段
```
TargetA: SpellImplicitTargetInfo   // 主隐式目标
TargetB: SpellImplicitTargetInfo   // 副隐式目标
TargetARadiusEntry / TargetBRadiusEntry  // 半径数据（来自 DBC）
ChainTargets: int32                // 链式跳跃数量
ImplicitTargetConditions: Condition[]  // 条件过滤
```

## 2. Flow Diagram

### SelectSpellTargets 总流程

```
SelectSpellTargets()
│
├─ 1. SelectExplicitTargets()          // 处理客户端传入的显式目标
│
├─ 2. for each effect:
│   ├─ SelectEffectImplicitTargets(TargetA)   // 主目标
│   └─ SelectEffectImplicitTargets(TargetB)   // 副目标
│
├─ 3. SelectEffectTypeImplicitTargets()  // 按 EffectType 补充默认目标
│
└─ 4. 校验：SPELL_ATTR1_REQUIRE_ALL_TARGETS / channeled / immune
```

### SelectEffectImplicitTargets 分发

```
SelectEffectImplicitTargets(effectInfo, targetType, targetIndex, processedMask)
│
├─ [共享目标优化] NEARBY/CONE/AREA/LINE:
│   同一 effect 的 TargetA==TargetB 且半径相同 → 共享一次搜索结果
│   processedEffectMask 防止重复搜索
│
└─ switch (SelectionCategory):
    ├─ DEFAULT ──┬─ ObjectType=SRC/DEST → 设置 SrcPos/DestPos
    │            ├─ ObjectType=DEST → SelectImplicitCasterDestTargets / TargetDest / DestDest
    │            └─ ObjectType=UNIT → SelectImplicitCasterObjectTargets / TargetObject
    │
    ├─ CHANNEL  → SelectImplicitChannelTargets
    ├─ NEARBY   → SelectImplicitNearbyTargets → SearchNearbyTarget → SelectImplicitChainTargets
    ├─ CONE     → SelectImplicitConeTargets → WorldObjectSpellConeTargetCheck
    ├─ AREA     → SelectImplicitAreaTargets → SearchAreaTargets
    ├─ TRAJ     → SelectImplicitTrajTargets → 抛物线碰撞
    └─ LINE     → SelectImplicitLineTargets → WorldObjectSpellLineTargetCheck
```

### SearchAreaTargets 流程

```
SearchAreaTargets(targets, effectInfo, range, position, referer, objectType, checkType, condList)
│
├─ GetSearcherTypeMask → 确定搜索的容器类型（Unit/GO/Corpse）
├─ WorldObjectSpellAreaTargetCheck(range, position, caster, referer, spellInfo, checkType, condList, objectType)
│   ├─ 距离检查: IsWithinDist(position, range)
│   ├─ 友敌检查: IsTargetExplicitForCaster / IsFriendlyTo
│   ├─ 条件检查: ConditionMgr::IsObjectMeetingCondition
│   └─ 免疫检查: IsImmuneToSpell / IsImmunedToDamage
│
└─ SearchTargets(searcher, ...) → 网格搜索
```

### SearchChainTargets 流程

```
SearchChainTargets(targets, chainTargets, target, objectType, selectType, isChainHeal)
│
├─ 计算 jumpRadius:
│   ├─ RANGED → 7.5y (Multi-Shot)
│   ├─ MELEE → 5.0y (Cleave/Swipe)
│   └─ MAGIC → isChainHeal ? 12.5y : 10.0y
│
├─ 计算 searchRadius:
│   ├─ CHAIN_FROM_CASTER → spell max range
│   ├─ ChainFromInitialTarget → jumpRadius
│   └─ default → jumpRadius * chainTargets
│
├─ SearchAreaTargets(tempTargets, searchRadius, chainSource, ...)
│
└─ while chainTargets > 0:
    ├─ [ChainHeal] 找 jumpRadius 内最大 HP deficit 的目标
    ├─ [其他] 找 jumpRadius 内最近的目标（需 LOS）
    ├─ 找到 → targets.push, chainSource = found, chainTargets--
    └─ 未找到 → 链结束
```

### SelectImplicitAreaTargets 的 Reference 解析

```
SelectImplicitAreaTargets:
│
├─ 解析 referer:
│   ├─ CASTER/SRC/DEST → m_caster
│   ├─ TARGET → m_targets.GetUnitTarget()
│   └─ LAST → m_UniqueTargetInfo 中最后一个匹配的目标
│
├─ 解析 center:
│   ├─ SRC → m_targets.GetSrcPos()
│   ├─ DEST → m_targets.GetDstPos()
│   └─ CASTER/TARGET/LAST → referer 自身位置
│
├─ 特殊目标:
│   ├─ TARGET_UNIT_CASTER_AND_PASSENGERS → 施法者 + 所有乘客
│   ├─ TARGET_UNIT_TARGET_ALLY_OR_RAID → 单目标或团队 AoE
│   ├─ TARGET_UNIT_CASTER_AND_SUMMONS → 施法者 + 所有召唤物
│   ├─ TARGET_UNIT_AREA_THREAT_LIST → 威胁列表中的单位
│   └─ TARGET_UNIT_AREA_TAP_LIST → Tap 列表中的玩家
│
└─ 默认: SearchAreaTargets(targets, radius, center, referer, ...)
    ├─ MaxAffectedTargets → RandomResize 随机截断
    └─ FURTHEST_ENEMY → 按距离降序排列后截断
```

## 3. Key Design Decisions

1. **ImplicitTarget → StaticData 查表而非 switch/case** → 每个 Targets 枚举值（1-152）通过 `_data[index]` 一次查表获得 ObjectType/ReferenceType/SelectionCategory/CheckType/DirectionType。新增目标类型只需在数组中加一行，无需修改分发逻辑。

2. **SelectionCategory 驱动算法选择，而非 ImplicitTarget 值本身** → 同一个算法（如 AREA）被多种 ImplicitTarget 复用（TARGET_UNIT_SRC_AREA_ENEMY、TARGET_UNIT_DEST_AREA_ALLY 等），区别仅在于 Reference 和 Check 不同。这避免了为每种目标写独立的搜索函数。

3. **ReferenceType 决定搜索原点，与 SelectionCategory 正交** → AREA 目标可以以 CASTER、TARGET、LAST、SRC、DEST 为中心。NEARBY 只用 CASTER。这种正交设计让目标组合爆炸（152 种）被 5 个 Reference × 7 个 Category × 9 个 Check 约束。

4. **TargetA + TargetB 两阶段解析** → TargetA 通常定义"位置/参考目标"（如 TARGET_DEST_TARGET_ENEMY），TargetB 定义"在该位置上搜索"（如 TARGET_UNIT_AREA_ENEMY）。两阶段解析让 Blizzard 这种 TargetA=Dest + TargetB=Area 的组合自然工作。

5. **共享目标优化（processedEffectMask）** → 多个 effect 如果 TargetA/TargetB 和半径完全相同，只搜索一次，结果共享给所有匹配的 effect。避免同一帧对同一区域重复搜索。

6. **Chain 的跳跃模型而非简单截断** → Chain 不是"在范围内找 N 个最近目标"，而是"从初始目标开始，每次跳跃 jumpRadius 找最近/最优目标，链式传播"。ChainHeal 特殊处理：选最大 HP deficit 的目标而非最近的。

7. **MaxAffectedTargets + RandomResize** → Area 目标找到所有候选后，如果超过 MaxAffectedTargets，随机裁剪而非按距离排序裁剪。这模拟了 WoW 中 AoE 不总是打最近目标的机制。

8. **Script Hook 介入** → 每个选择阶段都有 `CallScriptObjectAreaTargetSelectHandlers` / `CallScriptObjectTargetSelectHandlers` / `CallScriptDestinationTargetSelectHandlers`，允许脚本修改搜索结果、替换目标、或修改目标位置。

## 4. Reusable Patterns

### Pattern 1: 数据驱动的目标分类表
**适用**: 任何需要大量目标类型但行为可归类的系统。
**做法**: 枚举值 → 查表获得 (ObjectType, Reference, Category, Check, Direction)。Category 驱动算法，Reference 驱动原点，Check 驱动过滤。新增类型零代码改动。

### Pattern 2: 两阶段目标解析 (TargetA + TargetB)
**适用**: 目标选择需要先确定"在哪里"再确定"选谁"的场景。
**做法**: TargetA 解析参考位置/参考目标，TargetB 在该位置上执行空间搜索。例如 Blizzard: TargetA=DestTargetEnemy（确定地面位置），TargetB=AreaEnemy（在该位置搜索敌人）。

### Pattern 3: Reference 正交分解
**适用**: 同一搜索算法需要支持多种原点的场景。
**做法**: ReferenceType 独立于 SelectionCategory。AREA 可以以 CASTER/TARGET/LAST/SRC/DEST 为中心。新增原点类型不影响搜索算法。

### Pattern 4: 链式跳跃搜索
**适用**: Chain Lightning / Chain Heal 等链式技能。
**做法**: 从初始目标开始，每次在 jumpRadius 内搜索下一个目标（ChainHeal 选最大 deficit，其他选最近），找到后更新 chainSource 继续跳跃，直到 chainTargets 用尽或无合法目标。

### Pattern 5: 共享搜索结果优化
**适用**: 多个 effect 使用相同目标参数时避免重复搜索。
**做法**: 用 processedEffectMask 记录已搜索的 effect。后续 effect 如果 TargetA/TargetB 和半径完全匹配，直接复用结果。

### Pattern 6: Script Hook 介入点
**适用**: 需要允许脚本修改目标选择结果的系统。
**做法**: 在搜索完成后、AddUnitTarget 之前，调用脚本 hook。脚本可以增删目标、替换目标、修改位置。
