## Context

当前 skill-go 的目标选择系统存在以下问题：

1. `SelectTargets()` 只检查 `TargetA`，忽略 `TargetB` 的两阶段语义
2. AoE 目标通过外部注入的 `AoESelector` 接口解析，SpellInfo 无法自描述
3. `targeting` 包已实现 Descriptor/Category/Check 但未接入 spell pipeline
4. 无友敌过滤（`GetUnitsInRadius` 不分敌友）
5. Chain 是简单距离截断，非 TC 的跳跃模型
6. 无 ReferenceType（搜索原点只有 caster/dest）
7. 无 DirectionType（Cone 无方向）
8. 无 Script Hook 介入点

TC 参考文档：`tc-references/target-selection.md`

## Goals / Non-Goals

**Goals:**
- 按 TC 的 `SpellImplicitTargetInfo::StaticData` 模式实现数据驱动查表
- ImplicitTarget 枚举扩展至 ~50 个值，覆盖核心 Unit/Area/Cone/Line/Channel/Dest 类型
- 实现 TargetA + TargetB 两阶段解析
- 实现 ReferenceType（Caster/Target/Last/Src/Dest）决定搜索原点
- 实现 DirectionType（11 种方向）用于 Cone 和 Dest 目标
- 实现 CheckType 友敌过滤接入 spell pipeline
- 实现 Chain 跳跃模型（jumpRadius 逐跳传播）
- 移除 AoESelector 接口，SpellInfo 自描述目标行为
- 全量迁移现有 4 个技能
- 共享搜索优化
- Script Hook 介入点

**Non-Goals:**
- 不实现 TC 全部 152 个 ImplicitTarget（NYI 的 ~15 个跳过，Vehicle/Passenger/Corpse 等后续按需添加）
- 不实现 Traj（抛物线碰撞）— 无投射物系统
- 不实现 Entry 条件过滤 — 无 DBC Condition 系统
- 不实现 GameObject/Corpse/Item 目标类型 — 当前只有 Unit
- 不实现 MaxAffectedTargets + RandomResize — 后续按需添加
- 不实现 ThreatList/TapList/Clump 等特殊 Area 源 — 后续按需添加

## Decisions

### D1: targeting 包按 TC 结构重写，而非原地升级

**选择**: 完全重写 `pkg/targeting/`

**理由**: 当前包缺少 ReferenceType、DirectionType、ObjectType 三个维度，且 `Targetable` 接口与 `unit.Unit` 不兼容。重写比修补更清晰，且包目前未被任何代码调用（零迁移成本）。

**替代方案**: 在现有包上增量添加字段 — 会导致 Descriptor 结构臃肿，且 Targetable 接口需要大改才能适配 Unit。

### D2: ImplicitTarget 枚举值与 TC 对齐（使用 TC 的数字编号）

**选择**: ImplicitTarget 常量使用 TC 的原始编号（如 `TargetUnitCaster = 1`，`TargetUnitTargetEnemy = 6`），中间跳过未实现的编号。

**理由**: 与 TC 的 `_data` 查表对齐，新增目标类型时直接参照 TC 编号，无需维护映射关系。也便于未来对照 TC 调试。

**替代方案**: 连续 iota 编号 — 需要维护两套映射（Go 枚举 → TC 编号），增加混淆。

### D3: StaticData 查表用 Go 数组，非 map

**选择**: `var targetData [MaxTarget]StaticData` 固定大小数组

**理由**: TC 用 `std::array<StaticData, TOTAL_SPELL_TARGETS>`，Go 的 `[N]T` 数组等价。O(1) 索引访问，零分配，编译期大小确定。

### D4: SelectEffectTargets 替代 SelectTargets，按 effect 逐个解析

**选择**: 对每个 effect 的 TargetA 和 TargetB 分别调用 `selectEffectImplicitTargets()`，按 SelectionCategory 分发。

**理由**: 对齐 TC 的 `SelectSpellTargets()` 流程。TargetA 通常解析参考位置/参考目标，TargetB 在该位置上搜索。当前 `SelectTargets()` 的单体/AoE 二分支逻辑无法处理 TargetA=Dest + TargetB=Area 的组合。

### D5: 移除 AoESelector，目标选择由 Engine 内部驱动

**选择**: 删除 `AoESelector` 接口、`WithAoE()` CastOption、`Spell.AoESelector/AoECenter/AoEExcludeID` 字段。目标选择通过 `SpellEffectInfo.TargetA/TargetB + Radius + ChainTargets` 自描述。

**理由**: TC 中目标选择完全由 SpellInfo 的数据驱动，不需要外部注入。`AoESelector` 是当前简化设计的产物，与 TC 架构不兼容。Living Bomb 的爆炸目标选择可由 `TargetUnitAreaEnemy + Radius` 自动解析。

**替代方案**: 保留 AoESelector 作为 override 机制 — 增加复杂度，且 TC 用 Script Hook 解决 override 需求。

### D6: Radius 数据用 SpellEffectInfo.Radius 字段，非 MiscValue

**选择**: `SpellEffectInfo` 新增 `Radius float64` 字段

**理由**: TC 用 `TargetARadiusEntry` / `TargetBRadiusEntry`（DBC 查表）。我们没有 DBC，但可以用静态配置数据或直接在 SpellInfo 定义中设置 Radius。MiscValue 语义混乱（有时是半径，有时是其他），需要专用字段。

**替代方案**: RadiusIndex 查表 — 需要维护半径配置表，当前技能数量少，直接设值更简单。后续可迁移到查表。

### D7: ChainTargets 字段加在 SpellEffectInfo 上

**选择**: `SpellEffectInfo` 新增 `ChainTargets int32` 字段

**理由**: 对齐 TC 的 `SpellEffectInfo.ChainTargets`。Chain 跳跃数量是 effect 级别的属性，不是 spell 级别的。

### D8: 友敌过滤基于 Entity.Type 对比

**选择**: 延续当前 `passesCheck` 逻辑：Enemy = 不同 Type，Ally = 相同 Type

**理由**: 当前 `entity.EntityType` 有 Player/Creature/Pet 区分，足以判断敌友关系。Party/Raid 过滤需要额外分组信息，当前暂用 Ally 代替（同 Type 即视为同队）。

### D9: 共享搜索用 processedEffectMask 位掩码

**选择**: 用 `uint32` 位掩码记录已搜索的 effect index，相同 TargetA/TargetB + 半径的 effect 共享结果

**理由**: 对齐 TC 的 `processedAreaEffectsMask`。实现简单，O(1) 判断。

### D10: Script Hook 在搜索后、AddTarget 前介入

**选择**: 在 `SearchAreaTargets` / `SearchConeTargets` / `SearchChainTargets` 返回结果后，调用 `Registry.CallTargetSelectHook(spellID, effectIndex, targets)` 允许脚本修改

**理由**: 对齐 TC 的 `CallScriptObjectAreaTargetSelectHandlers`。当前 `script.Registry` 已有 `RegisterSpellHook` 机制，新增 `HookOnTargetSelect` 即可。

## Risks / Trade-offs

- **[Breaking API change]** → 所有使用 `WithAoE()` 的测试代码需要重写。Mitigation: 只有 Living Bomb 使用，改动范围可控。
- **[ImplicitTarget 编号跳跃]** → 使用 TC 编号意味着枚举值不连续（如 1, 6, 15, 16...），Go 的 iota 不可用。Mitigation: 显式赋值，加注释标注 TC 编号。
- **[Area aura 友敌过滤]** → `tickAreaAura` 当前无友敌过滤，接入后 Blizzard 的 tick 行为会改变。Mitigation: Blizzard 的 `TargetB: TargetUnitAreaEnemy` 已隐含 Enemy 过滤语义，修复是正确的。
- **[Chain 跳跃模型复杂度]** → jumpRadius / searchRadius / LOS 检查增加实现复杂度。Mitigation: 先实现基础跳跃（无 LOS），后续按需添加。
- **[TargetB 解析顺序依赖]** → TargetB 的搜索依赖 TargetA 解析出的 DestPos/SrcPos。Mitigation: 严格按 TargetA 先、TargetB 后的顺序解析，与 TC 一致。
