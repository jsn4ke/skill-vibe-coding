## Context

当前 effect 系统在 `effect.ProcessAll()` 中对所有 effect 做一次性处理，没有区分 Launch 和 Hit 阶段。`spell.HandleEffects(mode)` 是空壳（`_ = mode`）。TC 的 effect pipeline 有四个阶段，每个 effect handler 内部用 mode guard 决定是否执行。详细机制参考 `tc-references/effect-pipeline.md`。

现有文件：
- `skill-go/pkg/effect/effect.go` — ProcessAll + 15 个 effect handler（全部无 mode guard）
- `skill-go/pkg/spell/spell.go` — HandleEffects 空壳、EffectHandleMode 枚举已定义
- `skill-go/pkg/script/script.go` — 只有 HookOnEffectHit，缺少 Launch/LaunchTarget/HitTarget
- `skill-go/pkg/engine/engine.go` — CastSpell / Advance 驱动链

约束：仅使用当前机制，不引入新抽象。TC 的模式是 handler 内部 switch mode，不是分发层路由。

## Goals / Non-Goals

**Goals:**
- 实现 TC 四阶段 effect 分发：Launch → LaunchTarget → Hit → HitTarget
- 每个 effect handler 按 TC 映射绑定正确的 mode（handler 内部 mode guard）
- 新增 3 个脚本钩子（HookOnEffectLaunch / HookOnEffectLaunchTarget / HookOnEffectHitTarget）
- 迁移现有 skill 的脚本注册到正确的钩子
- 所有现有测试通过

**Non-Goals:**
- 不引入新抽象层（不用 effect 接口、不用分发表按 mode 路由）
- 不实现 TC 的 Damage/Healing 中间存储（Launch 阶段计算存到 TargetInfo，Hit 阶段施加）——当前阶段直接在 HIT_TARGET 施加
- 不实现 BeforeHit / OnHit / AfterHit 目标级钩子（超出本次范围）
- 不实现 CalcCritChance / CalcDamage 计算阶段（超出本次范围）
- 不新增 SpellAttribute 规则约束（超出本次范围）

## Decisions

### Decision 1: Handler 内部 mode guard（对齐 TC）

每个 effect handler 的第一行添加 mode guard：`if ctx.Mode != X { return }`。

**理由**: TC 的模式。Handler 在 4 个阶段都被调用，但只在匹配的 mode 执行。不需要修改分发层。

**替代方案**: 分发层维护 mode→handler 映射表。否决原因：与 TC 不一致，且新增 effect 需要修改两个地方。

Mode 映射表（来自 TC 分析）：

| Effect | Mode |
|--------|------|
| EffectSchoolDamage | LaunchTarget |
| EffectHeal | LaunchTarget |
| EffectHealPct | LaunchTarget |
| EffectApplyAura | HitTarget |
| EffectEnergize | HitTarget |
| EffectEnergizePct | HitTarget |
| EffectTriggerSpell | LaunchTarget |
| EffectWeaponDamage | LaunchTarget |
| EffectSummon | Launch |
| EffectDispel | HitTarget |
| EffectDummy | HitTarget |
| EffectTeleportUnits | HitTarget |
| EffectCharge | LaunchTarget + HitTarget |
| EffectKnockBack | HitTarget |
| EffectLeap | HitTarget |

### Decision 2: ProcessAll 重构为四阶段循环

重构 `ProcessAll` 为按阶段顺序处理：
1. 对每个 effect 调用 HandleEffects(Launch) — 无目标
2. 对每个 target + 每个 effect 调用 HandleEffects(LaunchTarget) — 有目标
3. 对每个 effect 调用 HandleEffects(Hit) — 无目标
4. 对每个 target + 每个 effect 调用 HandleEffects(HitTarget) — 有目标

**理由**: 对齐 TC 的 handle_immediate → HandleLaunchPhase → _handle_immediate_phase + DoProcessTargetContainer 流程。

### Decision 3: 脚本钩子按 mode 分发

在 HandleEffects 中，根据 mode 调用不同的脚本钩子：
- HandleLaunch → HookOnEffectLaunch
- HandleLaunchTarget → HookOnEffectLaunchTarget
- HandleHit → HookOnEffectHit
- HitTarget → HookOnEffectHitTarget

**理由**: 对齐 TC 的 CallScriptEffectHandlers 按 mode 映射到不同 HookList。现有 skill 使用 HookOnEffectHit 的地方需要迁移到 HitTarget（大多数场景是对目标操作）。

### Decision 4: 区域效果（无目标）在 Launch/Hit 无目标阶段处理

区域效果的 caster-as-target 模式保留在 Launch 和 Hit 阶段（无目标参数），LaunchTarget 和 HitTarget 阶段处理每个被选中的目标。

### Decision 5: 渐进式迁移 — 现有 skill 钩子迁移

- Living Bomb 44457: `HookOnEffectHit` → `HookOnEffectHitTarget`（拦截 Dummy，对目标施放周期法术）
- Living Bomb 44461: `HookOnEffectHit` → `HookOnEffectHitTarget`（拦截 SchoolDamage 后传播）
- Arcane Missiles: 无迁移（仅使用 AuraHookOnPeriodic）

## Risks / Trade-offs

- **[测试可能失败]** 所有 effect handler 加入 mode guard 后，如果 ProcessAll 仍然只传一个 mode，所有 handler 都会 return。→ 缓解：ProcessAll 重构和 handler mode guard 在同一个 change 中完成，同时验证。
- **[HookOnEffectHit 语义变更]** 现有 skill 使用 HookOnEffectHit 作为通用 effect 钩子，改为 HitTarget 后语义不同。→ 缓解：迁移所有现有 skill 的注册，不保留旧语义。
- **[Damage 计算在 LaunchTarget 但施加在 HitTarget]** 当前简化方案不区分计算和施加，直接在 handler 的 mode 中同时做两件事。未来需要分离时再重构。→ 可接受：当前阶段对齐 mode 绑定是首要目标。
