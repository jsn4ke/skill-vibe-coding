## Why

项目目前有 Fireball（弹道投射+DoT）和 Blizzard（引导 AoE 区域伤害）两个技能，但缺少**引导型周期触发法术 (Channeled Periodic Trigger Spell)** 这一 WoW 核心机制。奥术飞弹是这一模式的代表——每次 tick 不是直接计算伤害，而是触发一个独立法术实例。实现它将引入 `AuraPeriodicTriggerSpell` 类型，使框架支持"法术触发法术"的通用模式。

## What Changes

- 在 `pkg/aura/` 中新增 `AuraPeriodicTriggerSpell` Aura 类型
- 在 `pkg/aura/` 的 `TickPeriodic` 中为 `PeriodicTriggerSpell` 类型添加 tick 处理分支
- 在 `skill-go/skills/arcane-missiles/` 中实现奥术飞弹技能包
- 新增触发法术效果查找机制（根据 TriggerSpellID 获取效果定义并执行）
- 添加引导取消联动 Aura 移除（与 Blizzard 一致的 cancel hook 模式）

## Capabilities

### New Capabilities
- `arcane-missiles-skill`: 奥术飞弹技能实现——引导 3 秒，每秒触发一发飞弹法术（Spell 7268），每发 24+0.132×SP 奥术伤害，含技能包、单元测试、时间线测试
- `periodic-trigger-spell`: 周期触发法术机制——AuraPeriodicTriggerSpell 类型定义、TickPeriodic 中的触发路径、触发法术效果执行回调

### Modified Capabilities

（无现有 spec 需要修改）

## Impact

- **pkg/aura/aura.go**: 新增 `AuraPeriodicTriggerSpell` 常量，修改 `TickPeriodic` 添加触发路径
- **pkg/effect/effect.go**: 可能需要暴露效果处理函数供 tick 回调使用
- **skills/arcane-missiles/**: 新建技能包目录
- **无破坏性变更**: 新增功能，不影响现有 Fireball 和 Blizzard
