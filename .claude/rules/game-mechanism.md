# Game Mechanism Implementation Rule

When implementing or designing game skill/mechanism systems in this project, reference TrinityCore's implementation patterns using the `tc-mechanism-ref` skill before writing implementation code.

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

## Behavior

1. **Identify the mechanism topic** from the keywords above
2. **Invoke `tc-mechanism-ref`** with the corresponding topic to analyze TrinityCore's implementation
3. **Use TC patterns as reference** when designing and implementing — naturally incorporate insights into the work
4. **No mandatory confirmation** — this is a relaxed rule; reference TC and proceed without blocking

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

## Skip Conditions

- User explicitly says "不需要参考 TC" or "skip TC reference"
- The mechanism is already analyzed in `tc-references/` — reuse the cached analysis directly
- The task is purely about project configuration, not game mechanism design

## Reference Trace

When TC patterns are referenced, briefly note it in the implementation context (e.g., "参考 TC 的 Aura 三层架构"). No formal citation required — just naturally acknowledge the source of design patterns.
