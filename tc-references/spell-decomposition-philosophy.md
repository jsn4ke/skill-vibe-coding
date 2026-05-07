# TC 法术拆解设计哲学

> Source: TrinityCore 综合 | 生成: 2026-05-06 | Topic: spell decomposition, effect composition, trigger spell chain, aura-driven behavior, skill design strategy

本文档是**跨机制**的策略指南，综合 Spell Cast Flow、Effect Pipeline、TriggerSpell、Aura System、Spell Script 等机制，阐述 TC 如何将复杂技能拆解为可组合的原子单元。

---

## 1. 核心原则

### 1.1 四层原子模型

```
┌─────────────────────────────────────────────────────┐
│  Spell（容器）— 最多持有 3 个 Effect 槽位           │
│    EFFECT_0: 做一件事                                │
│    EFFECT_1: 做一件事                                │
│    EFFECT_2: 做一件事                                │
│                                                      │
│  每个 Effect 只做一件事：                             │
│    造成伤害、挂 Aura、恢复能量、触发子法术、...       │
│                                                      │
│  复杂度不靠堆逻辑，靠组合：                           │
│    Spell → Effect → TriggerSpell → Spell → ...       │
│    Spell → EffectApplyAura → Aura Tick → TriggerSpell │
└─────────────────────────────────────────────────────┘
```

**一句话总结：Spell 是容器，Effect 是原子操作，TriggerSpell 是组合手段，Aura 是持续状态。**

### 1.2 设计约束

| 约束 | 含义 |
|------|------|
| 一个 Spell 最多 3 个 Effect | EFFECT_0/1/2，超出必须拆成子法术 |
| 每个 EffectType 做一件事 | SchoolDMG 只做伤害，ApplyAura 只挂光环 |
| 复杂行为 = 多个 Spell 组合 | 通过 TriggerSpell、Aura Tick、Proc 事件串联 |
| 数据驱动优先 | 优先用 DBC 字段（TriggerSpell、AuraPeriod），脚本(Dummy+Script)是兜底 |

---

## 2. 五种拆解模式

### 模式 A：单 Spell 多 Effect（最简单）

**适用**：效果数量 ≤ 3 且无时序依赖。

```
Spell (11366 Pyroblast)
  EFFECT_0 = SchoolDamage        ← 即时火焰伤害
  EFFECT_1 = ApplyAura(PeriodicDamage, period=3000ms)  ← 12 秒 DoT
```

不需要 TriggerSpell，两个 Effect 在同一 Spell 的 HIT_TARGET 阶段按顺序执行。

**TC 行为**：
- `HandleEffects(unit, eff0, HIT_TARGET)` → `EffectSchoolDamage()` → 造成即时伤害
- `HandleEffects(unit, eff1, HIT_TARGET)` → `EffectApplyAura()` → 挂上 PeriodicDamage Aura
- 两个 Effect 独立工作，互不干扰

**关键 DBC 字段**：
- `Effect[0..2]`：EffectType 枚举值
- `EffectBasePoints[0..2]`：每个 effect 的数值
- `EffectAuraPeriod[0..2]`：周期性 effect 的 tick 间隔

### 模式 B：TriggerSpell 链式触发（多步串联）

**适用**：效果数量 > 3，或有**时序依赖**（先 A 再 B），或需要**条件分支**。

#### 例 1：Deep Wounds（流血 DoT）

```
天赋 Aura (12162 Deep Wounds)
  EFFECT_0 = ApplyAura(ProcTriggerSpell)
  → Proc 条件：近战暴击
  → Proc 触发时：

    子法术（Dummy spell，由 AuraScript 触发）
      EFFECT_0 = Dummy → SpellScript 计算流血伤害
      → 再 CastSpell(12721)：

        子子法术 (12721 Deep Wounds Periodic)
          EFFECT_0 = ApplyAura(PeriodicDamage)
          → 每 tick 造成流血伤害
```

**链路**：`Aura Proc → Dummy Spell → Periodic DoT Spell`（3 层嵌套）

**为什么这么复杂？**
- 天赋本身是被动 Aura（永远挂在玩家身上）
- 触发条件是近战暴击（Proc 机制）
- 伤害计算依赖武器伤害 × 天赋等级（需要脚本）
- 最终效果是周期性流血（需要独立 Aura 管理周期）

#### 例 2：Living Bomb（活体炸弹）

```
入口 Spell (44457 Living Bomb)
  EFFECT_0 = SchoolDamage          ← 即时伤害
  EFFECT_1 = ApplyAura(Dummy)      ← 挂 Dummy Aura，BasePoints 存爆炸法术 ID
  EFFECT_2 = ApplyAura(PeriodicDamage) ← 每 tick 造成持续伤害

  Aura 到期/被驱散时：
    AuraScript::AfterEffectRemove(EFFECT_1)
      → 读取 aurEff.GetAmount() = 爆炸法术 ID
      → CastSpell(target, explosion_spell_id)

    爆炸子法术
      EFFECT_0 = SchoolDamage      ← AoE 爆炸伤害
```

**链路**：`入口 Spell → 即时伤害 + Dummy Aura + DoT Aura → Aura 到期 → 爆炸子法术`

**巧妙之处**：
- 用 Dummy Aura 的 BasePoints 存储"要触发的法术 ID"，数据驱动
- 到期触发由 AuraScript 的 `AfterEffectRemove` 拦截，不需要定时器
- 三个阶段（即时/持续/爆炸）分散在两个 Spell 中，各自职责清晰

### 模式 C：Aura PeriodicTick → TriggerSpell（周期触发）

**适用**：每固定间隔触发一次子法术。DoT/HoT 的通用模式。

#### 例：Arcane Missiles（奥术飞弹）

```
引导 Spell (5143 Arcane Missiles)
  EFFECT_0 = ApplyAura(PeriodicTriggerSpell, TriggerSpell=导弹法术, Period=1000ms)
  → 引导期间每 1 秒：

    PeriodicTick 触发
      → Aura 系统读取 TriggerSpell 字段
      → CastSpell(missile_spell) → 对目标造成一次伤害
```

**链路**：`Channel Spell → PeriodicTriggerSpell Aura → 每 tick 触发导弹子法术`

**关键机制**：
- `SPELL_AURA_PERIODIC_TRIGGER_SPELL`：Aura 内置的周期触发能力，纯数据驱动
- `EffectAuraPeriod`：tick 间隔（ms）
- `EffectTriggerSpell`：每 tick 触发的子法术 ID
- 引导期间由 Unit.Update 驱动 Aura tick

#### 一般化模板

```
任何需要"每 X 毫秒做 Y"的场景：
  Spell → ApplyAura(PeriodicTriggerSpell, period=X, triggerSpell=Y_spell)
  Y_spell 的 Effect 只做一件事（通常是 SchoolDamage 或 ApplyAura）
```

### 模式 D：Proc Aura → TriggerSpell（条件触发）

**适用**：在特定战斗事件发生时触发（受击、暴击、施法、击杀等）。

#### 例 1：Sword and Board（剑盾合击）

```
被动 Aura (46951 Sword and Board)
  EFFECT_0 = ApplyAura(ProcTriggerSpell)
  → spell_proc 表定义：
    - ProcFlags：特定技能命中（复仇/毁灭打击/雷霆一击/震荡猛击）
    - SpellFamilyName：Warrior
    - SpellFamilyMask：过滤 qualifying 技能

  → Proc 触发时：
    AuraScript::HandleProc
      → 重置 Shield Slam 的冷却
      → 玩家可以立即使用 Shield Slam
```

**链路**：`被动 Aura → Proc 匹配 → 重置冷却/触发子法术`

**关键机制**：
- `SPELL_AURA_PROC_TRIGGER_SPELL`（42）：Proc 触发型 Aura
- `spell_proc` 表：定义触发条件（ProcFlags、SpellFamilyMask 等）
- Proc 由战斗事件驱动（受击、造成伤害、施法等）

#### 例 2：Blazing Speed（炽热疾速）

```
被动 Aura (31641 Blazing Speed)
  EFFECT_0 = ApplyAura(ProcTriggerSpell, TriggerSpell=31643)
  → Proc 条件：被近战攻击命中

  → Proc 触发时：
    → 自动 CastSpell(31643) — 移动加速 Buff
```

**链路**：`被动 Aura → 被近战命中 → 触发移动加速 Buff`

**与模式 C 的区别**：
- 模式 C（PeriodicTrigger）：固定时间间隔触发
- 模式 D（ProcTrigger）：特定战斗事件触发

### 模式 E：Dummy + SpellScript（万能出口）

**适用**：标准 EffectType 无法表达的行为。

```
DBC 数据：
  Spell EFFECT_0 = SPELL_EFFECT_DUMMY

引擎行为：
  EffectDummy() → 几乎为空（只处理 PetAura 和 DB 脚本启动）

脚本拦截：
  SpellScript::Register() {
    OnEffectHitTarget += SpellEffectFn(Handle, EFFECT_0, SPELL_EFFECT_DUMMY)
  }

  void Handle(SpellEffIndex) {
    // 这里写任意逻辑：复杂伤害公式、条件分支、多步触发...
  }
```

**这不是 hack，是 TC 的一等公民模式**。大量复杂技能通过 Dummy + Script 实现。

**何时用 Dummy vs 标准 Effect**：

| 场景 | 选择 |
|------|------|
| 纯伤害/治疗，数值公式简单 | EffectSchoolDamage / EffectHeal |
| 挂标准 Aura | EffectApplyAura |
| 触发固定 ID 的子法术 | EffectTriggerSpell（数据驱动） |
| 伤害公式依赖武器/属性/状态 | Dummy + Script |
| 需要多步条件判断 | Dummy + Script |
| 行为无法归入任何标准 EffectType | Dummy + Script |

---

## 3. 决策流程图

设计一个技能时，按以下顺序决策：

```
1. 这个技能有多少种效果？
   │
   ├─ 1~3 种，无时序依赖
   │   → 模式 A：单 Spell 多 Effect
   │      EFFECT_0/1/2 各做一件事
   │
   ├─ 需要固定间隔重复触发
   │   → 模式 C：PeriodicTriggerSpell Aura
   │      Spell → ApplyAura → 每 tick 触发子法术
   │
   ├─ 需要在特定战斗事件触发
   │   → 模式 D：ProcTriggerSpell Aura
   │      Spell → ApplyAura(Proc) → 事件匹配时触发
   │
   ├─ 效果 > 3 种，或有时序依赖，或有条件分支
   │   → 模式 B：TriggerSpell 链
   │      入口 Spell → EffectTriggerSpell → 子 Spell → ...
   │
   └─ 行为无法用标准 EffectType 表达
       → 模式 E：Dummy + SpellScript
          EffectDummy + OnEffectHitTarget hook

2. 选定模式后，逐个 Effect 检查：
   │
   ├─ 能用标准 EffectType 吗？
   │   → 优先使用，数据驱动
   │
   ├─ 需要 TriggerSpell 吗？
   │   → 加一个 EffectTriggerSpell effect
   │
   └─ 标准类型不够用？
       → 用 Dummy + Script 补充
```

---

## 4. TriggerSpell 的目标传递规则

子法术通过 `NeedsToBeTriggeredByCaster()` 决定目标来源：

```
父 Spell 有目标 targetUnit
  → EffectTriggerSpell 触发子 Spell
    │
    ├─ 子 Spell NeedsToBeTriggeredByCaster() == true
    │   → 继承父 Spell 的目标（LAUNCH_TARGET 路径传递）
    │
    └─ 子 Spell NeedsToBeTriggeredByCaster() == false
        → 子 Spell 自行定位目标（self-cast、AoE、implicit target）
```

**原则**：子法术尽量自己决定目标，只有需要显式单目标的子法术才继承父法术目标。

---

## 5. 补充机制：spell_linked_spell

除了 EffectTriggerSpell 和 Proc Aura 之外，TC 还有一个**数据库表驱动的法术关联机制**：

| 字段 | 含义 |
|------|------|
| `spell_trigger` | 触发条件法术 ID |
| `spell_effect` | 被触发的法术 ID |
| `type` | 触发时机：0=施放时, 1=命中时, 2=移除时 |
| `search` | 匹配方式 |

**用途**：简单的法术关联不需要写脚本或 DBC TriggerSpell 字段，直接在 spell_linked_spell 表中配置。

**示例**：某些形态切换会在施放时自动触发关联法术。

**与 TriggerSpell 的区别**：
- `EffectTriggerSpell`：写在 DBC Effect 数据中，是 Spell 自身的属性
- `spell_linked_spell`：数据库配置，是法术之间的外部关联关系

---

## 6. 关键架构约束：Aura 不能直接触发 Spell Effect

### 两套独立的 Effect 体系

TC 有两套完全不同的 Effect 类型，**它们不互通**：

| | Spell Effect | Aura Effect (AuraType) |
|---|---|---|
| 定义位置 | `SpellEffectInfo.Effect` | `SpellEffectInfo.ApplyAuraName` |
| 枚举前缀 | `SPELL_EFFECT_XXX` (~190 种) | `SPELL_AURA_XXX` (~300 种) |
| 执行者 | `Spell::HandleEffects()` | `AuraEffect` 内部处理 |
| 生命周期 | 随 Spell 一次性执行 | 随 Aura 持续存在 |

**不存在 Aura → Spell Effect 的直接路径。** Spell Effect Pipeline 是 Spell 的内部机制，Aura 无法调用。

### Aura 只有两件事可做

**1. 自身内置行为（不经过 Spell）：**

```
SPELL_AURA_PERIODIC_DAMAGE → Aura tick 时直接调用 Unit::DealDamage()
SPELL_AURA_MOD_STAT        → Aura Apply 时直接修改属性值
SPELL_AURA_MOD_SPEED       → Aura Apply 时直接修改移动速度
```

这些是 Aura 系统内部处理的，不需要也不经过 Spell。

**2. 触发一个 Spell（由 Spell 执行其 Effects）：**

```
SPELL_AURA_PERIODIC_TRIGGER_SPELL → 每 tick → CastSpell(sub_spell) → sub_spell 的 Effect pipeline
SPELL_AURA_PROC_TRIGGER_SPELL     → Proc   → CastSpell(sub_spell) → sub_spell 的 Effect pipeline
```

子 Spell 有自己的 SpellInfo 和 Effect 列表，走完整的 Spell Effect Pipeline。

### 完整架构图

```
                    ┌─── Spell ──→ Effect Pipeline (SpellEffect)
                    │                ├─ SchoolDamage
                    │                ├─ ApplyAura ──→ Aura (AuraEffect lifecycle)
                    │                ├─ TriggerSpell ──→ 子 Spell ──→ 子 Spell 的 Effects
                    │                └─ Dummy ──→ SpellScript
                    │
入口 Spell ──→ Effect ─┤
                    │
                    └─── Aura 事件 ──→ 两条路：
                          │
                          ├─ 简单行为 → Aura 内部直接处理（DealDamage、ModStat）
                          │             不经过 Spell
                          │
                          └─ 复杂行为 → CastSpell(子 Spell)
                                        → 子 Spell 的 Effect Pipeline
                                        （不是直接触发 Effect）
```

### 设计含义

设计技能时，如果 Aura 的 tick/proc 需要做**超出 AuraType 内置能力**的事，就必须创建一个子 Spell 来承载那些 Effect。不能想着"让 Aura 直接执行一个 SchoolDamage effect"。

| 需求 | 正确做法 | 错误做法 |
|------|---------|---------|
| DoT 每 tick 造成固定伤害 | `SPELL_AURA_PERIODIC_DAMAGE`（Aura 内置） | ~~触发 Spell 做 SchoolDamage~~（可以但不必要） |
| DoT 每 tick 触发复杂逻辑 | `PeriodicTriggerSpell` → 子 Spell → 子 Spell 的 Effects | ~~Aura 直接调用 Effect~~ |
| Proc 时挂 Buff | `ProcTriggerSpell` → 子 Spell → ApplyAura | ~~Aura 直接 ApplyAura Effect~~ |

## 7. Aura 作为持续行为载体的角色总结

Aura 不仅仅是 Buff/Debuff。在 TC 中，Aura 是**持续行为的状态机**：

| Aura 类型 | 行为 | 对应拆解模式 |
|-----------|------|-------------|
| `PeriodicDamage` | 每 tick 造成伤害（Aura 内置） | 模式 A（Pyroblast DoT） |
| `PeriodicTriggerSpell` | 每 tick 触发子法术 | 模式 C（Arcane Missiles） |
| `ProcTriggerSpell` | 战斗事件触发子法术 | 模式 D（Sword and Board） |
| `Dummy` + AuraScript | 脚本拦截，自定义行为 | 模式 E（Living Bomb 爆炸） |
| `ModStat` / `ModResistance` | 属性修改（Aura 内置） | 标准 Buff/Debuff |
| `Mechanic` | 免疫/控制（Aura 内置） | 控制效果 |

**关键**：Aura 有完整的生命周期事件（Apply / Remove / Periodic Tick / Proc），每个事件都可以挂 AuraScript 拦截。这使得 Aura 成为连接即时效果和持续行为的桥梁。

---

## 8. 本项目映射

| TC 概念 | 本项目对应 | 状态 |
|---------|-----------|------|
| SpellInfo + Effects | `spell.Info` + `spell.Effect` 列表 | ✅ 已实现 |
| EffectType dispatch | `effect.Handler` 按 type 分发 | ✅ 已实现 |
| EffectTriggerSpell | `effect.EffectTriggerSpell` handler | ✅ 已实现 |
| EffectApplyAura | `effect.EffectApplyAura` handler | ✅ 已实现 |
| EffectDummy + Script | `effect.EffectDummy` + `RegisterScripts` hook | ✅ 已实现 |
| Aura lifecycle | `AuraMgr.ApplyAura` + Aura 事件 | ✅ 已实现 |
| PeriodicTriggerSpell | Aura PeriodicTick 中触发子法术 | ✅ 已实现 |
| ProcTriggerSpell | Proc 系统 | ❌ 待实现 |
| spell_linked_spell | 无对应（可考虑） | ❌ 待评估 |
| AuraScript hooks | Aura 相关回调 | 部分（OnRemove 有） |

### 本项目中的组合示例

设计新技能时，参考以下已实现的模式：

```go
// 模式 A：单 Spell 多 Effect
var Info = spell.Info{
    Effects: []spell.Effect{
        {Type: effect.SchoolDamage, ...},
        {Type: effect.ApplyAura, ...},
    },
}

// 模式 B：TriggerSpell 链
var ParentInfo = spell.Info{
    Effects: []spell.Effect{
        {Type: effect.TriggerSpell, TriggerSpellID: childSpellID, ...},
    },
}

// 模式 C：PeriodicTrigger（通过 Aura 的 TriggerSpellID）
var ChannelInfo = spell.Info{
    Effects: []spell.Effect{
        {Type: effect.ApplyAura, AuraType: aura.PeriodicTriggerSpell,
         AuraPeriod: 1000, TriggerSpellID: tickSpellID, ...},
    },
}

// 模式 E：Dummy + RegisterScripts
var ComplexInfo = spell.Info{
    Effects: []spell.Effect{
        {Type: effect.Dummy, ...},
    },
}
func RegisterScripts(reg *spell.Registry, caster *unit.Unit, eng *engine.Engine) {
    reg.RegisterSpellHook(ComplexInfo.ID, spell.HookOnEffectHit, func(ctx spell.EffectContext) { ... })
}
```

---

## 9. 技能设计检查清单

设计一个新技能时，逐项检查：

- [ ] 列出所有需要的效果（伤害、治疗、Buff、DoT、控制、召唤等）
- [ ] 效果数量 ≤ 3 且无时序依赖？→ 模式 A
- [ ] 需要固定间隔重复？→ 模式 C（PeriodicTriggerSpell）
- [ ] 需要战斗事件触发？→ 模式 D（Proc Aura，目前待实现）
- [ ] 效果 > 3 或有时序/条件依赖？→ 模式 B（TriggerSpell 链）
- [ ] 有无法用标准 EffectType 表达的行为？→ 模式 E（Dummy + Script）
- [ ] 子法术的目标从哪来？→ 父法术传递 vs 子法术自行定位
- [ ] Aura 到期/被驱散时需要做什么？→ AfterEffectRemove hook
- [ ] 需要资源消耗/GCD/冷却？→ 入口法术处理，子法术 triggered 跳过
