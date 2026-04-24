## Architecture

```
skills/
├── .claude/skills/tc-mechanism-ref/
│   ├── SKILL.md                    # Skill 定义（触发条件 + 工作流）
│   └── references/
│       └── search-map.md           # TC 源码搜索路径映射
│
└── tc-references/                  # 分析结果知识库（按需积累）
    ├── aura-stacking.md
    ├── spell-cast-flow.md
    └── cooldown-charge.md
```

## Skill 触发

- **方式**: 用户输入 `/tc-mechanism-ref <topic>` 或自然语言提到"参考 TC 的 X 机制"
- **输入**: 机制方向关键词或短描述
- **描述关键词**: TrinityCore、TC、机制、流程、spell flow、aura、cooldown、targeting、proc 等

## 核心工作流

```
用户输入 topic
    │
    ▼
┌──────────────────────┐
│ 1. 检查知识库缓存     │  tc-references/<topic>.md 是否存在？
│    存在 → 直接返回    │
└──────────┬───────────┘
           │ 不存在
           ▼
┌──────────────────────┐
│ 2. 查找搜索路径       │  references/search-map.md 提供
│    确定目标文件范围    │  topic → 源码路径映射
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ 3. 搜索 TC 源码       │  Grep/Glob 在 ../TrinityCore/ 中
│    定位核心实现       │  搜索关键词、枚举、结构体
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ 4. 深度读取 + 分析    │  Read 关键文件，提取：
│    - 核心数据结构      │    struct/enum 关键字段
│    - 状态/流程图       │    ASCII diagram
│    - 关键设计决策      │    WHY 不只是 WHAT
│    - 通用模式提炼      │    去掉 WoW 特定概念
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│ 5. 输出 + 存储        │  展示给用户
│    保存到知识库        │  写入 tc-references/<topic>.md
└──────────────────────┘
```

## 输出格式模板

每个机制分析包含四个标准段落：

### 1. 核心数据结构
- 关键 struct/enum 的字段摘要（不贴整段代码）
- 字段用途简述

### 2. 流程图
- ASCII 状态机或时序图
- 标注关键分支和条件

### 3. 关键设计决策
- 每个决策用 "决策 → 原因" 格式
- 关注 TC 为什么这样设计，不只是做了什么

### 4. 可借鉴的通用模式
- 去掉 WoW 特定概念（spell family、DBC 等）
- 提炼到通用游戏开发层面
- 指出适用场景和不适用场景

## Search Map 设计

`references/search-map.md` 维护 topic → 源码路径的映射表：

| Topic | 关键词 | 搜索路径 |
|-------|--------|----------|
| spell cast flow | Spell::prepare, cast, finish | Spells/Spell.cpp, Spell.h |
| aura system | Aura, AuraEffect, AuraApplication | Spells/Auras/ |
| aura stacking | stack, refresh, hastack | Spells/Auras/SpellAuras.cpp |
| cooldown/charge | SpellHistory, cooldown, charge | Spells/SpellHistory.cpp |
| target selection | SelectTarget, ImplicitTarget | Spells/Spell.cpp, SpellInfo.cpp |
| proc system | ProcFlag, ProcTrigger | Spells/SpellMgr.h, Auras/SpellAuraEffects.cpp |
| effect pipeline | SpellEffect, HandleEffects | Spells/SpellEffects.cpp |
| diminishing returns | DiminishGroup, DiminishReturn | Spells/SpellInfo.cpp |
| spell script | SpellScript, AuraScript, Hook | Spells/SpellScript.cpp, Auras/ |

## 约束

- **只分析机制和流程**，不涉及具体 WoW 技能实现
- **TC 路径固定**为 `../TrinityCore`
- **知识库文件**使用英文文件名、中文内容
- 分析粒度适中——足够理解设计意图，不需要逐行解读
