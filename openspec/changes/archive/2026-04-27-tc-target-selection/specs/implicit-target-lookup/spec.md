## ADDED Requirements

### Requirement: ImplicitTargetInfo StaticData lookup table

The `pkg/targeting` package SHALL define a `StaticData` struct with five orthogonal dimensions: `ObjectType`, `ReferenceType`, `SelectionCategory`, `CheckType`, `DirectionType`. Each `ImplicitTarget` value SHALL map to a `StaticData` entry via a fixed-size array `targetData[ImplicitTarget]`, providing O(1) lookup. New ImplicitTarget values SHALL be addable by adding a row to the array with zero code changes to the dispatch logic.

#### Scenario: Lookup TargetUnitCaster returns correct StaticData
- **WHEN** `targetData[TargetUnitCaster]` is accessed
- **THEN** it SHALL return `{ObjectType: ObjUnit, ReferenceType: RefCaster, SelectionCategory: SelectDefault, CheckType: CheckDefault, DirectionType: DirNone}`

#### Scenario: Lookup TargetUnitAreaEnemy returns correct StaticData
- **WHEN** `targetData[TargetUnitAreaEnemy]` is accessed
- **THEN** it SHALL return `{ObjectType: ObjUnit, ReferenceType: RefSrc, SelectionCategory: SelectArea, CheckType: CheckEnemy, DirectionType: DirNone}`

#### Scenario: Lookup TargetUnitConeEnemy returns correct StaticData
- **WHEN** `targetData[TargetUnitConeEnemy]` is accessed
- **THEN** it SHALL return `{ObjectType: ObjUnit, ReferenceType: RefCaster, SelectionCategory: SelectCone, CheckType: CheckEnemy, DirectionType: DirFront}`

#### Scenario: Lookup NYI target returns NYI category
- **WHEN** `targetData[TargetNone]` or any unimplemented target value is accessed
- **THEN** it SHALL return `{SelectionCategory: SelectNYI}`

### Requirement: ImplicitTarget enum uses TC-aligned numbering

The `ImplicitTarget` enum SHALL use TrinityCore's original numeric values (e.g., `TargetUnitCaster = 1`, `TargetUnitTargetEnemy = 6`, `TargetUnitSrcAreaEnemy = 15`). Unimplemented indices between defined values SHALL be left as gaps. The enum SHALL cover at minimum: all Group A core unit targets (1, 5, 6, 21, 25, 27, 92), Group C core area targets (15, 16, 20, 30, 31, 56), Group D core cone targets (24, 54, 59, 104), Group E line targets (133, 134, 135), Group F channel targets (76, 77), and Group H core destination targets (18, 28, 29, 47-50, 53, 62, 63, 87).

#### Scenario: TargetUnitCaster has TC value 1
- **WHEN** `TargetUnitCaster` constant is inspected
- **THEN** its value SHALL be 1

#### Scenario: TargetUnitTargetEnemy has TC value 6
- **WHEN** `TargetUnitTargetEnemy` constant is inspected
- **THEN** its value SHALL be 6

#### Scenario: TargetUnitSrcAreaEnemy has TC value 15
- **WHEN** `TargetUnitSrcAreaEnemy` constant is inspected
- **THEN** its value SHALL be 15

### Requirement: ImplicitTargetInfo accessor methods

The `ImplicitTargetInfo` type (wrapping an `ImplicitTarget` value) SHALL provide accessor methods `GetObjectType()`, `GetReferenceType()`, `GetSelectionCategory()`, `GetCheckType()`, `GetDirectionType()`, and `CalcDirectionAngle() float64` that return the corresponding StaticData fields. `CalcDirectionAngle()` SHALL convert DirectionType to radians (Front=0, Back=π, Right=-π/2, Left=π/2, etc.; Random returns a random angle in [0, 2π)).

#### Scenario: CalcDirectionAngle for DirFront
- **WHEN** `ImplicitTargetInfo(TargetUnitConeEnemy).CalcDirectionAngle()` is called
- **THEN** it SHALL return 0.0

#### Scenario: CalcDirectionAngle for DirBack
- **WHEN** a target with DirectionType=DirBack is queried
- **THEN** `CalcDirectionAngle()` SHALL return π

#### Scenario: IsArea returns true for Area and Cone categories
- **WHEN** `ImplicitTargetInfo(TargetUnitAreaEnemy).IsArea()` is called
- **THEN** it SHALL return true

#### Scenario: IsArea returns false for Default category
- **WHEN** `ImplicitTargetInfo(TargetUnitCaster).IsArea()` is called
- **THEN** it SHALL return false

### Requirement: SpellEffectInfo gains Radius and ChainTargets fields

`SpellEffectInfo` SHALL include a `Radius float64` field for target selection radius and a `ChainTargets int32` field for chain jump count. These fields replace the ad-hoc use of `MiscValue` for radius.

#### Scenario: Radius field is accessible on SpellEffectInfo
- **WHEN** a `SpellEffectInfo` is defined with `Radius: 10.0`
- **THEN** `effect.Radius` SHALL equal 10.0

#### Scenario: ChainTargets field is accessible on SpellEffectInfo
- **WHEN** a `SpellEffectInfo` is defined with `ChainTargets: 5`
- **THEN** `effect.ChainTargets` SHALL equal 5
