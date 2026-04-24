## Why

当前技能系统已覆盖单体施法（火球术）和投射物延迟命中，但缺少两种关键机制：**引导施法（channeled spell）**和**范围持续伤害（persistent area aura）**。暴风雪（Blizzard）是魔兽世界中经典的法师AoE技能，同时涉及这两种机制，是验证和扩展技能系统的理想选择。

## What Changes

- 新增 `skill-go/skills/blizzard/` 包，实现暴风雪技能
- 支持引导施法：`IsChanneled = true`，8秒引导期间每1秒触发一次伤害
- 支持范围目标选择：8码半径内的敌方单位，复用已有 `targeting.SelectArea`，每次 tick 重新选目标（与 TC FillTargetMap 流程一致）
- 支持持续区域光环（Persistent Area Aura）：引导者在施法位置生成一个8秒的地面AoE
- 验证 `spell.StateChanneling` 状态转换：Preparing → Channeling → Finished
- 新增暴风雪测试覆盖引导生命周期、AoE目标选择、周期伤害、取消/打断

## Capabilities

### New Capabilities

- `blizzard-skill`: 暴风雪技能实现，包括引导施法流程、范围目标选择、周期性AoE伤害计算、法力消耗

### Modified Capabilities

- `fireball-spell`: 无需修改，暴风雪复用已有的 spell/aura/targeting 基础设施

## Impact

- 新增 `skill-go/skills/blizzard/` 目录（blizzard.go + blizzard_test.go）
- 可能需要在 `spell.SpellInfo` 或 `spell.Spell` 中扩展引导相关的字段（如 tick 周期回调）
- `aura.Manager` 的 `TickPeriodic` 已满足需求，无需修改
- `server/main.go` 演示代码将新增暴风雪模拟场景
