## Architecture

```
skill-go/
├── pkg/
│   ├── entity/         # Entity + Position + UnitState (最小化)
│   ├── stat/           # StatSet + Modifier chain (最小化)
│   ├── event/          # EventBus simple pub/sub (最小化)
│   ├── timer/          # 基于 std time 的调度封装 (最小化)
│   ├── combat/         # 战斗状态 + 伤害计算 (最小化)
│   │
│   ├── spell/          # 施法状态机 (完整)
│   ├── effect/         # 效果管线 (完整)
│   ├── aura/           # Aura 三层架构 (完整)
│   ├── cooldown/       # 冷却 + 充能 + GCD (完整)
│   ├── targeting/      # 目标选择 (完整)
│   ├── proc/           # 触发系统 (完整)
│   ├── diminishing/    # 递减 (完整)
│   └── script/         # 脚本 Hook (完整)
│
├── server/
│   └── main.go         # 演示入口
└── go.mod
```

## 依赖层次

```
Layer 2: 技能核心层 (完整实现，参考 TC)
  spell · effect · aura · cooldown · targeting · proc · diminishing · script
        │
        │ 依赖
        ▼
Layer 1: 战斗支撑层 (最小化)
  combat (战斗状态 + 伤害公式 + 命中结果)
        │
        │ 依赖
        ▼
Layer 0: 基础设施层 (最小化)
  entity · stat · event · timer
```

## 各模块设计

### Layer 0: 基础设施层

#### entity — 实体基础
- `Entity` struct: ID, Position(x,y,z,facing), UnitState
- `Position` struct + 距离/面向计算方法
- `UnitState` flag: Alive, Dead, InCombat, Stunned, Rooted, Silenced, Pacified, Charmed
- Go 组合模式，无继承链

#### stat — 属性系统
- `StatSet` struct: Health, Mana, AttackPower, SpellPower, CritChance, Haste
- `Modifier` chain: 基础值 + flat修正 + 百分比修正 → 最终值
- 支持 Aura 对属性的临时修改

#### event — 事件总线
- 简单 pub/sub，泛型事件类型
- 事件列表: OnDamageDealt, OnDamageTaken, OnHealDealt, OnSpellCast, OnAuraApplied, OnAuraRemoved, OnDeath, OnMovement, OnCombatEnter
- 用于驱动 Proc 系统和 Aura 中断

#### timer — 定时器封装
- 基于 `time.AfterFunc` 的轻量封装
- 注册延迟回调（施法完成）
- 注册周期回调（Aura tick）
- 取消回调（施法中断）
- Tick 驱动或实时驱动

### Layer 1: 战斗支撑层

#### combat — 战斗系统
- 战斗状态管理: 进入/脱离战斗
- 伤害公式: baseValue + levelScale + coefficient × stat + modifiers
- 命中结果枚举: Hit, Crit, Miss, Immune
- 暴击判定: random < CritChance

### Layer 2: 技能核心层

#### spell — 施法状态机
参考 TC Spell.cpp:
- 状态: Preparing → Casting → Launched → Channeling → Finished
- 流程: prepare → CheckCast → update → cast → HandleEffects → finish
- 验证链: 存活、冷却、资源、射程、状态限制
- 支持瞬发/施法/引导/蓄力
- 中断处理: 移动、沉默、眩晕、死亡

#### effect — 效果管线
参考 TC SpellEffects.cpp:
- SpellEffectName 枚举: SchoolDamage, Heal, ApplyAura, TriggerSpell, Energize, Summon, ...
- 每个 SpellInfo 包含多个 SpellEffectInfo
- 效果分发: 按类型路由到对应处理函数
- 数值计算: BasePoints + RealPointsPerLevel + BonusCoefficient × Stat

#### aura — Aura 三层架构
参考 TC Auras/:
- Aura (生命周期管理)
- AuraEffect (单个效果: 数值、周期、堆叠)
- AuraApplication (目标上的实例)
- 堆叠规则: 刷新/增加层数/互斥
- AuraType 枚举: PeriodicDamage, PeriodicHeal, ModStat, ModStun, ModRoot, ...
- Proc 触发: 事件匹配 → 概率判定 → 触发
- 中断条件: 移动/攻击/施法/受伤/死亡

#### cooldown — 冷却与充能
参考 TC SpellHistory:
- SpellCooldown: 单个技能冷却
- CategoryCooldown: 分类共享冷却
- ChargeEntry: 充能恢复队列
- GCD: 全局冷却管理
- SchoolLockout: 法术系封锁
- 修正器: 急速影响恢复速率

#### targeting — 目标选择
参考 TC SpellInfo + Spell:
- 5 维度正交分解: SelectionCategory, ReferenceType, ObjectType, CheckType, Direction
- 显式目标 vs 隐式目标
- 选择算法: 单体(Nearby), 区域(Area), 锥形(Cone), 链式(Chain)
- 距离/面向/数量过滤

#### proc — 触发系统
参考 TC SpellMgr Proc:
- ProcFlags: 触发事件（伤害/治疗/攻击/施法/死亡/...）
- ProcChance / PPM: 触发概率
- 内置冷却 / 充能次数
- 事件匹配: SpellTypeMask × SpellPhaseMask × HitMask

#### diminishing — 递减
参考 TC SpellInfo:
- DiminishGroup: 递减分组（晕、恐惧、魅惑、...）
- DiminishReturnType: none/standard/50%/immune
- DiminishMaxLevel / DurationLimit

#### script — 脚本扩展
参考 TC SpellScript + AuraScript:
- SpellScript hooks: OnCast, OnHit, OnEffectHit, ...
- AuraScript hooks: OnApply, OnRemove, OnPeriodic, OnProc, ...
- PreventDefaultAction / PreventHitEffect
- 注册表: 技能ID → 脚本实例

## 实现优先级

每个技能层模块实现前，通过 tc-mechanism-ref 参考 TC 对应机制设计。

```
Phase 1: Layer 0 全部 (entity → stat → event → timer)
Phase 2: Layer 1 (combat)
Phase 3: spell 状态机 (让一个瞬发技能跑通)
Phase 4: effect 管线 (效果类型)
Phase 5: aura 系统 (三层架构)
Phase 6: cooldown (冷却+充能+GCD)
Phase 7: targeting (目标选择)
Phase 8: proc (触发系统)
Phase 9: diminishing (递减)
Phase 10: script (脚本扩展)
```

## 约束

- **Go 标准库**，不引入外部依赖
- **支撑层最小化**，只做技能需要的
- **技能层完整化**，参考 TC 全部核心机制
- **无网络层**，初期是本地演示/测试
- **每个模块实现前**通过 tc-mechanism-ref 参考 TC 设计
- **每个具体技能实现前**通过 skill-design-decompose 拆解设计
