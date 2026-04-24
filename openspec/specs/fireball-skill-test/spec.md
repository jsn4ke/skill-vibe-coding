## ADDED Requirements

### Requirement: Every implemented skill must have a passing skill test
Each skill implemented in the `skills/` directory SHALL have a corresponding `<skill>_test.go` file that tests the complete skill lifecycle. The test file MUST pass via `go test ./skills/...`.

#### Scenario: Fireball has a skill test
- **WHEN** `go test ./skills/fireball/` is run
- **THEN** all tests SHALL pass, covering: cast lifecycle, projectile delay, damage calculation, DoT aura, cancel behavior

#### Scenario: New skill without test fails the rule
- **WHEN** a skill is added to `skills/<name>/` without a `<name>_test.go`
- **THEN** this SHALL be flagged as a missing test during implementation

### Requirement: Fireball skill package with definition and cast function
The `skills/fireball/` package SHALL contain the Fireball SpellInfo definition and a `CastFireball` function that encapsulates the full cast flow including projectile delay and aura creation.

#### Scenario: CastFireball executes full lifecycle
- **WHEN** CastFireball is called with a valid caster and target
- **THEN** it SHALL return a Spell in StateFinished with damage on TargetInfo and aura added to AuraManager

#### Scenario: CastFireball returns failure for invalid cast
- **WHEN** CastFireball is called with insufficient mana
- **THEN** it SHALL return the appropriate SpellCastResult error
