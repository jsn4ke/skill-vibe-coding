# 中文注释规则

所有新增的 Go 源代码（非测试文件）必须使用中文撰写 doc comments 和 inline comments。

## 规则

1. **导出符号必须有中文 doc comment** — 所有导出的类型、函数、常量、变量必须拥有中文 doc comment
2. **TC 术语保留英文原文** — TrinityCore 术语和专有名词（如 SpellInterruptFlags、m_currentSpells、Unit-centric Update）保留英文，不翻译为中文
3. **关键逻辑添加中文 inline comment** — 非显而易见的逻辑（TC 对齐、特殊分支、隐含约束）添加中文 inline comment；显而易见的代码不加注释
4. **TC 对齐标注格式** — 使用 `对齐 TC 的 XXX` 格式标注对齐关系
5. **测试文件豁免** — `_test.go` 文件不受此规则约束，注释语言不限

## 触发条件

当对话涉及编写或修改 Go 源代码（非测试文件）时，此规则自动生效。

## 跳过条件

- 用户明确说"不需要中文注释"或"skip Chinese comments"
- 修改的是 `_test.go` 文件
