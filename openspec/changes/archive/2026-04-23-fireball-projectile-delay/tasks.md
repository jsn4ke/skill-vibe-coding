## 1. Data Model

- [x] 1.1 Add `Speed float64` and `MinDuration uint32` fields to `SpellInfo` in `pkg/spell/info.go`
- [x] 1.2 Add `HitTimer int32` field to `Spell` for tracking projectile travel countdown

## 2. Distance and Delay Calculation

- [x] 2.1 Add `GetTargetPosition(targetID uint64) Position` to Caster interface in `pkg/spell/spell.go`
- [x] 2.2 Implement hit delay calculation in `Spell.Cast()`: if Speed > 0, compute `hitDelay = max(distance / Speed, MinDuration)`, set HitTimer, enter StateLaunched

## 3. StateLaunched Update Loop

- [x] 3.1 Update `Spell.Update()` to handle StateLaunched: decrement HitTimer, when it reaches 0 call `HandleImmediate()` to process effects
- [x] 3.2 Ensure StateLaunched path still checks caster alive, movement interrupt (projectile should NOT be interruptible by movement after launch — remove AttrBreakOnMove check for Launched state)

## 4. Wire Fireball

- [x] 4.1 Update Fireball SpellInfo in `server/main.go` with Speed value (e.g., 20.0 yards/sec based on TC data)
- [x] 4.2 Update main.go Update calls: first Update(3500) for cast, then Update(hitDelay) for projectile

## 5. Tests

- [x] 5.1 Update existing tests: full cast lifecycle now requires two Update calls (cast time + hit delay)
- [x] 5.2 Add test: no damage applied before projectile arrives
- [x] 5.3 Add test: hit delay calculation with known distance and speed
- [x] 5.4 Add test: MinDuration clamp and 5-yard distance clamp
- [x] 5.5 Update Caster interface implementations (testUnit in tests, Unit in main.go) with GetTargetPosition
- [x] 5.6 Run `go test -race ./...` to verify no concurrency issues
