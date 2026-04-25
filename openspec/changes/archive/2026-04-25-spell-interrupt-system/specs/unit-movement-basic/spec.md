## ADDED Requirements

### Requirement: Unit tracks position changes via SetPosition

The `Unit` struct SHALL provide a `SetPosition(pos entity.Position)` method that updates the unit's entity position. The Unit SHALL track whether its position changed during the current update frame.

#### Scenario: SetPosition updates entity position
- **WHEN** `unit.SetPosition(entity.Position{X: 5, Y: 0, Z: 0})` is called
- **THEN** `unit.Entity.Pos.X` SHALL be 5

#### Scenario: Position change detected within same frame
- **WHEN** unit position was at X=0 and `SetPosition({X: 5, Y: 0, Z: 0})` is called
- **THEN** the unit SHALL mark its position as changed for the current frame

### Requirement: Unit.IsMoving reflects position change from previous frame

The `Unit.IsMoving()` method SHALL return true when the unit's position has changed since the previous `Advance()` call. The movement state is evaluated once per frame at the end of `Unit.Update()`.

#### Scenario: IsMoving returns true after position change
- **WHEN** unit's position at frame N was X=0
- **AND** `SetPosition({X: 5})` is called before frame N+1's Update
- **AND** Unit.Update() runs for frame N+1
- **THEN** `IsMoving()` SHALL return true during and after that Update

#### Scenario: IsMoving returns false when position unchanged
- **WHEN** unit's position does not change between two consecutive frames
- **AND** Unit.Update() runs
- **THEN** `IsMoving()` SHALL return false

### Requirement: Engine.Advance triggers movement-based interrupt checks

After all Unit.Update() calls in Engine.Advance(), the engine SHALL NOT need additional logic. Movement detection happens inside Unit.Update() which compares current position to previous frame position.

#### Scenario: Movement detected automatically during simulation
- **WHEN** a unit's position changes via SetPosition between frames
- **THEN** the next Unit.Update() SHALL detect the movement and set isMoving=true
