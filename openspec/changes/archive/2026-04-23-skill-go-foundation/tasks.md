## Tasks

- [x] Phase 1: Layer 0 — 基础设施层
  - [x] 1.1 初始化 Go 项目 (go.mod, 目录结构)
  - [x] 1.2 entity 包: Entity, Position, UnitState
  - [x] 1.3 stat 包: StatSet, Modifier chain
  - [x] 1.4 event 包: EventBus pub/sub
  - [x] 1.5 timer 包: 延迟/周期回调封装

- [x] Phase 2: Layer 1 — 战斗支撑层
  - [x] 2.1 combat 包: 战斗状态, 伤害公式, 命中结果

- [x] Phase 3: Layer 2 — spell 施法状态机
  - [x] 3.1 tc-mechanism-ref 查阅 spell-cast-flow
  - [x] 3.2 spell 包: SpellState, SpellInfo, 验证链, 状态转移

- [x] Phase 4: Layer 2 — effect 效果管线
  - [x] 4.1 tc-mechanism-ref 查阅 effect-pipeline
  - [x] 4.2 effect 包: SpellEffectName 枚举, 效果分发, 数值计算

- [x] Phase 5: Layer 2 — aura 系统
  - [x] 5.1 tc-mechanism-ref 查阅 aura-system 和 aura-stacking
  - [x] 5.2 aura 包: Aura, AuraEffect, AuraApplication, 堆叠, 周期效果

- [x] Phase 6: Layer 2 — cooldown 冷却充能
  - [x] 6.1 tc-mechanism-ref 查阅 cooldown-charge
  - [x] 6.2 cooldown 包: SpellHistory, 冷却, 充能, GCD, 系封锁

- [x] Phase 7: Layer 2 — targeting 目标选择
  - [x] 7.1 tc-mechanism-ref 查阅 target-selection
  - [x] 7.2 targeting 包: 5维度分解, 选择算法

- [x] Phase 8: Layer 2 — proc 触发系统
  - [x] 8.1 tc-mechanism-ref 查阅 proc-system
  - [x] 8.2 proc 包: ProcFlags, 概率判定, 事件匹配

- [x] Phase 9: Layer 2 — diminishing 递减
  - [x] 9.1 tc-mechanism-ref 查阅 diminishing-returns
  - [x] 9.2 diminishing 包: 分组, 递减类型, 持续时间上限

- [x] Phase 10: Layer 2 — script 脚本扩展
  - [x] 10.1 tc-mechanism-ref 查阅 spell-script
  - [x] 10.2 script 包: SpellScript hooks, AuraScript hooks, 注册表

- [x] Phase 11: 集成演示
  - [x] 11.1 main.go 演示入口, 跑通完整技能流程
  - [x] 11.2 用 skill-design-decompose 拆解一个示例技能并实现
