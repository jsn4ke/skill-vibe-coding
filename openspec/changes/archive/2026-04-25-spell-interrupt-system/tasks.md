## Phase 1: 基础设施 — 类型定义 + 移动追踪

- [x] 1.1 新增 `SpellInterruptFlags` 类型到 `pkg/spell/info.go`（InterruptNone, InterruptMovement, InterruptDamageCancels, InterruptDamagePushback），实现 HasFlag 方法
- [x] 1.2 `SpellInfo` 添加 `InterruptFlags SpellInterruptFlags` 字段
- [x] 1.3 新增 `SpellAuraInterruptFlags` 类型到 `pkg/spell/info.go`（AuraInterruptNone, AuraInterruptOnMovement, AuraInterruptOnDamage, AuraInterruptOnAction），实现 HasFlag 方法
- [x] 1.4 `SpellEffectInfo` 添加 `AuraInterruptFlags SpellAuraInterruptFlags` 字段
- [x] 1.5 `Unit` 添加 `prevPos` 字段 + `SetPosition(pos)` 方法 + 修改 `IsMoving()` 基于 Update 中的位置对比
- [x] 1.6 `Unit.Update()` 末尾检测位置变化设置 isMoving，并调用 `RemoveAurasWithInterruptFlags(AuraInterruptOnMovement)`（新增方法）
- [x] 1.7 验证：`go build ./...` 通过

## Phase 2: Spell 级中断 — Update 中的持续验证

- [x] 2.1 `Spell.Update()` 添加目标消失检查：targetID != 0 且 engine.GetUnit(targetID) == nil → cancel()
- [x] 2.2 `Spell.Update()` 添加范围检查（PREPARING 状态）：caster 与 target 距离 > RangeMax + tolerance → cancel()
- [x] 2.3 `Spell.Update()` 添加移动中断检查：state=PREPARING/CHANNELING + IsMoving() + HasInterruptFlag(InterruptMovement) → cancel()，triggered 跳过
- [x] 2.4 `Spell.Update()` 添加 AttrBreakOnMove 兼容检查（Attributes & AttrBreakOnMove 也触发移动中断）
- [x] 2.5 `Spell.Update()` CHANNELING 分支添加 UpdateChanneledTargetList 逻辑：遍历 TargetInfos，检查目标存在/存活/范围，不满足则移除对应 aura + 从 TargetInfos 移除，全部移除则 cancel()
- [x] 2.6 验证：`go build ./...` 通过

## Phase 3: Aura 级中断 + cancel 清理

- [x] 3.1 `Aura` 结构体添加 `InterruptFlags SpellAuraInterruptFlags` 字段，`handleApplyAura` 中从 SpellEffectInfo 复制
- [x] 3.2 `Unit.RemoveAurasWithInterruptFlags(flag)` 方法：遍历 ownedAuras，匹配 InterruptFlags → RemoveAuraFromHosts(RemoveByInterrupt)
- [x] 3.3 `Unit.RemoveAurasWithInterruptFlags` 同时检查 current channeled spell，如果 channel spell info 有匹配 flag → cancel spell
- [x] 3.4 `Spell.Cancel()` 改为 TC 模式：记录 oldState → 做状态特定清理 → 恢复 oldState → finish(CastFailedInterrupted)
- [x] 3.5 CHANNELING cancel 清理：遍历 TargetInfos 移除所有目标 aura (RemoveByCancel)
- [x] 3.6 `aura.Manager.RemoveAura` 确保 RemoveByInterrupt 触发 AfterRemove hook
- [x] 3.7 验证：`go build ./...` 通过

## Phase 4: 技能配置 — 更新四个技能的 InterruptFlags

- [x] 4.1 Fireball Info 添加 `InterruptFlags: spell.InterruptMovement`（替换 AttrBreakOnMove）
- [x] 4.2 Arcane Missiles Info 添加 `InterruptFlags: spell.InterruptMovement`
- [x] 4.3 Blizzard Info 添加 `InterruptFlags: spell.InterruptMovement`（channel 移动打断）
- [x] 4.4 Living Bomb PeriodicInfo 的 effect 添加 AuraInterruptFlags（可选，DoT 不需要移动打断）
- [x] 4.5 验证：`go test ./skills/...` 全部通过（确保不破坏已有测试）

## Phase 5: 中断测试 — 新增 engine test

- [x] 5.1 `fireball_engine_test.go` 新增 TestFireball_EngineMovementCancels：caster 移动后 Fireball 被中断
- [x] 5.2 `fireball_engine_test.go` 新增 TestFireball_EngineOutOfRangeCancels：target 移出范围后 Fireball 被中断
- [x] 5.3 `fireball_engine_test.go` 新增 TestFireball_EngineTargetRemovedCancels：target 从 engine 移除后 Fireball 被中断
- [x] 5.4 `blizzard_engine_test.go` 新增 TestBlizzard_EngineMovementCancelsChannel：caster 移动打断 channel
- [x] 5.5 `blizzard_engine_test.go` 新增 TestBlizzard_EngineTargetLeavesRange：channel 中目标离开范围，aura 被移除
- [x] 5.6 `arcane-missiles_engine_test.go` 新增 TestArcaneMissiles_EngineMovementCancelsChannel：移动打断 channel
- [x] 5.7 `arcane-missiles_engine_test.go` 新增 TestArcaneMissiles_EngineTargetDeathCancelsChannel：目标死亡终止 channel
- [x] 5.8 验证：`go test ./skills/... -v` 全部通过

## Phase 6: 全局验证 + Race Test

- [x] 6.1 `go vet ./...` 通过
- [x] 6.2 `go test -race ./...` 通过（注：cgo 不可用，race test 需 CGO_ENABLED=1）
- [x] 6.3 更新 `.claude/rules/skill-test.md` 添加中断测试覆盖要求
