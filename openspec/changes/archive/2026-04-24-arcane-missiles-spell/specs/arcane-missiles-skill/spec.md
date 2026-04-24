## ADDED Requirements

### Requirement: Arcane Missiles spell info definition
The system SHALL define a SpellInfo for Arcane Missiles (ID 5143) with: instant cast (CastTime=0), 3000ms channel duration, 85 mana cost, 30yd range, IsChanneled=true, AttrChanneled+AttrBreakOnMove attributes, and one effect of type EffectApplyAura with AuraPeriodicTriggerSpell aura type, 1000ms period, TriggerSpellID=7268.

#### Scenario: SpellInfo fields are correctly populated
- **WHEN** the Arcane Missiles SpellInfo is inspected
- **THEN** ID SHALL be 5143, Duration SHALL be 3000, PowerCost SHALL be 85, IsChanneled SHALL be true, and Effects[0].TriggerSpellID SHALL be 7268

### Requirement: Arcane Missiles cast lifecycle
The system SHALL provide a CastArcaneMissiles function that: creates a Spell, sets the target, calls Prepare(), advances through channel duration (3000ms), and creates a PeriodicTriggerSpell Aura on the target with 1s tick period.

#### Scenario: Full cast lifecycle completes
- **WHEN** CastArcaneMissiles is called with a valid caster and target
- **THEN** the spell SHALL enter StateChanneling, the Aura SHALL be applied to the target, and after 3000ms the spell SHALL finish with CastOK

#### Scenario: Three missiles fire during channel
- **WHEN** the channel runs for 3000ms with 1000ms ticks
- **THEN** exactly 3 OnSpellHit events SHALL be published (at 1000ms, 2000ms, 3000ms)

### Requirement: Arcane Missiles damage per missile
Each triggered missile SHALL deal 24 + 0.132 × casterSpellPower arcane damage per hit, with independent crit rolls at 1.5x multiplier.

#### Scenario: Damage without crit at 100 SP
- **WHEN** a missile hits a target with caster spellPower=100 and no crit
- **THEN** damage SHALL be 24 + 0.132 × 100 = 37.2

#### Scenario: Damage with crit at 100 SP
- **WHEN** a missile crits with caster spellPower=100
- **THEN** damage SHALL be (24 + 13.2) × 1.5 = 55.8

### Requirement: Arcane Missiles channel cancel removes Aura
When the channel is cancelled (movement, interrupt), the PeriodicTriggerSpell Aura SHALL be immediately removed from the target, and no further missiles SHALL fire.

#### Scenario: Cancel after 1 tick
- **WHEN** the channel is cancelled after 1500ms (1 tick completed)
- **THEN** 1 OnSpellHit SHALL have fired, the Aura SHALL be removed, and no further ticks SHALL occur

### Requirement: Arcane Missiles timeline test
The skill package SHALL include an arcane_missiles_timeline_test.go that independently verifies the full event timeline: SpellCastStart, SpellLaunch, 3× SpellHit, AuraApplied, AuraExpired.

#### Scenario: Timeline output shows complete event sequence
- **WHEN** the timeline test runs
- **THEN** the rendered timeline SHALL show events in order: SpellCastStart at 0ms, SpellLaunch at 0ms, SpellHit at 1000ms, SpellHit at 2000ms, SpellHit at 3000ms, AuraExpired at 3000ms
