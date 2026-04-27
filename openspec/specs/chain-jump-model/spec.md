## ADDED Requirements

### Requirement: Chain jump model with jumpRadius

When `SpellEffectInfo.ChainTargets > 1`, the chain selection SHALL use a jump model: starting from the initial target, each subsequent target SHALL be found within `jumpRadius` of the previous target. The jump radius SHALL be determined by spell damage class: Melee=5.0, Ranged=7.5, Magic=10.0 (ChainHeal=12.5). The chain SHALL stop when `ChainTargets` jumps are exhausted or no valid target is found within jumpRadius.

#### Scenario: Chain Lightning hits 3 targets
- **GIVEN** a spell with ChainTargets=3, jumpRadius=10.0, CheckType=CheckEnemy
- **AND** initial target A is at (0, 0, 0)
- **AND** enemy B is at (8, 0, 0) (within 10.0 of A)
- **AND** enemy C is at (16, 0, 0) (within 10.0 of B, but >10.0 from A)
- **WHEN** chain selection executes
- **THEN** targets SHALL be selected in order: A → B → C

#### Scenario: Chain stops when no target within jumpRadius
- **GIVEN** a spell with ChainTargets=5, jumpRadius=10.0
- **AND** initial target A is at (0, 0, 0)
- **AND** only enemy B is within 10.0 of A
- **AND** no enemy is within 10.0 of B
- **WHEN** chain selection executes
- **THEN** only A and B SHALL be selected (chain stops after 1 jump)

#### Scenario: Chain excludes already-selected targets
- **GIVEN** a spell with ChainTargets=3
- **AND** initial target A and enemy B are within jumpRadius of each other
- **WHEN** chain selection searches for the next target from B
- **THEN** target A SHALL NOT be selected again

### Requirement: ChainHeal selects by HP deficit

When the chain target type is `TargetUnitTargetChainHealAlly` (TC value 45), the chain SHALL select the ally with the largest HP deficit (maxHealth - currentHealth) within jumpRadius, rather than the nearest ally.

#### Scenario: ChainHeal picks most damaged ally
- **GIVEN** a ChainHeal spell with ChainTargets=2
- **AND** ally B (80% HP) is at distance 5 from initial target
- **AND** ally C (30% HP) is at distance 7 from initial target
- **WHEN** chain selection executes
- **THEN** ally C SHALL be selected first (larger HP deficit)

### Requirement: ChainTargets field on SpellEffectInfo

`SpellEffectInfo.ChainTargets` SHALL specify the maximum number of chain jumps (not including the initial target). A value of 0 or 1 means no chain behavior (single target only). The initial target is added by the NEARBY/DEFAULT selection; chain targets are added by the chain jump algorithm.

#### Scenario: ChainTargets=0 means no chain
- **GIVEN** a spell effect with ChainTargets=0
- **WHEN** target selection runs
- **THEN** only the initial target SHALL be selected (no chain jumps)

#### Scenario: ChainTargets=5 means up to 5 jumps
- **GIVEN** a spell effect with ChainTargets=5
- **WHEN** chain selection executes
- **THEN** up to 5 additional targets SHALL be added via chain jumps
