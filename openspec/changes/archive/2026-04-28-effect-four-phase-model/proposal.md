## Why

当前 effect 系统只有一个处理阶段（所有 effect 一次性处理），没有区分 Launch 和 Hit。TC 的 effect pipeline 有四个阶段（Launch / LaunchTarget / Hit / HitTarget），每个 effect handler 只在特定 mode 执行。这导致：伤害类 effect 无法在发射时计算、命中时施加；Aura 应用时机不正确；脚本钩子粒度不够（只有一个 HookOnEffectHit）。需要按 TC 机制实现四阶段模型，使现有和未来的 effect 行为正确。

## What Changes

- **BREAKING**: `ProcessAll` 重构为四阶段分发：Launch → LaunchTarget → Hit → HitTarget
- **BREAKING**: 每个 effect handler 内部添加 mode guard（`if ctx.Mode != X { return }`），按 TC 映射绑定正确的 mode
- **BREAKING**: `spell.HandleEffects(mode)` 从空壳变为真正的分发入口
- 新增 3 个脚本钩子：`HookOnEffectLaunch`、`HookOnEffectLaunchTarget`、`HookOnEffectHitTarget`（保留现有 `HookOnEffectHit`）
- 现有 `HookOnEffectHit` 的语义从"唯一的 effect 钩子"变为"HIT 阶段（无目标）的 effect 钩子"，对齐 TC 的 `OnEffectHit`
- 迁移所有现有 skill 的 `RegisterScripts` 中 `HookOnEffectHit` 的使用，按 TC 语义分配到正确的钩子
- 所有 effect handler 重新绑定 mode，对照 TC 映射表校验

## Capabilities

### New Capabilities

- `effect-four-phase`: effect 四阶段分发模型，包含 mode guard 机制、按阶段分发流程、脚本钩子按 mode 分发

### Modified Capabilities

- `script-system-hooks`: 新增 3 个 effect 脚本钩子（HookOnEffectLaunch / HookOnEffectLaunchTarget / HookOnEffectHitTarget），现有 HookOnEffectHit 语义变更
- `living-bomb-skill`: RegisterScripts 中 HookOnEffectHit 需迁移到正确的阶段钩子
- `arcane-missiles-skill`: 无脚本迁移（仅使用 AuraHookOnPeriodic），但 MissileInfo 的 EffectSchoolDamage handler 需绑定 LAUNCH_TARGET mode

## Impact

- **核心文件**: `skill-go/pkg/effect/effect.go`（ProcessAll 重构）、`skill-go/pkg/spell/spell.go`（HandleEffects 实现）、`skill-go/pkg/script/script.go`（新增钩子枚举）
- **引擎文件**: `skill-go/pkg/engine/engine.go`（CallLaunchHook 等适配）
- **所有 skill**: `skill-go/skills/*/` 的 RegisterScripts 可能需要迁移钩子注册
- **测试文件**: 所有 `*_engine_test.go` 需验证四阶段行为正确
