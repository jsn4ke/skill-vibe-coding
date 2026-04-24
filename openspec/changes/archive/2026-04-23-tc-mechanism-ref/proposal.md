## Why

设计游戏技能系统时，需要频繁参考 TrinityCore (TC) 的成熟机制实现（施法流程、Aura 系统、冷却充能等）。当前每次都需要手动去 TC 源码中搜索和阅读，效率低且容易遗漏关键设计决策。需要一个 agent skill，输入机制方向即可自动检索 TC 源码，输出结构化的机制分析和通用设计模式。

## What Changes

- 创建 agent skill `tc-mechanism-ref`（位于 `.claude/skills/tc-mechanism-ref/`）
- 包含 SKILL.md 定义触发条件和工作流程
- 包含 references/ 目录存放 TC 源码搜索路径映射和关键词索引
- 创建 `tc-references/` 目录作为分析结果的知识库
- Skill 支持的工作流：
  1. 接收机制方向关键词（如 "aura stacking"、"spell cast flow"）
  2. 检查 `tc-references/` 是否已有对应分析，有则直接返回
  3. 无则搜索 `../TrinityCore/src/server/game/Spells/` 相关代码
  4. 输出结构化摘要：核心数据结构、流程图（ASCII）、关键设计决策、可借鉴的通用模式
  5. 保存分析结果到 `tc-references/<topic>.md`

## Capabilities

### New Capabilities
- `tc-mechanism-ref`: TrinityCore 机制查阅 skill，用于游戏技能系统设计时的参考分析

### Modified Capabilities
（无需修改现有 capability）

## Impact

- **文件**: 新增 `.claude/skills/tc-mechanism-ref/SKILL.md` + `references/` 目录
- **文件**: 新增 `tc-references/` 目录（知识库，按需积累）
- **依赖**: 依赖 `../TrinityCore` 路径可访问
- **范围**: 仅限 skills 项目内使用
