## Context

The skill-go project has a complete framework with 10 packages (entity, stat, event, timer, combat, spell, effect, aura, cooldown, targeting, proc, diminishing, script). All effect handlers exist as stubs. The Fireball spell is the first real spell implementation — it validates the framework end-to-end by exercising: standard cast flow with 3.5s cast time, school damage effect, aura-based periodic damage, and cooldown/GCD integration.

The skill design document at `skill-designs/fireball.md` contains the full WoW data mapping. All components map directly to existing framework types with zero custom mechanisms needed.

## Goals / Non-Goals

**Goals:**
- Implement Fireball as a complete, working spell using only existing framework types
- Fill in the `handleSchoolDamage` and `handleApplyAura` effect handler stubs with real logic
- Enable damage calculation with base points + random variance + SP coefficient scaling
- Enable aura periodic damage with SP coefficient scaling on ticks
- Validate the complete cast lifecycle through tests

**Non-Goals:**
- No new framework abstractions or package additions
- No spell script hooks for Fireball (no special behavior beyond data)
- No combat log or network layer integration
- No haste rating affecting cast time (future enhancement)
- No spell hit rating mechanics (assumed 100% hit for now)

## Decisions

### 1. Damage Formula: Direct Calculation in Effect Handler

**Decision**: `handleSchoolDamage` computes `damage = basePoints + rand(-variance, +variance) + coeff * casterSpellPower` and stores it in Context.FinalDamage.

**Rationale**: Simple, testable, follows TC's CalculateSpellDamage pattern. No intermediate modifiers for this first spell — talent/aura modifiers can be added via the existing stat.Modifier chain later.

**Alternative considered**: Separate DamageCalculator service → over-engineering for one spell. The effect handler IS the calculator.

### 2. SpellEffectInfo Extended Fields

**Decision**: Add `AuraType uint16`, `AuraPeriod int64` (milliseconds), and `BaseDieSides int32` fields to `SpellEffectInfo` to carry aura and variance data.

**Rationale**: `EffectApplyAura` needs to know WHICH aura type to apply. `AuraPeriod` drives the tick interval. `BaseDieSides` provides random variance range. These are static data fields that belong on SpellEffectInfo alongside existing BasePoints and BonusCoeff.

**Alternative considered**: Separate AuraInfo struct → adds indirection for no benefit since these fields are 1:1 with the effect.

### 3. Aura Tick SP Scaling via AuraEffect.Amount + BonusCoeff

**Decision**: Store the coefficient on AuraEffect (carried from SpellEffectInfo.BonusCoeff). During TickPeriodic, calculate `tickDamage = effect.Amount + effect.BonusCoeff * casterSP`.

**Rationale**: The aura needs to remember its scaling coefficient for the full duration. Storing it on AuraEffect keeps the tick self-contained. The caster's SP is resolved from the stat system at tick time (not snapshot at cast time), matching TC behavior for most DoTs.

**Alternative considered**: Snapshot SP at cast time → simpler but doesn't match TC behavior and prevents future buffs from affecting ongoing DoTs.

### 4. Context-Based Effect Communication

**Decision**: `effect.Context` carries CasterSpellPower resolved once, then each handler reads it. Results stored on Context (BaseDamage, FinalDamage, AppliedAura).

**Rationale**: Avoids each handler independently querying stats. Single resolution point for SP. Context is the standard pattern in the existing framework.

### 5. Test Strategy: Unit Tests on Handlers + Integration Test on Full Cast

**Decision**: Unit tests for `handleSchoolDamage` and `handleApplyAura` with controlled inputs. One integration test creating a real Spell, running Prepare → Update → verify damage + aura.

**Rationale**: Unit tests verify calculation correctness. Integration test verifies the framework wiring. This gives confidence without over-testing internals.

## Risks / Trade-offs

- **[Risk] No spell hit mechanics** → Mitigation: Assume 100% hit for now. Add HitChance stat and roll later. The framework already has HitResult enum with Miss/Immune.
- **[Risk] No variance seeding** → Mitigation: Use math/rand with fixed seed in tests for deterministic assertions. Production uses random seed.
- **[Risk] Aura tick timing precision** → Mitigation: Timer scheduler uses 10ms ticker. DoT ticks (2s interval) have sufficient precision. Sub-100ms ticks would need a different approach but Fireball doesn't require it.
- **[Trade-off] Caster SP resolved at tick time, not snapshotted** → If future DoTs need snapshotting, add a `SnapshotSpellPower` field to AuraEffect. Currently not needed.
