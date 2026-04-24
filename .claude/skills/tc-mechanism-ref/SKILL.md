---
name: tc-mechanism-ref
description: >
  Look up TrinityCore's implementation of a specific game mechanism or flow (spell cast flow,
  aura stacking, cooldown/charge, target selection, proc system, effect pipeline, diminishing
  returns, spell scripting, etc.). This skill should be used when the user wants to understand
  HOW TrinityCore implements a mechanism before designing their own version. Only analyzes
  mechanisms and flows — does not cover specific WoW spell data or content.
---

# TC Mechanism Reference

Search and analyze TrinityCore source code to extract mechanism designs for reference.

## Input

The user provides a **topic** — a mechanism or flow direction to investigate. Examples:

- "spell cast flow" — the full cast lifecycle
- "aura stacking" — how auras stack, refresh, and interact
- "cooldown charge" — cooldown, category cooldown, charge recovery
- "target selection" — how spells pick targets
- "proc system" — trigger-on-event mechanics
- "effect pipeline" — how spell effects are processed
- "diminishing returns" — crowd control diminishing returns
- "spell script" — the hook-based scripting system

## Workflow

### Step 1: Check Knowledge Base Cache

Check if `tc-references/<topic-slug>.md` already exists. If so, read and present it. Skip to Step 5 for storage (already cached).

To form the slug: lowercase the topic, replace spaces with hyphens (e.g., "Aura Stacking" → `aura-stacking.md`).

### Step 2: Resolve Search Scope

Read `references/search-map.md` (bundled with this skill) to find:
- Target source paths in `../TrinityCore/src/server/game/`
- Keywords to grep for
- Key files to read

If the topic is not in the map, infer search paths based on these heuristics:
- Spell casting → `Spells/Spell.cpp`, `Spells/Spell.h`
- Aura/buff → `Spells/Auras/`
- Cooldown → `Spells/SpellHistory.cpp`
- Targeting → `Spells/Spell.cpp`, `Spells/SpellInfo.cpp`
- Proc → `Spells/SpellMgr.h`, `Spells/Auras/SpellAuraEffects.cpp`
- Effects → `Spells/SpellEffects.cpp`
- Enums/constants → `Miscellaneous/SharedDefines.h`, `Spells/SpellDefines.h`, `Spells/Auras/SpellAuraDefines.h`
- Data structures → `DataStores/DB2Structure.h`

### Step 3: Search TrinityCore Source

Use Grep to find relevant code in `../TrinityCore/`. Start broad, then narrow:

1. **Broad search**: grep the topic keywords across `Spells/` directory
2. **Narrow**: identify the core files (usually 2-4 files contain the main logic)
3. **Pinpoint**: grep for specific struct/enum/function names

### Step 4: Analyze and Structure

Read the identified core files. Extract and organize into four sections:

#### Section 1: Core Data Structures
- Key struct/enum fields with brief purpose notes
- Do NOT paste entire code blocks — summarize field roles
- Focus on fields that drive the mechanism

#### Section 2: Flow Diagram
- Draw an ASCII state machine, sequence diagram, or data flow diagram
- Label key branches, conditions, and state transitions
- Use this style:

```
┌─────────┐   condition   ┌─────────┐
│ State A │──────────────▶│ State B │
└─────────┘               └─────────┘
     │                         │
     │ error                   │ done
     ▼                         ▼
┌─────────┐               ┌─────────┐
│ Cancel  │               │ Finish  │
└─────────┘               └─────────┘
```

#### Section 3: Key Design Decisions
- Each decision in "Decision → Reason" format
- Focus on WHY TrinityCore chose this approach
- Example: "Separate SpellInfo from Spell instance → allows static data sharing across all cast instances without locking"

#### Section 4: Reusable Patterns
- Strip WoW-specific concepts (spell family names, DBC, specific aura types)
- Elevate to general game development patterns
- State applicability: when this pattern fits and when it doesn't

### Step 5: Output and Store

1. Present the analysis to the user
2. Save to `tc-references/<topic-slug>.md` with a header:

```markdown
# <Topic>

> Source: TrinityCore | Generated: <date> | Topic: <keywords>

<four sections>
```

## Constraints

- **Mechanisms only** — never analyze specific WoW spells, items, or content
- **TC path** is always `../TrinityCore` relative to the skills project root
- **Knowledge base files** use English filenames, Chinese content
- **Depth** — enough to understand design intent, not line-by-line walkthrough
- **No code writing** — this skill is purely analytical, never generates implementation code
- If the user asks to implement something based on the analysis, remind them to exit this skill and use a proposal or implementation workflow
