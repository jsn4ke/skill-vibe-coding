## ADDED Requirements

### Requirement: Living Bomb cast spell (44457)

The `skills/living-bomb` package SHALL define a SpellInfo for spell 44457 "Living Bomb": instant cast, 35yd range, GCD, with a single EffectDummy effect. The `CastLivingBomb` function SHALL create a spell instance, and upon HandleImmediate, the registered script hook SHALL intercept the Dummy effect and cast spell 217694 on the target with SpellValues[2]=1 (canSpread=true).

#### Scenario: Player casts Living Bomb on enemy target
- **WHEN** CastLivingBomb(caster, targetID, auraMgr, bus) is called
- **THEN** a Spell(44457) SHALL be created, Prepare() SHALL succeed with CastOK
- **AND** upon HandleImmediate, spell 217694 SHALL be cast on targetID with TriggeredFullMask

#### Scenario: Living Bomb passes canSpread flag to periodic spell
- **WHEN** the script intercepts 44457's Dummy effect
- **THEN** the triggered spell 217694 SHALL have SpellValues[2] = 1

### Requirement: Living Bomb periodic spell (217694)

The package SHALL define a SpellInfo for spell 217694 "Living Bomb Periodic": instant, 4s duration, with EffectApplyAura (PeriodicDamage, 1s period) and EffectDummy (script hook point). The periodic aura SHALL tick 4 times (t=1s/2s/3s/4s), each tick dealing `basePoints + 0.06 × SP` fire damage.

#### Scenario: Periodic aura ticks 4 times
- **WHEN** aura 217694 is applied to a target with 4s duration and 1s period
- **THEN** it SHALL tick exactly 4 times at t=1000ms, 2000ms, 3000ms, 4000ms

#### Scenario: Periodic damage per tick
- **WHEN** aura 217694 ticks with BasePoints=0 and BonusCoeff=0.06
- **THEN** each tick SHALL deal `0 + 0.06 × spellPower` damage

### Requirement: Living Bomb explosion spell (44461)

The package SHALL define a SpellInfo for spell 44461 "Living Bomb Explode": instant, no cost, with EffectDummy and EffectSchoolDamage (BonusCoeff=0.14, Radius=10yd, TargetA=TargetUnitAreaEnemy). The explosion SHALL select all enemies within 10yd of the bomb carrier (excluding the carrier itself), deal fire damage, and optionally spread Living Bomb.

#### Scenario: Explosion hits enemies within 10yd
- **WHEN** spell 44461 is cast and 3 enemies are within 10yd of the bomb carrier
- **THEN** all 3 enemies SHALL receive fire damage (`basePoints + 0.14 × SP`)

#### Scenario: Explosion excludes bomb carrier
- **WHEN** spell 44461 is cast on target A (the bomb carrier)
- **THEN** target A SHALL NOT be in the TargetInfos (excluded by AoE selection)

#### Scenario: Explosion spreads Living Bomb when canSpread=true
- **WHEN** spell 44461's OnEffectHit hook fires with SpellValues[2] > 0
- **AND** enemy B is hit by the explosion
- **THEN** spell 217694 SHALL be cast on enemy B with SpellValues[2] = 0

#### Scenario: Explosion does NOT spread when canSpread=false
- **WHEN** spell 44461's OnEffectHit hook fires with SpellValues[2] = 0
- **THEN** NO new spell 217694 SHALL be cast on hit targets

### Requirement: Aura expiry triggers explosion (not death/dispel)

The aura from spell 217694 SHALL trigger explosion spell 44461 ONLY when it expires naturally (RemoveMode == RemoveByExpire). Death, dispel, and other removal modes SHALL NOT trigger the explosion.

#### Scenario: Natural expiry triggers explosion
- **WHEN** aura 217694 expires after 4s (RemoveByExpire)
- **THEN** spell 44461 SHALL be cast on the aura's target

#### Scenario: Target death does NOT trigger explosion
- **WHEN** aura 217694 is removed due to target death (RemoveByDeath)
- **THEN** spell 44461 SHALL NOT be cast

#### Scenario: Dispel does NOT trigger explosion
- **WHEN** aura 217694 is dispelled (RemoveByDispel)
- **THEN** spell 44461 SHALL NOT be cast

### Requirement: Spread chain terminates after one hop

When explosion spreads Living Bomb (SpellValues[2]=0), the spread copy's aura SHALL NOT spread further when it expires.

#### Scenario: Spread copy explosion does not spread
- **WHEN** aura 217694 (spread copy, SpellValues[2]=0) expires
- **AND** its explosion 44461 hits enemies
- **THEN** NO new spell 217694 SHALL be cast on hit enemies

### Requirement: Living Bomb skill tests

The `skills/living-bomb` package SHALL include `living_bomb_test.go` and `living_bomb_timeline_test.go` covering: full cast lifecycle, resource consumption, aura application, periodic damage ticks, explosion on expiry, no explosion on death, AoE damage, spread mechanic, spread chain termination, and timeline event ordering.

#### Scenario: Test file exists and passes
- **WHEN** `go test ./skills/living-bomb/... -v` is run
- **THEN** all tests SHALL pass

#### Scenario: Timeline test verifies event sequence
- **WHEN** a full Living Bomb lifecycle is simulated (cast → 4s DoT → expiry → explosion → spread)
- **THEN** the timeline SHALL show events in order: SpellCastStart(44457) → SpellLaunch(44457) → AuraApplied(217694) → 4×AuraTick → AuraExpired(217694) → SpellHit(44461) × AoE targets
