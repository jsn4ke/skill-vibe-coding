# Effect Pipeline (效果管线)

> Source: TrinityCore | Generated: 2026-04-28 | Topic: effect pipeline, handle mode, launch phase, hit phase, effect handler

## 1. 核心数据结构

### SpellEffectHandleMode 枚举

| Mode | 含义 | 目标 |
|------|------|------|
| `SPELL_EFFECT_HANDLE_LAUNCH` | 发射阶段（无目标） | 目的地/地面效果 |
| `SPELL_EFFECT_HANDLE_LAUNCH_TARGET` | 发射阶段（有目标） | 每个被选中的目标 |
| `SPELL_EFFECT_HANDLE_HIT` | 命中阶段（无目标） | 目的地/地面效果 |
| `SPELL_EFFECT_HANDLE_HIT_TARGET` | 命中阶段（有目标） | 每个被命中的目标 |

### HandleEffects 函数签名

```cpp
void Spell::HandleEffects(
    Unit* pUnitTarget,
    Item* pItemTarget,
    GameObject* pGoTarget,
    Corpse* pCorpseTarget,
    SpellEffectInfo const& spellEffectInfo,
    SpellEffectHandleMode mode
);
```

- 每次调用处理**一个** effect + **一个** mode
- target 参数在 LAUNCH/HIT 模式为 nullptr（无目标），在 LAUNCH_TARGET/HIT_TARGET 模式传入具体 Unit
- 内部统一模板：设置上下文 → 调用脚本钩子 → 如果未 PreventDefault 则执行默认处理

### Spell 脚本钩子与 Mode 的映射

| Mode | 对应的 HookList | 脚本 Hook 类型枚举 |
|------|----------------|-------------------|
| `LAUNCH` | `OnEffectLaunch` | `SPELL_SCRIPT_HOOK_EFFECT_LAUNCH` |
| `LAUNCH_TARGET` | `OnEffectLaunchTarget` | `SPELL_SCRIPT_HOOK_EFFECT_LAUNCH_TARGET` |
| `HIT` | `OnEffectHit` | `SPELL_SCRIPT_HOOK_EFFECT_HIT` |
| `HIT_TARGET` | `OnEffectHitTarget` | `SPELL_SCRIPT_HOOK_EFFECT_HIT_TARGET` |

## 2. 流程图

### 即时法术（instant / cast time=0）

```
Spell::cast()
│
├── HandleLaunchPhase()
│   │
│   ├─ LAUNCH (无目标):
│   │  for each effect:
│   │    HandleEffects(nullptr, eff, LAUNCH)
│   │    → CallScriptHook(effIdx, LAUNCH)    ← OnEffectLaunch
│   │    → 默认处理
│   │
│   ├─ PreprocessSpellLaunch()
│   │  → 对每个 target 计算暴击/免疫
│   │
│   └─ LAUNCH_TARGET (每目标):
│      for each target, each effect:
│        HandleEffects(unit, eff, LAUNCH_TARGET)
│        → CallScriptHook(effIdx, LAUNCH_TARGET)  ← OnEffectLaunchTarget
│        → 默认处理
│        → damage/healing 累加到 targetInfo
│        → AoE 衰减计算
│
├── handle_immediate()
│   │
│   ├─ _handle_immediate_phase()
│   │  for each effect:
│   │    HandleEffects(nullptr, eff, HIT)
│   │    → CallScriptHook(effIdx, HIT)      ← OnEffectHit
│   │    → 默认处理
│   │
│   └─ DoProcessTargetContainer()
│      for each target:
│        ├─ DoTargetSpellHit()              ← 处理每个目标
│        │  for each effect:
│        │    HandleEffects(unit, eff, HIT_TARGET)
│        │    → CallScriptHook(effIdx, HIT_TARGET)  ← OnEffectHitTarget
│        │    → 默认处理
│        │
│        └─ DoDamageAndTriggers()
│           → 施加伤害/治疗
│           → 触发 Proc
│
└── finish(SPELL_CAST_OK)
```

### 弹道法术（speed > 0）

```
Spell::cast()
│
├── HandleLaunchPhase()          ← 发射时刻
│   └─ (同上: LAUNCH + LAUNCH_TARGET)
│
├── state = LAUNCHED             ← 等待弹道飞行
│
└── handle_delayed(t_offset)     ← 弹道到达时
    │
    ├─ _handle_immediate_phase()
    │  └─ HIT (无目标)
    │
    └─ DoProcessTargetContainer()
       └─ HIT_TARGET (每目标)
```

### 引导法术（channeled）

```
Spell::cast()
│
├── HandleLaunchPhase()          ← 引导开始
│   └─ LAUNCH + LAUNCH_TARGET
│
├── ProcessEffects()             ← 引导入口：立即处理效果
│   └─ (调用 handle_immediate 的 HIT 阶段)
│
└── state = CHANNELING           ← 引导持续，由 Update 驱动
```

### HandleEffects 内部执行模板（所有 mode 统一）

```cpp
void Spell::HandleEffects(target..., effInfo, mode) {
    // 1. 设置上下文
    effectHandleMode = mode;
    unitTarget = target;
    effectInfo = &effInfo;

    // 2. 计算基础伤害
    damage = CalculateDamage(effInfo, unitTarget, &variance);

    // 3. 调用脚本钩子（先于默认处理）
    bool preventDefault = CallScriptEffectHandlers(effIndex, mode);

    // 4. 如果脚本没有阻止，执行默认处理
    if (!preventDefault)
        (this->*SpellEffectHandlers[effect])();
}
```

## 3. 关键设计决策

### 决策: 每个 Effect Handler 内部用 mode guard 过滤

**原因**: Handler 被 4 个阶段各调用一次（共 4 次），但只在匹配的 mode 才执行。这避免了在分发层维护 mode→effect 的映射表，每个 effect 自己声明自己只在哪个 mode 工作。

**模式**: 每个 handler 的第一行几乎总是 `if (effectHandleMode != X) return;`

### 决策: LAUNCH_TARGET = 计算，HIT_TARGET = 施加

**原因**: 弹道法术需要在发射时确定伤害值（服务端计算），但等到命中时才实际施加效果（Aura、能量恢复等）。两阶段分离支持弹道延迟。

**含义**:
- `EffectSchoolDMG` 在 `LAUNCH_TARGET` 计算伤害 → 存到 `targetInfo.Damage`
- `EffectApplyAura` 在 `HIT_TARGET` 才真正创建并应用 Aura
- 弹道飞行期间伤害值已经确定，但效果尚未施加

### 决策: 脚本拦截先于默认处理（Prevent Pattern）

**原因**: 脚本可以通过 `PreventHitDefaultEffect(effIndex)` 阻止引擎默认处理，完全自定义该 effect 的行为。Dummy effect 就是依赖这个模式——引擎默认处理几乎为空，真正的逻辑由脚本注入。

### 决策: 脚本钩子按 mode 分发到不同的 HookList

**原因**: 同一个法术可能需要在发射和命中时执行不同的脚本逻辑。4 个独立的 HookList 允许脚本精确注册到需要的阶段。例如火球术可能在 Launch 时记录日志，在 HitTarget 时触发点燃。

### 决策: Damage/Healing 在 LAUNCH_TARGET 阶段累加到 TargetInfo

**原因**: Launch 阶段计算的结果存储在 `targetInfo.Damage` / `targetInfo.Healing`，不直接施加。Hit 阶段从 TargetInfo 读取并施加。这允许 Launch 阶段对多个 target 的伤害进行修正（如 AoE 衰减、目标数量上限），然后 Hit 阶段施加最终值。

## 4. EffectType → Mode 映射表

### LAUNCH_TARGET（发射时计算，每目标）

| EffectType | 说明 |
|-----------|------|
| `EffectSchoolDMG` | 计算法术伤害，累加到 m_damage |
| `EffectWeaponDamage` / `EffectWeaponPercentDamage` / `EffectNormalizedWeaponDamage` | 武器伤害计算 |
| `EffectHeal` | 治疗值计算 |
| `EffectTriggerSpell` | 在发射时触发子法术 |
| `EffectCharge` | 启动冲锋移动（路径生成 + MoveCharge） |
| `EffectEnergizePct`（部分变体） | 百分比能量恢复计算 |

### LAUNCH（发射时处理，无目标）

| EffectType | 说明 |
|-----------|------|
| `EffectSummonType` | 创建召唤物 |
| `EffectTriggerSpell`（无目标触发） | 触发无目标的子法术 |
| `EffectLeapBack` | 后跳发射 |
| 各种 `EffectSummon*` | 召唤类效果在发射时创建 |

### HIT_TARGET（命中时施加，每目标）

| EffectType | 说明 |
|-----------|------|
| `EffectDummy` | 脚本钩子挂载点（主要入口） |
| `EffectApplyAura` | 创建并应用 Aura 到目标 |
| `EffectEnergize` | 恢复目标能量 |
| `EffectDispel` | 驱散目标 Buff |
| `EffectTeleportUnits` | 传送目标 |
| `EffectLeap` | 跳跃目标到目标位置 |
| `EffectKnockBack` | 击退目标 |
| `EffectCharge`（攻击部分） | 冲锋到达后发起攻击 + 触发子法术 |
| `EffectInstakill` | 即死效果 |
| `EffectAddComboPoints` | 增加连击点 |
| `EffectResurrect` | 复活目标 |
| `EffectInterruptCast` | 打断目标施法 |
| `EffectPullTowards` | 拉向目标 |

### HIT（命中时处理，无目标）

| EffectType | 说明 |
|-----------|------|
| 持续区域效果 | 地面 AoE 目的地的处理 |
| `EffectCreateAreaTrigger` | 创建区域触发器 |
| `EffectCreateDynamicObject` | 创建动态对象 |

### 多 Mode 效果（特殊）

| EffectType | Mode 行为 | 说明 |
|-----------|----------|------|
| `EffectCharge` | `LAUNCH_TARGET`: 移动; `HIT_TARGET`: 攻击 | 冲锋分两阶段：先移动后攻击 |
| `EffectTriggerSpell` | `LAUNCH_TARGET` + `LAUNCH` | 两种发射模式都接受 |
| `EffectWeaponDamage` + `SPELL_ATTR0_CU_SHARE_DAMAGE` | `LAUNCH_TARGET` | 伤害均分到所有目标 |

## 5. 可借鉴的通用模式

### 模式: Handler 内部 mode guard（自选模式）

Handler 不依赖外部路由，自己决定在哪个 mode 工作。系统在所有 4 个 mode 都调用 handler，handler 用 `if (mode != X) return` 过滤。

**优点**: 新增 effect 不需要修改分发层代码，只需在 handler 内部声明 mode。
**适用场景**: 同类处理函数数量大、每个函数有独立的 phase 需求。

### 模式: 计算-施加两阶段分离

LAUNCH_TARGET 阶段计算数值并存入中间存储，HIT_TARGET 阶段从中间存储读取并施加。中间间隔允许弹道飞行期间的修正。

**适用场景**: 有延迟的投射物系统、需要在施加前做全局修正（如 AoE 伤害上限）。

### 模式: 统一调用模板 + 脚本拦截

所有 mode 走同一个 HandleEffects 模板：设置上下文 → 调用脚本 → 默认处理。脚本先于默认处理执行，可以阻止默认行为。

**适用场景**: 需要可扩展性的系统，允许外部代码替代内置行为。
