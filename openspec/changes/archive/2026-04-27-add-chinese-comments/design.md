## Context

当前 skill-go 项目包含 25 个非测试 Go 源文件，其中约一半完全无注释（entity, stat, event, script, combat, cooldown, diminishing, proc, targeting, timer, timeline/renderer, server/main.go），另一半有英文 doc comments（engine, unit, spell, aura, effect）或简短英文注释（技能实现文件）。项目作者为中文使用者，.claude/rules/game-mechanism.md 中已使用中文关键词和说明，但代码注释尚未统一为中文。

## Goals / Non-Goals

**Goals:**
- 所有非测试 Go 源文件的导出类型和函数拥有中文 doc comments
- 关键逻辑（TC 对齐、设计决策、非显而易见的行为）有中文 inline comments
- 新增规则文件确保未来代码也使用中文注释
- 注释风格统一：doc comments 用中文，保留 TC 术语的英文原文（如 "TC Unit-centric Update"）

**Non-Goals:**
- 不修改测试文件的注释（测试函数名已足够自描述）
- 不修改 .claude/rules/ 中已有文件的内容
- 不改变任何运行时行为或 API
- 不引入新的代码逻辑或重构

## Decisions

### 1. 注释语言策略：纯中文 + 英文术语保留

**决定**：doc comments 全部使用中文，但 TC 术语和专有名词保留英文原文。

**替代方案**：中英双语注释（每条注释写两遍）——  rejected，因为维护成本高且信息冗余。

**理由**：项目面向中文开发者，中文注释阅读效率最高。TC 术语（如 "SpellInterruptFlags"、"m_currentSpells"）翻译后反而增加理解成本，保留英文原文更准确。

### 2. 注释补齐范围：仅非测试文件

**决定**：只补齐 25 个非测试 Go 源文件，不涉及 9 个测试文件。

**理由**：测试函数名（如 `TestHandleSchoolDamage_WithSpellPower`）已足够自描述，且测试代码变更频繁，注释维护成本高于收益。

### 3. 注释风格：doc comments + 关键 inline comments

**决定**：
- 所有导出类型（type）、函数（func）、常量（const）、变量（var）必须有中文 doc comment
- 非导出成员仅在逻辑非显而易见时添加 inline comment
- TC 对齐说明以 `// 对齐 TC 的 XXX` 格式标注

**替代方案**：每个函数都加详细注释 —— rejected，过度注释降低代码可读性。

### 4. 规则文件位置：.claude/rules/comment-language.md

**决定**：新建 `.claude/rules/comment-language.md`，作为项目级规则。

**理由**：与现有规则文件（go-race-test.md, game-mechanism.md, skill-test.md）保持一致的组织方式。

## Risks / Trade-offs

- [注释与代码不同步] → 规则文件约束新增代码，但无法自动检测存量代码的注释过时。可接受风险，注释过时是所有项目的通病。
- [英文开发者阅读障碍] → 项目明确面向中文开发者，此为预期 trade-off。
- [注释工作量] → 25 个文件逐个添加注释，工作量中等。按包分批执行可控制节奏。