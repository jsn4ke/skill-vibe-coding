## Context

Skill implementations currently live in `server/main.go` as inline demo code. Tests live in `pkg/spell/spell_test.go` which is a package-level test, not a skill-level test. As more skills get implemented (Frostbolt, Pyroblast, etc.), this structure won't scale.

## Goals / Non-Goals

**Goals:**
- Extract Fireball into a standalone `skills/fireball/` package with its own test
- The test file serves as both validation and living documentation for the skill
- Add a project rule enforcing `<skill>_test.go` for every implemented skill

**Non-Goals:**
- No generic skill framework or abstraction — each skill is a simple package
- No test infrastructure beyond standard `go test`
- Not extracting other spells yet (Frostbolt stays in main.go for now)

## Decisions

### 1. `skills/<name>/` Package Structure

**Decision**: Each skill gets its own package under `skills/<name>/` with `<name>.go` (definition + cast function) and `<name>_test.go`.

**Rationale**: Simple, flat, discoverable. No registry or plugin system needed. `go test ./skills/...` runs all skill tests.

### 2. CastFireball Function Signature

**Decision**: `CastFireball(caster Caster, targetID uint64, auraMgr *aura.Manager) (*spell.Spell, SpellCastResult)` — takes interfaces, returns the spell for inspection.

**Rationale**: The caller provides the dependencies (caster, aura manager). Returns the spell so tests can inspect damage, state, etc. The function handles Prepare → Update(castTime) → Update(hitDelay) → aura creation internally.

### 3. Rule Location

**Decision**: Add to `.claude/rules/skill-test.md`.

**Rationale**: Follows existing rule pattern (like `go-race-test.md`). Triggered by skill implementation keywords.

## Risks / Trade-offs

- **[Risk] Caster interface in spell package vs skills package** → Mitigation: skills package imports spell package, uses spell.Caster interface directly. No circular dependency.
- **[Trade-off] Each skill is a separate package** → More directories but better isolation. Acceptable for a small number of skills.
