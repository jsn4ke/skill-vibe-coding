## ADDED Requirements

### Requirement: ProcessAll implements four-phase effect dispatch

The `effect.ProcessAll` function SHALL process effects in four sequential phases: Launch, LaunchTarget, Hit, HitTarget. Each phase SHALL iterate over effects (and targets where applicable) and call `Process(ctx)` with the corresponding `EffectHandleMode`.

#### Scenario: Instant spell processes all four phases
- **WHEN** an instant spell with EffectSchoolDamage is cast
- **THEN** ProcessAll SHALL execute Launch (no target), LaunchTarget (per target, damage calculated), Hit (no target), HitTarget (per target) in order

#### Scenario: Area effect processes Launch and Hit without targets
- **WHEN** an area spell with no TargetInfos processes effects
- **THEN** Launch and Hit phases SHALL process with caster as target (for area aura creation)
- **AND** LaunchTarget and HitTarget phases SHALL process for each TargetInfo entry

### Requirement: Each effect handler enforces mode guard

Every effect handler function (e.g., handleSchoolDamage, handleApplyAura) SHALL check `ctx.Mode` at entry and return immediately if the mode does not match the handler's designated mode, aligned with TC's pattern.

Mode assignments per effect type:
- `EffectSchoolDamage` → LaunchTarget
- `EffectHeal` → LaunchTarget
- `EffectHealPct` → LaunchTarget
- `EffectApplyAura` → HitTarget
- `EffectEnergize` → HitTarget
- `EffectEnergizePct` → HitTarget
- `EffectTriggerSpell` → LaunchTarget
- `EffectWeaponDamage` → LaunchTarget
- `EffectSummon` → Launch
- `EffectDispel` → HitTarget
- `EffectDummy` → HitTarget
- `EffectTeleportUnits` → HitTarget
- `EffectCharge` → LaunchTarget + HitTarget (dual mode)
- `EffectKnockBack` → HitTarget
- `EffectLeap` → HitTarget

#### Scenario: SchoolDamage only executes in LaunchTarget mode
- **WHEN** handleSchoolDamage is called with ctx.Mode = HandleHitTarget
- **THEN** the handler SHALL return immediately without calculating damage

#### Scenario: ApplyAura only executes in HitTarget mode
- **WHEN** handleApplyAura is called with ctx.Mode = HandleLaunchTarget
- **THEN** the handler SHALL return immediately without creating an aura

#### Scenario: SchoolDamage executes in LaunchTarget mode
- **WHEN** handleSchoolDamage is called with ctx.Mode = HandleLaunchTarget
- **THEN** the handler SHALL calculate and set damage normally

#### Scenario: ApplyAura executes in HitTarget mode
- **WHEN** handleApplyAura is called with ctx.Mode = HandleHitTarget
- **THEN** the handler SHALL create the aura normally

#### Scenario: Charge executes in both LaunchTarget and HitTarget modes
- **WHEN** handleCharge is called with ctx.Mode = HandleLaunchTarget
- **THEN** the handler SHALL execute the launch-phase behavior
- **WHEN** handleCharge is called with ctx.Mode = HandleHitTarget
- **THEN** the handler SHALL execute the hit-phase behavior

### Requirement: Spell.HandleEffects dispatches by mode

`Spell.HandleEffects(mode)` SHALL iterate over all effects, create a Context for each, call the script hook corresponding to the mode, and call Process(ctx) unless PreventDefault is set.

#### Scenario: HandleEffects calls correct script hook per mode
- **WHEN** HandleEffects(HandleLaunch) is called
- **THEN** it SHALL call script hooks matching HookOnEffectLaunch
- **WHEN** HandleEffects(HandleLaunchTarget) is called
- **THEN** it SHALL call script hooks matching HookOnEffectLaunchTarget
- **WHEN** HandleEffects(HandleHit) is called
- **THEN** it SHALL call script hooks matching HookOnEffectHit
- **WHEN** HandleEffects(HandleHitTarget) is called
- **THEN** it SHALL call script hooks matching HookOnEffectHitTarget

### Requirement: Script hooks fire before default handler in each phase

In each phase of ProcessAll, the script hook SHALL be called before Process(ctx). If ctx.PreventDefault is set by the script, Process(ctx) SHALL be skipped.

#### Scenario: Script prevents default in HitTarget phase
- **WHEN** a script registered for HookOnEffectHitTarget sets PreventDefault = true
- **THEN** the default effect handler SHALL NOT be called for that effect

### Requirement: EffectSchoolDamage accumulates damage in LaunchTarget phase

`handleSchoolDamage` SHALL calculate damage (basePoints + variance + bonusCoeff × spellPower) during the LaunchTarget phase and set ctx.FinalDamage. The damage SHALL be accumulated into TargetInfo.Damage.

#### Scenario: Damage accumulated in LaunchTarget phase
- **WHEN** ProcessAll runs for a spell with EffectSchoolDamage
- **THEN** ctx.FinalDamage SHALL be set during the LaunchTarget phase
- **AND** TargetInfo.Damage SHALL reflect the accumulated value
