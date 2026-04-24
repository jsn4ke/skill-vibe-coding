## Why

Fireball spell logic is currently scattered across `server/main.go` (spell definition, aura creation, DoT tick demo) and `pkg/spell/spell_test.go` (unit tests). There is no dedicated skill-level test file that validates the complete Fireball behavior end-to-end. Every implemented skill should have a `<skill>_test.go` that exercises the full lifecycle and can be used as a contract test.

## What Changes

- Create `skill-go/skills/fireball/fireball.go` containing the Fireball SpellInfo definition and a `CastFireball` function that encapsulates the full cast + projectile + aura flow
- Create `skill-go/skills/fireball/fireball_test.go` with comprehensive skill-level tests
- Add a project rule requiring every implemented skill to have a corresponding `<skill>_test.go` that passes
- Update `server/main.go` to use the extracted Fireball skill package instead of inline definitions

## Capabilities

### New Capabilities

### Modified Capabilities

## Impact

- New directory: `skill-go/skills/fireball/` — skill package with definition + test
- `server/main.go` — import and use fireball package instead of inline SpellInfo
- `.claude/rules/` — new rule for mandatory skill tests
