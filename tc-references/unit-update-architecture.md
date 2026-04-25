# Unit Update Architecture (Unit 驱动架构)

> Source: TrinityCore | Generated: 2026-04-25 | Topic: unit update, engine tick, spell lifecycle, aura ownership

---

## 1. 核心数据结构

### 1.1 Unit 的四大容器

Unit 是 TC 中所有战斗实体的基类（Player / Creature 均继承自 Unit）。Unit 内部维护四组核心容器，分别管理法术和光环的生命周期：

| 容器 | 类型 | 键 | 用途 |
|------|------|----|------|
| `m_currentSpells` | `std::array<Spell*, 4>` | CurrentSpellTypes 枚举索引 | 当前正在施放的法术（每种类型最多一个） |
| `m_ownedAuras` | `std::multimap<uint32, Aura*>` | SpellId | 本 Unit **拥有**（创建）的所有 Aura 实例 |
| `m_appliedAuras` | `std::multimap<uint32, AuraApplication*>` | SpellId | 施加在本 Unit 身上的所有 Aura 应用 |
| `_spellHistory` | `std::unique_ptr<SpellHistory>` | — | 冷却 / 充能 / 法术历史记录 |

**CurrentSpellTypes 枚举（Unit.h:595）：**
```
CURRENT_MELEE_SPELL    = 0   // 近战自动攻击
CURRENT_GENERIC_SPELL  = 1   // 普通法术（读条 / 瞬发）
CURRENT_CHANNELED_SPELL = 2  // 引导法术
CURRENT_AUTOREPEAT_SPELL = 3 // 自动重复（如射击）
```

### 1.2 ownedAuras vs appliedAuras 的区别

这是 TC 架构中非常关键的设计：

- **`m_ownedAuras`**：Unit **创建了哪些 Aura**。 Aura 的生命周期由 owner 驱动（owner 的 `_UpdateSpells` 中调用 `Aura::UpdateOwner`）。例如法师给战士施放了一个 buff，这个 Aura 存储在**法师**的 `m_ownedAuras` 中。

- **`m_appliedAuras`**：**哪些 Aura 施加在 Unit 身上**。 `AuraApplication` 是 Aura 在目标上的"投影"，负责管理可见性、效果应用/移除等。同一个 Aura 可能有多个 Application（多目标场景）。

- 两者是 **多对多** 关系：一个 Aura（owned）可能被应用到多个目标上（每个目标一个 Application）；一个 Unit 身上可能有多个人施加的同 ID Aura。

### 1.3 法术执行的三条路径

在 `Spell::prepare()`（Spell.cpp:3420）中，根据法术属性和触发标志，法术走不同的执行路径：

| 路径 | 条件 | 行为 | 帧延迟 |
|------|------|------|--------|
| **路径 A: 触发瞬发** | `TRIGGERED_CAST_DIRECTLY` 标志 且非持续引导 | 直接调用 `cast(true)`，跳过 SetCurrentCastSpell、SendSpellStart | 同帧同步完成 |
| **路径 B: 普通瞬发** | `m_casttime == 0` 且 `CURRENT_GENERIC_SPELL` | 注册为 currentSpell → SendSpellStart → `cast(true)` | 同帧同步完成 |
| **路径 C: 读条/延迟** | `m_casttime > 0` 或有弹道延迟 | 注册 SpellEvent 到事件队列 → 由 Update 链驱动 | 跨帧，由 timer 驱动 |

关键代码（Spell.cpp:3562-3596）：
```cpp
// 路径 A：直接触发
if ((_triggeredCastFlags & TRIGGERED_CAST_DIRECTLY) && (!m_spellInfo->IsChanneled() || !m_spellInfo->GetMaxDuration()))
    cast(true);
else
{
    // 路径 B/C：是否可以立即释放
    bool willCastDirectly = !m_casttime && GetCurrentContainer() == CURRENT_GENERIC_SPELL;
    
    // 注册为当前法术
    if (!willCastDirectly || !(_triggeredCastFlags & TRIGGERED_IGNORE_CAST_IN_PROGRESS))
        unitCaster->SetCurrentCastSpell(this);
    
    SendSpellStart();
    
    if (willCastDirectly)
        cast(true);   // 路径 B：同帧完成
    // 路径 C：不调用 cast()，等待 Spell::update 中 timer 倒数到 0 再 cast
}
```

---

## 2. 流程图

### 2.1 主更新链：Map → Unit → Spell/Aura

```
Map::Update(t_diff)
│
├── 1. 更新所有 Player 的 WorldSession（处理网络包）
│
├── 2. Player::Update(t_diff)
│       ├── Unit::Update(t_diff)                          // ← 核心入口
│       │   ├── WorldObject::Update(t_diff)
│       │   │   └── m_Events.Update(diff)                 // 处理事件队列
│       │   │       └── SpellEvent::Execute()             // 法术事件回调
│       │   │           └── Spell::update(p_time)         // 法术状态机推进
│       │   │               └── PREPARING: timer 倒数 → cast()
│       │   │               └── CHANNELING: timer 倒数 → finish()
│       │   │
│       │   └── _UpdateSpells(t_diff)                     // ← 法术+光环更新
│       │       ├── SpellHistory::Update()                // 冷却/充能更新
│       │       ├── 清理 FINISHED 法术指针                  // m_currentSpells[x] = nullptr
│       │       ├── 遍历 m_ownedAuras → Aura::UpdateOwner // Aura 生命驱动
│       │       │   └── Aura::Update(diff)
│       │       │   └── AuraEffect::Update(diff)          // 周期效果 tick
│       │       ├── 清理过期 Aura (RemoveOwnedAura)
│       │       └── ClientUpdate 可见 Aura
│       │
│       │   // 之后：战斗管理器、攻击计时器、移动、AI...
│       └── ...
│
└── 3. 通过 ObjectUpdater 访问附近的 Creature/GameObject
        └── Creature::Update(t_diff) → Unit::Update(t_diff) → 同上
```

**关键时序约束（Unit.cpp:425 注释）：**
> "Spells must be processed with event system BEFORE they go to _UpdateSpells. Or else we may have some SPELL_STATE_FINISHED spells stalled in pointers, that is bad."

这意味着 `m_Events.Update()`（在 `WorldObject::Update` 中）必须在 `_UpdateSpells`（在 `Unit::Update` 中）之前执行。这样当 `_UpdateSpells` 清理 FINISHED 法术时，事件系统已经完成了当帧的法术处理。

### 2.2 法术三条执行路径详细流程

```
WorldObject::CastSpell(spellId, targets, args)     // Object.cpp:2253
│
├── new Spell(caster, spellInfo, triggerFlags)      // 创建 Spell 对象
└── spell->prepare(targets)                          // Spell.cpp:3420
    │
    ├── m_spellState = SPELL_STATE_PREPARING
    ├── _spellEvent = new SpellEvent(this)
    ├── m_caster->m_Events.AddEvent(_spellEvent, +1ms)  // 注册到事件队列
    │
    ├── [验证] CheckCast(true) → 失败则 finish()
    ├── [计算] m_casttime = CalcCastTime()
    │
    ├── ┌─ 路径 A: TRIGGERED_CAST_DIRECTLY ──────────────┐
    │   │  cast(true) → _cast(true)                       │
    │   │   → handle_immediate()                          │
    │   │    → _handle_immediate_phase()  // 效果执行     │
    │   │    → DoProcessTargetContainer()  // 处理目标    │
    │   │    → _handle_finish_phase()                     │
    │   │    → finish()                  // 同帧完成       │
    │   └──────────────────────────────────────────────────┘
    │
    ├── ┌─ 路径 B: 瞬发法术 (m_casttime==0, GENERIC) ───┐
    │   │  SetCurrentCastSpell(this)                      │
    │   │  SendSpellStart()                               │
    │   │  TriggerGlobalCooldown()                        │
    │   │  cast(true) → _cast(true)                      │
    │   │   → m_delayMoment==0 ? handle_immediate() : 延迟│
    │   │    → 同上，同帧完成                              │
    │   └──────────────────────────────────────────────────┘
    │
    └── ┌─ 路径 C: 读条法术 (m_casttime>0) ─────────────┐
        │  SetCurrentCastSpell(this)                      │
        │  SendSpellStart()                               │
        │  TriggerGlobalCooldown()                        │
        │  // 不调用 cast()！                              │
        │  // SpellEvent 在每帧 Spell::update() 中：     │
        │  //   PREPARING: m_timer -= difftime           │
        │  //   timer==0 → cast(!m_casttime=false)       │
        │  //     → _cast(skipCheck=false)               │
        │  //     → m_delayMoment > 0 ? 延迟处理 :       │
        │  //       handle_immediate()                    │
        └──────────────────────────────────────────────────┘
```

### 2.3 ownedAuras 与 appliedAuras 数据流

```
施法者 (Caster)                          目标 (Target)
═══════════════                          ═══════════════

m_ownedAuras                             m_appliedAuras
┌──────────────┐                         ┌──────────────────┐
│ spellId → Aura ───────────────────────→│ spellId → AuraApp│
│              │    Aura 创建时产生       │  (AuraApplication)│
│  Aura 对象   │    多个 Application     │                  │
│  - 持续时间  │                         │  - 可见性状态     │
│  - 周期 tick │                         │  - 效果数值       │
│  - 效果列表  │                         │  - 栈数           │
└──────────────┘                         └──────────────────┘
       │                                        │
       ▼                                        ▼
  _UpdateSpells()                         _ApplyAura() / _UnapplyAura()
  → Aura::UpdateOwner()                   → 效果注册/移除
  → Aura::Update()                        → 客户端同步
  → AuraEffect::Update()

所有权关系:
  - Aura 的生命周期由 owner 的 _UpdateSpells 驱动
  - AuraApplication 的注册/注销由 Aura 自身管理
  - 同一个 Aura 可能对应多个 Application (多目标)
  - 同一个目标可能有多个 Application (不同来源的同 ID buff)
```

### 2.4 SpellEvent 生命周期（延迟法术）

```
Spell::prepare()
  → new SpellEvent(this)
  → m_caster->m_Events.AddEvent(_spellEvent, +1ms)

每帧 WorldObject::Update():
  → m_Events.Update(diff)
    → SpellEvent::Execute(e_time, p_time)
      → Spell::update(p_time)

SpellEvent::Execute 状态机:
┌─────────────────────────────────────────────────────────┐
│ PREPARING:                                               │
│   timer 倒数 → 0 时调用 cast()                          │
│   → cast() 内部调用 _cast()                             │
│     → 若 m_delayMoment > 0 → m_spellState = LAUNCHED   │
│     → 若 m_delayMoment == 0 → handle_immediate() + finish│
├─────────────────────────────────────────────────────────┤
│ LAUNCHED (有弹道延迟):                                   │
│   首次: SetDelayStart(e_time)                           │
│         handle_delayed(0) → 计算 m_delayMoment          │
│         重新调度到 delayMoment 时刻                       │
│   后续: handle_delayed(t_offset) → 处理命中目标          │
│         若还有未命中目标 → 重新调度                       │
│         全部命中 → finish()                              │
├─────────────────────────────────────────────────────────┤
│ FINISHED:                                                │
│   若 IsDeletable() → 返回 true (事件完成，删除)         │
│   否则 → 保留事件，下帧再检查                            │
└─────────────────────────────────────────────────────────┘

未完成的事件自动重新注册:
  m_caster->m_Events.AddEvent(this, e_time + 1ms, false)
```

---

## 3. 关键设计决策

### 决策 1: Unit-centric 所有权（非全局管理器）

**决策**: 每个Unit自己持有 `m_currentSpells`、`m_ownedAuras`、`m_appliedAuras`、`_spellHistory`，而不是用一个全局的 SpellManager / AuraManager 来管理所有实体的法术和光环。

**原因**:
- **生命周期绑定**: 当 Unit 离开世界（死亡、传送、销毁），其所有法术和光环自然清理，不需要全局管理器做散列清理。
- **内存局部性**: 每个 Unit 的法术/光环在内存上属于同一实体，减少跨实体的散列查找。
- **并发安全**: 不同 Unit 的法术更新天然隔离（TC 是单线程 Map 更新，但设计上为并行化留下了空间）。
- **简化所有权**: 清晰知道"谁拥有什么"，Aura 的 owner 就是创建它的 Unit。

### 决策 2: owned vs applied 分离

**决策**: Aura 和 AuraApplication 分成两个独立的容器挂在 Unit 上。

**原因**:
- **多目标 Aura**: 一个 AoE buff（如 PAL 的 devotion aura）owner 是施法者自己，但 Application 分布在多个队友身上。
- **单侧更新**: owner 的 `_UpdateSpells` 驱动 Aura 的 timer tick；Application 只在目标身上做效果注册/数值计算。
- **栈/叠加逻辑**: 叠加判断在 Application 侧（目标侧），而非 Owner 侧。两个不同施法者的同名 buff 在同一目标上通过 Application 的栈逻辑处理。
- **安全移除**: Owner 离开世界时清理 ownedAuras → 所有 Application 被级联移除，不需要遍历所有目标。

### 决策 3: 瞬发法术同帧执行（确定性）

**决策**: 路径 A（`TRIGGERED_CAST_DIRECTLY`）和路径 B（普通瞬发）都在 `prepare()` 调用栈内直接完成 `cast()` → `handle_immediate()` → `finish()`，不跨帧。

**原因**:
- **原子性**: 触发法术（如 buff 触发的伤害）在触发点立即完成，保证效果与触发原因在同一逻辑帧内不可分割。
- **确定性**: 瞬发法术不需要担心跨帧状态变化（目标移动、死亡等），CheckCast 的结果与执行在同一帧一致。
- **减少事件队列压力**: 大量触发法术如果都走事件队列，会导致队列膨胀和调度开销。

### 决策 4: SpellEvent 用于延迟法术处理

**决策**: 所有法术在 `prepare()` 时都会创建 `SpellEvent` 并注册到 `m_caster->m_Events`（Spell.cpp:3450-3451）。但对于瞬发法术，`cast()` 在 `prepare()` 内同步执行并 `finish()`，所以 SpellEvent 后续执行时发现状态已经是 FINISHED 就直接清理。

**原因**:
- **统一的生命周期管理**: 不需要区分"有事件"和"无事件"的法术，所有 Spell 对象都通过 SpellEvent 管理其清理。
- **读条法术**: SpellEvent 每帧被触发（+1ms 间隔），驱动 `Spell::update()` 中的 timer 倒数。
- **弹道延迟**: `m_delayMoment` 分支（Spell.cpp:3908）将法术状态切换为 LAUNCHED，SpellEvent 重新调度到命中时刻。
- **清理保障**: SpellEvent 的析构函数会 cancel 未完成的法术，防止内存泄漏。

### 决策 5: _UpdateSpells 清理顺序

**决策**: 在 `_UpdateSpells()` 中（Unit.cpp:2952），先清理 FINISHED 法术指针，再更新 ownedAuras。

**原因**:
- **避免悬挂指针**: 如果先更新 Aura（可能触发新的法术），再清理旧法术指针，m_currentSpells 中可能残留已 finish 的 Spell 指针。
- **事件优先**: 代码注释（Unit.cpp:425）明确说明 Spells 必须在 event system 中处理完 BEFORE 进入 _UpdateSpells。这保证了当一个法术在事件系统中 finish 后，_UpdateSpells 中能正确清理。
- **Aura 触发安全**: Aura 的 tick 可能触发新法术。如果旧法术指针未清理，新法术的 SetCurrentCastSpell 可能被旧指针干扰。

---

## 4. 可借鉴的通用模式

### 模式 1: Entity-centric Update（实体驱动更新）

**模式描述**: 每个 Entity（Unit）持有自己的子系统（Spells、Auras、History），由上层调度器（Map）统一调用 `Update(diff)`，Entity 内部按固定顺序驱动子系统更新。

**适用场景**:
- 游戏实体内部有多个需要协同更新的子系统
- 子系统之间有顺序依赖（法术 → 光环 → 战斗 → 移动）
- 需要确定性更新顺序

**TC 实现**:
```
Map::Update → 遍历所有 Player/Creature
  → Unit::Update
    → WorldObject::Update (事件队列)
    → _UpdateSpells (法术 + 光环)
    → CombatManager::Update
    → AttackTimers
    → Movement
```

**简化替代**: 如果不需要复杂的子系统和严格的更新顺序，可以用 Component 模式，每个 Component 独立注册到全局调度器。但当子系统之间有强依赖（法术必须在光环前更新），Entity-centric 更可靠。

### 模式 2: Owned vs Applied 分离（拥有 vs 施加）

**模式描述**: 将"效果的创建者"和"效果的承受者"分离成两个数据结构，分别在各自的 Entity 上管理。

**适用场景**:
- 一个效果可能同时影响多个目标（AoE buff/debuff）
- 效果的生命周期由创建者驱动（如施法者死亡 → buff 消失）
- 需要按创建者查询效果（"我给谁加了什么 buff"）
- 需要按承受者查询效果（"我身上有什么 debuff"）

**TC 实现**:
- `Aura` (owned, on caster): 管理持续时间、周期 tick、效果列表
- `AuraApplication` (applied, on target): 管理可见性、效果数值、栈数

**简化替代**: 如果效果总是单目标、不需要按创建者查询，可以用单一结构同时存储在目标上。但如果需要支持 AoE 效果或创建者驱动的生命周期，Owned/Applied 分离是更安全的选择。

### 模式 3: 同帧 vs 延迟执行策略（Same-frame vs Deferred Execution）

**模式描述**: 根据操作的特性选择同步执行（在调用栈内直接完成）或延迟执行（注册到事件队列，由 Update 链驱动）。

**适用场景**:
- 瞬时效果（瞬发法术、触发效果）→ 同帧执行，保证原子性和确定性
- 持续效果（读条法术、弹道飞行、周期 tick）→ 延迟执行，由 Update 铱驱动 timer
- 混合场景：同一次调用中可能产生链式触发（A 触发 B，B 触发 C），全部在同帧完成

**TC 实现**:
- `TRIGGERED_CAST_DIRECTLY`: 完全跳过注册，直接执行
- 普通瞬发: 注册但不等 Update，prepare 内直接 cast(true)
- 读条/弹道: 注册到 SpellEvent，每帧 Spell::update 推进 timer

**简化替代**: 如果游戏不需要读条或弹道延迟，所有法术都可以走同帧路径。但保留延迟机制的设计空间很重要——一旦需要加入弹道或读条，重构成本很高。

### 模式 4: 事件队列作为统一调度器

**模式描述**: `m_Events`（EventProcessor）挂在 WorldObject 上，SpellEvent / Aura 触发等都通过事件队列调度。每帧 `m_Events.Update(diff)` 处理到期事件。

**适用场景**:
- 需要精确的延迟调度（"3 秒后触发"、"命中时刻 = 当前 + 弹道时间"）
- 多个异步流程需要统一调度
- 需要在同一帧内按时间顺序处理事件

**TC 实现**:
- SpellEvent 注册时 `AddEvent(this, +1ms)` → 下一帧就被执行
- 弹道延迟: `AddEvent(this, delayStart + m_delayMoment)` → 精确到毫秒的延迟
- 每帧 SpellEvent 末尾自动重新注册 `AddEvent(this, e_time + 1)`，直到法术完成

**简化替代**: 简单的 tick-based 系统（每个 tick 固定步进，不需要亚 tick 精度）可以用固定的 update loop 替代事件队列。但事件队列的优势在于可以处理"在 tick 中间的某个精确时刻触发"的场景。

### 模式 5: 防御性清理（Defensive Cleanup）

**模式描述**: 在 Update 循环中使用 iterator 安全模式，先递增迭代器再操作当前元素，防止迭代中修改容器导致崩溃。

**TC 实现**（Unit.cpp:2971-2976）:
```cpp
for (m_auraUpdateIterator = m_ownedAuras.begin(); m_auraUpdateIterator != m_ownedAuras.end();)
{
    Aura* i_aura = m_auraUpdateIterator->second;
    ++m_auraUpdateIterator;        // 先递增！
    i_aura->UpdateOwner(time, this);  // UpdateOwner 可能会移除其他 Aura
}
```

`m_auraUpdateIterator` 是成员变量，这样在 `UpdateOwner` 间接调用的 `RemoveOwnedAura` 中可以跳过即将被移除的元素，保证迭代安全。

**通用原则**: 任何在遍历中可能修改容器的场景都需要这种防御模式——成员变量记录迭代位置，先递增再操作。
