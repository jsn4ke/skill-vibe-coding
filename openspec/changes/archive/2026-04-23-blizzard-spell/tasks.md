## 1. Spell Info Definition

- [x] 1.1 Create `skill-go/skills/blizzard/` package with `blizzard.go`
- [x] 1.2 Define `Info` SpellInfo: ID=10, Name="Blizzard", CastTime=0, Duration=8000, PowerCost=320, RangeMax=30, IsChanneled=true, Attributes=AttrChanneled
- [x] 1.3 Define Blizzard effect: EffectApplyAura, AuraPeriodicDamage, BasePoints=25, BonusCoeff=0.042, AuraPeriod=1000, TargetA=TargetDestTargetEnemy, TargetB=TargetUnitAreaEnemy, MiscValue=radius=8

## 2. Cast Function

- [x] 2.1 Implement `CastBlizzard(caster spell.Caster, targetID uint64, auraMgr *aura.Manager, bus *event.Bus) (*spell.Spell, spell.SpellCastResult)`
- [x] 2.2 Create spell, set Bus, call Prepare(), check result
- [x] 2.3 On successful cast, create area-level Aura (8s, 1s tick, PeriodicDamage) with center position and radius, add to auraMgr
- [x] 2.4 On spell cancel, call `auraMgr.RemoveAurasBySpellID()` to remove the aura immediately (match TC behavior)

## 3. Tick Target Re-selection

- [x] 3.1 Extend aura tick mechanism to support re-selecting targets each tick via `targeting.SelectArea(center, radius)`
- [x] 3.2 Each tick: re-query enemies in area, apply damage to currently-in-range targets only
- [x] 3.3 Units entering area between ticks are picked up on next tick; units leaving are excluded

## 4. Tests

- [x] 4.1 Create `blizzard_test.go` with testUnit/testPos helpers (reuse fireball pattern)
- [x] 4.2 Test: cast lifecycle — Prepare → Channeling → Finished
- [x] 4.3 Test: aura applied with correct type, duration, tick period
- [x] 4.4 Test: periodic damage with SP bonus (25 + 0.042 × 100 = 29.2)
- [x] 4.5 Test: cancel during channeling → CastFailedInterrupted AND aura removed
- [x] 4.6 Test: mana consumed on cast
- [x] 4.7 Test: enemy entering area after cast receives damage on next tick
- [x] 4.8 Test: enemy leaving area after cast stops receiving damage
- [x] 4.9 Run `go test ./skills/blizzard/...` and verify all pass

## 5. Demo Integration

- [x] 5.1 Add Blizzard simulation to `server/main.go` alongside existing Fireball demo
- [x] 5.2 Verify timeline output shows SpellCastStart, SpellLaunch, AuraApplied, AuraTick ×8, AuraExpired
