## ADDED Requirements

### Requirement: SelectEffectTargets replaces SelectTargets

The `Spell` type SHALL implement `SelectEffectTargets()` that iterates each effect's `TargetA` and `TargetB`, calling `selectEffectImplicitTargets()` for each. This replaces the current `SelectTargets()` which only examines `TargetA` and uses an external `AoESelector`.

#### Scenario: Two-phase TargetA + TargetB resolution
- **GIVEN** a spell effect with `TargetA: TargetDestTargetEnemy` and `TargetB: TargetUnitAreaEnemy`
- **WHEN** `SelectEffectTargets()` is called
- **THEN** TargetA SHALL resolve the destination position (target enemy's position as DestPos)
- **AND** TargetB SHALL search for all enemies in radius around that DestPos

#### Scenario: TargetA-only resolution (no TargetB)
- **GIVEN** a spell effect with `TargetA: TargetUnitTargetEnemy` and `TargetB: TargetNone`
- **WHEN** `SelectEffectTargets()` is called
- **THEN** only TargetA SHALL be resolved, adding the target enemy to TargetInfos

### Requirement: SelectionCategory dispatches to algorithm

`selectEffectImplicitTargets()` SHALL dispatch based on `SelectionCategory`:
- `SelectDefault` → resolve reference target (caster/target/dest) and add to TargetInfos
- `SelectNearby` → find single nearest unit within radius matching CheckType
- `SelectArea` → find all units in radius from resolved center matching CheckType
- `SelectCone` → find all units in cone (center, direction, angle, radius) matching CheckType
- `SelectLine` → find all units in line/rectangle (caster→dest, width, radius) matching CheckType
- `SelectChannel` → get target from current channel spell
- `SelectNYI` → no-op (log warning)

#### Scenario: SelectDefault with TargetUnitCaster
- **GIVEN** TargetA = `TargetUnitCaster` (RefCaster, SelectDefault, ObjUnit)
- **WHEN** `selectEffectImplicitTargets()` processes this target
- **THEN** the caster SHALL be added to TargetInfos

#### Scenario: SelectArea with TargetUnitAreaEnemy
- **GIVEN** TargetA = `TargetUnitAreaEnemy` (RefSrc, SelectArea, CheckEnemy)
- **AND** DestPos is set to position (10, 0, 0)
- **AND** Radius = 8.0
- **WHEN** `selectEffectImplicitTargets()` processes this target
- **THEN** all enemy units within 8.0 units of DestPos SHALL be added to TargetInfos

#### Scenario: SelectCone with TargetUnitConeEnemy
- **GIVEN** TargetA = `TargetUnitConeEnemy` (RefCaster, SelectCone, CheckEnemy, DirFront)
- **AND** Radius = 10.0
- **WHEN** `selectEffectImplicitTargets()` processes this target
- **THEN** all enemy units within a frontal cone (caster facing, 90° arc, 10.0 range) SHALL be added to TargetInfos

### Requirement: ReferenceType resolves search center

The `resolveCenter()` function SHALL resolve the search center position based on `ReferenceType`:
- `RefCaster` → caster's position
- `RefTarget` → current target unit's position
- `RefLast` → last added target's position
- `RefSrc` → spell's SourcePos
- `RefDest` → spell's DestPos

#### Scenario: RefSrc resolves to DestPos
- **GIVEN** a spell with DestPos set to (20, 0, 0) and ReferenceType = RefSrc
- **WHEN** `resolveCenter()` is called
- **THEN** it SHALL return (20, 0, 0)

#### Scenario: RefCaster resolves to caster position
- **GIVEN** a caster at position (5, 0, 5) and ReferenceType = RefCaster
- **WHEN** `resolveCenter()` is called
- **THEN** it SHALL return (5, 0, 5)

### Requirement: CheckType filters by friend/foe

The `passesCheck()` function SHALL filter target candidates based on `CheckType`:
- `CheckDefault` → no filter (all pass)
- `CheckEnemy` → target's EntityType differs from caster's
- `CheckAlly` → target's EntityType matches caster's
- `CheckParty` → target is in same party as caster (fallback: same EntityType)
- `CheckRaid` → target is in same raid as caster (fallback: same EntityType)
- `CheckSummoned` → target is owned/summoned by caster

#### Scenario: CheckEnemy filters allies
- **GIVEN** caster is Player type and a candidate is also Player type
- **WHEN** `passesCheck(CheckEnemy, caster, candidate)` is called
- **THEN** it SHALL return false

#### Scenario: CheckEnemy passes enemies
- **GIVEN** caster is Player type and a candidate is Creature type
- **WHEN** `passesCheck(CheckEnemy, caster, candidate)` is called
- **THEN** it SHALL return true

### Requirement: Shared search optimization

When multiple effects share the same TargetA/TargetB values and Radius, the search SHALL execute only once and results SHALL be shared across matching effects. A `processedEffectMask uint32` bit mask SHALL track which effects have already been processed.

#### Scenario: Two effects with same Area target share search
- **GIVEN** Effect[0] has TargetA=TargetUnitAreaEnemy, Radius=8.0
- **AND** Effect[1] has TargetA=TargetUnitAreaEnemy, Radius=8.0
- **WHEN** `SelectEffectTargets()` processes both effects
- **THEN** the area search SHALL execute only once
- **AND** both effects SHALL receive the same target set

### Requirement: Script Hook intervention point

After `SearchAreaTargets` / `SearchConeTargets` / `SearchChainTargets` returns results and before targets are finalized, the system SHALL call `Registry.CallTargetSelectHook(spellID, effectIndex, targets)` allowing scripts to modify the target list (add, remove, or replace targets).

#### Scenario: Script removes a target from area selection
- **GIVEN** a script registered via `Registry.RegisterSpellHook(spellID, HookOnTargetSelect, handler)`
- **AND** SearchAreaTargets found 3 targets
- **WHEN** the hook handler removes 1 target
- **THEN** only 2 targets SHALL be added to TargetInfos

### Requirement: Remove AoESelector interface

The `AoESelector` interface, `WithAoE()` CastOption, and `Spell.AoESelector/AoECenter/AoEExcludeID` fields SHALL be removed. All target selection SHALL be driven by SpellInfo's TargetA/TargetB + Radius + ChainTargets data.

#### Scenario: CastSpell without WithAoE
- **GIVEN** a spell with TargetA=TargetUnitAreaEnemy and Radius=8.0
- **WHEN** `eng.CastSpell(caster, &Info, engine.WithTarget(2))` is called
- **THEN** the area search SHALL be driven by SpellInfo data alone
- **AND** no AoESelector injection SHALL be required
