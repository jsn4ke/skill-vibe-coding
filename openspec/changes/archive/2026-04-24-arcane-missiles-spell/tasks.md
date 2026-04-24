## 1. 框架扩展 — PeriodicTriggerSpell 机制

- [x] 1.1 在 `pkg/aura/aura.go` 中新增 `AuraPeriodicTriggerSpell` 常量（在 AuraProcTriggerSpell 之前）
- [x] 1.2 在 `AuraEffect` struct 中添加 `TriggerSpellID uint32` 字段
- [x] 1.3 在 `pkg/effect/effect.go` 的 `handleApplyAura` 中，将 `SpellEffectInfo.TriggerSpellID` 传播到 AuraEffect 的 TriggerSpellID 字段

## 2. 奥术飞弹技能实现

- [x] 2.1 创建 `skill-go/skills/arcane-missiles/` 目录和 `arcane_missiles.go`
- [x] 2.2 定义父法术 `Info` SpellInfo（ID 5143，引导 3s，85 法力，EffectApplyAura + AuraPeriodicTriggerSpell + TriggerSpellID 7268）
- [x] 2.3 定义触发法术 `MissileInfo` SpellInfo（ID 7268，instant，EffectSchoolDamage, BasePoints=24, BonusCoeff=0.132）
- [x] 2.4 实现 `CastTriggeredSpell(caster Caster, targetID uint64, info *spell.SpellInfo, bus *event.Bus)` 辅助函数：创建 Spell → TriggeredFullMask → 设置目标 → 设置 Bus → Prepare → 自动走完 HandleImmediate → Finish
- [x] 2.5 实现 `CastArcaneMissiles(caster, targetID, auraMgr, bus)` 函数：创建父 Spell → 设置目标 → Prepare → 进入引导 → 创建 PeriodicTriggerSpell Aura → 注册 OnCancel hook → 模拟 3s 引导，每秒 tick 调用 CastTriggeredSpell(MissileInfo)

## 3. 单元测试

- [x] 3.1 创建 `arcane_missiles_test.go`：测试父法术 SpellInfo 字段、触发法术 MissileInfo 字段、完整施法生命周期、每发飞弹伤害计算 (24+0.132×SP)、独立暴击判定、引导取消行为
- [x] 3.2 创建 `arcane_missiles_timeline_test.go`：测试完整事件时间线（SpellCastStart → SpellLaunch → 3×SpellHit → AuraExpired）、tick 次数验证、时间线输出

## 4. 构建验证

- [x] 4.1 运行 `go build ./...` 确保编译通过
- [x] 4.2 运行 `go test ./skills/... -v` 确保所有测试通过
- [x] 4.3 运行 `go test -race ./...` 确保无竞态问题
