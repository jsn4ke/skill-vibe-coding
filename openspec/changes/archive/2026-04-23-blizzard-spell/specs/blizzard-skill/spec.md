## ADDED Requirements

### Requirement: Blizzard spell definition
The system SHALL define a Blizzard spell (ID 10, Rank 1) with the following properties:
- Name: "Blizzard"
- Cast time: 0ms (instant cast, channeled)
- Duration: 8000ms (8 seconds channel)
- Power cost: 320 mana
- Range: 30 yards
- AoE radius: 8 yards
- IsChanneled: true
- School: Frost
- Effects: one Persistent Area Aura with periodic damage
- Slow effect: NOT part of base spell (added by Improved Blizzard talent, future extension)

#### Scenario: Spell info fields are correctly populated
- **WHEN** the Blizzard spell Info is inspected
- **THEN** ID SHALL be 10, Name SHALL be "Blizzard", CastTime SHALL be 0, Duration SHALL be 8000, PowerCost SHALL be 320, IsChanneled SHALL be true

### Requirement: Blizzard cast enters channeling state
When a caster casts Blizzard, the spell SHALL enter `StateChanneling` with Timer set to Duration (8000ms). The spell SHALL NOT use projectile delay (Speed = 0).

#### Scenario: Successful cast transitions to channeling
- **WHEN** a valid caster casts Blizzard with a valid target position
- **THEN** spell state SHALL be `StateChanneling` and Timer SHALL be 8000

#### Scenario: Channel completes naturally
- **WHEN** Update() is called until Timer reaches 0
- **THEN** spell state SHALL be `StateFinished` with result `CastOK`

### Requirement: Blizzard applies persistent area aura
Upon cast, Blizzard SHALL create an area-level Aura with:
- Type: `AuraPeriodicDamage`
- Duration: 8 seconds
- Tick period: 1 second
- Base damage per tick: 25
- SP coefficient per tick: 0.042
- Total expected ticks: 8
- The aura SHALL hold the center position and radius for target re-selection each tick

#### Scenario: Aura is applied to target area
- **WHEN** Blizzard is cast
- **THEN** an aura of type `AuraPeriodicDamage` SHALL be created with Duration 8s and tick Period 1s

#### Scenario: Aura deals periodic damage with spell power bonus
- **WHEN** the aura ticks with caster SpellPower = 100
- **THEN** each tick SHALL deal 25 + 0.042 × 100 = 29.2 damage

### Requirement: Blizzard re-selects targets each tick
Each tick SHALL re-select enemy units within 8 yards of the target position using `targeting.SelectArea`. This matches TC's DynObjAura.FillTargetMap flow. Units entering the area after cast SHALL be hit on the next tick; units leaving SHALL stop being hit.

#### Scenario: Enemies in range are targeted
- **WHEN** Blizzard ticks at a position with 3 enemies within 8 yards
- **THEN** all 3 enemies SHALL receive tick damage

#### Scenario: Enemy enters area after cast
- **WHEN** an enemy moves into the Blizzard area between ticks
- **THEN** the next tick SHALL select and damage that enemy

#### Scenario: Enemy leaves area after cast
- **WHEN** an enemy moves out of the Blizzard area between ticks
- **THEN** the next tick SHALL NOT select or damage that enemy

### Requirement: Blizzard consumes mana
Blizzard SHALL consume 320 mana upon cast, unless `TriggeredIgnorePower` flag is set.

#### Scenario: Mana is deducted on cast
- **WHEN** a caster with sufficient mana casts Blizzard
- **THEN** 320 mana SHALL be deducted from the caster's power

### Requirement: Blizzard cancel removes aura immediately
If the spell is cancelled while in `StateChanneling`, the spell SHALL finish with `CastFailedInterrupted` AND the area aura SHALL be removed immediately via `auraMgr.RemoveAurasBySpellID()`. This matches TC behavior: cancel → DynamicObject destroyed → Aura removed.

#### Scenario: Cancel during channeling removes aura
- **WHEN** Cancel() is called during StateChanneling
- **THEN** spell state SHALL be `StateFinished` with result `CastFailedInterrupted` AND the Blizzard aura SHALL be removed from the aura manager
