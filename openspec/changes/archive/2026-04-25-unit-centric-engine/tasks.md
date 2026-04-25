## Phase 1: Unit + Engine 基础结构

- [x] 1.1 创建 `pkg/unit/unit.go`：定义 `Unit` struct（Entity、Stats、History、activeSpells、ownedAuras、appliedAuras、engine 反向引用）
- [x] 1.2 实现 `Unit.Update(diff)` — 调用 updateSpells + updateAuras，顺序与 TC 对齐
- [x] 1.3 实现 `Unit.updateSpells(diff)` — 遍历 activeSpells 调用 Spell.Update(diff)，清理 finished spells
- [x] 1.4 实现 `Unit.updateAuras(diff)` — 遍历 ownedAuras 执行 aura tick，清理 expired auras，触发 OnExpire hook
- [x] 1.5 实现 `Unit.AddActiveSpell(s)` / `Unit.RemoveActiveSpell(s)` — activeSpells 管理
- [x] 1.6 实现 `Unit.AddOwnedAura(a)` / `Unit.AddAppliedAura(a)` — 双注册入口
- [x] 1.7 创建 `pkg/engine/engine.go`：定义 `Engine` struct（units map、timeline renderer、aura manager、event bus、script registry）
- [x] 1.8 实现 `Engine.New(bus, registry)` 构造函数
- [x] 1.9 实现 `Engine.AddUnit(entity, stats)` — 创建 Unit 并注册
- [x] 1.10 实现 `Engine.Tick(diff)` — 统一驱动所有 Unit.Update(diff)
- [x] 1.11 实现 `Engine.GetUnit(id)` — 按 ID 查询
- [x] 1.12 实现 `Engine.GetUnitsInRadius(center, radius)` — 区域查询（area aura 用）
- [x] 1.13 编译验证：`go build ./pkg/...` + 所有现有测试通过

## Phase 2: Aura Manager 重构

- [x] 2.1 重构 `pkg/aura/aura.go` Manager — 添加 `AuraHost` 接口打破循环依赖，保留旧 API 向后兼容
- [x] 2.2 `Manager.ApplyAura(owner, target AuraHost, aura)` — 双注册到 owner.ownedAuras + target.appliedAuras
- [x] 2.3 `Manager.RemoveAuraFromHosts(aura, mode)` — 双向移除 + 触发 hook + 发布事件
- [x] 2.4 `Unit.FindAppliedAura(spellID, casterID)` — 在 unit 包中实现，查询 appliedAuras
- [x] 2.5 `Unit.FindAreaAura(spellID)` / `Unit.FindOwnedAura(spellID, targetID)` — 在 unit 包中实现
- [x] 2.6 `TickPeriodic` 和 `TickPeriodicArea` 保留 — 向后兼容，待 Phase 5-8 迁移后删除
- [x] 2.7 `Aura.Tick(diff, sp, bus, onTick)` 和 `Aura.TickArea(...)` 方法 — 在 Aura 上实现，Unit.updateAuras 调用
- [x] 2.8 编译验证：`go build ./...` + 所有现有测试通过

## Phase 3: Engine.CastSpell — 三条执行路径

- [x] 3.1 实现 `Engine.CastSpell(caster, spellInfo, opts)` — 统一的法术施放入口
- [x] 3.2 路径 A: triggered instant — Prepare 内 Cast(true) 同帧完成，不入 activeSpells
- [x] 3.3 路径 B: normal instant — 注册到 caster.activeSpells + Prepare 内同步完成
- [x] 3.4 路径 C: delayed — 注册到 caster.activeSpells + 设置 timer，等 Engine.Advance 驱动
- [x] 3.5 `CastOption` 函数选项模式 — WithTarget、WithDestPos、WithSpellValues、WithAoE、WithTriggered
- [x] 3.6 编译验证：`go build ./pkg/...`

## Phase 4: Timeline Renderer 接入 Engine

- [x] 4.1 Engine.Advance 中调用 `renderer.SetTime(currentTime)` 在驱动 Unit.Update 之前
- [x] 4.2 Engine 持有时间累加器 `currentTime time.Duration`，每次 Advance 累加 diff
- [x] 4.3 编译验证：`go build ./pkg/...`

## Phase 5: Fireball 迁移（验证用技能）

- [x] 5.1 创建 `skills/fireball/fireball_engine_test.go` — 使用 Engine.Advance 驱动的新测试
- [x] 5.2 保留旧 CastFireball + legacy tests — 向后兼容
- [x] 5.3 Engine-driven timeline: CastSpell → Advance loop → aura creation → tick loop
- [x] 5.4 测试验证：`go test ./skills/fireball/... -v` — 新旧测试全部通过
- [x] 5.5 时间线验证：0ms CastStart, 3500ms Launch, 4000ms Hit, 6000/8000/10000/12000ms AuraTick, 12000ms Expired

## Phase 6: Arcane Missiles 迁移

- [x] 6.1 重构 `skills/arcane-missiles/arcane_missiles.go` — CastArcaneMissiles 改为调用 engine.CastSpell
- [x] 6.2 处理引导法术 + PeriodicTriggerSpell aura tick 触发子法术的流程
- [x] 6.3 重构测试文件
- [x] 6.4 测试验证：`go test ./skills/arcane-missiles/... -v`

## Phase 7: Blizzard 迁移

- [x] 7.1 重构 `skills/blizzard/blizzard.go` — CastBlizzard 改为调用 engine.CastSpell
- [x] 7.2 处理 AoE area aura tick 在 Unit.updateAuras 中的 resolve targets
- [x] 7.3 重构测试文件
- [x] 7.4 测试验证：`go test ./skills/blizzard/... -v`

## Phase 8: Living Bomb 迁移

- [x] 8.1 重构 `skills/living-bomb/living_bomb.go` — 三体结构改为 engine.CastSpell
- [x] 8.2 脚本 hook 中触发新法术走 engine.CastSpell（triggered instant 路径）
- [x] 8.3 重构测试文件
- [x] 8.4 测试验证：`go test ./skills/living-bomb/... -v`

## Phase 9: 全局验证 + server/main.go

- [x] 9.1 所有技能测试通过：`go test ./skills/... -v`
- [x] 9.2 Race detection：`go test -race ./...`
- [x] 9.3 重构 `server/main.go` — Engine 初始化示例，替代空壳
- [x] 9.4 删除 `server/main.go` 中的临时 Unit 定义（已迁移到 pkg/unit）
- [x] 9.5 清理：移除 aura.Manager 中已废弃的 TickPeriodic/TickPeriodicArea 方法
