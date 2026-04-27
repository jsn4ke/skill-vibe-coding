## Why

项目作者为中文使用者，但当前所有 Go 代码注释均为英文，部分核心文件甚至完全无注释。这导致：1) 中文开发者阅读代码时需要额外翻译成本；2) 无注释文件缺乏对设计意图和 TC 对齐关系的说明；3) 没有规则约束未来新增代码的注释语言，容易再次出现注释风格不一致的情况。

## What Changes

- 为所有 25 个非测试 Go 源文件补齐中文注释（doc comments + 关键逻辑 inline comments）
- 对已有英文注释的核心文件（engine, unit, spell, aura, effect），将注释改为中文或增加中文补充
- 对零注释文件（entity, stat, event, script, combat, cooldown, diminishing, proc, targeting, timer, timeline/renderer, server/main.go），新增中文 doc comments 和关键逻辑注释
- 对 4 个技能实现文件（fireball, blizzard, arcane-missiles, living-bomb），补齐中文注释
- 新增 `.claude/rules/comment-language.md` 规则文件，要求所有新增 Go 代码必须使用中文注释

## Capabilities

### New Capabilities
- `chinese-comments`: 为所有 Go 源文件补齐中文注释，覆盖 doc comments、TC 对齐说明、关键逻辑 inline comments
- `comment-language-rule`: 新增规则文件，强制要求未来所有 Go 代码使用中文注释

### Modified Capabilities
（无现有 spec 需要修改，这是纯注释/规则层面的变更）

## Impact

- 影响范围：skill-go/ 下所有 25 个非测试 Go 源文件 + 1 个新增规则文件
- 不影响任何运行时行为或 API
- 不涉及 breaking changes
- 测试文件暂不纳入本次变更范围（测试函数名已足够自描述）