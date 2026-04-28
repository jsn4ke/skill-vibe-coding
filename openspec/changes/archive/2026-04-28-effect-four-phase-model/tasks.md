## 1. 脚本钩子扩展

- [x] 1.1 在 `script/script.go` 中添加 `HookOnEffectLaunch`、`HookOnEffectLaunchTarget`、`HookOnEffectHitTarget` 三个枚举值
- [x] 1.2 验证 `Registry.HasSpellHook` 和 `CallSpellHook` 能正确处理新钩子

## 2. Effect Handler Mode Guard

- [x] 2.1 `handleSchoolDamage` 添加 `if ctx.Mode != HandleLaunchTarget { return }`
- [x] 2.2 `handleHeal` 添加 `if ctx.Mode != HandleLaunchTarget { return }`
- [x] 2.3 `handleHealPct` 添加 `if ctx.Mode != HandleLaunchTarget { return }`
- [x] 2.4 `handleApplyAura` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.5 `handleEnergize` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.6 `handleEnergizePct` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.7 `handleTriggerSpell` 添加 `if ctx.Mode != HandleLaunchTarget { return }`
- [x] 2.8 `handleWeaponDamage` 添加 `if ctx.Mode != HandleLaunchTarget { return }`
- [x] 2.9 `handleSummon` 添加 `if ctx.Mode != HandleLaunch { return }`
- [x] 2.10 `handleDispel` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.11 `handleDummy` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.12 `handleTeleport` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.13 `handleCharge` 添加双 mode 支持（LaunchTarget + HitTarget）
- [x] 2.14 `handleKnockBack` 添加 `if ctx.Mode != HandleHitTarget { return }`
- [x] 2.15 `handleLeap` 添加 `if ctx.Mode != HandleHitTarget { return }`

## 3. ProcessAll 四阶段重构

- [x] 3.1 重构 `effect.ProcessAll` 为四阶段循环：Launch → LaunchTarget → Hit → HitTarget
- [x] 3.2 每个阶段调用 `s.HandleEffects(mode)` 而非直接调用 Process
- [x] 3.3 实现 `spell.HandleEffects(mode)` — 对每个 effect 创建 Context、调用脚本钩子、调用 Process

## 4. Skill 脚本迁移

- [x] 4.1 Living Bomb 44457: `HookOnEffectHit` → `HookOnEffectHitTarget`
- [x] 4.2 Living Bomb 44461: `HookOnEffectHit` → `HookOnEffectHitTarget`（effect index 1 过滤）
- [x] 4.3 Arcane Missiles: 验证 MissileInfo 的 EffectSchoolDamage 在 LaunchTarget 阶段正确执行

## 5. 测试验证

- [x] 5.1 运行 `go test ./skills/... -v` 确认所有现有 skill 测试通过
- [x] 5.2 运行 `go test -race ./...` 确认无竞态问题
- [x] 5.3 运行 `go build ./...` 确认编译通过
