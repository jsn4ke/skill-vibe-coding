# Spell Interrupt & Cancel Mechanism

> Source: TrinityCore | Generated: 2026-04-25 | Topic: spell interrupt, cancel, movement break, range check, channel interrupt

## Section 1: Core Data Structures

### SpellInterruptFlags (施法中断标志)
控制 `Spell::update()` 中的中断行为，挂在 SpellInfo 上作为静态数据：

| Flag | 含义 |
|------|------|
| `Movement` (0x01) | 施法者移动时中断 PREPARING 状态的法术 |
| `DamagePushback` (0x200) | 受到伤害时施法时间推回 |
| `DamageCancels` (0x400) | 受到伤害时直接取消施法 |
| `Stun` (0x04) | 眩晕时中断（实际无此 flag 的法术也会被中断） |
| `Combat` (0x08) | 进入战斗时中断（也用于重置自动攻击计时器） |

### SpellAuraInterruptFlags (Aura 中断标志)
控制 aura 和 channel 法术的中断，由 `Unit::RemoveAurasWithInterruptFlags()` 驱动：

| Flag | 含义 |
|------|------|
| `Moving` (0x08) | 移动时移除 aura |
| `Turning` (0x10) | 转向时移除 aura |
| `Damage` (0x02) | 受到伤害时移除 aura |
| `Action` (0x04) | 执行动作时移除 aura |
| `Attacking` (0x1000) | 攻击时移除 aura |
| `Mount` (0x2000) | 上坐骑时移除 aura |
| `Shapeshifting` (0x8000) | 变形时移除 aura |

### CurrentSpellTypes (法术槽位)
Unit 持有多个"当前法术"槽位：

| Slot | 用途 |
|------|------|
| `CURRENT_GENERIC_SPELL` | 普通施法（有吟唱时间的） |
| `CURRENT_CHANNELED_SPELL` | 引导法术 |
| `CURRENT_MELEE_SPELL` | 近战自动攻击 |
| `CURRENT_AUTOREPEAT_SPELL` | 自动重复射击 |

## Section 2: Flow Diagram

### 中断检查的三个层次

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 1: Spell::update() — 每个 tick 检查                      │
│  (由 Unit::_UpdateSpells → spell->update(diff) 调用)             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  update(diff):                                                  │
│    ├── UpdatePointers() 失败 → cancel()                        │
│    ├── 目标 GUID 不为空但目标对象不存在 → cancel()              │
│    ├── 施法者正在移动 && CheckMovement() != OK → cancel()      │
│    │       │                                                    │
│    │       ├── PREPARING + InterruptFlags.Movement → FAILED    │
│    │       └── CHANNELING + !IsMoveAllowedChannel → FAILED     │
│    │                                                            │
│    └── switch(state):                                           │
│        ├── PREPARING: timer -= diff, timer=0 → cast()          │
│        └── CHANNELING:                                          │
│            ├── UpdateChanneledTargetList()                      │
│            │   ├── 目标离开范围 → 移除该目标的 aura             │
│            │   ├── 目标死亡/消失 → 移除该目标                   │
│            │   └── 所有目标都没了 → timer=0, 清理 aura, 结束   │
│            └── timer -= diff, timer=0 → finish(OK)             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  Layer 2: Unit::RemoveAurasWithInterruptFlags() — 事件驱动      │
│  (由移动、受伤、坐骑、变形等事件触发)                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  触发点:                                                        │
│    ├── Unit::InterruptMovementBasedAuras()                     │
│    │   ├── Turned → RemoveAurasWithInterruptFlags(Turning)     │
│    │   └── Relocated → RemoveAurasWithInterruptFlags(Moving)   │
│    │                                                            │
│    ├── 受到伤害时                                               │
│    │   └── RemoveAurasWithInterruptFlags(Damage)               │
│    │                                                            │
│    └── 各种 Unit 状态变化                                       │
│                                                                 │
│  处理流程:                                                      │
│    1. 遍历 m_interruptableAuras                                │
│       └── aura 有对应 InterruptFlag → RemoveAura(BY_INTERRUPT) │
│    2. 检查 CURRENT_CHANNELED_SPELL                             │
│       └── channel 法术有对应 ChannelInterruptFlag              │
│           → InterruptNonMeleeSpells() → spell->cancel()        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  Layer 3: Spell::cancel() — 清理执行                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  cancel():                                                      │
│    ├── 记录 oldState = 当前状态                                 │
│    ├── 设 m_spellState = FINISHED                              │
│    ├── switch(oldState):                                        │
│    │   ├── PREPARING:                                           │
│    │   │   ├── CancelGlobalCooldown()  // 退 GCD              │
│    │   │   └── SendInterrupted() + SendCastResult(FAILED)     │
│    │   ├── CHANNELING:                                          │
│    │   │   ├── 遍历所有目标 → RemoveOwnedAura(BY_CANCEL)      │
│    │   │   └── SendChannelUpdate() + SendInterrupted()        │
│    │   └── LAUNCHED:                                            │
│    │       └── SendInterrupted()                               │
│    ├── RemoveDynObject / RemoveGameObject                      │
│    ├── 恢复 m_spellState = oldState                            │
│    └── finish(FAILED_INTERRUPTED)                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Channel 法术目标范围检查流程

```
UpdateChanneledTargetList() — 每个 Update tick 调用:

  ┌──────────────────────────────────────────────┐
  │ 遍历 m_UniqueTargetInfo                      │
  │   ├── 目标对象不存在 → 跳过（从 channel 移除）│
  │   ├── 目标已死亡但不是有效死亡目标 → 跳过     │
  │   ├── 目标有 aura:                           │
  │   │   └── 施法者与目标距离 > range + tolerance│
  │   │       → 移除目标 aura                    │
  │   │       → 从 channel 目标列表移除           │
  │   └── 目标 aura 被 dispel → 从列表移除       │
  │                                              │
  │ 所有目标都移除了 → return false               │
  │   → timer = 0, 移除所有 aura, 结束 channel   │
  └──────────────────────────────────────────────┘
```

## Section 3: Key Design Decisions

**Decision: `cancel()` 恢复 oldState 再调 `finish()`**
→ Reason: `finish()` 需要根据原始状态做不同清理。PREPARING 中断退 GCD，CHANNELING 中断移除 Aura。先设 FINISHED 处理中断逻辑，再恢复原始状态给 `finish()`。

**Decision: Channel 法术的范围检查在 `update()` 里持续做**
→ Reason: 目标可以在 channel 期间移动出范围。不是一次性的 pre-cast 检查，而是每 tick 检查。`UpdateChanneledTargetList()` 负责。

**Decision: 两套 Interrupt Flag 系统（Spell vs Aura）**
→ Reason: Spell 的中断标志控制**正在施放的法术**（PREPARING/CHANNELING 状态），Aura 的中断标志控制**已施加的光环**。Channel 法术同时受两者影响——它既有 Spell 状态又有 Aura。

**Decision: `CheckRange` 在 `cast()` 里做 strict=false 宽容检查**
→ Reason: 非严格模式给 10% 容差（`MAX_SPELL_RANGE_TOLERANCE`），因为网络延迟可能导致施法者和目标位置略有差异。

**Decision: 移动中断通过 `_positionUpdateInfo.Relocated` 触发**
→ Reason: 不是每帧检查位置，而是由移动系统标记 `Relocated` 标志。`InterruptMovementBasedAuras()` 只在位置确实改变时调用，效率更高。

## Section 4: Reusable Patterns

### Pattern 1: 双层中断 — Spell 级 + Aura 级
施放中的法术和已施加的 aura 有独立的中断机制。Channel 法术横跨两者。
**适用**: 任何有持续效果的技能系统。

### Pattern 2: Update 驱动的持续验证
`Spell::update()` 不仅推进 timer，还持续验证条件（目标存在、范围、移动）。条件不满足就 cancel。
**适用**: 需要持续条件的技能（吟唱、引导、持续效果）。

### Pattern 3: Flag-based 中断映射
中断条件是 bitmask flag，不是硬编码的 if-else。新中断类型只需加一个 flag。
**适用**: 可扩展的状态效果系统。

### Pattern 4: cancel → oldState → finish 三段式清理
cancel 做清理，但让 finish 知道"从哪个状态被中断的"，从而做状态特定的收尾。
**适用**: 有复杂状态机的对象取消逻辑。

### Pattern 5: Channel 目标列表动态维护
Channel 不是一次性确定目标。每 tick 检查：目标还活着吗？还在范围内吗？Aura 还在吗？不满足就移除。全部移除就结束 channel。
**适用**: 持续性 AoE、beam、link 类技能。
