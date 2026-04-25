## Context

上一个 change（`unit-centric-engine`）引入了 Engine + Unit 架构，但保留了所有 legacy CastXxx 函数和旧测试作为向后兼容。结果是两套系统并存：engine-driven 测试和 legacy 测试各自跑，技能代码里 CastXxx 函数仍然自己调 `s.Update()` 和 `auraMgr.AddAura()`。

## Why

Legacy 代码直接违反了已建立的架构原则：

1. **Cast 函数自驱动 Update** — `CastFireball`、`CastBlizzard`、`CastArcaneMissiles`、`CastLivingBomb`、`castPeriodicSpell`、`castExplosionSpell`、`CastTriggeredSpell` 全部内部调用 `s.Update()`，绕过 Engine
2. **auraMgr.AddAura() 走 legacy 全局 map** — aura 不注册到 Unit.ownedAuras / Unit.appliedAuras，Engine.Advance 驱动不到
3. **Script hooks 内部触发法术走 legacy** — AfterRemove hook 调 `castExplosionSpell()`，OnEffectHit hook 调 `castPeriodicSpell()`，这些函数都不走 engine
4. **两套测试并存** — legacy test 用 `testUnit` mock + 手动 `a.Tick()`，engine test 手动构建 aura 绕过 script hook。两套都不完整

Engine 架构已经到位且可用，legacy 代码现在是纯粹的债务。

## What Changes

- 删除所有 `CastXxx` legacy 入口函数（7 个函数）
- Script hooks 内部改为调用 `Engine.CastSpell(triggered)` 路径
- 删除所有 legacy `*_test.go`（4 个技能各一个），仅保留 `*_engine_test.go`
- 删除所有 `testUnit` / `testPos` mock 类型
- Engine test 中移除手动 aura 构建 hack，改由 script hook 或 engine 效果管线自动创建 aura
- `aura.Manager` 的 legacy 方法（AddAura、RemoveAura、FindAura 等）标记为最终删除或保留为纯查询 API

## Capabilities

### Modified Capabilities

- `skill-cast-entry`: 所有技能入口统一为 `eng.CastSpell()` + SpellInfo，无 per-skill Cast 函数
- `script-hooks-triggered`: Script hook 内部触发法术走 `Engine.CastSpell(WithTriggered)` 路径
- `skill-tests`: 所有测试统一使用 Engine + Unit，无 mock，无手动 aura 构建

## Impact

- **skills/fireball/**: 删除 CastFireball、legacy test、testUnit mock
- **skills/blizzard/**: 删除 CastBlizzard、legacy test、testUnit mock
- **skills/arcane-missiles/**: 删除 CastArcaneMissiles + CastTriggeredSpell、legacy test、testUnit mock
- **skills/living-bomb/**: 删除 CastLivingBomb + castPeriodicSpell + castExplosionSpell、legacy test、testUnit mock
- **pkg/aura/**: legacy AddAura/RemoveAura/FindAura 不再被技能代码使用
- **pkg/engine/**: 可能需要增加 triggered spell 的 aura 创建支持
- **pkg/spell/**: SpellInfo 保留，Spell 生命周期不变
