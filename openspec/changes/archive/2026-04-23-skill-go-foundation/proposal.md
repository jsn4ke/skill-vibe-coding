## Why

skills 项目需要一个 Go 实现的技能系统。技能系统本身（施法、Aura、冷却、目标选择等）参考 TrinityCore 完整实现，但周边支撑系统（实体、属性、事件、计时器、战斗）以最小化方式实现，仅提供技能运行所需的基础能力。需要一个 change 来规划整个项目的模块层次、依赖关系和实现优先级。

## What Changes

- 建立 Go 项目结构 `skill-go/`
- 实现 Layer 0 基础设施（entity、stat、event、timer）
- 实现 Layer 1 最小化战斗支撑（combat）
- 实现 Layer 2 核心技能系统（spell、effect、aura、cooldown、targeting、proc、diminishing、script）
- 每层实现前通过 tc-mechanism-ref 参考 TC 设计
- 每个具体技能实现前通过 skill-design-decompose 拆解设计

## Capabilities

### New Capabilities
- `skill-go-foundation`: Go 技能系统的完整项目框架，包含最小化支撑层和完整技能层

### Modified Capabilities
（无需修改现有 capability）

## Impact

- **代码**: 新增 Go 项目 `skill-go/`，预计 20-30 个包
- **依赖**: 仅 Go 标准库，无外部依赖
- **规模**: 支撑层约 500-1000 行，技能层约 3000-5000 行
