# Spell Cast Flow (施法流程)

> Source: TrinityCore | Generated: 2026-04-23 | Topic: spell cast flow, state machine, lifecycle

## 1. 核心数据结构

### SpellState 枚举
| 状态 | 值 | 含义 |
|------|-----|------|
| NULL | 0 | 初始/无状态 |
| PREPARING | 1 | 施法条进行中（有施法时间的法术） |
| LAUNCHED | 2 | 已发射，弹道飞行中/延迟命中 |
| CHANNELING | 3 | 引导中，持续施法 |
| FINISHED | 4 | 完成，即将销毁 |
| IDLE | 5 | 自动射击等待状态 |

### SpellEffectHandleMode 枚举
| 模式 | 含义 |
|------|------|
| LAUNCH | 发射阶段（施法者端） |
| LAUNCH_TARGET | 发射阶段（目标端） |
| HIT | 命中阶段（施法者端） |
| HIT_TARGET | 命中阶段（目标端） |

### Spell 类关键字段
- `m_spellInfo` — 静态法术定义（只读共享）
- `m_spellState` — 当前状态
- `m_caster` — 施法者 WorldObject
- `m_targets` — 目标信息
- `m_timer` — 施法/引导倒计时
- `m_casttime` — 施法时间（受急速修正后）
- `m_UniqueTargetInfo` — 运行时选中的目标列表
- `m_triggeredCastFlags` — 触发标记（忽略冷却/消耗等）
- `m_selfContainer` — 自引用指针，用于安全取消

## 2. 流程图

```
                    ┌───────────┐
                    │   NULL    │
                    └─────┬─────┘
                          │ prepare()
                          ▼
                    ┌───────────┐
              ┌────▶│ PREPARING │◀──── 瞬发: casttime=0
              │     └─────┬─────┘      直接调 cast(true)
              │           │
              │     timer > 0 时       ┌──────────────────┐
              │     update() 倒计时    │ 中断检查:         │
              │           │            │ - 目标消失 → cancel│
              │           │ timer=0    │ - 移动打断 → cancel│
              │           ▼            │ - 施法者死亡→ cancel│
              │     ┌───────────┐      └──────────────────┘
              │     │   cast()  │
              │     └─────┬─────┘
              │           │
              │     ┌─────┴─────────────────────┐
              │     │ 1. UpdatePointers          │
              │     │ 2. CheckCast (二次验证)     │
              │     │ 3. 递减检查                 │
              │     │ 4. SelectSpellTargets      │
              │     │ 5. TakePower / TakeReagents │
              │     │ 6. SendSpellCooldown       │
              │     └─────┬───────────────────────┘
              │           │
              │           ▼
              │     ┌───────────┐
              │     │ LAUNCHED  │
              │     └─────┬─────┘
              │           │
              │     ┌─────┴──────────────────┐
              │     │ 有 LaunchDelay?        │
              │     │  是 → 等待延迟          │
              │     │  否 → HandleLaunchPhase │
              │     └─────┬──────────────────┘
              │           │
              │     ┌─────┴──────────────────┐
              │     │ 引导法术?               │
              │     │  是 → CHANNELING        │
              │     │  否 → handle_immediate  │
              │     └─────┬──────────────────┘
              │           │
              │           ▼
              │     ┌───────────┐         ┌──────────────┐
              │     │ CHANNELING│────────▶│  FINISHED    │
              │     │ (引导倒计时)│  完成/中断 │ (清理资源)   │
              │     └───────────┘         └──────────────┘
              │
              │     cancel() 路径:
              │     ┌───────────┐
              └─────│ 任何状态   │
                    │ → FINISHED │
                    │ + 清理     │
                    └───────────┘
```

### handle_immediate 流程
```
handle_immediate()
    │
    ├── HandleLaunchPhase()     ← 处理 Launch 阶段效果
    │   └── 对每个效果: HandleEffect(LAUNCH, LAUNCH_TARGET)
    │
    ├── HandleHitPhase()         ← 处理命中阶段效果
    │   └── 对每个目标+效果: HandleEffect(HIT, HIT_TARGET)
    │
    ├── ProcSkillsAndAuras       ← 触发 Proc
    │
    ├── CallScriptAfterHitHandlers
    │
    └── finish(SPELL_CAST_OK)
```

## 3. 关键设计决策

### 决策: SpellInfo 与 Spell 实例分离
**原因**: SpellInfo 是静态数据（从 DBC 加载），所有施法共享同一份。Spell 是每次施法的运行时实例。这避免了数据锁定和内存浪费。

### 决策: CheckCast 分两次执行
**原因**: 第一次在 prepare() 时做基本检查（冷却、资源、射程），第二次在 cast() 时做完整检查（含递减、交易状态等）。中间间隔是施法条的等待时间——目标状态可能已经变化。

### 决策: 四阶段效果处理 (Launch/LaunchTarget/Hit/HitTarget)
**原因**: 区分发射端和命中端，支持弹道延迟。发射阶段处理施法者端逻辑（如消耗资源），命中阶段处理目标端逻辑（如造成伤害）。弹道法术在 Launch 和 Hit 之间有时间差。

### 决策: cancel() 恢复旧状态再调 finish()
**原因**: cancel() 先将状态设为 FINISHED，处理中断清理，然后恢复到旧状态再调 finish()。这样 finish() 中可以根据原始状态做不同清理（PREPARING 退 GCD，CHANNELING 移除 Aura）。

### 决策: 自引用指针 (m_selfContainer)
**原因**: Spell 对象由 Unit 持有。cancel()/finish() 可能导致 Spell 被删除。通过 m_selfContainer 双指针，cancel 后检查 `*m_selfContainer == this` 来判断是否还被引用，防止 use-after-free。

### 决策: 触发标记 (TriggeredCastFlags) 控制跳过逻辑
**原因**: 触发法术（由天赋/Aura/脚本触发）通常不消耗资源、不触发冷却、不做完整验证。通过位标记精确控制跳过哪些步骤，避免为每种触发场景写不同路径。

## 4. 可借鉴的通用模式

### 模式: 静态定义 + 运行时实例
- 适用于所有需要大量同类对象的系统
- 静态数据只加载一份，运行时实例只持有差异状态
- Go 实现: SpellInfo 作为值类型存储在 map 中，Spell 持有 *SpellInfo 指针

### 模式: 分阶段效果处理
- 适用于需要区分"发射"和"命中"的系统（远程攻击、弹道、延迟效果）
- 即使游戏没有弹道，也可以用两阶段来分离"施法者端逻辑"和"目标端逻辑"
- Go 实现: SpellEffectHandleMode 枚举 + 分发函数表

### 模式: 验证链
- 每个检查返回具体的错误码，不使用 bool
- 检查按成本排序：先检查便宜的（状态标记），再检查昂贵的（目标查询）
- Go 实现: 定义 SpellCastResult 枚举，CheckCast() 返回枚举值

### 模式: 状态机驱动更新
- update() 由外部定时器驱动（TC 用 heartbeat 50ms）
- PREPARING 状态倒计时，到 0 触发 cast()
- CHANNELING 状态倒计时，到 0 触发 finish()
- Go 实现: timer.Scheduler 注册回调，或用 select + ticker

### 适用场景
- 回合制游戏不需要施法条 → 可简化为 prepare → cast 直接连接
- 无弹道游戏 → Launch 和 Hit 可以合并为一个阶段
- 无引导机制 → 可省略 CHANNELING 状态
