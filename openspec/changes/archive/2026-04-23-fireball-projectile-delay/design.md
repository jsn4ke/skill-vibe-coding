## Context

TC source analysis revealed Fireball's damage is delayed by projectile travel time. The server stores `Speed` (yards/sec), `LaunchDelay` (sec), and `MinDuration` (sec) in `SpellMisc.db2`, loaded into `SpellInfo`. When a spell has Speed > 0, after cast completes it enters `StateLaunched` with `hitDelay = LaunchDelay + max(distance / Speed, MinDuration)`. Effects are processed only when the delay expires.

Our current `HandleImmediate` always processes effects instantly. The `StateLaunched` path exists in the state machine but only fires when `LaunchDelay > 0` — it doesn't handle the Speed-based travel time calculation.

## Goals / Non-Goals

**Goals:**
- Add Speed and MinDuration to SpellInfo
- Implement per-target hit delay calculation: `hitDelay = LaunchDelay + max(distance / Speed, MinDuration)`
- Make StateLaunched actually delay effect processing until hit timer expires
- Update Fireball with a reasonable Speed value

**Non-Goals:**
- No SPELL_ATTR9_MISSILE_SPEED_IS_DELAY_IN_SEC flag support (flat delay mode) — not needed for Fireball
- No chain projectile logic (SPELL_ATTR4_BOUNCY_CHAIN_MISSILES) — not needed for single-target Fireball
- No SpellEvent scheduler system — we use Update() tick-based approach instead
- No LaunchDelay for now (Fireball has LaunchDelay = 0 in SpellMisc)

## Decisions

### 1. Per-Target TimeDelay on TargetInfo

**Decision**: Add `TimeDelay int32` field to `TargetInfo`. During `Cast()`, if Speed > 0, calculate per-target delay and store it. The Update loop checks elapsed time against each target's TimeDelay.

**Rationale**: TC uses per-target delay because chain spells and area spells can have different distances. For Fireball (single target) it's one delay, but the per-target design is future-proof.

**Alternative considered**: Single delay on Spell → breaks for multi-target spells later.

### 2. Update-Based Timer Instead of Event System

**Decision**: Use the existing `Update(diffMs)` loop to count down a single `hitTimer` on the Spell. When it reaches 0, process effects. This matches our timer-less architecture.

**Rationale**: TC uses a separate SpellEvent system with the global event scheduler. Our framework uses `Update(diffMs)` driven by the game loop. Adding a hitTimer field to Spell is simpler and consistent.

**Alternative considered**: Use timer.Scheduler → overkill for this; introduces async complexity.

### 3. Minimum Distance Clamp at 5 Yards

**Decision**: Use `max(distance, 5.0)` for distance calculation, matching TC's `GetDistance()` minimum clamp.

**Rationale**: Prevents near-zero travel time at point-blank range, which would make projectile spells behave identically to instant spells at melee range.

### 4. Distance from Entity.Position

**Decision**: Use `entity.Position.DistanceTo()` for distance calculation. Spell needs access to caster and target positions via Caster interface and a new Target interface.

**Rationale**: The Caster interface already has `GetPosition()`. We need target position for distance calculation. Add `GetTargetPosition()` or compute distance in Cast() using target lookup.

**Alternative considered**: Pass distance as parameter → caller shouldn't need to know about projectile mechanics.

## Risks / Trade-offs

- **[Risk] Caster interface needs target position** → Mitigation: Add `GetTargetPosition(targetID uint64) Position` to Caster interface, or use the distance from entity.Position directly in Spell.
- **[Risk] Tests need to account for delayed effects** → Mitigation: Tests call Update(hitDelayMs) to advance past the projectile travel time.
- **[Trade-off] Single hitTimer instead of per-target timers** → Simpler but won't support multi-target projectiles with different distances in the future. Acceptable for now since Fireball is single-target.
