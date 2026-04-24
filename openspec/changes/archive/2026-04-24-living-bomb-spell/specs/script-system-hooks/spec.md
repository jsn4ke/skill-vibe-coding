## ADDED Requirements

### Requirement: AuraContext carries RemoveMode and Aura reference

The `script.AuraContext` struct SHALL include a `RemoveMode aura.RemoveMode` field and an `Aura *aura.Aura` field, enabling script hooks to filter by removal reason and access aura data (including SpellValues).

#### Scenario: AuraContext provides RemoveMode on aura expiry
- **WHEN** an aura expires and the AfterRemove hook is called
- **THEN** AuraContext.RemoveMode SHALL be `aura.RemoveByExpire`

#### Scenario: AuraContext provides RemoveMode on aura dispel
- **WHEN** an aura is dispelled and the AfterRemove hook is called
- **THEN** AuraContext.RemoveMode SHALL be `aura.RemoveByDispel`

#### Scenario: Script can access Aura SpellValues
- **WHEN** a script hook receives an AuraContext with an Aura that has SpellValues
- **THEN** the script SHALL be able to read `ctx.Aura.SpellValues[key]`

### Requirement: aura.Manager calls script hooks on lifecycle events

The `aura.Manager` SHALL accept a `*script.Registry` and call registered hooks during aura lifecycle events: AfterRemove (in RemoveAura and TickPeriodic expiry), and AfterApply (in AddAura).

#### Scenario: AfterRemove hook called on aura expiry
- **WHEN** an aura expires in TickPeriodic
- **THEN** the Manager SHALL call `registry.CallAuraHook(spellID, AuraHookAfterRemove, ctx)` with RemoveMode=RemoveByExpire BEFORE calling RemoveAura

#### Scenario: AfterRemove hook NOT called on non-expiry removal in TickPeriodic
- **WHEN** an aura is removed by death or dispel (outside TickPeriodic expiry path)
- **THEN** RemoveAura SHALL call AfterRemove hook with the appropriate RemoveMode

#### Scenario: AfterApply hook called on aura application
- **WHEN** a new aura is added via AddAura (not a refresh)
- **THEN** the Manager SHALL call `registry.CallAuraHook(spellID, AuraHookAfterApply, ctx)`

### Requirement: effect.ProcessAll calls OnEffectHit script hook

The `effect.ProcessAll` function SHALL call `registry.CallSpellHook` with `HookOnEffectHit` for each effect before executing the default handler. If `ctx.PreventDefault` is set, the default handler SHALL be skipped.

#### Scenario: Script intercepts Dummy effect
- **WHEN** a spell with EFFECT_0 = EffectDummy has a registered OnEffectHit hook for EFFECT_0
- **AND** the hook sets PreventDefault = true
- **THEN** the default handleDummy handler SHALL NOT be called

#### Scenario: Script runs custom logic on effect hit
- **WHEN** a script hook is registered for a spell's OnEffectHit
- **THEN** the hook SHALL execute with SpellContext containing the Spell, and the script can trigger new spells via the context

### Requirement: Spell carries SpellValues

The `spell.Spell` struct SHALL include a `SpellValues map[uint8]float64` field. This field is populated by scripts when triggering new spells, and propagated to created auras.

#### Scenario: SpellValues passed between spells
- **WHEN** spell A's script creates spell B with SpellValues[2] = 1
- **THEN** spell B's SpellValues[2] SHALL be 1

#### Scenario: SpellValues propagated to Aura
- **WHEN** an aura is created from a spell that has SpellValues
- **THEN** the aura's SpellValues SHALL reflect the spell's SpellValues

### Requirement: Aura carries SpellValues

The `aura.Aura` struct SHALL include a `SpellValues map[uint8]float64` field. Scripts can read SpellValues from the Aura during lifecycle hooks.

#### Scenario: SpellValues stored in Aura
- **WHEN** a script creates an aura with SpellValues[2] = 1
- **THEN** when the aura's AfterRemove hook fires, `ctx.Aura.SpellValues[2]` SHALL be 1

### Requirement: Spell.SelectTargets supports AoE target selection

The `Spell.SelectTargets()` method SHALL support `TargetUnitAreaEnemy` and similar area-based implicit targets. When an area target type is detected, it SHALL use `targeting.TargetSelector` to select multiple targets around a center point.

#### Scenario: AoE target selection finds enemies in radius
- **WHEN** a spell has an effect with TargetA = TargetUnitAreaEnemy and Radius = 10
- **AND** there are 3 enemies within 10 yards of the center point
- **THEN** SelectTargets SHALL populate TargetInfos with 3 entries

#### Scenario: AoE target selection excludes specified ID
- **WHEN** SelectTargets is called with an exclude target
- **THEN** the excluded target SHALL NOT appear in TargetInfos
