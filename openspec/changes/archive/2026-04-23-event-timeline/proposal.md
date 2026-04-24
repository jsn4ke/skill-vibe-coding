## Why

The skill system has a fully implemented event bus (`pkg/event/`) but it is never used — all subsystems (spell, aura, effect) operate silently. There is no way to observe spell behavior over time, making debugging and skill design verification difficult. The current console demo only prints ad-hoc `fmt.Printf` statements rather than structured, replayable event output.

## What Changes

- Wire the existing `event.Bus` into spell lifecycle (cast start, launch, hit, cancel)
- Wire `event.Bus` into aura lifecycle (apply, tick, expire)
- Add `event.Bus` injection path through `spell.Spell` (or `spell.Caster`)
- Create a `TimelineRenderer` that subscribes to events and outputs a structured ASCII timeline
- Update `server/main.go` to use the timeline renderer instead of ad-hoc prints
- Update `skills/fireball/` to publish events during `CastFireball`

## Capabilities

### New Capabilities
- `event-timeline`: Structured event emission from spell/aura systems and ASCII timeline rendering

### Modified Capabilities

## Impact

- `pkg/spell/spell.go`: Add bus field, publish events at state transitions
- `pkg/aura/aura.go`: Publish events on apply/tick/expire
- `pkg/event/event.go`: Possibly add new event types if needed
- `skills/fireball/fireball.go`: Accept and use bus parameter
- `server/main.go`: Wire timeline renderer, remove ad-hoc prints
- New file: `pkg/timeline/` or inline renderer in event subscriber
