## Context

The skill system has a fully implemented `event.Bus` (pub/sub) in `pkg/event/` with 17 event types defined, but it is never wired into any subsystem. All spell/aura/effect operations execute silently. The only output mechanism is ad-hoc `fmt.Printf` in `server/main.go`.

The `event.Bus` uses a simple synchronous publish model — `Publish()` calls all registered handlers inline. This is sufficient for a single-threaded demo and avoids concurrency complexity.

## Goals / Non-Goals

**Goals:**
- Wire `event.Bus` into spell lifecycle: cast start, launch, hit, cancel/finish
- Wire `event.Bus` into aura lifecycle: apply, tick, expire
- Create a `TimelineRenderer` that collects events and outputs a formatted ASCII timeline
- Demonstrate with Fireball: cast → projectile → hit → DoT ticks
- Minimal changes to existing package APIs — inject bus rather than restructuring

**Non-Goals:**
- Real-time animation or browser rendering
- Event persistence or replay files (can add later as a subscriber)
- Async/event-loop event dispatch — synchronous is fine
- Wire events into every subsystem (proc, cooldown, diminishing) — spell + aura is enough for now

## Decisions

### 1. Bus injection via `spell.Spell` field (not Caster interface)

Add `Bus *event.Bus` as a field on `spell.Spell`. Pass it via `NewSpell()` or set it directly.

**Why not Caster interface?** Adding `GetBus()` to Caster would break all existing Caster implementations (test mocks, `Unit` in main.go, `testUnit` in fireball tests). Adding a field to `Spell` is non-breaking — nil bus = no events emitted (silent mode).

**Why not a global bus?** A global `event.DefaultBus` is tempting but makes testing harder and couples all packages to a singleton. Explicit injection is cleaner.

### 2. Aura Manager accepts Bus on construction

`aura.NewManager()` → `aura.NewManager(bus *event.Bus)`.

Manager already owns all aura operations (add, remove, tick). Publishing events there is natural. Nil bus = no events.

### 3. New event types: `OnSpellCastStart`, `OnSpellLaunch`

The existing `OnSpellCast` and `OnSpellHit` cover some cases, but we need:
- `OnSpellCastStart` — when cast begins (Prepare succeeds, StatePreparing)
- `OnSpellLaunch` — when projectile is launched (StateLaunched, before flight)
- `OnSpellCancel` — when cast is interrupted/cancelled
- `OnAuraExpired` — when aura expires (distinct from removed by dispel)

### 4. TimelineRenderer as a subscriber

A `TimelineRenderer` struct that subscribes to all relevant event types, collects events with relative timestamps, and renders an ASCII timeline on `Render()`.

```
TimelineRenderer
  ├── SubscribeAll(bus)
  ├── Events []TimelineEvent (collected during simulation)
  └── Render() string
```

Each `TimelineEvent` stores: offset (time from start), event type, source→target, detail string.

### 5. Simulation time via explicit clock

The demo runs in simulation time (no real delays). `TimelineRenderer` tracks a virtual clock. The caller advances time explicitly. For now, the renderer uses event order + known timings to reconstruct relative timestamps from the spell's perspective.

Simpler approach: just record a sequential index and reconstruct timeline from spell info (CastTime, HitTimer, AuraPeriod).

## Risks / Trade-offs

- **Breaking `aura.NewManager()` signature** → Acceptable; only called in main.go and tests. Quick fix everywhere.
- **Bus nil checks everywhere** → Every publish site needs `if bus != nil`. Trade-off for backward compatibility. Worth it to avoid breaking changes.
- **Timeline accuracy** → The renderer reconstructs timing from spell info, not wall clock. If spell timings change mid-cast, the timeline may be slightly off. Acceptable for demo purposes.
