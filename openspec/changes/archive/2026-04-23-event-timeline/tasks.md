## 1. Event Types

- [x] 1.1 Add new event types to `pkg/event/event.go`: `OnSpellCastStart`, `OnSpellLaunch`, `OnSpellCancel`, `OnAuraExpired`

## 2. Spell Bus Integration

- [x] 2.1 Add `Bus *event.Bus` field to `spell.Spell` struct
- [x] 2.2 Publish `OnSpellCastStart` in `Prepare()` when state becomes `StatePreparing`
- [x] 2.3 Publish `OnSpellLaunch` in `Cast()` when state becomes `StateLaunched`
- [x] 2.4 Publish `OnSpellHit` in `HandleImmediate()` after effects are processed
- [x] 2.5 Publish `OnSpellCancel` in `Cancel()` before finishing

## 3. Aura Bus Integration

- [x] 3.1 Change `aura.NewManager()` to accept `*event.Bus` parameter
- [x] 3.2 Publish `OnAuraApplied` in `AddAura()` when a new aura is added (not refreshed)
- [x] 3.3 Publish `OnAuraTick` in `TickPeriodic()` when a tick fires
- [x] 3.4 Publish `OnAuraExpired` in `TickPeriodic()` when aura expires
- [x] 3.5 Update all callers of `aura.NewManager()` (main.go, fireball tests)

## 4. TimelineRenderer

- [x] 4.1 Create `pkg/timeline/renderer.go` with `TimelineRenderer` struct
- [x] 4.2 Implement `SubscribeAll(bus *event.Bus)` to subscribe to spell and aura events
- [x] 4.3 Implement `Render() string` that outputs formatted ASCII timeline table

## 5. Wire Up Demo

- [x] 5.1 Update `server/main.go` to create bus, pass to aura manager, create renderer
- [x] 5.2 Update `skills/fireball/fireball.go` to accept bus and set it on the spell
- [x] 5.3 Update `skills/fireball/fireball_test.go` to pass nil bus (backward compat)
- [x] 5.4 Run `go test -race ./...` and verify all tests pass
- [x] 5.5 Run `go run ./server/` and verify timeline output
