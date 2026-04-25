## MODIFIED Requirements

### Requirement: aura.Manager calls script hooks on lifecycle events

The `aura.Manager` SHALL accept a `*script.Registry` and call registered hooks during aura lifecycle events: AfterRemove (in RemoveAura and TickPeriodic expiry, including RemoveByInterrupt), and AfterApply (in AddAura).

#### Scenario: AfterRemove hook called on aura interrupt removal
- **WHEN** an aura is removed by interrupt (RemoveByInterrupt)
- **THEN** the Manager SHALL call `registry.CallAuraHook(spellID, AuraHookAfterRemove, ctx)` with RemoveMode=RemoveByInterrupt

#### Scenario: AfterRemove hook called on aura expiry
- **WHEN** an aura expires in TickPeriodic
- **THEN** the Manager SHALL call `registry.CallAuraHook(spellID, AuraHookAfterRemove, ctx)` with RemoveMode=RemoveByExpire BEFORE calling RemoveAura

#### Scenario: AfterRemove hook called on aura dispel
- **WHEN** an aura is dispelled
- **THEN** the Manager SHALL call `registry.CallAuraHook(spellID, AuraHookAfterRemove, ctx)` with RemoveMode=RemoveByDispel

#### Scenario: AfterApply hook called on aura application
- **WHEN** a new aura is added via AddAura (not a refresh)
- **THEN** the Manager SHALL call `registry.CallAuraHook(spellID, AuraHookAfterApply, ctx)`
