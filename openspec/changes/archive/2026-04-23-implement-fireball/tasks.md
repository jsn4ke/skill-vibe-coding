## 1. Data Model Extensions

- [x] 1.1 Add `AuraType`, `AuraPeriod`, and `BaseDieSides` fields to `SpellEffectInfo` in `pkg/spell/info.go`
- [x] 1.2 Add `BonusCoeff` field to `AuraEffect` in `pkg/aura/aura.go`

## 2. Effect Handler Implementation

- [x] 2.1 Implement `handleSchoolDamage` in `pkg/effect/effect.go` — calculate damage from basePoints + variance + coeff×SP, handle crit multiplier
- [x] 2.2 Implement `handleApplyAura` in `pkg/effect/effect.go` — create Aura with AuraEffects from SpellEffectInfo, set stack rule to StackRefresh
- [x] 2.3 Add CasterSpellPower resolution to `effect.Context` so handlers can read SP without querying stats directly

## 3. Aura Tick SP Scaling

- [x] 3.1 Update `TickPeriodic` in `pkg/aura/aura.go` to calculate tick damage using `Amount + BonusCoeff × casterSpellPower`

## 4. Spell Integration

- [x] 4.1 Ensure `Spell.Cast()` resolves caster SP into `effect.Context` before calling `ProcessAll`
- [x] 4.2 Update `server/main.go` Fireball SpellInfo to use correct data (ID 25306, CastTime 3500, PowerCost 410, BasePoints 678, BonusCoeff 1.0, aura fields)

## 5. Tests

- [x] 5.1 Add unit tests for `handleSchoolDamage` — verify damage formula with known SP, zero SP, and crit
- [x] 5.2 Add unit tests for `handleApplyAura` — verify aura creation with correct type, period, and stack rule
- [x] 5.3 Add unit tests for DoT tick SP scaling — verify tick damage = Amount + coeff×SP
- [x] 5.4 Add integration test for full Fireball cast lifecycle — Prepare → Update(3500) → verify damage + aura
- [x] 5.5 Run `go test -race ./...` to verify no concurrency issues
