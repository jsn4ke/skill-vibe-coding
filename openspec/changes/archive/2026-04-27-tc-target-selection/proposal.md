## Why

当前目标选择系统过于简化：`SelectTargets()` 只检查 `TargetA`，AoE 目标通过外部注入的 `AoESelector` 接口解析，无友敌过滤，chain 是简单截断而非跳跃模型，`targeting` 包已实现但未接入 spell pipeline。TrinityCore 的目标选择采用 5 维正交分解（ObjectType × ReferenceType × SelectionCategory × CheckType × DirectionType）的数据驱动查表设计，153 种 ImplicitTarget 值通过查表获得全部属性，由 SelectionCategory 驱动算法分发。需要对标 TC 重写目标选择系统，使 SpellInfo 自描述目标行为，消除外部注入依赖。

## What Changes

- **BREAKING**: 重写 `pkg/targeting/` 包，按 TC 的 `SpellImplicitTargetInfo::StaticData` 模式实现数据驱动查表，包含 5 维属性（ObjectType, ReferenceType, SelectionCategory, CheckType, DirectionType）
- **BREAKING**: 扩展 `ImplicitTarget` 枚举至 ~50 个值，覆盖 TC 核心 Unit/Area/Cone/Line/Channel/Dest 目标类型，其余标记 NYI
- **BREAKING**: 重写 `Spell.SelectTargets()` 为 `SelectEffectTargets()`，对每个 effect 的 TargetA 和 TargetB 分别解析，按 SelectionCategory 分发到不同算法（Default/Nearby/Cone/Area/Line/Channel）
- 新增 ReferenceType（Caster/Target/Last/Src/Dest）决定搜索原点
- 新增 DirectionType（Front/Back/Left/Right 等 11 种）用于 Cone 和 Dest 目标
- 新增 CheckType 友敌过滤接入 spell pipeline（Enemy/Ally/Party/Raid/Summoned/Entry）
- 新增 Chain 跳跃模型：从初始目标逐跳搜索，jumpRadius 约束，替代当前简单距离截断
- 新增 `SpellEffectInfo.ChainTargets` 字段（链式跳跃数量）
- 新增 `SpellEffectInfo.Radius` 字段（目标选择半径，替代 MiscValue 滥用）
- **BREAKING**: 移除 `AoESelector` 接口和 `WithAoE()` CastOption，SpellInfo 自描述目标行为
- 新增共享搜索优化：多 effect 同 TargetA/TargetB + 同半径时复用搜索结果
- 新增 Script Hook 介入点：目标选择结果可被脚本修改
- 全量迁移现有 4 个技能（Fireball/Blizzard/ArcaneMissiles/LivingBomb）到新目标系统

## Capabilities

### New Capabilities
- `implicit-target-lookup`: ImplicitTarget 数据驱动查表系统 — StaticData 5 维属性、ImplicitTargetInfo 类型、查表访问方法
- `target-resolution`: 目标解析引擎 — SelectEffectTargets 分发、Reference 解析、Check 过滤、共享搜索优化、Script Hook 介入
- `chain-jump-model`: 链式跳跃搜索 — jumpRadius 约束、逐跳传播、ChainHeal HP deficit 优先、LOS 检查

### Modified Capabilities
- `living-bomb-skill`: 移除 AoESelector 外部注入，改用 SpellInfo 自描述的 TargetUnitAreaEnemy + Radius 字段

## Impact

- `pkg/targeting/` — 完全重写，新增 ~500 行
- `pkg/spell/info.go` — ImplicitTarget 扩展、SpellEffectInfo 新增 ChainTargets/Radius 字段
- `pkg/spell/spell.go` — SelectTargets() 重写为 SelectEffectTargets()，移除 AoESelector 相关字段
- `pkg/engine/engine.go` — 移除 WithAoE() CastOption，CastSpell 目标解析改为内部驱动
- `pkg/effect/effect.go` — 适配新目标系统，area aura 创建逻辑调整
- `pkg/unit/unit.go` — tickAreaAura 友敌过滤接入
- `skills/fireball/` — 适配（主要无需改动，单目标）
- `skills/blizzard/` — 适配（TargetA+TargetB 两阶段解析）
- `skills/arcane-missiles/` — 适配（单目标，主要无需改动）
- `skills/living-bomb/` — 重写目标选择，移除 AoESelector 注入
- 所有 `*_engine_test.go` — 移除 WithAoE() 调用，适配新 API
