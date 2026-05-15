# Interrupt Mechanism (完整版)

> Source: TrinityCore | Generated: 2026-05-15 | Topic: interrupt, stun, silence, pacify, fear, CC cascade, aura interrupt flags, channel interrupt, mechanic immunity

## Section 1: Core Data Structures

### SpellInterruptFlags — 施法中断标志（控制 PREPARING/CHANNELING 阶段法术的中断）

挂在 SpellInfo 上作为静态数据，由 `Spell::update()` 每帧检查。

| Flag | 值 | 含义 |
|------|-----|------|
| `Movement` | 0x01 | 施法者移动时中断 PREPARING 状态 |
| `DamagePushbackPlayerOnly` | 0x02 | 仅玩家：受伤时施法时间推回 |
| `Stun` | 0x04 | 眩晕中断（注释：useless，即使没有此 flag 也会被中断） |
| `Combat` | 0x08 | 进入战斗时中断 + 重置自动攻击计时器 |
| `DamageCancelsPlayerOnly` | 0x10 | 仅玩家：受伤直接取消 |
| `MeleeCombat` | 0x20 | NYI |
| `Immunity` | 0x40 | NYI |
| `DamageAbsorb` | 0x80 | 被吸收的伤害触发推回 |
| `ZeroDamageCancels` | 0x100 | 零伤害也触发取消 |
| `DamagePushback` | 0x200 | 受伤时施法时间推回（通用） |
| `DamageCancels` | 0x400 | 受伤直接取消（通用） |

### SpellAuraInterruptFlags — Aura 中断标志（控制已施加 aura 的移除）

挂在 SpellInfo 上，由 `Unit::RemoveAurasWithInterruptFlags()` 驱动。

| Flag | 值 | 含义 |
|------|-----|------|
| `HostileActionReceived` | 0x01 | 受到敌对行为 |
| `Damage` | 0x02 | 受到伤害 |
| `Action` | 0x04 | 执行动作 |
| `Moving` | 0x08 | 移动 |
| `Turning` | 0x10 | 转向 |
| `Anim` | 0x20 | 动画 |
| `Dismount` | 0x40 | 下坐骑 |
| `UnderWater` | 0x80 | 进入水中 |
| `AboveWater` | 0x100 | 离开水中 |
| `Sheathing` | 0x200 | 收武器 |
| `Interacting` | 0x400 | 交互（NPC 等） |
| `Looting` | 0x800 | 拾取 |
| `Attacking` | 0x1000 | 攻击 |
| `ItemUse` | 0x2000 | 使用物品 |
| `DamageChannelDuration` | 0x4000 | 受伤减少 channel 时长 |
| `Shapeshifting` | 0x8000 | 变形 |
| `ActionDelayed` | 0x10000 | 延迟动作 |
| `Mount` | 0x20000 | 上坐骑 |
| `Standing` | 0x40000 | 站起 |
| `LeaveWorld` | 0x80000 | 离线 |
| `StealthOrInvis` | 0x100000 | 潜行/隐形 |
| `InvulnerabilityBuff` | 0x200000 | 无敌 buff |
| `EnterWorld` | 0x400000 | 上线 |
| `PvPActive` | 0x800000 | PvP 激活 |
| `NonPeriodicDamage` | 0x1000000 | 非周期性伤害 |
| `LandingOrFlight` | 0x2000000 | 起飞/降落 |
| `Release` | 0x4000000 | 释放灵魂 |
| `DamageCancelsScript` | 0x8000000 | NYI |
| `EnteringCombat` | 0x10000000 | 进入战斗 |
| `Login` | 0x20000000 | 登录 |
| `Summon` | 0x40000000 | 召唤 |
| `LeavingCombat` | 0x80000000 | 离开战斗 |
| **NOT_VICTIM** | 复合 | HostileActionReceived \| Damage \| NonPeriodicDamage |
| **AnyDamageMask** | 复合 | Damage \| NonPeriodicDamage \| DamageCancelsScript |

### SpellAuraInterruptFlags2 — 扩展 Aura 中断标志

| Flag | 值 | 含义 |
|------|-----|------|
| `Falling` | 0x01 | 落下 |
| `Swimming` | 0x02 | 游泳 |
| `NotMoving` | 0x04 | NYI |
| `Ground` | 0x08 | 着地 |
| `Transform` | 0x10 | 变身 |
| `Jump` | 0x20 | 跳跃 |
| `ChangeSpec` | 0x40 | 切换专精 |
| `AbandonVehicle` | 0x80 | 离开载具 |
| `StartOfRaidEncounterAndStartOfMythicPlus` | 0x100 | 团本/M+ 开始 |
| `EndOfRaidEncounterAndStartOfMythicPlus` | 0x200 | 团本/M+ 结束 |
| `Disconnect` | 0x400 | NYI |
| `EnteringInstance` | 0x800 | 进入副本 |
| `DuelEnd` | 0x1000 | 决斗结束 |
| `LeaveArenaOrBattleground` | 0x2000 | 离开竞技场/战场 |
| `ChangeTalent` | 0x4000 | 切换天赋 |
| `ChangeGlyph` | 0x8000 | 切换铭文 |
| `SeamlessTransfer` | 0x10000 | NYI |
| `WarModeLeave` | 0x20000 | 离开战争模式 |
| `TouchingGround` | 0x40000 | NYI |
| `ChromieTime` | 0x80000 | NYI |
| `SplineFlightOrFreeFlight` | 0x100000 | NYI |
| `ProcOrPeriodicAttacking` | 0x200000 | NYI |
| `ChallengeModeStart` | 0x400000 | 大秘境开始 |
| `StartOfEncounter` | 0x800000 | 战斗开始 |
| `EndOfEncounter` | 0x1000000 | 战斗结束 |
| `ReleaseEmpower` | 0x2000000 | 蓄力释放 |

### ChannelInterruptFlags — 引导法术中断标志

没有独立的枚举类型。**复用 `SpellAuraInterruptFlags` 和 `SpellAuraInterruptFlags2`**，作为 SpellInfo 的 `ChannelInterruptFlags` / `ChannelInterruptFlags2` 字段存储。

通过 `SpellInfo::HasChannelInterruptFlag(flag)` 检查，在 `RemoveAurasWithInterruptFlags()` 中同步检查 channel 法术。

### SpellPreventionType — 施法阻止类型

每个 SpellInfo 有一个 `PreventionType` 字段，标识该法术会被哪种 CC 阻止：

| Type | 值 | 含义 | 对应 CC |
|------|-----|------|---------|
| `NONE` | 0 | 不受任何 CC 阻止 | — |
| `SILENCE` | 1 | 被沉默阻止 | SPELL_AURA_MOD_SILENCE |
| `PACIFY` | 2 | 被安抚阻止 | SPELL_AURA_MOD_PACIFY |
| `NO_ACTIONS` | 4 | 被禁止行动阻止 | SPELL_AURA_MOD_NO_ACTIONS |

### UnitState — 单位状态标志

| State | 值 | 含义 | 如何进入 |
|-------|-----|------|---------|
| `STUNNED` | 0x08 | 眩晕 | SPELL_AURA_MOD_STUN |
| `ROOT` | 0x400 | 定身 | SPELL_AURA_MOD_ROOT |
| `CONFUSED` | 0x800 | 迷惑 | SPELL_AURA_MOD_CONFUSE |
| `FLEEING` | 0x80 | 恐惧逃跑 | SPELL_AURA_MOD_FEAR |
| `CASTING` | 0x8000 | 正在施法 | 施法进入 |

**复合状态：**
- `UNIT_STATE_CONTROLLED` = CONFUSED | STUNNED | FLEEING
- `UNIT_STATE_LOST_CONTROL` = CONTROLLED | POSSESSED | JUMPING | CHARGING

### Mechanics 枚举 — 机制类型（用于免疫判定）

| Mechanic | 值 | 含义 |
|----------|-----|------|
| `CHARM` | 1 | 惑控 |
| `DISORIENTED` | 2 | 迷惑 |
| `DISARM` | 3 | 缴械 |
| `FEAR` | 5 | 恐惧 |
| `ROOT` | 7 | 定身 |
| `SILENCE` | 9 | 沉默 |
| `SLEEP` | 10 | 催眠 |
| `SNARE` | 11 | 减速 |
| `STUN` | 12 | 眩晕 |
| `FREEZE` | 13 | 冰冻 |
| `KNOCKOUT` | 14 | 击倒 |
| `POLYMORPH` | 17 | 变形 |
| `BANISH` | 18 | 放逐 |
| `HORROR` | 24 | 惊骇 |
| `INTERRUPT` | 26 | 打断 |
| `DAZE` | 27 | 眩晕（弱） |
| `IMMUNE_SHIELD` | 29 | 无敌护盾 |

### Unit 上的关键容器

| 容器 | 用途 |
|------|------|
| `m_interruptableAuras` | 可中断 aura 的子集（forward_list），高效遍历 |
| `m_interruptMask` / `m_interruptMask2` | 所有可中断 aura 的 flag 并集（快速判断"有没有 aura 需要检查"） |
| `m_currentSpells[4]` | 四个法术槽位：GENERIC / CHANNELED / MELEE / AUTOREPEAT |

## Section 2: Flow Diagram

### 完整中断链路：从 CC 施加到法术/aura 中断

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  CC Aura Apply 阶段（眩晕/沉默/恐惧/迷惑/安抚/禁止行动）                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  例如：眩晕法术命中 target                                                    │
│    │                                                                        │
│    ▼                                                                        │
│  AuraEffect::HandleAuraModStun(aurApp, REAL, apply=true)                   │
│    │                                                                        │
│    ├─ target->SetControlled(true, UNIT_STATE_STUNNED)                       │
│    │    │                                                                   │
│    │    ├─ if (state & UNIT_STATE_CONTROLLED)                               │
│    │    │    └─ CastStop()                                                  │
│    │    │         └─ 遍历 m_currentSpells[GENERIC/AUTOREPEAT/CHANNELED]      │
│    │    │              └─ InterruptSpell(slot) → spell->cancel()            │
│    │    │                   ├─ PREPARING → 退 GCD + 发送中断消息              │
│    │    │                   └─ CHANNELING → 移除目标 aura + 发送中断消息       │
│    │    │                                                                   │
│    │    ├─ AddUnitState(UNIT_STATE_STUNNED)                                 │
│    │    └─ SetStunned(true)                                                 │
│    │         ├─ SetTarget(Empty)                                            │
│    │         ├─ SetUnitFlag(UNIT_FLAG_STUNNED)                              │
│    │         ├─ StopMoving()                                                │
│    │         ├─ SetRooted(true)                                             │
│    │         └─ CastStop()  ← 第二次 CastStop                               │
│    │                                                                        │
│    └─ [眩晕不清除已有 aura] — 眩晕只中断施法中的法术，不按 flag 移除 aura      │
│                                                                             │
│  沉默走不同路径：                                                             │
│  AuraEffect::HandleAuraModSilence(aurApp, REAL, apply=true)                │
│    ├─ target->SetSilencedSchoolMask(miscValue)                              │
│    └─ InterruptSpellsWithPreventionTypeOnAuraApply(target, SILENCE)        │
│         └─ 遍历所有 currentSpells                                           │
│              └─ if spell.PreventionType & SILENCE → InterruptSpell()       │
│                                                                             │
│  安抚走类似路径：                                                             │
│  AuraEffect::HandleAuraModPacify(aurApp, REAL, apply=true)                 │
│    ├─ target->SetUnitFlag(UNIT_FLAG_PACIFIED)                              │
│    └─ InterruptSpellsWithPreventionTypeOnAuraApply(target, PACIFY)         │
│                                                                             │
│  禁止行动走类似路径：                                                          │
│  AuraEffect::HandleAuraModNoActions(aurApp, REAL, apply=true)              │
│    ├─ target->SetUnitFlag2(UNIT_FLAG2_NO_ACTIONS)                          │
│    └─ InterruptSpellsWithPreventionTypeOnAuraApply(target, NO_ACTIONS)     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### RemoveAurasWithInterruptFlags 链路（事件驱动的 aura 移除）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  事件触发点（Unit.cpp 中的调用点）                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  受伤时:                                                                    │
│    Unit::ProcDamageAndSpellFor() / Unit::HandleProcDamageAndSpell()         │
│      └─ RemoveAurasWithInterruptFlags(Damage | NonPeriodicDamage)           │
│                                                                             │
│  移动时:                                                                    │
│    Unit::UpdatePosition()                                                   │
│      ├─ Relocated → RemoveAurasWithInterruptFlags(Moving)                   │
│      └─ Turned → RemoveAurasWithInterruptFlags(Turning)                     │
│                                                                             │
│  上坐骑/下坐骑:                                                              │
│    Unit::Mount() / Unit::Dismount()                                         │
│      └─ RemoveAurasWithInterruptFlags(Mount / Dismount)                     │
│                                                                             │
│  变形/潜行/进入战斗/离开战斗/进入水/离开水/等                                  │
│    → RemoveAurasWithInterruptFlags(对应 flag)                               │
│                                                                             │
└─────────────┬───────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  RemoveAurasWithInterruptFlags(flag, source) 模板函数                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. if (!HasInterruptFlag(flag)) return                                    │
│     └─ 快速退出：m_interruptMask 中没有此 flag，说明没有任何 aura 需要检查    │
│                                                                             │
│  2. 遍历 m_interruptableAuras:                                              │
│     for each aurApp:                                                        │
│       if (aura.SpellInfo.HasAuraInterruptFlag(flag)                        │
│           && !IsInterruptFlagIgnoredForSpell(flag))                         │
│         → RemoveAura(aura, AURA_REMOVE_BY_INTERRUPT)                       │
│                                                                             │
│  3. 检查 channel 法术:                                                      │
│     if (currentChanneledSpell && spell.SpellInfo.HasChannelInterruptFlag()) │
│       → InterruptNonMeleeSpells(false)                                     │
│                                                                             │
│  4. UpdateInterruptMask()                                                  │
│     └─ 重新计算 m_interruptMask = 所有 interruptableAuras 的 flag 并集       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 施法前 CC 阻止检查（CheckCast 路径）

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Spell::CheckCast() — 新施法前的 CC 阻止检查                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  if (UNIT_FLAG_STUNNED):                                                   │
│    ├─ usableWhileStunned (SPELL_ATTR5_ALLOW_WHILE_STUNNED)?                │
│    │   └─ mechanicCheck: 遍历 STUN aura，检查 mechanic mask 是否在           │
│    │       spell.GetAllowedMechanicMask() 中 → 不在则 SPELL_FAILED_STUNNED  │
│    └─ 否则 → SPELL_FAILED_STUNNED (除非 CheckSpellCancelsStun)             │
│                                                                             │
│  else if (IsSilenced(schoolMask) && PreventionType & SILENCE):             │
│    └─ SPELL_FAILED_SILENCED (除非 CheckSpellCancelsSilence)                 │
│                                                                             │
│  else if (UNIT_FLAG_PACIFIED && PreventionType & PACIFY):                  │
│    └─ SPELL_FAILED_PACIFIED (除非 CheckSpellCancelsPacify)                  │
│                                                                             │
│  else if (UNIT_FLAG_FLEEING):                                              │
│    ├─ usableWhileFeared? → mechanicCheck                                   │
│    └─ 否则 → SPELL_FAILED_FLEEING (除非 CheckSpellCancelsFear)              │
│                                                                             │
│  else if (UNIT_FLAG_CONFUSED):                                             │
│    ├─ usableWhileConfused? → mechanicCheck                                 │
│    └─ 否则 → SPELL_FAILED_CONFUSED (除非 CheckSpellCancelsConfuse)          │
│                                                                             │
│  else if (UNIT_FLAG2_NO_ACTIONS && PreventionType & NO_ACTIONS):           │
│    └─ SPELL_FAILED_NO_ACTIONS (除非 CheckSpellCancelsNoActions)             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### MechanicImmunity 在 Aura Apply 前的检查

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Aura::UpdateTargetMap() / Aura::TryCreate()                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  对每个 target:                                                             │
│    ├─ target.IsImmunedToSpell(spellInfo, effMask) → skip                   │
│    │   └─ 检查 IMMUNITY_MECHANIC 容器                                       │
│    │       如果 target 有 MECHANIC_STUN 免疫 → 眩晕 aura 不会被应用          │
│    │                                                                        │
│    └─ target.IsImmunedToSpellEffect(spellInfo, effect) → skip              │
│                                                                             │
│  MechanicImmunity 来源:                                                     │
│    ├─ SPELL_AURA_MECHANIC_IMMUNITY (特定机制免疫)                            │
│    ├─ SPELL_AURA_MECHANIC_IMMUNITY_MASK (机制掩码免疫)                       │
│    └─ 种族天赋 / 职业 buff / 装备效果                                       │
│                                                                             │
│  免疫 → aura 根本不会被应用 → 不触发 SetControlled → 不中断任何东西           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Section 3: Key Design Decisions

### Decision: 眩晕和沉默/安抚的中断路径不同

**眩晕** → `SetControlled()` → `CastStop()` → 直接取消所有当前法术（不检查 PreventionType）
**沉默** → `HandleAuraModSilence()` → `InterruptSpellsWithPreventionTypeOnAuraApply(SILENCE)` → 只取消 PreventionType & SILENCE 的法术
**安抚** → 类似沉默，但用 PACIFY 类型
**禁止行动** → 类似，用 NO_ACTIONS 类型

**Reason:** 眩晕是完全丧失控制（连移动都不行），所以无条件取消所有施法。沉默只禁止施放法术但不影响近战/移动，所以用 PreventionType 精确匹配。安抚禁止近战但允许法术，也是精确匹配。

### Decision: 眩晕不使用 RemoveAurasWithInterruptFlags

眩晕本身不会调用 `RemoveAurasWithInterruptFlags`。眩晕只是：
1. 取消当前施法中的法术
2. 阻止新施法

已存在的 aura（包括周期性触发 spell 的 aura）**不受眩晕影响**继续运行。只有当 aura 自身带有特定 SpellAuraInterruptFlags（如 Damage、Moving 等）并且在对应事件触发时才会被移除。

**Reason:** 眩晕控制的是"行动能力"，不是"已有魔法效果"。一个已经施加在目标身上的 DoT 或周期触发 aura 是独立的魔法实体，不因施法者被眩晕而消失。

### Decision: m_interruptableAuras 子集 + m_interruptMask 快速退出

Unit 不遍历所有 appliedAuras 检查中断，而是维护 `m_interruptableAuras`（forward_list 子集）和 `m_interruptMask`（flag 并集）。

**Reason:** 性能。每次受伤/移动都触发 `RemoveAurasWithInterruptFlags`，如果遍历所有 aura 会很慢。通过 mask 快速判断"没有任何 aura 需要检查"可以跳过大部分调用。

### Decision: ChannelInterruptFlags 复用 SpellAuraInterruptFlags

没有独立的 ChannelInterruptFlags 枚举。SpellInfo 有 `ChannelInterruptFlags` 字段，类型和 `AuraInterruptFlags` 相同。

**Reason:** Channel 法术同时具有"法术"和"aura"双重身份。引导期间它在目标上维持 aura，所以中断条件（受伤、移动等）和 aura 的中断条件相同。复用避免重复定义。

### Decision: PreventionType 是 per-spell 的固定属性

每个 SpellInfo 有一个 `PreventionType` 字段（不是 bitmask），标识该法术被哪种 CC 阻止。PreventionType 在数据定义时确定，不是运行时计算。

**Reason:** 不同类型的法术受不同 CC 影响。近战技能被安抚阻止但不受沉默影响，法术被沉默阻止但不受安抚影响。PreventionType 将这个映射关系固化在数据中。

### Decision: usableWhileStunned 做 mechanic mask 精确匹配

即使法术标记了 `SPELL_ATTR5_ALLOW_WHILE_STUNNED`，仍然检查当前眩晕 aura 的 mechanic mask 是否在 spell 的 AllowedMechanicMask 中。

**Reason:** 眩晕的来源有多种（STUN、SLEEP、FREEZE、KNOCKOUT 等），法术可能"可在眩晕中使用"但仅限特定类型的眩晕。例如 Ice Block（冰块）解冻后可用但不能在 Polymorph（变羊）中使用。精确匹配避免过度允许。

### Decision: IsInterruptFlagIgnoredForSpell 提供例外路径

`RemoveAurasWithInterruptFlags` 中对每个 aura 检查 `IsInterruptFlagIgnoredForSpell()`，提供特殊例外：
- `Moving` flag 但 target 有 `CAST_WHILE_WALKING` aura → 不中断
- `Action` flag 但来源 spell 允许在 channeling 时施放 → 不中断

**Reason:** 某些机制允许打破常规。"移动中断"可以被"施法时行走" aura 覆盖，"动作中断"可以被特定属性豁免。

## Section 4: Reusable Patterns

### Pattern: 三层中断架构（CC Apply → 事件驱动 → Update 检查）

```
Layer 1: CC Apply 时立即中断
  眩晕 → CastStop() → 取消所有当前法术
  沉默 → InterruptSpellsWithPreventionType() → 取消匹配的法术

Layer 2: 事件驱动中断
  受伤/移动/坐骑等事件 → RemoveAurasWithInterruptFlags()
  → 遍历 m_interruptableAuras → 按 flag 移除匹配 aura + channel spell

Layer 3: Update 持续检查
  Spell::update() → 检查移动/目标存在/范围
  → 不满足条件 → cancel()
```

**适用:** 需要多种中断触发源的复杂技能系统。

### Pattern: PreventionType 分层阻止

```
CC 类型           阻止范围                          施法阻止方式
────────────────────────────────────────────────────────────
眩晕 (Stun)       全部法术 + 全部动作               CastStop() 无条件
沉默 (Silence)    PreventionType & SILENCE 的法术   InterruptSpellsWithPreventionType()
安抚 (Pacify)     PreventionType & PACIFY 的法术    InterruptSpellsWithPreventionType()
恐惧 (Fear)       全部法术（类似眩晕）               SetControlled → CastStop()
迷惑 (Confuse)    全部法术（类似眩晕）               SetControlled → CastStop()
禁止行动          PreventionType & NO_ACTIONS       InterruptSpellsWithPreventionType()
```

**适用:** CC 系统需要精细控制"哪些技能被阻止"的游戏。

### Pattern: Mask + 子集 的快速中断过滤

```
1. 维护可中断对象子集 (m_interruptableAuras)
2. 维护 flag 并集 mask (m_interruptMask)
3. 中断事件触发时：
   a. if (!HasInterruptFlag(flag)) return;  ← O(1) 快速退出
   b. 遍历子集（不是全集）检查匹配
4. 每次子集变化后重新计算 mask
```

**适用:** 高频事件驱动的中断检查（每帧都可能触发）。

### Pattern: 免疫在 Apply 前拦截

```
CC 施加流程:
  效果命中 → 检查免疫 → 免疫则跳过 → 不进入 SetControlled → 不触发任何中断

免疫来源:
  - MechanicImmunity（机制免疫）
  - SchoolImmunity（学派免疫）
  - EffectImmunity（效果免疫）
  - StateImmunity（状态免疫）
```

**适用:** 需要防止 CC 叠加或 CC 链的系统。免疫在最早的环节拦截，避免需要"先应用再移除"的回滚。

### Pattern: CC Apply 和 CC Remove 的对称设计

```
Apply:
  SetControlled(true, STUNNED)
    → AddUnitState + SetStunned(true) + CastStop()

Remove:
  SetControlled(false, STUNNED)
    → if (still has stun aura) return  ← 防止多重 CC 互相清除
    → ClearUnitState + SetStunned(false)
    → ApplyControlStatesIfNeeded()  ← 检查是否还有其他 CC 需要重新应用
```

**适用:** 多重 CC 可能叠加的场景（两个眩晕 aura 同时存在，移除一个不能解除另一个的效果）。

### TC 中各 CC 类型的中断行为对比

| CC 类型 | AuraType | 进入路径 | 中断当前施法 | 移除已有 aura | 阻止新施法 |
|---------|----------|---------|-------------|-------------|-----------|
| 眩晕 | MOD_STUN | SetControlled → CastStop | 全部 | 否（不调 RemoveAuras） | 全部（usableWhileStunned 可豁免部分） |
| 沉默 | MOD_SILENCE | InterruptSpellsWithPreventionType(SILENCE) | PreventionType & SILENCE | 否 | PreventionType & SILENCE |
| 安抚 | MOD_PACIFY | InterruptSpellsWithPreventionType(PACIFY) | PreventionType & PACIFY | 否 | PreventionType & PACIFY |
| 恐惧 | MOD_FEAR | SetControlled → CastStop | 全部 | 否 | 全部（usableWhileFeared 可豁免部分） |
| 迷惑 | MOD_CONFUSE | SetControlled → CastStop | 全部 | 否 | 全部（usableWhileConfused 可豁免部分） |
| 禁止行动 | MOD_NO_ACTIONS | InterruptSpellsWithPreventionType(NO_ACTIONS) | PreventionType & NO_ACTIONS | 否 | PreventionType & NO_ACTIONS |

**关键结论：CC 永远不会自动移除已有 aura。** CC 只做两件事：(1) 中断正在施放的法术 (2) 阻止新施法。已有 aura（包括周期性触发 spell 的 aura）只有在自身 SpellAuraInterruptFlags 匹配且对应事件触发时才会被移除。
