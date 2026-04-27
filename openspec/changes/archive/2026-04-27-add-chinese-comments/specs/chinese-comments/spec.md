## ADDED Requirements

### Requirement: 所有导出符号必须有中文 doc comments

所有非测试 Go 源文件中的导出类型（type）、函数（func）、常量（const）、变量（var）SHALL 拥有中文 doc comment。doc comment SHALL 以中文撰写，TC 术语和专有名词 SHALL 保留英文原文。

#### Scenario: 导出类型缺少中文 doc comment
- **WHEN** 一个 Go 源文件包含导出类型但无中文 doc comment
- **THEN** 该类型 SHALL 被添加中文 doc comment，说明其用途和设计意图

#### Scenario: 导出函数缺少中文 doc comment
- **WHEN** 一个 Go 源文件包含导出函数但无中文 doc comment
- **THEN** 该函数 SHALL 被添加中文 doc comment，说明其行为和关键参数

### Requirement: TC 对齐说明以中文标注

当代码设计对齐 TrinityCore 实现时，SHALL 以 `// 对齐 TC 的 XXX` 格式在相关类型或字段处添加中文注释，说明对齐的 TC 概念。

#### Scenario: 核心类型对齐 TC
- **WHEN** 一个类型（如 Unit、Spell、Aura）的设计对齐 TC 的对应概念
- **THEN** 该类型的 doc comment 或 inline comment SHALL 包含 `对齐 TC 的 XXX` 说明

#### Scenario: 字段对齐 TC 成员变量
- **WHEN** 一个结构体字段对应 TC 的成员变量（如 m_currentSpells）
- **THEN** 该字段 SHALL 有 inline comment 标注 `对齐 TC 的 m_currentSpells`

### Requirement: 关键逻辑添加中文 inline comments

非显而易见的逻辑（如特殊分支、workaround、隐含约束）SHALL 添加中文 inline comment。显而易见的代码（如简单赋值、getter/setter）SHALL NOT 添加注释。

#### Scenario: 非显而易见的条件分支
- **WHEN** 代码包含非显而易见的条件判断（如特殊处理某类法术）
- **THEN** 该条件分支 SHALL 有中文 inline comment 说明原因

#### Scenario: 显而易见的代码
- **WHEN** 代码逻辑简单明了（如 `return u.id`）
- **THEN** SHALL NOT 添加注释

### Requirement: 英文注释替换为中文

已有的英文 doc comments 和 inline comments SHALL 被替换为等效的中文注释。TC 术语和专有名词 SHALL 保留英文原文。

#### Scenario: 已有英文 doc comment 的导出函数
- **WHEN** 一个导出函数有英文 doc comment
- **THEN** 该 doc comment SHALL 被替换为中文版本，保留 TC 术语英文原文

#### Scenario: 已有英文 inline comment
- **WHEN** 代码中有英文 inline comment
- **THEN** 该 comment SHALL 被替换为中文版本