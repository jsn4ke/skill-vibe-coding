# Game Mechanism Implementation Rule

When implementing or designing game skill/mechanism systems in this project, reference TrinityCore's implementation patterns using the `tc-mechanism-ref` skill before writing implementation code.

## Core Architecture Alignment

**The core architecture MUST be consistent with TrinityCore, not simplified or lightweighted.** These are non-negotiable design constraints:

1. **Unit-centric Update driving** — Unit holds active spells and applied/owned auras; Engine.Tick drives all Units uniformly. No component drives itself internally.
2. **Owned vs Applied aura separation** — Caster owns the aura (drives ticks, handles expiry), target has it applied (receives effects). These are two distinct container roles on two different Units.
3. **Instant same-frame / Non-instant via Update chain** — Instant and triggered spells execute synchronously within the current frame. Spells with cast time or travel time register into the Unit's active spell list and are driven by Update(diff).
4. **Spell lifecycle: create → register → Update-driven** — CastSpell creates a Spell and registers it; it does NOT self-drive by calling Update internally. The engine is the sole driver of time progression.
5. **Dummy effect + Script Hook pattern** — EffectDummy provides a hook mount point with no built-in behavior. Scripts intercept via HookOnEffectHit and supply custom logic. No shortcutting this pattern with "just call a function."

**Why:** Simplifying these patterns creates technical debt. Every shortcut taken now (flat global aura manager, Cast functions self-driving Update, no owned/applied separation) will need to be refactored later when more complex skills require the full architecture.

## TC Research Workflow

**All TC mechanism research MUST persist to `tc-references/` — no exceptions.** Whether in explore mode, implementation, or casual discussion, the rule is the same:

```
查 TC 的流程（任何模式下都一样）:
  1. 先查 tc-references/ 有没有 → 有就直接用
  2. 没有 → 用 tc-mechanism-ref skill 生成 → 自动存入 tc-references/
  3. 后续对话 → 直接读 tc-references/ → 不重复查 TC

禁止:
  ✗ 手动 grep TC → 讨论 → 不存文档 → 下次又查一遍
```

**This applies to explore mode too.** Explore can absolutely involve TC research, but it must follow the same flow: check reference first, use tc-mechanism-ref if missing, so the result gets persisted. The issue isn't "explore shouldn't touch TC" — it's "TC research must always produce a persisted artifact, regardless of what mode you're in."

## Trigger

This rule activates when the conversation involves implementing, designing, or building any of the following game mechanisms:

- **Spell/Cast**: spell, cast, casting, 施法, 法术, 技能
- **Aura/Buff**: aura, buff, debuff, 增益, 减益, 光环
- **Stacking**: stack, stacking, 堆叠
- **Cooldown**: cooldown, charge, gcd, 冷却, 充能
- **Targeting**: target, targeting, 目标选择
- **Proc**: proc, trigger, 触发, 触发器
- **Effect**: effect, pipeline, 效果, 效果管线, 管线
- **Diminishing Returns**: diminishing, dr, 递减
- **Scripting**: spell script, hook, 脚本
- **Interrupt**: interrupt, silence, stun, 打断, 沉默, 眩晕
- **Movement**: knockback, charge, leap, 击退, 冲锋, 跳跃
- **Skill Design**: 设计技能, 拆解技能, 技能拆解, skill design, 分解技能, 技能分析
- **Engine/World**: engine, world, tick, update, 引擎, 驱动, 游戏循环, unit update

## Behavior

1. **Identify the mechanism topic** from the keywords above
2. **Check `tc-references/`** for cached analysis — reuse if present
3. **Invoke `tc-mechanism-ref`** if no cached analysis exists
4. **Follow TC patterns as structural requirements** — core architecture must align, not approximate
5. **No mandatory confirmation** — this is a relaxed rule; reference TC and proceed without blocking

### Skill Design Trigger

When the user wants to design or decompose a specific skill (e.g., "设计一个火球术", "拆解这个技能"), invoke `skill-design-decompose` to systematically analyze the skill and output a design document before implementation.

### Skill + OpenSpec Propose Ordering

When `/opsx:propose` involves a game skill (matched by keywords above), the correct order is:

1. **Invoke `skill-design-decompose` FIRST** → generates `skill-designs/<name>.md` with full research (Wowhead data, TC cross-table analysis, mechanism reference)
2. **Then proceed with openspec artifacts** (proposal, design, specs, tasks), using the design document as primary input

Do NOT skip step 1 and jump directly to openspec artifacts. The design document ensures thorough research before scoping.

## Keyword → Topic Mapping

| Keywords | tc-mechanism-ref topic |
|----------|----------------------|
| spell, cast, 施法, 法术 | spell-cast-flow |
| aura, buff, debuff, 增益, 减益 | aura-system |
| stack, 堆叠 | aura-stacking |
| cooldown, charge, gcd, 冷却, 充能 | cooldown-charge |
| target, targeting, 目标选择 | target-selection |
| proc, trigger, 触发 | proc-system |
| effect, pipeline, 效果 | effect-pipeline |
| diminishing, dr, 递减 | diminishing-returns |
| script, hook, 脚本 | spell-script |
| engine, world, tick, update, 引擎, 驱动 | unit-update-architecture |

## Skip Conditions

- User explicitly says "不需要参考 TC" or "skip TC reference"
- The mechanism is already analyzed in `tc-references/` — reuse the cached analysis directly
- The task is purely about project configuration, not game mechanism design

## Reference Trace

When TC patterns are referenced, briefly note it in the implementation context (e.g., "参考 TC 的 Unit-centric Update 驱动"). No formal citation required — just naturally acknowledge the source of design patterns.
