# Aura PeriodicTick Handler

> Source: TrinityCore | Generated: 2026-05-09 | Topic: aura periodic tick, PeriodicTick, aura type dispatch

## Core Data Structures

- `AuraEffect` — 每个 aura 的效果实例，内含 `_periodicTimer`、`_period`、`_ticksDone`、`m_isPeriodic` 字段驱动周期 tick
- `AuraApplication` — Aura 在 target 上的应用实例，PeriodicTick 按每个 aurApp 逐一执行
- `SPELL_AURA_*` 枚举 — 在 `SpellAuraDefines.h` 中定义，约 30 种周期性类型（PERIODIC_DAMAGE、PERIODIC_HEAL、PERIODIC_TRIGGER_SPELL 等）
- `auraEffectHandlers[]` 函数指针表 — 按AuraType索引，Apply/Remove时调用的handler；**周期性AuraType全部指向 `HandleNoImmediateEffect`**，注释标明"implemented in AuraEffect::PeriodicTick"

### 关键字段

| 字段 | 作用 |
|------|------|
| `m_isPeriodic` | `CalculatePeriodic()` 中根据 AuraType switch 设置，标记是否周期性 |
| `_period` | tick 间隔(ms)，可被脚本 `CalcPeriodic` hook 修改 |
| `_periodicTimer` | 累积时间，>= _period 时触发 tick |
| `_ticksDone` | 已完成 tick 计数，超过 `totalTicks` 时停止 |

## Flow Diagram

```
┌─────────────────────────────┐
│ Unit::Update(diff)          │
│   → Aura::Update(diff)      │
│     → AuraEffect::Update()  │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│ AuraEffect::Update(diff)                │
│   if (!m_isPeriodic) return             │
│   _periodicTimer += diff                │
│   while (_periodicTimer >= _period)     │
│     _periodicTimer -= _period           │
│     ++_ticksDone                        │
│     CallScriptEffectUpdatePeriodic()    │
│     for each aurApp:                    │
│       PeriodicTick(aurApp, caster)      │
└─────────────┬───────────────────────────┘
              │
              ▼
┌───────────────────────────────────────────────┐
│ PeriodicTick(aurApp, caster)                   │
│   CallScriptEffectPeriodicHandlers() ← 脚本拦截│
│   switch (GetAuraType())                       │
│     PERIODIC_DAMAGE → HandlePeriodicDamage..() │
│     PERIODIC_HEAL   → HandlePeriodicHeal..()   │
│     PERIODIC_TRIGGER_SPELL → HandlePeriodicTr..│
│     PERIODIC_LEECH  → HandlePeriodicHealthLe.. │
│     PERIODIC_ENERGIZE → HandlePeriodicEnerg..  │
│     PERIODIC_DUMMY  → (scripts only)           │
│     ... 十几个 case                             │
└───────────────────────────────────────────────┘
```

## Key Design Decisions

1. **Apply handler 统一为 HandleNoImmediateEffect，tick 行为集中在 PeriodicTick** — Apply 时不做任何事（周期性效果在 apply 时无即时效果），所有行为在 PeriodicTick 的 switch 中分发。分工清晰：Apply handler 管 apply/remove 生命周期，PeriodicTick 管 tick 行为。

2. **switch-case 硬编码分发，而非虚函数/策略模式** — AuraType 是有限枚举(~30种)，且每种 tick 行为差异很大（damage 要算 resist/absorb，heal 要算 crit，trigger spell 要创建子施法），没有统一的"tick策略接口"。switch-case 是最直接的做法。

3. **脚本拦截在 switch 之前执行** — `CallScriptEffectPeriodicHandlers()` 返回 `prevented` bool，可以完全阻止默认 tick 行为。这允许脚本覆盖任意 AuraType 的 tick 逻辑。

4. **每种 AuraType 独立 handler 函数** — `HandlePeriodicDamageAurasTick`、`HandlePeriodicHealAurasTick` 等，每个函数独立处理计算（bonus、crit、absorb、resist）和结算。不是内联在 switch 里。

## Reusable Patterns

### 模式：枚举驱动的周期效果分发

**适用场景：** 需要支持多种周期性效果类型，且每种类型的 tick 行为差异显著。

**结构：**
1. 枚举定义所有 AuraType
2. Apply 时根据 AuraType 设置 `isPeriodic` 标记
3. Update 循环中累积 timer，到时触发 PeriodicTick
4. PeriodicTick 内 switch(AuraType) 分发到各自 handler
5. 脚本 hook 在 switch 之前，可拦截/覆盖默认行为

**不适用：** 如果 tick 行为高度统一（都是"造成 X 点伤害"只是数值不同），直接用数据驱动即可，不需要 switch。

### TC 中周期性 AuraType 完整列表

| AuraType | Handler 函数 | 行为 |
|----------|-------------|------|
| PERIODIC_DAMAGE | HandlePeriodicDamageAurasTick | 伤害 → resist/absorb |
| PERIODIC_WEAPON_PERCENT_DAMAGE | HandlePeriodicDamageAurasTick | 武器伤害% → 同上handler |
| PERIODIC_DAMAGE_PERCENT | HandlePeriodicDamageAurasTick | 最大生命%伤害 → 同上handler |
| PERIODIC_LEECH | HandlePeriodicHealthLeechAuraTick | 伤害 + 回血给施法者 |
| PERIODIC_HEALTH_FUNNEL | HandlePeriodicHealthFunnelAuraTick | 施法者流血 + 目标回血 |
| PERIODIC_HEAL | HandlePeriodicHealAurasTick | 治疗 → 可暴击 |
| OBS_MOD_HEALTH | HandlePeriodicHealAurasTick | 百分比治疗 → 同上handler |
| PERIODIC_MANA_LEECH | HandlePeriodicManaLeechAuraTick | 抽蓝 + 回蓝给施法者 |
| OBS_MOD_POWER | HandleObsModPowerAuraTick | 百分比能量回复 |
| PERIODIC_ENERGIZE | HandlePeriodicEnergizeAuraTick | 固定量能量回复 |
| PERIODIC_TRIGGER_SPELL | HandlePeriodicTriggerSpellAuraTick | 触发子法术 |
| PERIODIC_TRIGGER_SPELL_WITH_VALUE | HandlePeriodicTriggerSpellWithValueAuraTick | 带数值触发子法术 |
| PERIODIC_DUMMY | (无默认handler) | 完全由脚本处理 |
