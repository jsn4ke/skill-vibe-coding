## ADDED Requirements

### Requirement: Fireball spell definition
The system SHALL provide a Fireball spell (ID 25306) with the following data: cast time 3500ms, mana cost 410, range 35 yards, no cooldown, two effects (school damage and apply aura with periodic damage).

#### Scenario: SpellInfo contains correct Fireball data
- **WHEN** a Fireball SpellInfo is created
- **THEN** it SHALL have ID 25306, CastTime 3500, PowerCost 410, RangeMax 35, and exactly 2 SpellEffectInfo entries

#### Scenario: Fireball has school damage effect
- **WHEN** the first SpellEffectInfo of Fireball is inspected
- **THEN** EffectType SHALL be EffectSchoolDamage, BasePoints SHALL be 678 (average of 596-760), BonusCoeff SHALL be 1.0, and TargetA SHALL be TargetUnitTargetEnemy

#### Scenario: Fireball has apply aura effect
- **WHEN** the second SpellEffectInfo of Fireball is inspected
- **THEN** EffectType SHALL be EffectApplyAura, AuraType SHALL be AuraPeriodicDamage, BasePoints SHALL be 19, BonusCoeff SHALL be 0.125, AuraPeriod SHALL be 2000ms, and TargetA SHALL be TargetUnitTargetEnemy

### Requirement: School damage effect calculates damage with SP scaling
The handleSchoolDamage effect handler SHALL calculate damage as: basePoints + random variance + BonusCoeff × casterSpellPower. The result SHALL be stored in Context.FinalDamage.

#### Scenario: Damage with known spell power
- **WHEN** handleSchoolDamage processes an effect with BasePoints 678, BonusCoeff 1.0, and caster has 100 spell power
- **THEN** Context.FinalDamage SHALL be 778 + random variance (678 + 100)

#### Scenario: Damage with zero spell power
- **WHEN** handleSchoolDamage processes an effect with BasePoints 678, BonusCoeff 1.0, and caster has 0 spell power
- **THEN** Context.FinalDamage SHALL be 678 + random variance

#### Scenario: Crit doubles damage multiplier
- **WHEN** handleSchoolDamage processes with a crit result
- **THEN** Context.FinalDamage SHALL be multiplied by 1.5

### Requirement: Apply aura effect creates periodic damage aura
The handleApplyAura effect handler SHALL create an Aura with the correct SpellID, CasterID, TargetID, AuraType, Duration, and AuraEffects based on SpellEffectInfo data.

#### Scenario: Aura created from spell effect
- **WHEN** handleApplyAura processes an effect with AuraType AuraPeriodicDamage, BasePoints 19, AuraPeriod 2000
- **THEN** an Aura SHALL be created with AuraPeriodicDamage type, containing one AuraEffect with Amount 19, Period 2 seconds, and BonusCoeff 0.125

#### Scenario: Aura stacking uses refresh rule
- **WHEN** the same Fireball aura is applied to a target that already has it
- **THEN** the aura's duration SHALL be refreshed to 8 seconds without resetting the tick timer

### Requirement: Aura periodic damage ticks scale with SP
When TickPeriodic processes a periodic damage aura, each tick SHALL calculate damage as: Amount + BonusCoeff × casterSpellPower.

#### Scenario: DoT tick with spell power
- **WHEN** TickPeriodic processes a Fireball DoT aura with Amount 19, BonusCoeff 0.125, and caster has 100 spell power
- **THEN** tick damage SHALL be 31.25 (19 + 0.125 × 100)

#### Scenario: DoT expires after 4 ticks
- **WHEN** a Fireball DoT aura with 8s duration and 2s tick interval has been ticking for 8 seconds
- **THEN** exactly 4 ticks SHALL have fired and the aura SHALL be removed

### Requirement: Full Fireball cast lifecycle
The system SHALL execute the complete Fireball cast: Prepare → 3.5s cast time → effects dispatch → direct damage + DoT aura applied.

#### Scenario: Complete cast hits target with damage and DoT
- **WHEN** a caster with 100 SP casts Fireball at a target, and Update(3500) is called
- **THEN** the target SHALL receive school damage (678 + 100 + variance) and a periodic damage aura lasting 8 seconds with 2s tick interval

#### Scenario: Cast fails if caster has insufficient mana
- **WHEN** a caster with less than 410 mana attempts to cast Fireball
- **THEN** Prepare SHALL return CastFailedNoPower

#### Scenario: Cast fails if target is out of range
- **WHEN** the target is more than 35 yards from the caster
- **THEN** Prepare SHALL return CastFailedOutOfRange
