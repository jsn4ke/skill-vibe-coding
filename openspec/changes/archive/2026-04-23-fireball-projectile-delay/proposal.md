## Why

The initial Fireball implementation applied damage instantly when the 3.5s cast completed, matching only Spell.dbc data from Wowhead. TC source analysis reveals that Fireball is a projectile spell — damage should be applied when the projectile reaches the target, not at cast completion. The server uses `SpellMisc.Speed` (yards/second) to calculate `hitDelay = LaunchDelay + max(distance / Speed, MinDuration)`, then enters `StateLaunched` until the delay expires.

## What Changes

- Add `Speed` and `MinDuration` fields to `SpellInfo` (matching TC's SpellMisc fields)
- Implement dynamic `StateLaunched` delay: calculate per-target hit delay from distance and speed
- Split `HandleImmediate` into two paths: instant (Speed == 0) and projectile (Speed > 0)
- For projectile spells, delay effect processing (damage + aura) until the hit timer expires
- Update Fireball SpellInfo with Speed value from SpellMisc.db2
- Update tests to verify delayed hit behavior

## Capabilities

### New Capabilities

### Modified Capabilities
- `fireball-spell`: Add projectile travel time delay between cast completion and damage application. Damage is no longer instant at cast finish — it is delayed by `distance / Speed` milliseconds while the spell is in StateLaunched.

## Impact

- `pkg/spell/info.go`: Add Speed, MinDuration fields to SpellInfo
- `pkg/spell/spell.go`: Implement StateLaunched delay path, split HandleImmediate
- `pkg/entity/entity.go`: May need distance calculation method exposed for SpellInfo
- `server/main.go`: Update Fireball with Speed, adjust Update calls for projectile timing
- `pkg/spell/spell_test.go`: Update tests for delayed hit behavior
- `skill-designs/fireball.md`: Update design doc with projectile data
