## 1. 规则文件

- [x] 1.1 创建 `.claude/rules/comment-language.md`，定义中文注释规则（导出符号中文 doc comments、TC 术语保留英文、测试文件豁免）

## 2. 核心包 — 零注释文件补齐

- [x] 2.1 为 `pkg/entity/entity.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.2 为 `pkg/stat/stat.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.3 为 `pkg/event/event.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.4 为 `pkg/script/script.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.5 为 `pkg/combat/combat.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.6 为 `pkg/cooldown/cooldown.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.7 为 `pkg/diminishing/diminishing.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.8 为 `pkg/proc/proc.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.9 为 `pkg/targeting/targeting.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.10 为 `pkg/timer/timer.go` 添加中文 doc comments 和关键逻辑注释
- [x] 2.11 为 `pkg/timeline/renderer.go` 添加中文 doc comments 和关键逻辑注释

## 3. 核心包 — 已有英文注释文件替换

- [x] 3.1 将 `pkg/engine/engine.go` 的英文注释替换为中文
- [x] 3.2 将 `pkg/unit/unit.go` 的英文注释替换为中文
- [x] 3.3 将 `pkg/spell/spell.go` 的英文注释替换为中文
- [x] 3.4 将 `pkg/spell/info.go` 的英文注释替换为中文
- [x] 3.5 将 `pkg/aura/aura.go` 的英文注释替换为中文
- [x] 3.6 将 `pkg/effect/effect.go` 的英文注释替换为中文

## 4. 技能实现文件

- [x] 4.1 为 `skills/fireball/fireball.go` 补齐/替换中文注释
- [x] 4.2 为 `skills/blizzard/blizzard.go` 补齐/替换中文注释
- [x] 4.3 为 `skills/arcane-missiles/arcane_missiles.go` 补齐/替换中文注释
- [x] 4.4 为 `skills/living-bomb/living_bomb.go` 补齐/替换中文注释

## 5. 入口文件

- [x] 5.1 为 `server/main.go` 添加中文注释

## 6. 验证

- [x] 6.1 运行 `go build ./...` 确认编译通过
- [x] 6.2 运行 `go test ./...` 确认测试通过（`-race` 在 Windows 上有 DLL 兼容问题，非代码问题）
