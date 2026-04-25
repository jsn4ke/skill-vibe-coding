# Spell Script System (法术脚本系统)

> Source: TrinityCore | Generated: 2026-04-25 | Topic: spell script, hook, dummy, PreventDefault

## 1. 核心数据结构

### SpellScriptHookType (SpellScript 枚举)

定义了 SpellScript 可注册的所有 Hook 类型，每个枚举值对应法术生命周期中的一个拦截点：

| Hook 类型 | 对应阶段 | 绑定的 HookList | Handler 签名 |
|-----------|---------|----------------|-------------|
| `SPELL_SCRIPT_HOOK_ON_PRECAST` | 施法准备阶段 | `OnPrecast()` 虚函数覆写 | `void OnPrecast()` |
| `SPELL_SCRIPT_HOOK_BEFORE_CAST` | 施法条满，开始处理前 | `BeforeCast` | `void HandleCast()` |
| `SPELL_SCRIPT_HOOK_CHECK_CAST` | 施法条件检查 | `OnCheckCast` | `SpellCastResult CheckCast()` |
| `SPELL_SCRIPT_HOOK_OBJECT_AREA_TARGET_SELECT` | 区域目标选择 | `OnObjectAreaTargetSelect` | `void SetTargets(list<WO*>&)` |
| `SPELL_SCRIPT_HOOK_OBJECT_TARGET_SELECT` | 单体目标选择 | `OnObjectTargetSelect` | `void SetTarget(WO*&)` |
| `SPELL_SCRIPT_HOOK_DESTINATION_TARGET_SELECT` | 目的地选择 | `OnDestinationTargetSelect` | `void SetTarget(SpellDest&)` |
| `SPELL_SCRIPT_HOOK_ON_CAST` | 法术发射前 | `OnCast` | `void HandleCast()` |
| `SPELL_SCRIPT_HOOK_AFTER_CAST` | 法术发射后 | `AfterCast` | `void HandleCast()` |
| `SPELL_SCRIPT_HOOK_EFFECT_LAUNCH` | 效果发射时 (无目标) | `OnEffectLaunch` | `void HandleEffect(SpellEffIndex)` |
| `SPELL_SCRIPT_HOOK_EFFECT_LAUNCH_TARGET` | 效果发射时 (有目标) | `OnEffectLaunchTarget` | `void HandleEffect(SpellEffIndex)` |
| `SPELL_SCRIPT_HOOK_CALC_CRIT_CHANCE` | 暴击率计算 | `OnCalcCritChance` | `void CalcCrit(Unit*, float&)` |
| `SPELL_SCRIPT_HOOK_CALC_DAMAGE` | 伤害计算 | `CalcDamage` | `void CalcDmg(..., int32&, ...)` |
| `SPELL_SCRIPT_HOOK_CALC_HEALING` | 治疗计算 | `CalcHealing` | `void CalcHeal(..., int32&, ...)` |
| `SPELL_SCRIPT_HOOK_ON_RESIST_ABSORB_CALCULATION` | 抗性/吸收计算 | `OnCalculateResistAbsorb` | `void CalcAbsorb(...)` |
| `SPELL_SCRIPT_HOOK_EFFECT_HIT` | 效果命中目的地 | `OnEffectHit` | `void HandleEffect(SpellEffIndex)` |
| `SPELL_SCRIPT_HOOK_BEFORE_HIT` | 命中目标前 | `BeforeHit` | `void HandleBeforeHit(SpellMissInfo)` |
| `SPELL_SCRIPT_HOOK_EFFECT_HIT_TARGET` | 效果命中目标 | `OnEffectHitTarget` | `void HandleEffect(SpellEffIndex)` |
| `SPELL_SCRIPT_HOOK_HIT` | 命中处理 | `OnHit` | `void HandleHit()` |
| `SPELL_SCRIPT_HOOK_AFTER_HIT` | 命中完成后 | `AfterHit` | `void HandleHit()` |
| `SPELL_SCRIPT_HOOK_EMPOWER_STAGE_COMPLETED` | 蓄力阶段完成 | `OnEmpowerStageCompleted` | `void Handle(int32)` |
| `SPELL_SCRIPT_HOOK_EMPOWER_COMPLETED` | 蓄力完成 | `OnEmpowerCompleted` | `void Handle(int32)` |

### AuraScriptHookType (AuraScript 枚举)

Aura 生命周期相关 Hook，覆盖 Apply/Remove/Periodic/Proc/Absorb 等阶段。主要类别：

- **生命周期**: `EFFECT_APPLY`, `EFFECT_AFTER_APPLY`, `EFFECT_REMOVE`, `EFFECT_AFTER_REMOVE`
- **周期性**: `EFFECT_PERIODIC`, `EFFECT_UPDATE_PERIODIC`
- **计算类**: `EFFECT_CALC_AMOUNT`, `EFFECT_CALC_PERIODIC`, `EFFECT_CALC_SPELLMOD`, `EFFECT_CALC_CRIT_CHANCE`, `EFFECT_CALC_DAMAGE_AND_HEALING`
- **吸收类**: `EFFECT_ABSORB`, `EFFECT_AFTER_ABSORB`, `EFFECT_MANASHIELD`, `EFFECT_AFTER_MANASHIELD`, `EFFECT_SPLIT`
- **Proc**: `CHECK_PROC`, `CHECK_EFFECT_PROC`, `PREPARE_PROC`, `PROC`, `EFFECT_PROC`, `EFFECT_AFTER_PROC`, `AFTER_PROC`
- **区域/战斗**: `CHECK_AREA_TARGET`, `DISPEL`, `AFTER_DISPEL`, `ON_HEARTBEAT`, `ENTER_LEAVE_COMBAT`

### EffectHook / EffectBase / EffectHandler

```
EffectHook (SpellScriptBase 内部基类)
  |- _effIndex: uint8  (EFFECT_0/1/2, EFFECT_ALL, EFFECT_FIRST_FOUND)
  |- GetAffectedEffectsMask(spellInfo) -> uint32  // 位掩码，哪些 effect index 匹配
  |- CheckEffect(spellInfo, effIndex) -> bool     // 纯虚函数，子类决定匹配规则
  |
  +- EffectBase (SpellScript::EffectBase)
  |    |- _effName: uint16  (SPELL_EFFECT_DUMMY, SPELL_EFFECT_ANY 等)
  |    |- CheckEffect(): 比较 spellInfo.Effect == _effName
  |
  +- EffectBase (AuraScript::EffectBase)
  |    |- _auraType: uint16  (SPELL_AURA_ANY 或具体 aura type)
  |    |- CheckEffect(): 比较 spellInfo.ApplyAuraName == _auraType
  |
  +- TargetHook (SpellScript::TargetHook)
       |- _targetType, _area, _dest
       |- CheckEffect(): 比较 TargetA/B 与 _targetType
```

**关键**：EffectHandler 构造时同时传入 `(handler, effIndex, effName)`，注册时即绑定到**特定的 effect index + effect type**。运行时只对匹配的 effect 触发回调。

### HookList\<T\> 与 operator+=

```cpp
// 在脚本 Register() 中使用 += 注册 handler
HookList<EffectHandler> OnEffectHitTarget;
// 用法：
OnEffectHitTarget += SpellEffectFn(MyScript::HandleHitTarget, EFFECT_0, SPELL_EFFECT_DUMMY);
```

`HookList` 继承自 `::HookList<T>`，底层是 handler 对象的容器。`operator+=` 在 `SPELL_SCRIPT_STATE_REGISTRATION` 状态下将 handler 追加到列表。宏 (`SpellEffectFn`, `SpellCastFn` 等) 构造对应 Handler 类型并传入函数指针 + 效果匹配参数。

### ScriptFuncInvoker (类型擦除机制)

TC 使用 thunk-based 类型擦除来存储任意成员函数指针或静态函数：
- `ImplStorage`: 存储 `ScriptFunc` 的大小/对齐
- `Thunk`: 函数指针，负责将 `BaseClass&` `static_cast` 为 `ScriptClass&` 后调用 `ScriptFunc`
- 编译期 `static_assert` 校验签名正确性

### m_hitPreventEffectMask / m_hitPreventDefaultEffectMask

SpellScript 内部的两个 bitmask，用于 `PreventHitEffect` / `PreventHitDefaultEffect`：
- `PreventHitEffect(effIndex)`: 阻止该 effect 的**所有** handler（包括后续脚本和默认处理）
- `PreventHitDefaultEffect(effIndex)`: 仅阻止引擎默认处理，脚本 handler 仍执行

### SpellScriptState

脚本内部状态机：`NONE` -> `REGISTRATION` -> `LOADING` -> 运行时 Hook -> `UNLOADING`

---

## 2. 流程图

```
                          法术施放完整 Hook 流程
                          ========================

 Player::CastSpell / Unit::CastSpell
          |
          v
    Spell::Prepare()                   Spell 构造时调用
          |                             LoadScripts()
          |                               -> sScriptMgr->CreateSpellScripts(spellId, m_loadedScripts, this)
          |                               -> 每个 script->_Register() -> 调用 script->Register()
          |                                  (Register() 中通过 += 注册所有 handler)
          |
          v
 [1] OnPrecast()                       脚本虚函数覆写，施法准备阶段
          |
          v
 [2] BeforeCast handlers               HookList<CastHandler> BeforeCast
          |
          v
 [3] OnCheckCast handlers              HookList<CheckCastHandler> -> 返回 SpellCastResult
          |                               第一个非 OK 的结果被保留
          v
 [4] 目标选择阶段                       对每个 effect 的每个 implicit target:
     OnObjectAreaTargetSelect              CallScriptObjectAreaTargetSelectHandlers()
     OnObjectTargetSelect                  CallScriptObjectTargetSelectHandlers()
     OnDestinationTargetSelect             CallScriptDestinationTargetSelectHandlers()
          |
          v
 [5] OnCast handlers                    HookList<CastHandler> OnCast
          |
          v
 [6] AfterCast handlers                 HookList<CastHandler> AfterCast
          |
          v
 [7] 发射阶段 (对每个 effect)           mode = SPELL_EFFECT_HANDLE_LAUNCH / LAUNCH_TARGET
     OnEffectLaunch                      CallScriptEffectHandlers(effIndex, LAUNCH)
     OnEffectLaunchTarget                CallScriptEffectHandlers(effIndex, LAUNCH_TARGET)
          |
          v
 [8] CalcCritChance / CalcDamage / CalcHealing
          |
          v
 [9] 命中阶段 (对每个目标每个 effect)   HandleEffects() 被调用
          |
          +---> [9a] CallScriptEffectHandlers(effIndex, HIT)          OnEffectHit
          |         |
          |         +---> preventDefault = false?
          |         |       YES -> SpellEffectHandlers[effect](this)  引擎默认处理
          |         |
          |         +---> CallScriptBeforeHitHandlers(missInfo)       BeforeHit
          |         |
          |         +---> CallScriptEffectHandlers(effIndex, HIT_TARGET)  OnEffectHitTarget
          |         |         |
          |         |         +---> preventDefault = false?
          |         |         |       YES -> SpellEffectHandlers[effect](this)  引擎默认处理
          |         |         |
          |         |         |  *** Dummy 效果模式 (SPELL_EFFECT_DUMMY) ***
          |         |         |  SpellEffectHandlers[3] = EffectDummy()
          |         |         |  EffectDummy 本身几乎为空:
          |         |         |    - 检查 PetAura (特殊逻辑)
          |         |         |    - 触发 DB 脚本 ScriptsStart()
          |         |         |  真正的逻辑由脚本的 OnEffectHitTarget handler 实现
          |         |         |
          |         +---> CallScriptOnHitHandlers()                   OnHit
          |         |
          |         +---> CallScriptAfterHitHandlers()                AfterHit
          |
          v
     Spell 析构
          |
          +---> 每个 script->_Unload() -> script->Unload()
          +---> delete script
```

### Dummy Effect 拦截模式详图

```
  DBC 数据定义:                        SpellScript 注册:
  Spell 133 (Fire Blast)               class spell_fire_blast : public SpellScript
    Effect[0] = SPELL_EFFECT_DUMMY       {
    Effect[0].BasePoints = 0               void Register() override {
    (EffectDummy 本身不做有意义操作)          OnEffectHitTarget +=
                                                SpellEffectFn(HandleOnHitTarget,
                                                  EFFECT_0, SPELL_EFFECT_DUMMY);
                                           }
                                           void HandleOnHitTarget(SpellEffIndex) {
                                             // 这里写真正的伤害/效果逻辑
                                             int32 damage = ...;
                                             GetHitUnit()->DealDamage(GetCaster(), damage);
                                           }
                                         };

  运行时:
  HandleEffects(unit, item, go, corpse, effectInfo, SPELL_EFFECT_HANDLE_HIT_TARGET)
    |
    +-> CallScriptEffectHandlers(EFFECT_0, HIT_TARGET)
    |     |
    |     +-> 对每个 script->OnEffectHitTarget 中的 handler:
    |           if (handler.IsEffectAffected(spellInfo, EFFECT_0))  // _effName == DUMMY ✓
    |             handler.Call(script, EFFECT_0)
    |               -> spell_fire_blast::HandleOnHitTarget(EFFECT_0)
    |               -> 如果脚本调用了 PreventHitDefaultEffect(EFFECT_0):
    |                   m_hitPreventDefaultEffectMask |= (1 << 0)
    |
    +-> preventDefault = script->_IsDefaultEffectPrevented(EFFECT_0)
    |
    +-> if (!preventDefault)
          (this->*SpellEffectHandlers[3])()   // -> EffectDummy()
          // EffectDummy 只做 DB 脚本启动，不产生实际效果
```

---

## 3. 关键设计决策

### 3.1 Dummy Effect 作为 Hook 挂载点

**决策**: DBC 中大量法术使用 `SPELL_EFFECT_DUMMY` (枚举值 3) 作为 effect type，引擎的 `EffectDummy()` 处理函数几乎为空（仅处理 PetAura 和触发 DB 脚本）。

**原因**:
- Dummy 是一个 "Marker Effect"——它在 DBC 中声明了 "这个 effect 需要脚本处理"，但引擎本身不做任何通用处理
- 脚本通过 `OnEffectHitTarget += SpellEffectFn(Handle, EFFECT_0, SPELL_EFFECT_DUMMY)` 拦截到这个 effect，执行真正的自定义逻辑
- 解耦了数据定义（DBC）和行为实现（C++ 脚本），让 DBA 和程序员可以独立工作
- 同一个法术可以有多个 Dummy effect（EFFECT_0/1/2），每个对应不同的自定义行为

### 3.2 Hook 注册绑定到特定 Effect Type

**决策**: `EffectHandler` 构造时要求传入 `(handler, effIndex, effName)`，运行时通过 `IsEffectAffected()` 检查 DBC 数据是否匹配。

**原因**:
- 类型安全——如果 DBC 中 EFFECT_0 不是 `SPELL_EFFECT_DUMMY`，`_Validate()` 阶段会输出错误日志，handler 不会被执行
- 精确匹配——同一个脚本可以针对不同 effect 注册不同 handler，不会误触发
- 支持通配——`SPELL_EFFECT_ANY` 和 `EFFECT_ALL` 允许注册到任意 effect

### 3.3 PreventHitDefaultEffect vs PreventHitEffect 的区别

**决策**: 提供两个层级的阻止机制。

| 方法 | 行为 | 适用场景 |
|------|------|---------|
| `PreventHitDefaultEffect(effIndex)` | 仅阻止引擎默认处理 (`SpellEffectHandlers[]`) | 脚本想完全自定义该 effect 的行为 |
| `PreventHitEffect(effIndex)` | 阻止**所有**后续处理（包括其他脚本的 handler + 默认处理） | 脚本想完全取消该 effect |

**原因**: 多脚本共存时需要细粒度控制。脚本 A 可能想阻止默认处理但允许脚本 B 运行。`PreventHitDefaultEffect` 是更常用的 API。

**实现**: 两个 bitmask (`m_hitPreventEffectMask`, `m_hitPreventDefaultEffectMask`)。`CallScriptEffectHandlers()` 检查 `_IsEffectPrevented()` 决定是否跳过 handler 调用；`HandleEffects()` 检查 `_IsDefaultEffectPrevented()` 决定是否跳过默认处理。

### 3.4 SpellScript 与 AuraScript 分离

**决策**: 两个独立的脚本基类，各自有自己的 Hook 类型枚举和生命周期。

| | SpellScript | AuraScript |
|---|---|---|
| 生命周期 | 一次施法 | 持续性 Aura |
| 挂载对象 | Spell 实例 | Aura 实例 |
| Hook 数量 | ~20 个 | ~30 个 |
| 特有 Hook | Cast/Hit/Effect/Target | Apply/Remove/Periodic/Proc/Absorb |
| 状态栈 | 无（单层） | 有（ScriptStateStack，支持嵌套调用） |
| PreventDefault | `PreventHitDefaultEffect` | `PreventDefaultAction` |

**原因**: Spell（瞬时行为）和 Aura（持续状态）的生命周期完全不同。Spell 一次 cast 就结束，Aura 有 Apply/Remove/Periodic tick/Proc 等持续事件。分离后各自的 Hook 接口更清晰，状态管理更简单。

### 3.5 Hook 执行顺序（文档化）

**决策**: SpellScript.h 注释中明确记录了 18 步 Hook 执行顺序。

完整顺序:
1. `OnPrecast` - 施法准备阶段（施法条开始前）
2. `BeforeCast` - 施法条满，开始处理前
3. `OnCheckCast` - 施法条件检查
4. `OnObjectAreaTargetSelect` / `OnObjectTargetSelect` / `OnDestinationTargetSelect` - 目标选择
5. `OnCast` - 法术发射前
6. `AfterCast` - 法术发射后
7. `OnEffectLaunch` - 效果发射（无目标）
8. `OnCalcCritChance` - 暴击率计算（每个目标）
9. `OnEffectLaunchTarget` - 效果发射（有目标，每个目标）
10. `CalcDamage` / `CalcHealing` - 伤害/治疗计算
11. `OnCalculateResistAbsorb` - 抗性吸收计算
12. `OnEffectHit` - 效果命中目的地
13. `BeforeHit` - 命中目标前
14. `OnEffectHitTarget` - 效果命中目标（**Dummy 模式的主要 Hook**）
15. `OnHit` - 命中处理（伤害/procs 之前）
16. `AfterHit` - 命中完成后
17. `OnEmpowerStageCompleted` - 蓄力阶段完成
18. `OnEmpowerCompleted` - 蓄力释放

**原因**: 明确的顺序保证让脚本作者知道在哪个阶段可以访问哪些数据（例如 `GetHitUnit()` 只在 target hook 中可用）。`IsInTargetHook()` / `IsInHitPhase()` / `IsInEffectHook()` 等运行时检查强制这个约束。

### 3.6 Validate 阶段的 DBC 兼容性检查

**决策**: `_Validate()` 遍历所有已注册的 handler，调用 `GetAffectedEffectsMask()` 检查 DBC 数据是否匹配。不匹配则输出错误日志但不阻止加载。

**原因**: DBC 数据和脚本代码可能不同步。错误日志帮助开发者在测试阶段发现问题，但不会因为单个脚本错误导致整个服务器崩溃。

---

## 4. 可借鉴的通用模式

### 4.1 Dummy/Marker Effect 模式

**模式**: 在数据定义中使用一个特殊标记（如 `DUMMY`），引擎对标记不做通用处理，由外部脚本/插件拦截并提供行为。

**适用场景**:
- 需要在数据驱动和行为驱动之间解耦时
- 同一系统中需要大量不可预测的自定义行为时
- DBA/策划负责数据定义，程序员负责行为实现的分工模式

**不适用场景**:
- 行为完全可预测、可归类的效果（直接用枚举定义 effect type 更好）
- 不需要扩展性的简单系统

**在非 WoW 项目中的应用**: 技能系统的每个技能可以有 "effects" 数组，其中一个 effect type 是 "SCRIPT"。引擎遇到 SCRIPT 类型时不做处理，而是回调到注册的脚本函数。脚本系统根据技能 ID 查找并执行对应的 handler。

### 4.2 Hook-Based 可扩展性 + 类型安全注册

**模式**: 通过 `HookList<T> += Handler(handler, effIndex, effType)` 注册回调，编译期校验签名，运行时校验数据匹配。

**关键要素**:
1. **类型擦除存储**: `ScriptFuncInvoker` 使用 thunk + storage 联合体，避免虚函数开销
2. **编译期签名检查**: `static_assert(std::is_invocable_r_v<...>)` 确保回调签名正确
3. **运行时数据匹配**: `CheckEffect()` 确保回调只对匹配的 DBC effect 触发
4. **状态守卫**: `_PrepareScriptCall(hookType)` / `_FinishScriptCall()` 设置当前 hook 类型，API 方法内部检查 `IsInXxxHook()` 防止误用

**在非 WoW 项目中的应用**: 设计一个 `SkillScript` 基类，提供 `OnCastStart`, `OnHit`, `OnTick` 等 HookList。技能脚本继承基类，在 `Register()` 中用 `+=` 注册 handler。每个 handler 绑定到特定的 effect index，避免误触发。

### 4.3 脚本拦截先于默认处理 (Prevent Pattern)

**模式**: 引擎在每个处理步骤前先调用脚本，脚本可以通过 `Prevent` 系列方法阻止后续默认处理。

```
处理步骤:
  1. CallScriptXxxHandlers()        <- 脚本先执行
  2. if (!preventDefault)
       DefaultHandler()             <- 默认处理可被跳过
```

**优点**:
- 脚本可以完全替代默认行为
- 脚本可以只在特定条件下阻止默认行为
- 不需要修改引擎代码

**在非 WoW 项目中的应用**: 伤害计算管线中，每个阶段都先调用脚本 hook，脚本可以通过 `PreventDefaultAction` 阻止后续阶段。

### 4.4 何时使用此模式 vs 何时不使用

**使用此模式**:
- 需要 100+ 种不同行为且持续增长的技能系统
- 行为由 C++ 实现（性能要求高）
- 数据定义和行为实现由不同团队负责

**不使用此模式（更简单的方案）**:
- 行为可归类的系统——用策略模式 + 数据驱动即可
- 小型项目——直接在技能定义中用函数指针或闭包
- 脚本语言驱动的系统——用 Lua/Python 回调，不需要类型擦除的复杂性
