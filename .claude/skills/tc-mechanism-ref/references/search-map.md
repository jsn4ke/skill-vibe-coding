# TC Source Search Map

TrinityCore root: `../TrinityCore`
Source base: `../TrinityCore/src/server/game/`

## Directory Structure

```
src/server/game/
├── Spells/
│   ├── Spell.cpp / Spell.h          # Cast lifecycle, state machine, target selection
│   ├── SpellInfo.cpp / SpellInfo.h  # Static spell definition, effects, targeting rules
│   ├── SpellMgr.cpp / SpellMgr.h    # Global spell registry, proc system, spell chains
│   ├── SpellEffects.cpp             # Per-effect-type handlers (damage, heal, summon...)
│   ├── SpellHistory.cpp / .h        # Cooldown, charge, GCD, school lockout
│   ├── SpellScript.cpp / .h         # Hook-based spell scripting
│   ├── SpellDefines.h               # Cast flags, interrupt flags, aura interrupt flags
│   ├── SpellCastRequest.h           # Cast request from client
│   ├── Auras/
│   │   ├── SpellAuras.cpp / .h      # Aura lifecycle, stacking, application
│   │   ├── SpellAuraEffects.cpp / .h # Per-aura-type handlers (periodic, stat mod, CC...)
│   │   └── SpellAuraDefines.h       # AuraType enum (655+), AuraRemoveMode, ShapeshiftForm
│   └── TraitMgr.cpp / .h            # Talent/trait system
├── Miscellaneous/
│   └── SharedDefines.h              # SpellEffectName (300+), SpellCastResult, enums
├── DataStores/
│   └── DB2Structure.h               # DBC/DB2 data structures (SpellEffectEntry, etc.)
├── Entities/
│   ├── Unit.cpp / Unit.h            # Unit base (aura container, spell history, combat)
│   ├── Player.cpp / Player.h        # Player-specific spell logic
│   └── Creature.cpp / Creature.h    # Creature AI spell casting
└── Condition/
    └── ConditionMgr.h               # Conditional spell/aura requirements
```

## Topic → Search Mapping

### spell-cast-flow
**Core files:**
- `Spells/Spell.h` — SpellState enum, Spell class declaration
- `Spells/Spell.cpp` — prepare(), update(), cast(), finish(), cancel()

**Key terms:** `SpellState`, `prepare`, `CheckCast`, `cast`, `finish`, `cancel`, `SPELL_STATE_`

### aura-system
**Core files:**
- `Spells/Auras/SpellAuras.h` — Aura, AuraApplication, UnitAura, DynObjAura
- `Spells/Auras/SpellAuras.cpp` — Create, Apply, Remove, Refresh

**Key terms:** `Aura::TryCreate`, `AuraApplication`, `UnitAura`, `DynObjAura`, `_ApplyEffect`, `_RemoveEffect`

### aura-stacking
**Core files:**
- `Spells/Auras/SpellAuras.cpp` — TryRefreshStackOrCreate, AddStack, ModStackAmount

**Key terms:** `stackAmount`, `TryRefreshStack`, `AddStack`, `ModStackAmount`, `spellGroupStackRules`

### cooldown-charge
**Core files:**
- `Spells/SpellHistory.h` — CooldownEntry, ChargeEntry, GCD storage
- `Spells/SpellHistory.cpp` — AddCooldown, AddCharge, CancelCooldown, IsReady

**Key terms:** `CooldownEntry`, `ChargeEntry`, `CategoryEnd`, `OnHold`, `SchoolLockout`, `GlobalCooldownMgr`

### target-selection
**Core files:**
- `Spells/SpellInfo.h` — SpellImplicitTargetInfo, TargetDescriptor
- `Spells/Spell.cpp` — SelectSpellTargets, SearchAreaTargets, SelectImplicit*

**Key terms:** `SelectSpellTargets`, `ImplicitTarget`, `TargetDescriptor`, `TARGET_SELECT_CATEGORY_`, `TARGET_CHECK_`

### proc-system
**Core files:**
- `Spells/SpellMgr.h` — SpellProcEntry, ProcFlags enums
- `Spells/Auras/SpellAuraEffects.cpp` — TriggerProc, HandleProcTriggerSpell

**Key terms:** `ProcFlags`, `ProcChance`, `ProcsPerMinute`, `TriggerProc`, `SpellProcEntry`

### effect-pipeline
**Core files:**
- `Spells/SpellEffects.cpp` — All Effect* handlers
- `Spells/Spell.h` — HandleEffects, SpellEffectHandleMode

**Key terms:** `HandleEffects`, `SpellEffectHandleMode`, `SPELL_EFFECT_`, `EffectSchoolDMG`, `EffectApplyAura`

### diminishing-returns
**Core files:**
- `Spells/SpellInfo.h` — SpellDiminishInfo struct
- `Spells/SpellInfo.cpp` — GetDiminishInfo

**Key terms:** `DiminishGroup`, `DiminishReturnType`, `DiminishMaxLevel`, `DiminishDurationLimit`

### spell-script
**Core files:**
- `Spells/SpellScript.cpp` / `Spells/SpellScript.h` — SpellScript base, HookList
- `Spells/Auras/SpellAuras.h` — AuraScript (part of Aura system)

**Key terms:** `SpellScript`, `AuraScript`, `HookList`, `PreventHitEffect`, `PreventDefaultAction`, `_Hook`

### spell-modifier
**Core files:**
- `Spells/Spell.cpp` — ApplySpellModifier, CalculateSpellDamage

**Key terms:** `ApplySpellModifier`, `SPELLMOD`, `CalculateSpellDamage`, `BonusCoefficient`

### spell-interrupt
**Core files:**
- `Spells/Spell.cpp` — Cancel, Interrupt
- `Spells/SpellDefines.h` — SpellInterruptFlags, SpellAuraInterruptFlags

**Key terms:** `SpellInterruptFlags`, `SpellAuraInterruptFlags`, `CHANNEL_INTERRUPT_FLAG`, `AURA_INTERRUPT_FLAG`

## Key Enum Locations

| Enum | File | Purpose |
|------|------|---------|
| `SpellEffectName` | SharedDefines.h | 300+ effect types |
| `AuraType` | SpellAuraDefines.h | 655+ aura types |
| `SpellCastResult` | SharedDefines.h | 150+ cast failure reasons |
| `SpellState` | Spell.h | Cast state machine states |
| `SpellCastTargetFlags` | SpellDefines.h | Client target flags |
| `SpellInterruptFlags` | SpellDefines.h | Cast interruption rules |
| `SpellAuraInterruptFlags` | SpellDefines.h | Aura break conditions |
| `ProcFlags` | SpellMgr.h | Proc trigger events |
| `SpellCustomAttributes` | SpellInfo.h | Server-side behavior flags |
