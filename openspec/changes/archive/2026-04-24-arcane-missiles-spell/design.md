## Context

项目是一个 WoW 风格的战斗模拟框架 (skill-go)，目前已实现 Fireball（弹道投射 + DoT）和 Blizzard（引导 AoE 区域伤害）。框架的核心架构是：SpellInfo 静态定义 → Spell 运行时实例（FSM 驱动）→ Effect 处理器分发 → Aura 周期 tick。

现在需要引入第三种施法模式：**引导型周期触发法术**。与暴风雪的区别在于 tick 不直接计算伤害，而是触发一个独立法术。这是 WoW 中 `SPELL_AURA_PERIODIC_TRIGGER_SPELL` 的经典实现模式。

参考文档：`skill-designs/arcane-missiles.md`（完整技能拆解）、`tc-references/spell-cast-flow.md`（TC 施法流程）。

## Goals / Non-Goals

**Goals:**
- 在 Aura 系统中新增 `AuraPeriodicTriggerSpell` 类型
- 实现完整 Spell 实例的周期触发：tick → 创建 Spell 实例 → Prepare → HandleImmediate → 伤害
- 实现完整的奥术飞弹技能包（CastArcaneMissiles、两个 SpellInfo、测试）
- 引导取消联动 Aura 移除
- 时间线测试验证 3 tick 的完整事件序列

**Non-Goals:**
- 不实现目标追踪 (`TRACK_TARGET_IN_CHANNEL`)
- 不实现 proc 触发联动（每发飞弹触发 Clearcasting 等）
- 不实现天赋/装备修正器链
- 不实现多等级奥术飞弹（仅 Rank 1）

## Decisions

### Decision 1: 触发法术执行方式——完整 Spell 实例

**选择**: 每次 tick 创建完整 Spell 实例走完整 CastSpell 流程（复刻 TC 原版做法）

**替代方案**: 直接调用效果处理器（简化方案）

**理由**:
- TC 原版 `HandlePeriodicTriggerSpellAuraTick` 调用 `caster->CastSpell(target, triggerSpellId, TRIGGERED_FULL_MASK)`，每次触发都是完整法术实例
- 完整 Spell 实例保证触发法术走统一的 Prepare → Cast → HandleImmediate 流程
- 触发法术有自己的 SelectTargets、ProcessEffects、OnSpellHit 发布，完全复用现有管线
- 为未来 proc 触发和嵌套施法预留正确的基础
- CastFlags 使用 `TriggeredFullMask` 跳过 GCD/冷却/法力消耗，与 TC 行为一致

**实现细节**: 每次 tick 的 onTick 回调中：
1. 用触发法术的 SpellInfo (7268) 创建新 `Spell` 实例
2. 设置 `CastFlags = TriggeredFullMask`
3. 设置 `Targets.UnitTargetID = 目标`
4. 设置 `Bus` 以发布事件
5. 调用 `Prepare()` → 因为 CastTime=0，立即进入 `Cast(true)` → HandleImmediate → Finish
6. 每发飞弹有独立的 TargetInfo、Crit 判定、伤害计算

### Decision 2: 触发法术 SpellInfo 定义

**选择**: 在技能包内定义 `MissileInfo` 作为 SpellInfo（ID 7268，EffectSchoolDamage，BasePoints=24，BonusCoeff=0.132）

**替代方案**: 全局法术注册表 (`SpellRegistry`) 按 TriggerSpellID 查找

**理由**: 触发法术的 SpellInfo 和父法术在同一个技能包内，闭包捕获最直接。如果后续有跨技能包共享触发法术的需求，再引入注册表。

### Decision 3: TickPeriodic 的触发路径实现

**选择**: 在 `TickPeriodic` 的 onTick 回调中，由调用方（CastArcaneMissiles）传入的自定义回调创建并执行完整 Spell 实例

**理由**: 现有 `TickPeriodic` 已经接受 `onTick` 回调参数。`AuraPeriodicTriggerSpell` 的 tick 行为在回调中完成 Spell 创建和执行，无需修改 `TickPeriodic` 内部逻辑。TickPeriodic 保持通用——它只负责"每 Period 触发一次回调"。

### Decision 4: 引导取消联动

**选择**: 与 Blizzard 一致——注册 `OnCancel` hook，在回调中调用 `auraMgr.RemoveAura()`

**理由**: 已验证的模式，代码一致性好。

## Risks / Trade-offs

**[嵌套 Spell 实例的复杂性]** → 每次 tick 创建完整 Spell 实例增加了运行时开销（每个 tick 一个 Spell 对象 + SelectTargets + ProcessEffects）。对于 3 tick 的奥术飞弹可以接受。如果未来有高频触发场景（如每 0.1s 触发），需要考虑对象池优化。

**[TickPeriodic onTick 回调职责]** → onTick 回调需要创建 Spell、设置参数、调用 Prepare，代码量增加。缓解：提取 `CastTriggeredSpell(caster, targetID, info, bus)` 辅助函数封装通用逻辑。

**[无触发法术注册表]** → 每个技能包自己管理触发法术 SpellInfo，不支持跨技能共享。如果后续有"通用触发法术"需求需要引入注册表。当前只有一个触发法术，风险低。
