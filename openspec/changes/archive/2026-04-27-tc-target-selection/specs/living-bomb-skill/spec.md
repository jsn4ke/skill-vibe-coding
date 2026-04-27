## MODIFIED Requirements

### Requirement: Living Bomb explosion spell (44461)

The package SHALL define a SpellInfo for spell 44461 "Living Bomb Explode": instant, no cost, with EffectDummy and EffectSchoolDamage (BonusCoeff=0.14, Radius=10.0, TargetA=TargetUnitAreaEnemy). The explosion SHALL use the data-driven target selection system to automatically select all enemies within 10yd of the explosion center (the bomb carrier's position), excluding the carrier. The `AoESelector` interface SHALL NOT be used; target selection SHALL be driven entirely by SpellInfo's TargetA + Radius fields.

#### Scenario: Explosion hits enemies within 10yd
- **WHEN** spell 44461 is cast and 3 enemies are within 10yd of the bomb carrier
- **THEN** all 3 enemies SHALL receive fire damage (`basePoints + 0.14 × SP`)

#### Scenario: Explosion excludes bomb carrier
- **WHEN** spell 44461 is cast on target A (the bomb carrier)
- **THEN** target A SHALL NOT be in the TargetInfos (excluded by area target selection with caster exclude)

#### Scenario: Explosion spreads Living Bomb when canSpread=true
- **WHEN** spell 44461's OnEffectHit hook fires with SpellValues[2] > 0
- **AND** enemy B is hit by the explosion
- **THEN** spell 217694 SHALL be cast on enemy B with SpellValues[2] = 0

#### Scenario: Explosion does NOT spread when canSpread=false
- **WHEN** spell 44461's OnEffectHit hook fires with SpellValues[2] = 0
- **THEN** NO new spell 217694 SHALL be cast on hit targets

### Requirement: Living Bomb skill tests

The `skills/living-bomb` package SHALL include `living_bomb_engine_test.go` covering: full cast lifecycle via `eng.CastSpell()`, resource consumption, aura application, periodic damage ticks, explosion on expiry, no explosion on death, AoE damage via data-driven target selection (no WithAoE), spread mechanic, spread chain termination, and movement interrupt. Tests SHALL NOT use `WithAoE()` — all AoE targeting SHALL be resolved by the engine's target selection system from SpellInfo data.

#### Scenario: Test file exists and passes
- **WHEN** `go test ./skills/living-bomb/... -v` is run
- **THEN** all tests SHALL pass

#### Scenario: AoE targeting works without WithAoE
- **WHEN** Living Bomb explosion is cast via `eng.CastSpell(caster, &ExplosionInfo, engine.WithTriggered())`
- **THEN** the engine's target selection SHALL resolve enemies within Radius from the explosion center
- **AND** no `WithAoE()` call SHALL be present in the test code
