## Architecture

```
skills/
├── .claude/
│   ├── rules/
│   │   └── game-mechanism.md      # 新增：游戏机制实现规则
│   └── skills/
│       └── tc-mechanism-ref/      # 已有：TC 查阅 skill
└── tc-references/                 # 已有：分析知识库
```

## Rule 行为设计

### 触发条件

当对话中出现以下关键词，且上下文是实现/设计/构建意图时触发：

**英文关键词:**
- spell, cast, casting
- aura, buff, debuff
- cooldown, charge, gcd
- targeting, target selection
- proc, trigger
- effect, effect pipeline
- diminishing return, dr
- spell script, hook

**中文关键词:**
- 施法、法术、技能
- 增益、减益、光环
- 冷却、充能
- 目标选择
- 触发、触发器
- 效果、效果管线
- 递减
- 脚本

### 触发后的行为

```
检测到机制实现意图
    │
    ▼
┌─────────────────────────────┐
│ 1. 识别具体机制方向          │
│    从关键词推断 topic        │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ 2. 调用 tc-mechanism-ref    │
│    自动查阅 TC 对应实现      │
│    （利用 skill 5 步流程）   │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────────────┐
│ 3. 将 TC 分析融入实现       │
│    设计参考 TC 的模式        │
│    实现中体现借鉴的决策      │
└─────────────────────────────┘
```

### 宽松模式特征

- 不强制等待用户确认 TC 分析结果
- 不强制输出完整的 TC 分析报告
- 在设计和实现中自然体现 TC 参考
- 当 TC 知识库已有对应分析时，直接引用，不重复搜索
- 用户可以明确说"不需要参考 TC"来跳过

### 输出中的参考痕迹

实现代码或设计文档中，应在适当位置体现 TC 参考：
- 设计决策可以提及 "参考 TC 的 X 模式"
- 不需要逐条标注，自然融入即可

## 关键词 → Topic 映射

Rule 内置一个简单的映射表，将检测到的关键词转为 tc-mechanism-ref 的 topic：

| 检测关键词 | tc-mechanism-ref topic |
|-----------|----------------------|
| spell, cast, 施法, 法术 | spell-cast-flow |
| aura, buff, debuff, 增益, 减益, 光环 | aura-system |
| stack, 堆叠 | aura-stacking |
| cooldown, charge, gcd, 冷却, 充能 | cooldown-charge |
| target, targeting, 目标选择 | target-selection |
| proc, trigger, 触发 | proc-system |
| effect, pipeline, 效果, 管线 | effect-pipeline |
| diminishing, dr, 递减 | diminishing-returns |
| script, hook, 脚本 | spell-script |

## Rule 文件结构

```markdown
# Game Mechanism Implementation Rule

## 触发条件
<关键词列表>

## 行为要求
<宽松模式的工作流程>

## Topic 映射
<关键词 → tc-mechanism-ref topic>

## 约束
- 用户明确说"不需要参考 TC"时可跳过
- 知识库已有分析时直接引用
- 不强制完整分析报告
```

## 约束

- **项目内生效** — rule 位于 skills 项目的 `.claude/rules/`
- **依赖 tc-mechanism-ref** — skill 必须存在且可用
- **不修改 tc-mechanism-ref** — 仅消费其能力
- **宽松执行** — 不阻断用户工作流
