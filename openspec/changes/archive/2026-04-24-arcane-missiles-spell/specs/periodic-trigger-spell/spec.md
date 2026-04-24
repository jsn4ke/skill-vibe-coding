## ADDED Requirements

### Requirement: AuraPeriodicTriggerSpell type definition
The aura package SHALL define `AuraPeriodicTriggerSpell` as a new AuraType constant, distinct from existing types (AuraPeriodicDamage, AuraPeriodicHeal, etc.).

#### Scenario: Type constant exists and is distinct
- **WHEN** AuraPeriodicTriggerSpell is compared to other AuraType values
- **THEN** it SHALL be unique (not equal to AuraNone, AuraPeriodicDamage, or any existing type)

### Requirement: TickPeriodic handles PeriodicTriggerSpell aura type
The Manager.TickPeriodic method SHALL process auras with AuraPeriodicTriggerSpell type by firing the onTick callback every AuraPeriod milliseconds, identical to how it handles AuraPeriodicDamage. The tick behavior (what happens on each tick) is determined by the caller's onTick callback.

#### Scenario: PeriodicTriggerSpell aura ticks every 1000ms
- **WHEN** a PeriodicTriggerSpell aura with period=1000ms is ticked over 3000ms
- **THEN** the onTick callback SHALL be invoked exactly 3 times

#### Scenario: PeriodicTriggerSpell aura expires after duration
- **WHEN** a PeriodicTriggerSpell aura with duration=3000ms is ticked past its duration
- **THEN** the aura SHALL be removed and OnAuraExpired SHALL be published

### Requirement: Triggered spell creates full Spell instance
On each PeriodicTriggerSpell tick, the onTick callback SHALL create a new `spell.Spell` instance using the triggered spell's SpellInfo, set CastFlags to TriggeredFullMask, set the target, set the event bus, and call Prepare(). The triggered spell SHALL go through the complete Prepare → Cast → HandleImmediate → Finish lifecycle.

#### Scenario: Each tick creates and executes a full Spell
- **WHEN** the onTick callback fires for a PeriodicTriggerSpell aura
- **THEN** a new Spell SHALL be created with TriggeredFullMask, Prepare() SHALL be called, and the spell SHALL reach StateFinished with CastOK

#### Scenario: Triggered spell publishes OnSpellHit
- **WHEN** the triggered spell's HandleImmediate executes
- **THEN** an OnSpellHit event SHALL be published with the triggered spell's SpellID, the correct damage value, and crit status

### Requirement: AuraEffect carries TriggerSpellID
AuraEffect SHALL carry a TriggerSpellID field that identifies which spell to trigger on each periodic tick. This field is populated from SpellEffectInfo.TriggerSpellID when the aura is created.

#### Scenario: TriggerSpellID propagated from effect to aura
- **WHEN** an aura is created from a SpellEffectInfo with TriggerSpellID=7268
- **THEN** the AuraEffect.TriggerSpellID SHALL be 7268
