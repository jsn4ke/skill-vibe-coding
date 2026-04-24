## Why

在 skills 项目中实现游戏技能机制时，需要确保 Claude 始终参考 TrinityCore 的成熟设计，而非凭空设计。目前已拥有 `tc-mechanism-ref` skill 用于查阅 TC 实现，但缺乏一条规则在实现时自动触发参考流程。需要一个 Claude Code rule 文件，在检测到游戏机制相关任务时自动调用 tc-mechanism-ref，将 TC 的设计模式融入实现过程。

## What Changes

- 创建 `.claude/rules/game-mechanism.md` 规则文件
- 规则定义触发关键词（spell、aura、cooldown、targeting、proc、effect 及中文对应）
- 规则要求：检测到机制实现任务时，自动先通过 tc-mechanism-ref 查阅 TC 对应实现
- 宽松模式：自动参考并融入实现，不强制等待用户确认
- 输出中应体现 TC 参考的痕迹（提及参考了哪个机制、借鉴了什么模式）

## Capabilities

### New Capabilities
- `game-mechanism-rule`: Claude Code 规则，确保实现游戏机制时自动参考 TC 设计

### Modified Capabilities
（无需修改现有 capability）

## Impact

- **文件**: 新增 `.claude/rules/game-mechanism.md`
- **依赖**: 依赖 `tc-mechanism-ref` skill 和 `../TrinityCore` 路径可访问
- **范围**: 仅限 skills 项目内生效
