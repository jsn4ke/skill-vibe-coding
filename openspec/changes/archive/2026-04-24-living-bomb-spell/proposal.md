## Why

项目目前有 3 个技能（Fireball、Blizzard、Arcane Missiles），覆盖了标准施法、引导 AoE、引导周期触发三种模式。但框架缺少两项关键能力：**脚本驱动的法术拦截 (Dummy + Script)** 和 **Aura 过期回调 (AfterEffectRemove + RemoveMode)**。Living Bomb 的三体结构（44457 施放 → 217694 DoT → 44461 爆炸 + 传染）精确地需要这两项能力，是验证框架脚本系统的最佳试金石。

## What Changes

- 增强 `pkg/script/` 的 `AuraContext`，添加 `RemoveMode` 和 `Aura` 字段
- 在 `pkg/aura/Manager` 中接入 `script.Registry`，在 AddAura/RemoveAura/TickPeriodic 生命周期中调用脚本 hook
- 在 `pkg/effect/` 的 `ProcessAll` 中接入 `script.Registry`，每个效果处理前后调用 `OnEffectHit` hook
- 在 `pkg/spell/Spell` 中添加 `SpellValues map[uint8]float64` 字段，用于脚本间传递参数
- 扩展 `Spell.SelectTargets()` 支持 AoE 目标选择（接入 `targeting.TargetSelector`）
- 在 `skill-go/skills/living-bomb/` 中实现完整的三体活动炸弹技能包

## Capabilities

### New Capabilities

- `living-bomb-skill`: 活动炸弹三体技能——即时施放 (44457) → 周期 DoT 4s (217694) → AoE 爆炸 10yd + 传染 (44461)。含技能包、单元测试、时间线测试
- `script-system-hooks`: 脚本系统生命周期 hook 接入——AuraContext 增强 (RemoveMode + Aura)、effect.ProcessAll 脚本 hook、aura.Manager 脚本 hook、Spell.SelectTargets AoE 扩展、SpellValues 传递

### Modified Capabilities

（无现有 spec 需要修改——所有变更为纯新增或内部增强）

## Impact

- **pkg/script/script.go**: AuraContext 添加 RemoveMode + Aura 字段
- **pkg/aura/aura.go**: Manager 接收 `*script.Registry`，在 RemoveAura/TickPeriodic 中调用 hook；Aura 添加 SpellValues 字段
- **pkg/effect/effect.go**: ProcessAll 接收 `*script.Registry`，每个效果调用 OnEffectHit hook
- **pkg/spell/spell.go**: Spell 添加 SpellValues 字段；SelectTargets 扩展 AoE 支持
- **skills/living-bomb/**: 新建技能包目录（3 个 SpellInfo + CastXxx + 脚本注册 + 测试）
- **无破坏性变更**: 新增字段和方法，现有技能代码无需修改
