## ADDED Requirements

### Requirement: Spell publishes lifecycle events
The spell system SHALL publish events to an `event.Bus` (if provided) at each lifecycle transition: cast start, launch, hit, and cancel.

#### Scenario: Fireball cast start event
- **WHEN** `spell.Prepare()` succeeds and state becomes `StatePreparing`
- **THEN** an `OnSpellCastStart` event is published with SourceID=caster, SpellID=spell ID, Extra["castTime"]=cast time

#### Scenario: Fireball launch event
- **WHEN** `spell.Cast()` transitions state to `StateLaunched` (projectile in flight)
- **THEN** an `OnSpellLaunch` event is published with SourceID=caster, TargetID=target, SpellID=spell ID, Extra["speed"]=projectile speed

#### Scenario: Fireball hit event
- **WHEN** `spell.HandleImmediate()` processes effects and applies damage
- **THEN** an `OnSpellHit` event is published with SourceID=caster, TargetID=target, SpellID=spell ID, Value=damage amount, Extra["crit"]=crit boolean

#### Scenario: Spell cancel event
- **WHEN** `spell.Cancel()` is called during cast
- **THEN** an `OnSpellCancel` event is published with SourceID=caster, SpellID=spell ID, Extra["result"]=cast result

#### Scenario: No events when bus is nil
- **WHEN** `spell.Spell.Bus` is nil
- **THEN** no events are published and spell operates exactly as before

### Requirement: Aura Manager publishes lifecycle events
The aura manager SHALL publish events to an `event.Bus` (if provided) when auras are applied, tick, and expire.

#### Scenario: Aura applied event
- **WHEN** `aura.Manager.AddAura()` adds a new aura
- **THEN** an `OnAuraApplied` event is published with SourceID=caster, TargetID=target, SpellID=spell ID, Extra["auraType"]=aura type

#### Scenario: Aura tick event
- **WHEN** `aura.Manager.TickPeriodic()` processes a periodic tick
- **THEN** an `OnAuraTick` event is published with SourceID=caster, TargetID=target, SpellID=spell ID, Value=tick amount

#### Scenario: Aura expired event
- **WHEN** `aura.Manager.TickPeriodic()` detects an expired aura
- **THEN** an `OnAuraExpired` event is published with TargetID=target, SpellID=spell ID

#### Scenario: No events when manager bus is nil
- **WHEN** `aura.Manager` is created with nil bus
- **THEN** no events are published and aura operates exactly as before

### Requirement: TimelineRenderer produces ASCII timeline
A `TimelineRenderer` SHALL subscribe to spell and aura events, collect them, and render a formatted ASCII timeline showing the sequence of events with relative timestamps.

#### Scenario: Fireball full lifecycle timeline
- **WHEN** a Fireball is cast (3.5s cast, projectile hit, DoT applied, 4 ticks)
- **THEN** the rendered timeline shows events in order with correct relative timestamps: cast start at 0ms, hit at 3650ms, aura ticks at 5650/7650/9650/11650ms

#### Scenario: Timeline format
- **WHEN** `TimelineRenderer.Render()` is called
- **THEN** output is a table with columns: Time (ms), Event, Source→Target, Detail

#### Scenario: Empty timeline
- **WHEN** no events were collected
- **THEN** `Render()` returns an empty string or "no events" message
