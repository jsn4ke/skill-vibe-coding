## Context

技能系统已有单体施法（火球术）和投射物延迟，支持 `StateChanneling` 状态和 `IsChanneled` 标记，但引导期间没有任何周期效果处理——`Update()` 对 `StateChanneling` 只做计时器倒计时然后 Finish。

暴风雪是 WoW 法师经典 AoE：在目标位置生成暴风雪区域，8秒内每1秒对范围内敌人造成冰霜伤害。这是一个 Persistent Area Aura + Periodic Damage 的组合。

当前 aura 系统（`aura.Manager.TickPeriodic`）已支持周期伤害和过期检测，可直接复用。targeting 系统已有 `SelectArea` 支持 `TargetUnitAreaEnemy`。

## Goals / Non-Goals

**Goals:**
- 实现暴风雪技能（Rank 1, Spell ID 10）
- 验证引导施法的完整生命周期：Preparing → Channeling → Finished
- 引导期间每1秒对 AoE 范围内目标造成周期伤害
- 复用已有 targeting、aura、event 基础设施

**Non-Goals:**
- 不实现引导打断/推回（pushback）机制
- 不实现多 Rank 支持（仅 Rank 1）
- 不修改 `spell.Spell.Update()` 的核心逻辑——通过 aura 系统处理周期伤害

## Decisions

### Decision 1: 引导周期伤害通过 Aura 实现

WoW 的暴风雪本质上是施法者在目标位置创建一个 Persistent Area Aura。引导期间，aura 每1秒 tick 一次造成伤害。

**选择**：复用 `aura.Manager.TickPeriodic` 处理伤害 tick，不新增 spell 层的 tick 机制。

**理由**：
- 已有 aura 周期系统完善（Period、PeriodTimer、TicksDone）
- 已有 event 集成（OnAuraTick、OnAuraExpired）
- 避免在 spell 层引入重复的 tick 逻辑

**替代方案**：在 `Spell.Update()` 的 `StateChanneling` 分支中新增 tick 回调——更接近 TC 原始实现，但需要修改 spell 核心代码，对现有火球术无益。

### Decision 2: AoE 目标选择 — 每 tick 重选

暴风雪使用 `TargetUnitAreaEnemy`，在目标位置（`TargetDestTargetEnemy`）8码半径内选择敌方单位。TC 的 DynObjAura 每 tick 都重新 FillTargetMap。

**选择**：保留每 tick 重新选目标的流程（与 TC 一致），用简单的列表遍历替代 Cell 空间分区。

**理由**：
- 流程与 TC 一致：单位走入/走出区域会在下一 tick 更新
- 简化的是空间查询实现（列表遍历 vs Cell::VisitAllObjects），不是流程本身

### Decision 3: 引导流程

```
Prepare() → Cast() → StateChanneling, Timer=8000
                        │
                        ├── Update() 倒计时
                        │
                        └── Cast() 中创建 Aura (8s, 1s tick)
                            → auraMgr.TickPeriodic() 每 step 处理 tick
```

**选择**：在 `CastFireball` 风格的 `CastBlizzard` 函数中，spell 进入 Channeling 后立即创建 Persistent Area Aura，aura 的生命周期与引导同步。

### Decision 4: 伤害公式

参考 WoW 数据：
- Rank 1 (ID 10): BasePoints = 25/tick, SP coefficient = 0.042/tick
- 8 ticks × 1s = 8s channel

公式: `damage = 25 + 0.042 × SpellPower`

## Risks / Trade-offs

- **[Aura 过期与引导结束不同步]** Aura 依赖 simulation loop 的 `TickPeriodic` 调用，如果 step 精度不够可能偏移。→ 使用与火球 DoT 相同的 step (100ms)，1s tick 精度足够。
- **[AoE 目标选择时机]** 暴风雪每次 tick 重新选择目标（与 TC FillTargetMap 一致），用列表遍历替代 Cell 空间分区。精度为每 1s tick 一次（TC 是每帧，差异可接受）。
