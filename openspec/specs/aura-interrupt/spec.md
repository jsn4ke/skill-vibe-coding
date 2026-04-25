## ADDED Requirements

### Requirement: SpellAuraInterruptFlags type with bitmask operations

The system SHALL define `SpellAuraInterruptFlags` as a bitmask type with the following flags:
- `AuraInterruptNone` (0)
- `AuraInterruptOnMovement` (0x01) — remove aura when carrier moves
- `AuraInterruptOnDamage` (0x02) — remove aura when carrier takes damage
- `AuraInterruptOnAction` (0x04) — remove aura when carrier performs an action (cast, attack)

`SpellEffectInfo` SHALL carry an `AuraInterruptFlags` field.

#### Scenario: SpellEffectInfo carries AuraInterruptFlags
- **WHEN** an effect is defined with `AuraInterruptFlags: spell.AuraInterruptOnMovement`
- **THEN** `effect.AuraInterruptFlags.HasFlag(spell.AuraInterruptOnMovement)` SHALL return true

### Requirement: Aura stores InterruptFlags from SpellEffectInfo

When an aura is created via EffectApplyAura, the aura SHALL copy the `AuraInterruptFlags` from the corresponding SpellEffectInfo.

#### Scenario: Aura inherits InterruptFlags from effect
- **WHEN** a spell effect has AuraInterruptFlags = AuraInterruptOnMovement
- **AND** an aura is created from this effect
- **THEN** the aura's InterruptFlags SHALL include AuraInterruptOnMovement

### Requirement: RemoveAurasWithInterruptFlags removes matching auras

The `Unit` SHALL provide a method that removes all owned auras whose InterruptFlags match the given flag. Auras SHALL be removed with `aura.RemoveByInterrupt` mode.

This method SHALL also check the current channeled spell — if the channel spell's info has the matching channel interrupt flag, the channel SHALL be cancelled.

#### Scenario: Movement removes auras with AuraInterruptOnMovement
- **WHEN** a unit owns an aura with AuraInterruptOnMovement flag
- **AND** the unit moves
- **THEN** the aura SHALL be removed with RemoveByInterrupt mode

#### Scenario: Non-matching auras not removed
- **WHEN** a unit owns an aura with AuraInterruptOnDamage flag only
- **AND** the unit moves
- **THEN** the aura SHALL NOT be removed

### Requirement: Unit.Update triggers aura interrupt checks

At the end of `Unit.Update()`, the unit SHALL check for movement-based aura interrupts. If the unit moved during this frame, it SHALL call RemoveAurasWithInterruptFlags(AuraInterruptOnMovement).

#### Scenario: Moving unit loses movement-interruptible aura
- **WHEN** a unit has an aura with AuraInterruptOnMovement
- **AND** the unit changes position between frames
- **THEN** after Unit.Update(), the aura SHALL be removed

#### Scenario: Stationary unit keeps aura
- **WHEN** a unit has an aura with AuraInterruptOnMovement
- **AND** the unit does not change position
- **THEN** the aura SHALL remain active
