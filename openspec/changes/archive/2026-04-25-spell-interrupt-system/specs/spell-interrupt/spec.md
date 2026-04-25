## ADDED Requirements

### Requirement: SpellInterruptFlags type with bitmask operations

The system SHALL define `SpellInterruptFlags` as a bitmask type with the following flags:
- `InterruptNone` (0)
- `InterruptMovement` (0x01) — cancel on caster movement
- `InterruptDamageCancels` (0x02) — cancel on damage taken
- `InterruptDamagePushback` (0x04) — pushback cast time on damage (placeholder for future)

`SpellInfo` SHALL carry an `InterruptFlags SpellInterruptFlags` field.

#### Scenario: SpellInfo carries InterruptFlags
- **WHEN** a SpellInfo is defined with `InterruptFlags: spell.InterruptMovement`
- **THEN** `info.InterruptFlags.HasFlag(spell.InterruptMovement)` SHALL return true

#### Scenario: Multiple flags combine via bitwise OR
- **WHEN** a SpellInfo has `InterruptFlags: spell.InterruptMovement | spell.InterruptDamageCancels`
- **THEN** both HasFlag checks SHALL return true

### Requirement: Spell.Update cancels on caster movement

When a spell is in PREPARING or CHANNELING state, and the caster is moving, and the spell has `InterruptMovement` flag, the spell SHALL be cancelled.

Triggered spells (TriggeredFullMask) SHALL NOT be interrupted by movement.

#### Scenario: Fireball cast interrupted by movement
- **WHEN** Fireball (CastTime > 0, InterruptMovement) is in PREPARING state
- **AND** the caster moves (IsMoving() returns true)
- **THEN** the spell SHALL be cancelled with result CastFailedInterrupted

#### Scenario: Channel spell interrupted by movement
- **WHEN** Arcane Missiles (channel, InterruptMovement) is in CHANNELING state
- **AND** the caster moves
- **THEN** the spell SHALL be cancelled

#### Scenario: Triggered spell ignores movement interrupt
- **WHEN** a triggered spell (TriggeredFullMask) is active
- **AND** the caster moves
- **THEN** the spell SHALL NOT be cancelled by movement

### Requirement: Spell.Update cancels when target disappears

When a spell has a unit target (TargetID != 0) and the target unit no longer exists in the engine, the spell SHALL be cancelled.

#### Scenario: Cast cancelled when target removed
- **WHEN** a spell with targetID=2 is in PREPARING state
- **AND** unit 2 is removed from the engine
- **THEN** the spell SHALL be cancelled

#### Scenario: Channel cancelled when target disappears
- **WHEN** a channel spell's target disappears
- **THEN** the spell SHALL be cancelled

### Requirement: Spell.Update performs range check during cast

When a spell is in PREPARING state and has a unit target, the spell SHALL check that the target is within RangeMax (with tolerance) each Update tick. If the target moves out of range, the spell SHALL be cancelled.

Tolerance: `min(RangeMax * 0.1, MAX_RANGE_TOLERANCE)` added to max range.

#### Scenario: Cast cancelled when target moves out of range
- **WHEN** a spell with RangeMax=35 is in PREPARING state
- **AND** the target moves to distance 40 from caster
- **THEN** the spell SHALL be cancelled

#### Scenario: Cast continues with tolerance
- **WHEN** a spell with RangeMax=35 is in PREPARING state
- **AND** the target is at distance 37 (within 35 + 3.5 tolerance)
- **THEN** the spell SHALL NOT be cancelled

#### Scenario: Instant spells skip range check in Update
- **WHEN** an instant spell (CastTime=0) has been launched
- **THEN** no range check SHALL be performed in Update

### Requirement: Channel spells validate targets each tick

When a spell is in CHANNELING state, it SHALL validate each target in TargetInfos every Update tick. A target is removed from the channel if:
1. The target unit no longer exists
2. The target is dead
3. The target is out of range (RangeMax + tolerance)

If a target is removed, the corresponding aura SHALL be removed with RemoveByCancel. If ALL targets are removed, the spell SHALL be cancelled.

#### Scenario: Channel target moves out of range
- **WHEN** Arcane Missiles is channeling with target at distance 10
- **AND** the target moves to distance 45 (RangeMax=40)
- **THEN** the target's aura SHALL be removed
- **AND** the spell SHALL be cancelled (no remaining targets)

#### Scenario: Channel continues with some targets remaining
- **WHEN** Blizzard has 3 targets in the area
- **AND** 1 target moves out of range
- **THEN** that target's aura SHALL be removed
- **AND** the channel SHALL continue for remaining targets

#### Scenario: Channel target dies
- **WHEN** a channel spell's target dies
- **THEN** the target SHALL be removed from the channel

### Requirement: Cancel performs state-specific cleanup

`Spell.Cancel()` SHALL perform cleanup based on the spell's original state:
- PREPARING: no additional cleanup beyond finish (GCD placeholder)
- CHANNELING: remove all auras created by this spell from all targets
- LAUNCHED: no additional cleanup

The cancel function SHALL emit an OnSpellCancel event via the bus.

#### Scenario: Channel cancel removes all target auras
- **WHEN** a channel spell with aura on 2 targets is cancelled
- **THEN** both auras SHALL be removed with RemoveByCancel mode

#### Scenario: Cancel emits OnSpellCancel event
- **WHEN** a spell is cancelled
- **THEN** an event with Type=OnSpellCancel SHALL be published on the bus

### Requirement: AttrBreakOnMove maps to InterruptMovement

Existing `AttrBreakOnMove` in SpellAttribute SHALL be treated as setting `InterruptMovement` in InterruptFlags. This provides backward compatibility during migration.

#### Scenario: Spell with AttrBreakOnMove interrupted by movement
- **WHEN** a spell has `Attributes: spell.AttrBreakOnMove` but no explicit InterruptFlags
- **THEN** it SHALL still be interrupted by movement
