## Why

The skill-go framework has complete infrastructure (spell, effect, aura, cooldown, targeting, proc, diminishing, script, timer, combat) but no actual WoW spell has been fully implemented end-to-end. Fireball is the ideal first spell — it exercises the core cast flow (3.5s cast time), damage effect pipeline, and aura-based periodic damage, all using existing framework mappings with zero custom mechanisms.

## What Changes

- Add Fireball (Spell ID 25306) spell definition with correct data (cast time, mana cost, range, effects)
- Implement `handleSchoolDamage` effect handler with damage calculation formula (base + random variance + coeff × SP)
- Implement `handleApplyAura` effect handler to create Aura instances from SpellEffectInfo
- Enhance Aura periodic damage tick calculation to use BonusCoeff × SP scaling
- Wire the complete cast flow: Prepare → CheckCast → cast time countdown → Cast → effect dispatch → aura management
- Add unit tests for damage calculation, DoT ticks, and full cast lifecycle

## Capabilities

### New Capabilities

- `fireball-spell`: Complete Fireball spell implementation including spell definition, damage calculation, DoT aura application, and full cast lifecycle

### Modified Capabilities

## Impact

- `pkg/effect/effect.go`: Implement actual logic in `handleSchoolDamage` and `handleApplyAura` stubs
- `pkg/aura/aura.go`: Enhance TickPeriodic to support BonusCoeff-based SP scaling
- `pkg/spell/info.go`: Add AuraType and AuraPeriod fields to SpellEffectInfo
- `pkg/combat/combat.go`: Damage application integration
- `server/main.go`: Update demo to use real Fireball definition
- New test files for effect handlers and integration
