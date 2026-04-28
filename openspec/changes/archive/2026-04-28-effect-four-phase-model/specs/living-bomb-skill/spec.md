## MODIFIED Requirements

### Requirement: Living Bomb cast spell (44457)

The `skills/living-bomb` package SHALL define a SpellInfo for spell 44457 "Living Bomb": instant cast, 35yd range, GCD, with a single EffectDummy effect. The `CastLivingBomb` function SHALL create a spell instance, and upon HandleImmediate, the registered script hook SHALL intercept the Dummy effect via HookOnEffectHitTarget and cast spell 217694 on the target with SpellValues[2]=1 (canSpread=true).

#### Scenario: Player casts Living Bomb on enemy target
- **WHEN** CastLivingBomb(caster, targetID, auraMgr, bus) is called
- **THEN** a Spell(44457) SHALL be created, Prepare() SHALL succeed with CastOK
- **AND** upon HandleImmediate, spell 217694 SHALL be cast on targetID with TriggeredFullMask

#### Scenario: Living Bomb passes canSpread flag to periodic spell
- **WHEN** the script intercepts 44457's Dummy effect via HookOnEffectHitTarget
- **THEN** the triggered spell 217694 SHALL have SpellValues[2] = 1

### Requirement: Living Bomb explosion spell (44461)

The package SHALL define a SpellInfo for spell 44461 "Living Bomb Explode": instant, no cost, with EffectDummy and EffectSchoolDamage (BonusCoeff=0.14, Radius=10.0, TargetA=TargetUnitAreaEnemy). The explosion SHALL use the data-driven target selection system to automatically select all enemies within 10yd of the explosion center (the bomb carrier's position), excluding the carrier. The script hook for spread logic SHALL be registered on HookOnEffectHitTarget instead of HookOnEffectHit.

#### Scenario: Explosion spreads Living Bomb when canSpread=true
- **WHEN** spell 44461's HookOnEffectHitTarget hook fires for EFFECT_1 with SpellValues[2] > 0
- **AND** enemy B is hit by the explosion
- **THEN** spell 217694 SHALL be cast on enemy B with SpellValues[2] = 0

#### Scenario: Explosion does NOT spread when canSpread=false
- **WHEN** spell 44461's HookOnEffectHitTarget hook fires with SpellValues[2] = 0
- **THEN** NO new spell 217694 SHALL be cast on hit targets
