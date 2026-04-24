# Skill Test Rule

Every implemented skill must live in `skill-go/skills/<name>/` and have a corresponding `<skill>_test.go` that passes.

## Trigger

This rule activates when implementing, designing, or building game skills.

## Behavior

1. **Skill location**: All new skills must be implemented in `skill-go/skills/<name>/`, not in `server/main.go` or other packages
2. **Check for `<skill>_test.go`**: Every skill package must have a test file
3. **Run tests**: After any skill implementation, run:
   ```bash
   go test ./skills/...
   ```
4. **Do not mark skill implementation as complete** until its test file exists and all tests pass

## Required Test Coverage

Each skill test should cover at minimum:
- Full cast lifecycle (prepare → cast → hit)
- Resource consumption (mana/energy cost)
- Effect results (damage, healing, aura application)
- Cancel/interrupt behavior where applicable

## Timeline Verification

Every skill must have a `<skill>_timeline_test.go` that verifies the event timeline independently from `server/main.go`.

**Rules:**
1. **`server/main.go` is NOT the place for skill simulation logic** — it is infrastructure bootstrap only
2. Each skill package must include `<skill>_timeline_test.go` that:
   - Sets up its own `event.Bus`, `aura.Manager`, `timeline.TimelineRenderer`
   - Runs the simulation loop with its own `simMs` starting from 0
   - Asserts expected events (SpellCastStart, SpellHit, AuraTick, AuraExpired, etc.)
   - Asserts tick counts, damage values, and event ordering
3. Use `t.Log("\n" + output)` in a dedicated test to allow `go test -v` to print the full timeline

**CRITICAL: `renderer.SetTime()` must be called inside every simulation step.** If `SetTime()` is not called, all events record at 0ms — this is the #1 timeline bug. Every `AuraMgr.TickPeriodic` / `TickPeriodicArea` call MUST be preceded by `renderer.SetTime(time.Duration(simMs) * time.Millisecond)`. Do NOT extract tick helpers that skip `SetTime`.

**Timeline test structure:**
```go
func runXxxTimeline() string {
    // 1. Create caster, targets, bus, auraMgr, renderer
    // 2. Cast spell (via CastXxx or manually)
    // 3. Simulation loop:
    //    for simMs := int32(0); simMs < totalMs; simMs += stepMs {
    //        renderer.SetTime(time.Duration(simMs) * time.Millisecond)  // <-- MANDATORY
    //        auraMgr.TickPeriodic(...)
    //    }
    // 4. Return renderer.Render()
}
```

**Verification checklist before marking timeline complete:**
- [ ] Tick events show increasing time values (NOT all 0ms)
- [ ] Tick intervals match the spell's AuraPeriod
- [ ] Expiry/Spread events occur at correct times relative to ticks

## Skill Package Structure

```
skill-go/skills/
  <name>/
    <name>.go                # SpellInfo definition + CastXxx function
    <name>_test.go           # Skill-level unit tests
    <name>_timeline_test.go  # Timeline/event verification tests
```
