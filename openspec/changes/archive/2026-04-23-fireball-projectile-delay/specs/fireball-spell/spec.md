## MODIFIED Requirements

### Requirement: Full Fireball cast lifecycle
The system SHALL execute the complete Fireball cast: Prepare → 3.5s cast time → StateLaunched → projectile travel delay → effects dispatch → direct damage + DoT aura applied. Damage and aura are NOT applied at cast completion — they are delayed by the projectile travel time.

#### Scenario: Complete cast hits target with damage and DoT
- **WHEN** a caster with 100 SP casts Fireball at a target, Update(3500) completes the cast, then Update(hitDelayMs) advances past the projectile travel time
- **THEN** the target SHALL receive school damage (678 + 100 + variance) and a periodic damage aura lasting 8 seconds with 2s tick interval

#### Scenario: No damage before projectile arrives
- **WHEN** a caster casts Fireball, Update(3500) completes the cast (enters StateLaunched), but Update has NOT advanced past the hit delay
- **THEN** no damage SHALL be applied and no aura SHALL be created

#### Scenario: Cast fails if caster has insufficient mana
- **WHEN** a caster with less than 410 mana attempts to cast Fireball
- **THEN** Prepare SHALL return CastFailedNoPower

#### Scenario: Cast fails if target is out of range
- **WHEN** the target is more than 35 yards from the caster
- **THEN** Prepare SHALL return CastFailedOutOfRange

## ADDED Requirements

### Requirement: SpellInfo stores projectile speed
SpellInfo SHALL have Speed (float64, yards/second) and MinDuration (uint32, milliseconds) fields for projectile travel time calculation.

#### Scenario: Fireball has projectile speed
- **WHEN** Fireball SpellInfo is created with Speed > 0
- **THEN** the spell SHALL enter StateLaunched after cast completes, with hitDelay = max(distance / Speed, MinDuration)

#### Scenario: Instant spell has zero speed
- **WHEN** a spell has Speed = 0
- **THEN** the spell SHALL process effects immediately at cast completion (no StateLaunched delay)

### Requirement: Projectile hit delay is calculated from distance
The hit delay SHALL be calculated as: `hitDelay = max(distance / Speed, MinDuration)` in milliseconds, where distance is the caster-to-target distance in yards, with a minimum clamp of 5 yards.

#### Scenario: Hit delay at 30 yards with Speed 20
- **WHEN** Fireball (Speed=20 y/s) is cast at a target 30 yards away
- **THEN** hit delay SHALL be 1500ms (30 / 20 * 1000)

#### Scenario: Hit delay clamped by MinDuration
- **WHEN** Fireball (Speed=20 y/s, MinDuration=500ms) is cast at a target 5 yards away
- **THEN** hit delay SHALL be max(250, 500) = 500ms (MinDuration wins)

#### Scenario: Distance clamped to minimum 5 yards
- **WHEN** Fireball (Speed=20 y/s) is cast at a target 2 yards away
- **THEN** distance SHALL be clamped to 5 yards, hit delay = 250ms
