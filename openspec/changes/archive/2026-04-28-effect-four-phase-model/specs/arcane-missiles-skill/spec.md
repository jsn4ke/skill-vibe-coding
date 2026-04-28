## MODIFIED Requirements

### Requirement: Arcane Missiles damage per missile

Each triggered missile SHALL deal 24 + 0.132 × casterSpellPower arcane damage per hit, with independent crit rolls at 1.5x multiplier. The EffectSchoolDamage handler SHALL execute during the LaunchTarget phase (mode guard enforced).

#### Scenario: Damage without crit at 100 SP
- **WHEN** a missile hits a target with caster spellPower=100 and no crit
- **THEN** damage SHALL be 24 + 0.132 × 100 = 37.2
- **AND** the damage SHALL be calculated during the LaunchTarget phase

#### Scenario: Damage with crit at 100 SP
- **WHEN** a missile crits with caster spellPower=100
- **THEN** damage SHALL be (24 + 13.2) × 1.5 = 55.8
