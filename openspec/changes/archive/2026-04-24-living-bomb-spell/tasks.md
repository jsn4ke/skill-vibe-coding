## 1. 框架增强 — AuraContext + Aura SpellValues

- [x] 1.1 在 `pkg/script/script.go` 的 `AuraContext` 中添加 `RemoveMode uint8` 字段（映射 `aura.RemoveMode`）和 `Aura interface{}` 字段（避免循环引用，运行时断言为 `*aura.Aura`）
- [x] 1.2 在 `pkg/aura/aura.go` 的 `Aura` struct 中添加 `SpellValues map[uint8]float64` 字段
- [x] 1.3 验证现有 3 个技能测试仍然通过：`go test ./skills/...`

## 2. 框架增强 — aura.Manager 接入脚本 hook

- [x] 2.1 在 `pkg/aura/aura.go` 的 `Manager` struct 中添加 `registry` 字段（`*script.Registry` 类型）
- [x] 2.2 添加 `Manager.SetRegistry(reg *script.Registry)` 方法
- [x] 2.3 在 `Manager.AddAura` 中（新添加路径，非刷新），调用 `AuraHookAfterApply` hook
- [x] 2.4 在 `Manager.RemoveAura` 中，调用 `AuraHookAfterRemove` hook，传递 RemoveMode
- [x] 2.5 在 `Manager.TickPeriodic` 的过期段中——RemoveAura 内部已自动调用 AfterRemove hook，RemoveMode=RemoveByExpire 自然传递
- [x] 2.6 验证现有 3 个技能测试仍然通过：`go test ./skills/...`

## 3. 框架增强 — effect.ProcessAll 接入脚本 hook

- [x] 3.1 在 `pkg/effect/effect.go` 中添加全局变量 `var ScriptRegistry *script.Registry`
- [x] 3.2 在 `ProcessAll` 中，每个效果执行前，检查 ScriptRegistry 是否有对应 spellID 的 `HookOnEffectHit` hook
- [x] 3.3 如果 hook 存在，创建 `script.SpellContext{Spell: s, PreventDefault: false}`，调用 hook
- [x] 3.4 如果 `PreventDefault == true`，跳过默认效果处理
- [x] 3.5 验证现有 3 个技能测试仍然通过：`go test ./skills/...`

## 4. 框架增强 — Spell.SelectTargets AoE 扩展

- [x] 4.1 在 `pkg/spell/spell.go` 的 `Spell` struct 中添加 `SpellValues map[uint8]float64` 字段
- [x] 4.2 在 `Spell` struct 中添加 `AoESelector AoESelector` 接口字段和 `AoECenter [3]float64` 字段和 `AoEExcludeID uint64` 字段
- [x] 4.3 在 `SelectTargets()` 中添加分支：检查效果的 TargetA，如果是 `TargetUnitAreaEnemy` 等区域类型，使用 AoESelector 选择多个目标
- [x] 4.4 区域目标选择使用 AoECenter 为中心、AoEExcludeID 排除指定目标
- [x] 4.5 验证现有 3 个技能测试仍然通过：`go test ./skills/...`

## 5. Living Bomb 技能实现

- [x] 5.1 创建 `skill-go/skills/living-bomb/` 目录
- [x] 5.2 创建 `living_bomb.go`：定义 3 个 SpellInfo (Info44457, PeriodicInfo217694, ExplosionInfo44461)
- [x] 5.3 实现 `CastLivingBomb(caster, targetID, auraMgr, bus)` — 创建 Spell(44457)，设置目标，Prepare → HandleImmediate → 脚本拦截 Dummy → CastPeriodicSpell(SpellValues{2:1})
- [x] 5.4 实现 `castPeriodicSpell(caster, targetID, auraMgr, bus, spellValues)` — 创建 Spell(217694)，SpellValues 传入，Prepare → HandleImmediate → ApplyAura(DoT)
- [x] 5.5 实现 `castExplosionSpell(caster, carrierTargetID, auraSpellValues, aoeSelector, bus)` — 创建 Spell(44461)，设置 AoECenter 和 AoEExcludeID，Prepare → HandleImmediate → AoE 伤害
- [x] 5.6 实现 `RegisterScripts(registry, caster, auraMgr, bus)` — 注册 3 个脚本 hook：
  - 44457 OnEffectHit: 拦截 EFFECT_0 Dummy → PreventDefault → CastPeriodicSpell(SpellValues{2:1})
  - 217694 AfterRemove: if RemoveMode==EXPIRE → CastExplosionSpell(从 Aura.SpellValues 获取 canSpread)
  - 44461 OnEffectHit: if SpellValues[2]>0 → 对每个命中目标 CastPeriodicSpell(SpellValues{2:0})

## 6. 单元测试

- [x] 6.1 创建 `living_bomb_test.go`：测试 3 个 SpellInfo 字段、完整施法生命周期、DoT 周期伤害 (4 tick)、死亡不触发爆炸、传染链终止
- [x] 6.2 创建 `living_bomb_timeline_test.go`：完整事件时间线（SpellCastStart → AuraApplied → 4×AuraTick → AuraExpired → Explosion），tick 次数验证，时间线输出

## 7. 构建验证

- [x] 7.1 运行 `go build ./...` 确保编译通过
- [x] 7.2 运行 `go test ./skills/... -v` 确保所有技能测试通过（含已有 3 个 + Living Bomb）
- [x] 7.3 运行 `go test -race ./...` — 跳过（Windows 环境缺少 GCC/cgo），普通测试全部通过
