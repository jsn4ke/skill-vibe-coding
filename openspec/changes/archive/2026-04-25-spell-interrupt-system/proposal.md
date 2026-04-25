## Why

当前技能系统缺乏中断机制：施法者移动不会打断吟唱，目标离开范围不会取消法术，引导法术不会因目标消失而终止。TC 的中断系统是三层架构（Spell 级持续检查 + Aura 级事件驱动 + cancel 清理），我们只有 cancel() 的基本框架。这导致 Fireball 移动不中断、Blizzard 目标进出区域不响应、Arcane Missiles 目标死亡不终止等核心体验缺失。

## What Changes

- **移动基础设施**: Unit 支持 SetPosition() 位置变更，内部追踪 `isMoving` 状态（基于位置是否在本 frame 发生变化）
- **SpellInterruptFlags**: 施法中断标志系统（Movement、DamageCancels、DamagePushback），SpellInfo 携带标志，Spell.Update() 每帧检查
- **Spell.Update() 持续验证**: 目标消失检测、目标范围检查（PREPARING/CHANNELING 状态每 tick 验证条件）
- **Channel 目标动态维护**: UpdateChanneledTargetList() 等价逻辑，每 tick 检查 channel 目标是否存活/在范围内，不满足则移除对应 aura，全部移除则终止 channel
- **SpellAuraInterruptFlags**: Aura 中断标志系统（Moving、Damage、Action 等），AuraInfo 携带标志
- **RemoveAurasWithInterruptFlags()**: 事件驱动的 aura 移除，由移动/受伤/状态变化等事件触发，同时检查 channel 法术
- **cancel() 状态特定清理**: PREPARING 中断与 CHANNELING 中断的不同清理路径，参考 TC 的 oldState 恢复模式
- **引擎测试覆盖**: 所有四个技能的中断测试（移动打断、目标离范围、目标死亡、channel 清理）

## Capabilities

### New Capabilities
- `spell-interrupt`: Spell 级中断机制 — SpellInterruptFlags、Spell.Update() 中的持续条件验证（移动打断、目标消失、范围检查）、cancel() 状态特定清理
- `aura-interrupt`: Aura 级中断机制 — SpellAuraInterruptFlags、RemoveAurasWithInterruptFlags()、事件驱动的 aura 移除
- `unit-movement-basic`: 基础移动支持 — Unit.SetPosition()、isMoving 追踪、位置变化标记

### Modified Capabilities
- `script-system-hooks`: RegisterScripts 需要处理新的中断事件（OnInterrupt hook 或通过 AuraHookAfterRemove with RemoveByInterrupt）

## Impact

- **pkg/spell/**: SpellInfo 加 InterruptFlags 字段，Spell.Update() 加中断检查逻辑，CheckRange/CheckMovement 方法
- **pkg/aura/**: Aura 加 InterruptFlags，AuraManager 加 RemoveAurasWithInterruptFlags()
- **pkg/unit/**: SetPosition() + isMoving 追踪，Update() 中触发移动中断
- **pkg/engine/**: Engine 中集成中断触发点（Advance 中调用 InterruptMovementBasedAuras 等）
- **skills/**: 四个技能的 SpellInfo 添加正确的 InterruptFlags，新增中断相关的 engine test
- **pkg/spell/info.go**: 新增 SpellInterruptFlags / SpellAuraInterruptFlags 类型定义
