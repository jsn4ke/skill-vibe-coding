## Context

skill-go 是一个 WoW 风格的战斗模拟框架，已实现 4 个技能（Fireball、Blizzard、Arcane Missiles、Living Bomb）。当前架构存在根本性问题：**没有统一的 Update 驱动**。

时间推进分散在三处：Cast* 函数内部自驱动 Update、测试循环手动驱动 auraMgr.TickPeriodic、脚本 hook 中同步创建并驱动子法术。这导致每个技能的测试循环都是手工编写的、不可复用的模拟循环。

参考文档：`tc-references/unit-update-architecture.md`（TC 的 Unit 中心驱动架构完整分析）。

## Why

当前架构的三个核心问题：

1. **无统一驱动** — 每个 Cast* 函数自己调用 `spell.Update()`，每个测试循环自己写 `for simMs` 循环驱动 aura tick。Engine 不存在。
2. **无 Unit 持有关系** — aura.Manager 是全局 flat map（按 targetID 分组），没有 owned/applied 分离。施法者死亡时无法级联清理其拥有的 aura。AoE aura 的 tick 由 Manager 统一驱动，而非由 owner 驱动。
3. **Cast 函数自驱动** — Cast* 函数内部 `Prepare() → Update(CastTime) → Update(HitTimer)` 完整跑完 spell 生命周期，违反 TC 的"Engine 是唯一驱动者"原则。

TC 的做法是：Map.Update → Unit.Update → _UpdateSpells（清理 finished spells → 遍历 ownedAuras.UpdateOwner → 清理 expired auras）。所有时间推进来自唯一入口。

## What Changes

- 新建 `pkg/engine/` 包 — 统一的 `Engine` 结构，持有 `Unit` 列表，提供 `Tick(diff)` 方法驱动全局时间推进
- 新建 `pkg/unit/` 包 — `Unit` 结构，持有 `activeSpells`、`ownedAuras`、`appliedAuras`、`spellHistory`，提供 `Update(diff)` 方法
- 重构 `pkg/aura/` — Manager 的 Tick 逻辑下沉到 Unit 级别（Unit.UpdateAuras），Manager 保留为 Aura 工厂和全局查询
- 重构 `pkg/spell/` — Cast* 函数不再自驱动 Update，改为注册到 Unit 的 activeSpells 列表
- 实现三条执行路径 — triggered instant（同帧同步完成）、normal instant（同帧带注册）、delayed（注册 + Update 驱动）
- 重构所有技能的测试 — 统一使用 `engine.Tick(step)` 替代手动模拟循环

## Capabilities

### New Capabilities

- `engine`: 统一游戏引擎 — 持有 Unit 列表、统一 Tick(diff) 驱动、时间推进管理
- `unit`: Unit 实体 — 以 Unit 为中心的 spell/aura 所有权，Update 驱动链（spells → auras → cleanup）

### Modified Capabilities

- `aura-manager`: 从全局 flat tick 改为 Unit 级别驱动，保留 AddAura/RemoveAura/Find 等查询 API
- `spell-lifecycle`: Cast* 函数改为注册模式，不再自驱动 Update；instant 同帧 / 非瞬发走 Update 链
- `skill-tests`: 所有技能测试改为 engine.Tick 驱动，删除手动模拟循环

## Impact

- **pkg/engine/** (new): Engine struct, Tick method, time management
- **pkg/unit/** (new): Unit struct, activeSpells/ownedAuras/appliedAuras, Update method chain
- **pkg/aura/**: Manager 简化为工厂 + 查询层；tick 逻辑迁移到 Unit
- **pkg/spell/**: Spell.Update 保持不变；CastXxx 函数签名和流程变更
- **pkg/timeline/**: Renderer 接入 Engine（通过 Engine.SetTime 统一推进）
- **skills/**: 所有 4 个技能的 CastXxx 函数和测试重构
- **server/main.go**: 从空壳改为 Engine 初始化示例
- **Breaking change**: 所有 CastXxx 函数签名变更，测试需全部改写
