# Spell Timing Fields (施法时间字段)

> Source: TrinityCore | Generated: 2026-04-28 | Topic: CastTime, Speed, LaunchDelay, MinDuration, Duration, IsChanneled — DBC 定义、Spell::Update 驱动、命中延迟计算

## 1. 核心数据结构

### SpellMiscEntry (DBC 原始定义)

来自 `DB2Structure.h` 的 `SpellMiscEntry`，是 WoW 客户端数据表的直接映射：

| 字段 | 类型 | 单位 | 含义 |
|------|------|------|------|
| `CastingTimeIndex` | uint16 | 索引 | 指向 SpellCastTimes 表的索引，非直接毫秒值 |
| `DurationIndex` | uint16 | 索引 | 指向 SpellDuration 表的索引 |
| `Speed` | float | yards/sec | 弹道速度。0 = 无弹道 |
| `LaunchDelay` | float | 秒 | 发射后延迟，在 Launch 阶段之前等待 |
| `MinDuration` | float | 秒 | 弹道最小飞行时间（保底延迟） |

### SpellInfo (TC 运行时)

来自 `SpellInfo.h`，由 `SpellMgr.cpp` 从 DBC 加载：

```cpp
// 施法时间 — 不是直接毫秒值，通过 CastTimeEntry 查表 + 急速修正
SpellCastTimesEntry const* CastTimeEntry = nullptr;

// 速度三件套
float Speed = 0.0f;          // 弹道速度 (yards/sec)
float LaunchDelay = 0.0f;    // 发射延迟 (秒，非毫秒)
float MinDuration = 0.0f;    // 最小飞行时间 (秒，非毫秒)

// 持续时间 — 通过 DurationEntry 查表
SpellDurationEntry const* DurationEntry = nullptr;

// 引导标记 — 不是独立字段，是 Attributes 位标记
bool IsChanneled() const {
    return HasAttribute(SPELL_ATTR1_IS_CHANNELLED | SPELL_ATTR1_IS_SELF_CHANNELLED);
}
```

### 关键差异：TC vs 当前项目

| 方面 | TC | 当前项目 |
|------|-----|---------|
| CastTime | `CastTimeEntry` 索引 → 查表 → `CalcCastTime()` 急速修正 | 直接 `uint32` 毫秒值 |
| LaunchDelay | `float` 秒 | `uint32` 毫秒 |
| MinDuration | `float` 秒 | `uint32` 毫秒 |
| Speed | `float` yards/sec | `float` yards/sec (一致) |
| Duration | `DurationEntry` 索引 → 查表 | 直接 `uint32` 毫秒值 |
| IsChanneled | `Attributes` 位标记推导 | 独立 `bool` + `Attributes` 冗余 |

## 2. 流程图

### 完整时间线

```
时间轴:  0ms          CastTime         LaunchDelay      弹道飞行        命中
         │               │                │               │              │
         ▼               ▼                ▼               ▼              ▼
     ┌─────────┐   ┌──────────┐    ┌──────────┐   ┌──────────┐   ┌──────────┐
     │ PREPARE │──▶│   cast() │──▶│ LAUNCHED │──▶│ LAUNCHED │──▶│  HIT     │
     │ 倒计时   │   │ 选目标    │   │ 等Delay  │   │ 等弹道   │   │ 处理效果  │
     └─────────┘   │ 消耗资源  │   └──────────┘   └──────────┘   └──────────┘
                    │ LaunchPhase│        │               │
                    └──────────┘        │               │
                                        ▼               ▼
                               LaunchDelay=0?     Speed=0?
                               是 → 跳过          是 → 跳过
```

### cast() 中的决策逻辑

```
cast()
  │
  ├─ 1. 选目标 (SelectSpellTargets)
  ├─ 2. 消耗资源 (TakePower)
  ├─ 3. 状态 = SPELL_STATE_LAUNCHED
  │
  ├─ LaunchDelay == 0 ?
  │   是 → HandleLaunchPhase() 立即执行
  │   否 → 延迟到 handle_delayed() 中执行
  │
  ├─ m_delayMoment > 0 && !IsChanneled ?
  │   是 → 延迟命中路径 (handle_delayed)
  │        - m_immediateHandled = false
  │        - SetDelayStart(0)
  │        - 等待事件触发 handle_delayed(t_offset)
  │   否 → 立即命中路径 (handle_immediate)
  │
  └─ IsChanneled ?
      是 → handle_immediate 中处理 Hit 后
           进入 SPELL_STATE_CHANNELING
           Timer = GetMaxDuration()
      否 → handle_immediate → finish()
```

### handle_delayed() 的分阶段执行

```
handle_delayed(t_offset)
  │
  ├─ 1. LaunchDelay 未处理?
  │     launchMoment = floor(LaunchDelay * 1000)
  │     t_offset < launchMoment? → return launchMoment (等下次)
  │     否 → HandleLaunchPhase(), m_launchHandled = true
  │
  ├─ 2. m_delayMoment > t_offset?
  │     是 → return m_delayMoment (等弹道)
  │
  ├─ 3. _handle_immediate_phase()
  │     - 处理所有到时间的 TargetInfo
  │
  └─ 4. DoProcessTargetContainer()
        - 按 TimeDelay 过滤，只处理 TimeDelay <= t_offset 的目标
        - 剩余目标等待下次 handle_delayed
```

### 命中延迟计算 (每个目标)

```
AddUnitTarget() 中:
  hitDelay = LaunchDelay                    // 基础延迟
  
  if caster != target:                      // 不是自身
    if SPELL_ATTR9_MISSILE_SPEED_IS_DELAY_IN_SEC:
      hitDelay += max(Speed, MinDuration)   // Speed 被当作固定延迟秒数
    else if Speed > 0:
      dist = max(distance(caster, target), 5.0)  // 最小 5 码
      hitDelay += max(dist / Speed, MinDuration)  // 距离/速度，保底 MinDuration
  
    targetInfo.TimeDelay += floor(hitDelay * 1000)  // 转毫秒
```

## 3. 关键设计决策

### 决策: CastTime 通过索引查表而非直接存储
**原因**: WoW 的 `SpellMisc` 表只存 `CastingTimeIndex`，实际值在 `SpellCastTimes` 表中。这允许:
- 多个法术共享同一条施法时间记录
- 急速修正通过 `CalcCastTime()` 统一处理
- 热修时只改一条记录即影响所有引用法术

**我们项目的做法**: 直接存毫秒值，简化了查表，但牺牲了共享和热修能力。对于我们的场景是合理的简化。

### 决策: LaunchDelay 控制的是 LaunchPhase 的延迟，不是 Hit 的延迟
**原因**: TC 区分两个阶段:
- `LaunchPhase` 处理施法者端效果（如创建动态对象、设置目标位置）
- `HitPhase` 处理目标端效果（如伤害、光环应用）

`LaunchDelay` 让 LaunchPhase 也延迟执行。如果 LaunchDelay > 0，则 cast() 中不执行 HandleLaunchPhase()，而是等到 handle_delayed() 中 LaunchDelay 时间到达后再执行。

**当前项目差异**: 我们在 cast() 中总是立即执行 HandleLaunchPhase()，LaunchDelay 只影响 Hit 阶段。这与 TC 不一致。

### 决策: Speed 和 MinDuration 是叠加关系，不是二选一
**原因**: 公式 `hitDelay += max(dist / Speed, MinDuration)` 表示:
- 正常情况: 飞行时间 = 距离 / 速度
- 近距离保护: 如果距离太近导致飞行太快，MinDuration 保底

MinDuration 不是独立延迟，而是 Speed 路径的保底值。

### 决策: SPEED_IS_DELAY_IN_SEC 特殊属性改变 Speed 的语义
**原因**: 某些法术的 Speed 字段不是速度，而是固定延迟秒数。通过 `SPELL_ATTR9_MISSILE_SPEED_IS_DELAY_IN_SEC` 标记切换语义:
- 正常: `hitDelay += max(dist / Speed, MinDuration)`
- 固定延迟: `hitDelay += max(Speed, MinDuration)` (忽略距离)

用于不需要弹道但需要统一延迟的法术（如某些 AoE）。

### 决策: handle_delayed 支持多目标错开命中
**原因**: 每个目标有独立的 `TimeDelay`。Chain 法术（ChainTargets）和 Bouncy 属性会叠加前一个目标的延迟。这意味着:
- 第 1 个目标: LaunchDelay + dist/Speed
- 第 2 个目标 (chain): 前一目标延迟 + dist/Speed (LaunchDelay 只算一次)

这是 TC 比"所有目标同时命中"更精细的设计。

## 4. 可借鉴的通用模式

### 模式: 三层延迟 (Prepare → Launch → Hit)
- 适用: 需要区分"施法者准备"、"发射动画"、"命中效果"的系统
- 施法时间 (Prepare): 有施法条的法术
- 发射延迟 (LaunchDelay): 从施法完成到发射的时间
- 弹道飞行 (Speed + MinDuration): 从发射到命中的时间
- 三个阶段可以独立为 0，退化成更简单的模式

### 模式: 保底值 (MinDuration)
- 适用: 需要保证最小视觉/动画时间的系统
- `max(实际计算值, 保底值)` 的通用模式
- 避免近距离弹道瞬间的视觉问题

### 模式: 延迟事件的分批处理
- 适用: 多目标且命中时间不同的系统
- `handle_delayed(t_offset)` 按 TimeDelay 过滤已到期目标
- 未到期的继续等待，不阻塞已到期的目标
- 比统一延迟所有目标更精确

### 适用场景
- 实时战斗系统 → 完整三层延迟
- 回合制 → 只需 CastTime，Speed/LaunchDelay 为 0
- 无弹道 → Speed = 0，可用 LaunchDelay 模拟延迟命中
- 全部即时 → CastTime = Speed = LaunchDelay = 0，同帧完成
