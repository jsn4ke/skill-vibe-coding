---
name: skill-design-decompose
description: >
  Decompose a game skill into implementable components before coding. Given a skill name or
  description, this skill searches Wowhead (for WoW skills) or uses user-provided details,
  references TrinityCore's mechanism implementation via tc-mechanism-ref, maps components to
  the WoW framework (creating custom mechanisms only when WoW has no equivalent), and outputs
  a structured design document for discussion. This skill should be used when the user wants
  to design, analyze, or plan the implementation of a specific game skill.
---

# Skill Design Decompose

Systematically decompose a game skill into implementable components and output a design document.

## Input

The user provides either:
- A **WoW skill name** (e.g., "Fireball", "火球术") — the skill will be looked up on Wowhead
- A **custom skill description** — the user provides the skill concept and behavior

## Workflow

### Phase 1: Collect Skill Information

**If WoW skill:**
1. Use WebSearch to search Wowhead: `site:wowhead.com <skill name>` or `<skill name> wowhead spell`
2. Extract from the Wowhead spell page:
   - Spell effects and their types
   - Base values, scaling coefficients
   - Cast time, cooldown, duration
   - Range, targeting
   - Resource cost
   - Tooltip description for behavior details
3. If Wowhead search fails or is ambiguous, ask the user for clarification

**If custom skill:**
1. Use the user's description as the primary source
2. If the description is vague or missing key details, ask targeted questions:
   - What does the skill do on hit?
   - Does it apply any lasting effects?
   - What are the casting requirements?
   - How does it interact with other skills?

### Phase 1.5: TC Cross-Table Data Supplement

Wowhead only shows Spell.dbc data. Many spell properties live in related DBC tables linked by ID chains. After Phase 1, supplement with TC source data:

1. **Identify missing cross-table data**: Check if the spell might involve:
   - Projectile/missile behavior → **SpellMisc.db2** contains `Speed`, `LaunchDelay`, `MinDuration` (server-side gameplay data, NOT SpellVisual/SpellMissile which is client-side visual only)
   - Visual effects (SpellVisual.dbc) → client-side cosmetic, server does NOT use for gameplay
   - Spell categories (SpellCategory.dbc) → shared cooldowns
   - Spell difficulty (SpellDifficulty.dbc) → scaling by difficulty
   - Spell cooldown overrides (SpellCooldowns.dbc) → category cooldown
   - Interrupt/aura interrupt flags (in Spell.dbc columns not shown on Wowhead)

2. **Search TC source for spell ID**: Grep `../TrinityCore/` for the spell ID to find:
   - Hardcoded spell handlers or scripts
   - DBC structure definitions showing all fields
   - Server-side processing logic that references cross-table data

3. **Check DBC structure**: Read relevant DBC definitions from TC source:
   - `src/server/game/DataStores/DB2Structure.h` for all DB2 entry structures
   - `src/server/game/Spells/SpellInfo.cpp` for how SpellMisc fields load into SpellInfo
   - Key structures: `SpellMiscEntry` (Speed/LaunchDelay/MinDuration), `SpellVisualEntry`, `SpellVisualMissileEntry`

4. **Document findings**: Add cross-table data to the design document:
   - If projectile: note Speed from SpellMisc, travel time formula `delay = LaunchDelay + max(dist/Speed, MinDuration)`, and StateLaunched handling
   - If category cooldown: note category ID and shared spells
   - If special flags: note interrupt flags, aura interrupt conditions

### Phase 1.6: Triggered Spell Discovery and Decomposition

Many WoW spells trigger other spells (via `TriggerSpellID`, `SPELL_AURA_PERIODIC_TRIGGER_SPELL`, `SPELL_EFFECT_TRIGGER_SPELL`, etc.). These triggered/child spells are separate spell entities with their own SpellInfo, effects, and cross-table data — they **must** receive the same full decomposition as the parent spell.

1. **Identify all triggered spells**: Scan each effect for `TriggerSpellID` references. Common patterns:
   - `SPELL_AURA_PERIODIC_TRIGGER_SPELL` → periodic aura tick triggers a spell (e.g., Arcane Missiles tick → spell 7268)
   - `SPELL_EFFECT_TRIGGER_SPELL` → immediate spell trigger (e.g., Pyroblast触发 Combustion)
   - `SPELL_AURA_PROC_TRIGGER_SPELL` → proc-based spell trigger (e.g., Clearcasting)
   - Aura `TriggerSpellID` field in SpellEffectInfo

2. **Decompose each triggered spell**: For every discovered triggered spell, apply the same Phase 1 + Phase 1.5 + Phase 2 process:
   - Collect basic info from Wowhead (cast time, cost, range, school, effects)
   - Collect cross-table data (SpellMisc Speed/LaunchDelay, DBC fields)
   - Reference TC mechanism for its implementation pattern
   - Note how it differs from a player-castable spell (often hidden, no GCD, triggered flags)

3. **Generate independent design documents**: Each triggered spell gets its own standalone design document at `skill-designs/<spell-name-slug>-<id>.md`, at the same level as the parent spell's document. This is because:
   - Triggered spells are separate spell entities with their own SpellInfo, effects, and cross-table data
   - Multiple parent spells may reference the same triggered spell (reusability)
   - Independent documents are easier to reference during implementation

   The triggered spell document follows the same output template as any skill (Phase 4), with additional sections:
   - Trigger flags analysis (e.g., TRIGGERED_FULL_MASK components)
   - Parent spell reference (which spell triggers it and how)
   - Lifecycle diagram showing how it's created and executed from the parent context

4. **Cross-reference**: The parent spell's document links to the triggered spell's document. The parent's "效果拆解" section contains a brief reference (name, ID, link to standalone doc) instead of the full decomposition.

### Phase 2: TC Mechanism Reference

For each mechanism the skill involves, invoke `tc-mechanism-ref` to understand how TrinityCore implements it:

1. Identify relevant mechanisms from the skill's effects (damage, healing, aura, targeting, cooldown, etc.)
2. For each mechanism, check `tc-references/` for cached analysis first
3. If not cached, perform tc-mechanism-ref analysis on the relevant topic
4. Collect the key design patterns and data structures that apply

**Phase 2.1: Timing Verification (mandatory for periodic/channeled spells)**

When the skill involves any periodic mechanism (periodic damage, periodic heal, periodic trigger spell, channeled ticks), you MUST verify exact tick timing by tracing TC source code:

1. **First tick timing**: Check `AuraEffect::ResetPeriodic` and `CalculatePeriodic` in `SpellAuraEffects.cpp`:
   - `_periodicTimer` initial value determines when the first tick fires
   - Default: `_periodicTimer = 0` → first tick at t = period (NOT at t = 0)
   - `SPELL_ATTR5_EXTRA_INITIAL_PERIOD` flag: `_periodicTimer = _period` → first tick at t = 0 (immediate)
   - **Always state the first tick time explicitly** — do not assume or guess

2. **Tick count**: Derive from `duration / period` for normal auras, or `GetTotalTicks()` which accounts for extra tick flags

3. **Document in the design document**: Include a tick timeline table:
   ```
   | Tick # | Time  | Event           |
   |--------|-------|-----------------|
   | 1      | 1000ms| First missile   |
   | 2      | 2000ms| Second missile  |
   | 3      | 3000ms| Third missile   |
   ```

This step prevents the common mistake of writing tick times as 0/period/2×period when the correct timing is period/2×period/3×period.

### Phase 3: Framework Mapping and Decomposition

Map each skill component to the WoW framework:

1. **Effects**: Map to TC SpellEffectName enum values. If a direct mapping exists, note it. If not, design a custom effect type and explain why WoW's framework doesn't cover it.
2. **Aura**: Map to TC AuraType enum values. Same principle — map first, create only when necessary.
3. **Targeting**: Map to TC SpellImplicitTargetInfo patterns.
4. **Cooldown/Charge**: Map to TC SpellHistory patterns.
5. **Cast flow**: Determine if standard cast, instant, channeled, or charged cast.

For each custom mechanism created:
- Explain what it does
- Explain why no WoW mechanism fits
- Describe the design approach
- Note any risks or unknowns

### Phase 4: Output Design Document

Generate a design document following the output template in `references/output-template.md`.

Save to `skill-designs/<skill-name-slug>.md`.

Present the document to the user for discussion.

## Output Document Sections

The document must include these sections (see `references/output-template.md` for full template):

1. **Overview** — one paragraph positioning the skill
2. **Basic Info** — cast time, cooldown, cost, range, GCD (table format with WoW reference)
3. **Targeting** — target type, count, filtering, radius (table format with WoW reference)
4. **Effect Breakdown** — each effect as a subsection: type, values, target, WoW mapping
5. **Aura** (if applicable) — type, duration, stacking, proc, periodic, WoW mapping
6. **WoW Framework Mapping Summary** — consolidated table of all components
7. **Implementation Advice** — detailed design thinking:
   - Overall architecture: how data flows from definition to execution
   - Data configuration: what should be data-driven vs script-driven
   - Cast flow: full lifecycle from trigger to effect resolution, including branches
   - Effect execution: order, dependencies, calculation formulas, modifier chains
   - Aura lifecycle (if applicable): create/apply/refresh/remove, periodic ticks, stacking
   - Special logic: custom mechanisms, why WoW framework doesn't cover them, design rationale

## Constraints

- **Output is for discussion** — never generate code
- **Prioritize WoW framework** — only create custom mechanisms when no WoW equivalent exists
- **Implementation advice must be detailed** — cover logic and design thinking, not just "implement X"
- **Custom mechanisms must explain the gap** — why WoW framework is insufficient
- **Skill designs go to `skill-designs/`** directory
- **Use English filenames, Chinese content** for output documents
- **Triggered spells require independent design documents** — when an effect references a TriggerSpellID, the triggered spell must be decomposed into its own `skill-designs/<name>-<id>.md` file at the same level as the parent spell. The parent document links to it rather than embedding the full decomposition.
