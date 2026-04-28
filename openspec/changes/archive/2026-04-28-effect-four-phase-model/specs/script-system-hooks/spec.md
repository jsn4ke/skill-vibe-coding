## ADDED Requirements

### Requirement: Registry supports HookOnEffectLaunch

The `script.Hook` enum SHALL include `HookOnEffectLaunch`. This hook fires during the Launch phase (no target) of effect processing.

#### Scenario: HookOnEffectLaunch fires during Launch phase
- **WHEN** a spell with a registered HookOnEffectLaunch handler processes the Launch phase
- **THEN** the handler SHALL be called with a SpellContext containing the Spell and EffectIndex

### Requirement: Registry supports HookOnEffectLaunchTarget

The `script.Hook` enum SHALL include `HookOnEffectLaunchTarget`. This hook fires during the LaunchTarget phase (per target) of effect processing.

#### Scenario: HookOnEffectLaunchTarget fires for each target
- **WHEN** a spell with a registered HookOnEffectLaunchTarget handler processes the LaunchTarget phase
- **THEN** the handler SHALL be called once per target with a SpellContext containing the Spell and EffectIndex

### Requirement: Registry supports HookOnEffectHitTarget

The `script.Hook` enum SHALL include `HookOnEffectHitTarget`. This hook fires during the HitTarget phase (per target) of effect processing.

#### Scenario: HookOnEffectHitTarget fires for each target
- **WHEN** a spell with a registered HookOnEffectHitTarget handler processes the HitTarget phase
- **THEN** the handler SHALL be called once per target with a SpellContext containing the Spell and EffectIndex

## MODIFIED Requirements

### Requirement: effect.ProcessAll calls OnEffectHit script hook

The `effect.ProcessAll` function SHALL call script hooks for each effect in each phase: HookOnEffectLaunch (Launch), HookOnEffectLaunchTarget (LaunchTarget), HookOnEffectHit (Hit), HookOnEffectHitTarget (HitTarget). The hook is called before the default handler. If `ctx.PreventDefault` is set, the default handler SHALL be skipped for that effect in that phase.

#### Scenario: Script intercepts Dummy effect in HitTarget phase
- **WHEN** a spell with EFFECT_0 = EffectDummy has a registered HookOnEffectHitTarget handler for EFFECT_0
- **AND** the handler sets PreventDefault = true
- **THEN** the default handleDummy handler SHALL NOT be called

#### Scenario: Script runs custom logic on effect hit target
- **WHEN** a script hook is registered for a spell's HookOnEffectHitTarget
- **THEN** the hook SHALL execute with SpellContext containing the Spell, and the script can trigger new spells via the context

#### Scenario: Script runs in LaunchTarget phase
- **WHEN** a script hook is registered for a spell's HookOnEffectLaunchTarget
- **THEN** the hook SHALL execute with SpellContext before the default LaunchTarget handler
