## 1. Create Fireball Skill Package

- [x] 1.1 Create `skill-go/skills/fireball/fireball.go` with SpellInfo definition and `CastFireball` function
- [x] 1.2 Create `skill-go/skills/fireball/fireball_test.go` with full skill lifecycle tests (cast, projectile, damage, aura, cancel)

## 2. Update Main Demo

- [x] 2.1 Update `server/main.go` to import and use `skills/fireball` package instead of inline SpellInfo

## 3. Add Project Rule

- [x] 3.1 Create `.claude/rules/skill-test.md` requiring every skill to have a passing `<skill>_test.go`
- [x] 3.2 Run `go test ./skills/...` to verify all skill tests pass
