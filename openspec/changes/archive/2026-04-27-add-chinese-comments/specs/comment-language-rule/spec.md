## ADDED Requirements

### Requirement: 新增 Go 代码必须使用中文注释

所有新增的 Go 源代码（非测试文件）SHALL 使用中文撰写 doc comments 和 inline comments。此规则 SHALL 在 `.claude/rules/comment-language.md` 中定义。

#### Scenario: 新增导出类型
- **WHEN** 开发者新增一个导出类型
- **THEN** 该类型 SHALL 拥有中文 doc comment

#### Scenario: 新增导出函数
- **WHEN** 开发者新增一个导出函数
- **THEN** 该函数 SHALL 拥有中文 doc comment

#### Scenario: 新增 inline comment
- **WHEN** 开发者需要添加 inline comment
- **THEN** 该 comment SHALL 使用中文撰写

### Requirement: TC 术语保留英文原文

在中文注释中引用 TrinityCore 术语时（如 SpellInterruptFlags、m_currentSpells、Unit-centric Update），SHALL 保留英文原文，不翻译为中文。

#### Scenario: 注释中引用 TC 术语
- **WHEN** 中文注释需要提及 TC 的概念或术语
- **THEN** 该术语 SHALL 保留英文原文，可在括号中附上中文说明

### Requirement: 测试文件豁免

测试文件（`_test.go`）SHALL NOT 受中文注释规则约束。测试函数名已足够自描述，无需强制中文注释。

#### Scenario: 测试文件中的注释
- **WHEN** 开发者在测试文件中添加注释
- **THEN** 注释语言不受限制，中文或英文均可