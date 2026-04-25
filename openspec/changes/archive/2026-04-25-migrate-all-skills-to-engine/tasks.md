## Phase 1: Fireball 完全迁移

- [x] 1.1 重构 `skills/fireball/fireball.go` — 删除 `CastFireball` 函数，只保留 `Info` 定义
- [x] 1.2 重写 `skills/fireball/fireball_engine_test.go` — 覆盖 legacy test 所有断言（伤害值、projectile delay、DoT aura 属性、tick 伤害、cancel）
- [x] 1.3 删除 `skills/fireball/fireball_test.go` 和 testUnit/testPos mock
- [x] 1.4 验证：`go test ./skills/fireball/... -v` 通过

## Phase 2: Blizzard 完全迁移

- [x] 2.1 重构 `skills/blizzard/blizzard.go` — 删除 `CastBlizzard` 函数，添加 RegisterScripts (OnSpellLaunch + OnSpellCancel)
- [x] 2.2 channeled spell hook — OnSpellLaunch bus listener 创建 area aura + OnSpellCancel 清理
- [x] 2.3 重写 `skills/blizzard/blizzard_engine_test.go` — 覆盖 aura 属性、tick 伤害、cancel、area targets
- [x] 2.4 删除 `skills/blizzard/blizzard_test.go` 和 testUnit/testPos mock
- [x] 2.5 验证：`go test ./skills/blizzard/... -v` 通过

## Phase 3: Arcane Missiles 完全迁移

- [x] 3.1 重构 `skills/arcane-missiles/arcane_missiles.go` — 删除 `CastArcaneMissiles` + `CastTriggeredSpell`，添加 RegisterScripts
- [x] 3.2 PeriodicTriggerSpell — OnAuraTick bus listener 触发 missile，triggerSpellID 通过 Unit.tickSingleAura Extra 传递
- [x] 3.3 重写 `skills/arcane-missiles/arcane_missiles_engine_test.go` — 覆盖 aura 属性、missile 伤害、3 hits、mana、cancel
- [x] 3.4 删除 `skills/arcane-missiles/arcane_missiles_test.go` 和 testUnit/testPos mock
- [x] 3.5 验证：`go test ./skills/arcane-missiles/... -v` 通过

## Phase 4: Living Bomb 完全迁移

- [x] 4.1 重构 `skills/living-bomb/living_bomb.go` — 删除 `CastLivingBomb`、`castPeriodicSpell`、`castExplosionSpell`
- [x] 4.2 重写 `RegisterScripts` — 接收 `*unit.Unit` + `*engine.Engine`
- [x] 4.3 OnEffectHit hook (44457) — 调 `eng.CastSpell(caster, &PeriodicInfo, WithTriggered, WithTarget)`
- [x] 4.4 AfterRemove hook (217694) — 调 `eng.CastSpell(caster, &ExplosionInfo, WithTriggered, WithAoE)`
- [x] 4.5 OnEffectHit hook (44461) — 调 `eng.CastSpell(caster, &PeriodicInfo, WithTriggered, WithTarget)` 用于 spread
- [x] 4.6 Engine test 中移除手动 aura 构建 — hooks + effect pipeline 自动创建 aura
- [x] 4.7 重写 `skills/living-bomb/living_bomb_engine_test.go` — 覆盖 spread、chain terminate、no explode on dispel/death
- [x] 4.8 删除 `skills/living-bomb/living_bomb_test.go` 和 testUnit/testPos mock
- [x] 4.9 验证：`go test ./skills/living-bomb/... -v` 通过

## Phase 5: Engine 补充 + Hook 基础设施

- [x] 5.1 OnAuraCreated callback on Spell — effect pipeline 自动注册 aura (不需要额外 hook 点)
- [x] 5.2 Channeled spell — OnSpellLaunch bus listener (不需要新 hook 类型)
- [x] 5.3 PeriodicTriggerSpell — OnAuraTick bus listener + triggerSpellID in Extra
- [x] 5.4 effect.ScriptRegistry wired to engine's registry in engine.New()
- [x] 5.5 Unit.ModifyPower implemented (was stub), TC power type mapping (0→Mana)
- [x] 5.6 DestPos included in OnSpellLaunch event Extra

## Phase 6: 全局验证 + 清理

- [x] 6.1 所有技能测试通过：`go test ./skills/...`
- [x] 6.2 `go vet ./...` 通过
- [x] 6.3 aura.Manager legacy 方法无技能代码引用
- [x] 6.4 无 s.Update() 自驱动调用残留
- [x] 6.5 无 testUnit/testPos mock 残留
