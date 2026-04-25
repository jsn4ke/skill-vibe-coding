# Skill Test Rule

Every implemented skill must live in `skill-go/skills/<name>/` and have a corresponding `<skill>_engine_test.go` that passes.

## Trigger

This rule activates when implementing, designing, or building game skills.

## Behavior

1. **Skill location**: All new skills must be implemented in `skill-go/skills/<name>/`, not in `server/main.go` or other packages
2. **Check for `<skill>_engine_test.go`**: Every skill package must have an engine test file
3. **Run tests**: After any skill implementation, run:
   ```bash
   go test ./skills/...
   ```
4. **Do not mark skill implementation as complete** until its test file exists and all tests pass

## Required Test Coverage

Each skill engine test should cover at minimum:
- Full cast lifecycle via `eng.CastSpell()` + `eng.Simulate()`
- Resource consumption (mana/energy cost)
- Effect results (damage, healing, aura application)
- Cancel/interrupt behavior where applicable
- Spread/chain behavior for skills with AoE mechanics
- **Movement interrupt** — caster movement cancels PREPARING/CHANNELING spells (for spells with InterruptMovement)
- **Target out of range** — target moving out of range cancels cast (for spells with RangeMax)
- **Target removed/dead** — target disappearing or dying cancels channel spells

## Engine Test Pattern

All skill tests use the engine-driven architecture:

1. Create engine: `eng := engine.New()`
2. Add units: `eng.AddUnitWithID(id, entity, stats)`
3. Register hooks: `RegisterScripts(eng.Registry(), caster, eng, ...)` if the skill has script hooks
4. Cast spell: `eng.CastSpell(caster, &Info, engine.WithTarget(id), ...)`
5. Drive simulation: `eng.Simulate(totalMs, stepMs)`
6. Assert on `eng.Renderer().Render()` output

**Engine test structure:**
```go
func runXxxEngineTimeline() string {
    eng := engine.New()
    caster := eng.AddUnitWithID(1, entity.NewEntity(...), stat.NewStatSet())
    caster.Stats.SetBase(stat.SpellPower, 100)
    eng.AddUnitWithID(2, entity.NewEntity(...), stat.NewStatSet())

    RegisterScripts(eng.Registry(), caster, eng, ...)  // if needed

    eng.CastSpell(caster, &Info, engine.WithTarget(2))
    eng.Simulate(5000, 100)
    return eng.Renderer().Render()
}
```

**Verification checklist before marking test complete:**
- [ ] Timeline events show increasing time values (NOT all 0ms)
- [ ] Tick intervals match the spell's AuraPeriod
- [ ] Expiry/Spread events occur at correct times relative to ticks
- [ ] `t.Log("\n" + output)` in a dedicated test for verbose timeline output

## Skill Package Structure

```
skill-go/skills/
  <name>/
    <name>.go                # SpellInfo definition + RegisterScripts (if needed)
    <name>_engine_test.go    # Engine-driven tests (all test coverage in one file)
```

## Forbidden Patterns

- **No `CastXxx` functions** — All casting goes through `eng.CastSpell(caster, &Info, ...)`
- **No manual `renderer.SetTime()` loops** — `eng.Simulate()` handles time progression
- **No `AuraMgr.TickPeriodic()` calls** — Engine drives aura ticks through `Unit.Update()`
- **No `testUnit`/`testPos` mocks** — Use `eng.AddUnitWithID()` with real entities
- **No separate `<skill>_timeline_test.go`** — Timeline verification is part of the engine test file
