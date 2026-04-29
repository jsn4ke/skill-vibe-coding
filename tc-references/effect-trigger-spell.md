# EffectTriggerSpell

> Source: TrinityCore | Generated: 2026-04-29 | Topic: SPELL_EFFECT_TRIGGER_SPELL, SPELL_EFFECT_TRIGGER_SPELL_WITH_VALUE, trigger spell from effect

## Section 1: Core Data Structures

### 相关 Effect 类型（SharedDefines.h）

| 枚举值 | 名称 | 用途 |
|--------|------|------|
| 64 | `SPELL_EFFECT_TRIGGER_SPELL` | 标准触发法术，从 `TriggerSpell` 字段获取目标法术 ID |
| 142 | `SPELL_EFFECT_TRIGGER_SPELL_WITH_VALUE` | 触发法术并传递 base point 值给子法术 |
| 151 | `SPELL_EFFECT_TRIGGER_SPELL_2` | 第二种触发法术变体 |

### 关键字段（SpellEffectInfo）

- `TriggerSpell` — 目标法术 ID，通过 `sSpellMgr->GetSpellInfo()` 查找
- `MiscValue` — 对 SPELL_EFFECT_TRIGGER_SPELL 用作延迟时间（毫秒）
- `Effect` — 区分是哪种触发类型（64 vs 142 vs 151）

## Section 2: Flow Diagram

```
EffectTriggerSpell()
  │
  ├── phase != LAUNCH_TARGET && phase != LAUNCH → return
  │
  ├── triggered_spell_id = effectInfo->TriggerSpell
  │
  ├── [特殊 case 处理] — hardcoded spell ID 特殊逻辑（少数WoW特例）
  │
  ├── triggered_spell_id == 0 → warning, return
  │
  ├── spellInfo = sSpellMgr->GetSpellInfo(triggered_spell_id) → not found → error, return
  │
  ├── LAUNCH_TARGET 路径:
  │   ├── spellInfo->NeedsToBeTriggeredByCaster() == false → return（子法术自行定位目标）
  │   └── targets.SetUnitTarget(unitTarget)  + targetCount/targetIndex
  │
  ├── LAUNCH 路径:
  │   ├── NeedsToBeTriggeredByCaster() && has unit target mask → return（留给 LAUNCH_TARGET 处理）
  │   └── fallback: caster 自身或原目标
  │
  ├── delay = effectInfo->MiscValue（仅 SPELL_EFFECT_TRIGGER_SPELL）
  │
  └── m_Events.AddEventAtOffset(delay) →
      CastSpell(targets, TriggerSpell, args)
        - triggered: TRIGGERED_FULL_MASK ~(IGNORE_POWER_COST | IGNORE_REAGENT_COST)
        - WITH_VALUE 时: 传递 base points 给子法术
```

## Section 3: Key Design Decisions

1. **通过 sSpellMgr 数据驱动 lookup** — TriggerSpell 字段存 SpellID，运行时通过 `sSpellMgr->GetSpellInfo(triggered_spell_id)` 查找。不需要 spell script 介入。

2. **延迟触发用 Event 系统而非即时** — 使用 `m_Events.AddEventAtOffset(delay)` 实现延迟触发。MiscValue 字段作为延迟毫秒数。

3. **Triggered 但保留资源消耗** — 触发标记 `TRIGGERED_FULL_MASK & ~(IGNORE_POWER_COST | IGNORE_REAGENT_COST)` — 忽略 GCD/CD 但保留资源消耗。

4. **NeedsToBeTriggeredByCaster 决定目标传递** — 如果子法术自身能定位目标（如 AoE、self-cast），则 LAUNCH_TARGET 路径直接 return，不传递目标。只有需要显式目标的子法术才由触发者传递 target。

5. **SPELL_EFFECT_TRIGGER_SPELL_WITH_VALUE 传递 base points** — 将触发 effect 的 damage/value 作为子法术各 effect 的 base point 传入。

## Section 4: Reusable Patterns

### Pattern: Data-driven spell triggering from effects

- **适用**：当法术效果是"施放另一个法术"时，纯数据驱动（TriggerSpellID），无需脚本。
- **实现**：effect handler 从 SpellEffectInfo.TriggerSpell 取 ID → store lookup → CastSpell
- **延迟**：通过 MiscValue 字段指定延迟（ms），用定时器异步触发
- **目标传递**：根据子法术的目标类型决定是继承父法术目标还是自行定位

### Pattern: Trigger with value propagation

- **适用**：子法术需要接收父法术的 base point 作为参数
- **实现**：`TRIGGER_SPELL_WITH_VALUE` 将 value 映射到子法术的 `SPELLVALUE_BASE_POINT0+i`

### 对比：何时用 EffectTriggerSpell vs EffectDummy + Script

| 场景 | 推荐方式 |
|------|---------|
| 触发固定 ID 的子法术，无需额外参数 | EffectTriggerSpell（数据驱动） |
| 触发子法术并传递自定义值 | EffectTriggerSpellWithValue |
| 触发逻辑依赖复杂条件、状态、多步判断 | EffectDummy + Script Hook |
| 需要修改 SpellValues 等脚本间通信数据 | EffectDummy + Script Hook |
