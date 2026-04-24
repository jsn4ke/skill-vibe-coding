## Why

实现具体游戏技能前，需要一种系统化的方法将技能拆解为可实现的组件。当前缺乏标准流程来分析技能的组成要素（效果、aura、目标、冷却等）并将其映射到机制框架。需要一个 skill，输入技能名称或描述，自动搜索技能细节，参考 TC 框架进行拆解，输出结构化的技能设计文档用于思考和讨论。

## What Changes

- 创建 agent skill `skill-design-decompose`（位于 `.claude/skills/skill-design-decompose/`）
- 包含 SKILL.md 定义完整的 4 步工作流
- 包含 references/ 目录存放技能分类体系、输出模板
- 工作流：
  1. 判断技能来源（WoW 技能搜 Wowhead，自创技能用用户描述）
  2. 调用 tc-mechanism-ref 查 TC 对应机制
  3. 对照 WoW 框架拆解，优先匹配已有机制，无对应则自创
  4. 输出技能设计文档（含详细实现建议和设计思路）
- 更新 game-mechanism rule 增加触发条件

## Capabilities

### New Capabilities
- `skill-design-decompose`: 技能设计拆解 skill，将技能拆分为可实现的组件并输出设计文档

### Modified Capabilities
- `game-mechanism-rule`: 新增"设计具体技能"触发条件

## Impact

- **文件**: 新增 `.claude/skills/skill-design-decompose/SKILL.md` + `references/`
- **文件**: 修改 `.claude/rules/game-mechanism.md` 新增触发词
- **依赖**: 依赖 `tc-mechanism-ref` skill、`../TrinityCore` 路径、WebSearch 能力
- **范围**: 仅限 skills 项目内使用
