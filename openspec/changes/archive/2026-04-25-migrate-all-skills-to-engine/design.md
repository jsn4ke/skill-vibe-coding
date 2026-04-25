## Context

Engine + Unit 架构已经到位。4 个技能都有 engine_test 文件。问题是 legacy CastXxx 函数和旧测试还在，script hooks 走 legacy 路径。这个 change 是纯迁移 — 不引入新架构，只是让所有代码走已有的 engine 路径。

## Goals / Non-Goals

**Goals:**
- 删除所有 CastXxx legacy 函数（7 个）
- Script hooks 内部触发法术走 Engine.CastSpell(WithTriggered)
- 所有测试统一使用 Engine，删除 testUnit mock
- Engine test 中移除手动 aura 构建 hack
- `go test -race ./...` 全部通过

**Non-Goals:**
- 不实现 SpellEffectHandler 自动 aura 创建管线（初版保留 engine_test 中通过 hook 或手动创建 aura 的模式）
- 不删除 aura.Manager 的 legacy API（可能被其他包引用，但技能代码不再使用）
- 不重构 Spell.Update 内部逻辑
- 不引入新技能

## Decisions

### Decision 1: CastXxx 函数删除策略

**选择**: 直接删除所有 CastXxx 函数，不提供 wrapper。

**理由**: 这些函数的调用方只有旧测试和 script hooks。旧测试将被删除，script hooks 将改为直接调 `Engine.CastSpell()`。没有外部调用方需要兼容。

**涉及函数**:
- `fireball.CastFireball`
- `blizzard.CastBlizzard`
- `arcanemissiles.CastArcaneMissiles`
- `arcanemissiles.CastTriggeredSpell`
- `livingbomb.CastLivingBomb`
- `livingbomb.castPeriodicSpell`
- `livingbomb.castExplosionSpell`

### Decision 2: Script hooks 如何获取 Engine 引用

**选择**: RegisterScripts 接收 `*engine.Engine` 替代 `auraMgr + bus + aoeSelector` 散参数。

**理由**: 当前 RegisterScripts 接收 `caster, auraMgr, bus, aoeSelector` 四个参数，这些都可以从 Engine 获取。hook 内部需要调 `eng.CastSpell()` 来触发子法术。统一传入 Engine 让 hook 内部能访问所有子系统。

**实现**:
```go
// Before
func RegisterScripts(registry *script.Registry, caster spell.Caster,
    auraMgr *aura.Manager, bus *event.Bus, aoeSelector spell.AoESelector)

// After
func RegisterScripts(registry *script.Registry, caster *unit.Unit, eng *engine.Engine)
```

**Hook 内部触发法术**:
```go
// castPeriodicSpell → eng.CastSpell with WithTriggered + WithTarget
eng.CastSpell(caster, &PeriodicInfo,
    engine.WithTarget(targetID),
    engine.WithTriggered(),
    engine.WithSpellValues(spellValues),
)

// castExplosionSpell → eng.CastSpell with WithTriggered + WithAoE
eng.CastSpell(caster, &ExplosionInfo,
    engine.WithTriggered(),
    engine.WithAoE(selector, center, excludeID),
    engine.WithSpellValues(map[uint8]float64{2: canSpread}),
)
```

### Decision 3: Triggered instant spell 的 aura 创建

**选择**: Triggered instant spell 的 aura 创建通过 OnEffectHit hook 完成，不通过手动构建。

**理由**: 当前 engine_test 中手动构建 aura 是因为 legacy `castPeriodicSpell` 走 `auraMgr.AddAura` 不注册到 Unit。迁移后，hook 内部调用 `eng.CastSpell(WithTriggered)` → triggered instant path → `HandleImmediate()` → 效果处理。效果处理需要能识别 `EffectApplyAura` 并调用 `eng.AuraMgr().ApplyAura()`。

**两条实现路径**:

**路径 A (理想)**: Spell 效果管线自动处理 EffectApplyAura → 创建 aura → ApplyAura。这需要扩展 Spell.HandleImmediate 或添加 EffectHandler。

**路径 B (务实)**: 保持现有 script hook 模式 — 在 OnEffectHit hook 中拦截 Dummy effect，手动构建 aura 并调用 `eng.AuraMgr().ApplyAura(caster, target, a)`。

**选择路径 B** — 最小改动，与现有 hook 模式一致，不需要引入效果管线。

### Decision 4: Legacy tests 删除策略

**选择**: 完全删除 `*_test.go`（非 engine），将其中有价值的断言迁移到 `*_engine_test.go`。

**理由**: Legacy tests 的断言逻辑（伤害计算、aura 属性、cancel 行为等）有价值，但测试框架（testUnit mock、手动 Tick）需要替换。迁移策略是确保 engine_test 覆盖 legacy test 的所有断言点。

**各技能迁移要点**:

| 技能 | Legacy 断言 | Engine Test 覆盖方式 |
|------|-------------|---------------------|
| Fireball | 伤害值、projectile delay、DoT aura 属性、tick 伤害、cancel | timeline event 验证 + 数值断言 |
| Blizzard | aura 属性、tick 伤害、cancel、area targets | timeline event 验证 + tick count |
| Arcane Missiles | aura 属性、missile 伤害、3 hits、mana、cancel | timeline event 验证 |
| Living Bomb | aura 属性、spread、chain terminate、no explode on dispel/death | timeline event 验证 |

### Decision 5: Engine test 不再手动构建 aura

**选择**: Engine test 通过 script hooks 自动创建 aura，不手动构建。

**理由**: 当前 engine_test 中手动构建 aura 是因为 script hooks 走 legacy 路径。迁移 hooks 后，`eng.CastSpell()` → hook 触发 → 自动创建 aura。测试只需调 `eng.CastSpell()` + `eng.Simulate()` 即可。

**Before**:
```go
eng.CastSpell(caster, &Info, engine.WithTarget(2))
// 手动构建 aura
a := aura.NewAura(...)
eng.AuraMgr().ApplyAura(caster, target, a)
eng.Simulate(5000, 100)
```

**After**:
```go
eng.CastSpell(caster, &Info, engine.WithTarget(2))
// hooks 自动创建 aura → engine 自动驱动 tick
eng.Simulate(5000, 100)
```

### Decision 6: Channeled spell 的 aura 绑定

**选择**: Channeled spell（Blizzard、Arcane Missiles）的 aura 创建通过 OnSpellLaunch 或 StateChanneling entry hook 完成。

**理由**: Channeled spell 进入 StateChanneling 后需要创建 aura 并绑定到 OnCancel 清理。当前在 CastBlizzard 内部完成，迁移后在 script hook 中完成。

**实现**: RegisterScripts 注册 StateChanneling entry hook（或 OnSpellLaunch hook），在 hook 中创建 aura 并设置 spell.OnCancel。

## Risks / Trade-offs

**[Hook 内部需要 engine 引用]** → RegisterScripts 签名变更，所有调用方需更新。当前只有 engine_test 调用，影响可控。

**[Triggered spell 的 aura 创建依赖 hook 链完整性]** → 如果 hook 链断裂（注册遗漏、PreventDefault 误用），aura 不会被创建。缓解：每个技能的 engine_test 验证 aura 存在性。

**[Channeled spell 的 OnCancel 绑定]** → spell.OnCancel 需要在 hook 中设置，如果 hook 执行时 spell 还没进入 Channeling 状态，绑定会失败。缓解：hook 注册在 OnSpellLaunch 事件，此时 spell 已进入 Channeling。

**[Living Bomb 三层 hook 链]** → 44457 (OnEffectHit) → 217694 (AfterRemove) → 44461 (OnEffectHit)，每一层都通过 eng.CastSpell 触发下一层。需要确保 triggered instant 路径下 Prepare → HandleImmediate → Finish 流程中 hook 能正确触发。
