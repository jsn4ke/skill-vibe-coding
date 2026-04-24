# Skill Category System

Skill classification system used by `skill-design-decompose` for categorizing and mapping skills.

## Primary Categories

A skill can belong to multiple categories simultaneously (e.g., a spell that deals damage AND applies a debuff).

| Category | Description | TC Effect Examples | TC Aura Examples |
|----------|-------------|--------------------|------------------|
| 伤害 (Damage) | Directly or indirectly reduces health | `SPELL_EFFECT_SCHOOL_DAMAGE`, `SPELL_EFFECT_WEAPON_DAMAGE`, `SPELL_EFFECT_ENVIRONMENTAL_DAMAGE` | `SPELL_AURA_PERIODIC_DAMAGE` |
| 治疗 (Healing) | Restores health | `SPELL_EFFECT_HEAL`, `SPELL_EFFECT_HEAL_PCT`, `SPELL_EFFECT_HEAL_MAX_HEALTH` | `SPELL_AURA_PERIODIC_HEAL` |
| 控制 (Control) | Restricts target actions | `SPELL_EFFECT_APPLY_AURA` (with stun/root/fear/confuse aura) | `SPELL_AURA_MOD_STUN`, `SPELL_AURA_MOD_ROOT`, `SPELL_AURA_MOD_FEAR`, `SPELL_AURA_MOD_CONFUSE`, `SPELL_AURA_MOD_CHARM`, `SPELL_AURA_MOD_SILENCE`, `SPELL_AURA_MOD_PACIFY` |
| 增益 (Buff) | Strengthens self or allies | `SPELL_EFFECT_APPLY_AURA` (with stat/speed/power aura) | `SPELL_AURA_MOD_STAT`, `SPELL_AURA_MOD_INCREASE_SPEED`, `SPELL_AURA_MOD_ATTACK_POWER`, `SPELL_AURA_MOD_POWER_REGEN`, `SPELL_AURA_MOD_RESISTANCE` |
| 减益 (Debuff) | Weakens enemies | `SPELL_EFFECT_APPLY_AURA` (with reduction aura) | `SPELL_AURA_MOD_DECREASE_SPEED`, `SPELL_AURA_MOD_RESISTANCE` (negative), `SPELL_AURA_MOD_STAT` (negative) |
| 召唤 (Summon) | Creates units or objects | `SPELL_EFFECT_SUMMON`, `SPELL_EFFECT_SUMMON_PET`, `SPELL_EFFECT_SUMMON_OBJECT_WILD`, `SPELL_EFFECT_CREATE_ITEM` | `SPELL_AURA_MOUNTED` |
| 移动 (Movement) | Changes position | `SPELL_EFFECT_TELEPORT_UNITS`, `SPELL_EFFECT_CHARGE`, `SPELL_EFFECT_LEAP`, `SPELL_EFFECT_KNOCK_BACK`, `SPELL_EFFECT_JUMP`, `SPELL_EFFECT_PULL_TOWARDS` | `SPELL_AURA_MOD_INCREASE_SPEED`, `SPELL_AURA_ALLOW_FLIGHT` |
| 资源 (Resource) | Modifies resource values | `SPELL_EFFECT_ENERGIZE`, `SPELL_EFFECT_ENERGIZE_PCT`, `SPELL_EFFECT_POWER_DRAIN`, `SPELL_EFFECT_POWER_BURN` | `SPELL_AURA_MOD_POWER_REGEN`, `SPELL_AURA_PERIODIC_ENERGIZE` |
| 驱散 (Dispel) | Removes effects | `SPELL_EFFECT_DISPEL`, `SPELL_EFFECT_DISPEL_MECHANIC` | `SPELL_AURA_MECHANIC_IMMUNITY`, `SPELL_AURA_SCHOOL_IMMUNITY` |
| 触发 (Trigger) | Triggers other skills/effects | `SPELL_EFFECT_TRIGGER_SPELL`, `SPELL_EFFECT_TRIGGER_SPELL_WITH_VALUE`, `SPELL_EFFECT_DUMMY`, `SPELL_EFFECT_SCRIPT_EFFECT` | Proc auras via `SPELL_AURA_PROC_TRIGGER_SPELL`, `SPELL_AURA_PROC_TRIGGER_DAMAGE` |

## Cast Type Classification

| Cast Type | Description | TC Pattern |
|-----------|-------------|------------|
| 瞬发 (Instant) | No cast time, may trigger GCD | `CastTimeEntry = 0` |
| 施法 (Cast) | Has cast bar, interruptible | `CastTimeEntry > 0`, `SPELL_INTERRUPT_FLAG_INTERRUPT` |
| 引导 (Channeled) | Effect ticks during cast, interruptible | `AttributesEx & CHANNELLED_SPELL` |
| 蓄力 (Charged) | Release determines power | Custom / `SPELL_EFFECT_XXX` with charge mechanic |
| 被动 (Passive) | Always active, no cast action | `Attributes & PASSIVE_SPELL` |

## Target Type Classification

| Target Type | Description | TC Pattern |
|-------------|-------------|------------|
| 自身 (Self) | Affects caster only | `TARGET_UNIT_CASTER` |
| 单体 (Single) | One target unit | `TARGET_UNIT_TARGET_*` |
| 区域 (Area) | All units in radius around point | `TARGET_DEST_*` + radius |
| 锥形 (Cone) | Units in frontal cone | `TARGET_UNIT_CONE_*` |
| 链式 (Chain) | Bounces between targets | `TARGET_UNIT_TARGET_*` + ChainTargets |
| 轨迹 (Trajectory) | Projectile path | `TARGET_DEST_TRAJ` |
| 方向 (Direction) | Line or ray | `TARGET_UNIT_LINE_*` |

## Aura Duration Classification

| Duration Type | Description | TC Pattern |
|---------------|-------------|------------|
| 即时 (Instant) | No lasting effect | No aura applied |
| 定时 (Timed) | Lasts X seconds | `SpellDurationEntry` with specific values |
| 永久 (Permanent) | Until explicitly removed | `SpellDurationEntry = -1` |
| 跟随 (Follow) | While caster is alive/channeling | Linked to caster/channel state |

## Resource Type Reference

Common resource types a skill may consume or modify:

| Resource | TC Power Type | Typical Usage |
|----------|---------------|---------------|
| 法力 (Mana) | `POWER_MANA` (0) | Spells |
| 怒气 (Rage) | `POWER_RAGE` (1) | Warrior abilities |
| 能量 (Energy) | `POWER_ENERGY` (3) | Rogue abilities |
| 连击点 (Combo Points) | `POWER_COMBO_POINTS` (4) | Finisher scaling |
| 符文 (Runic Power) | `POWER_RUNIC_POWER` (6) | Death Knight |
| 聚气 (Focus) | `POWER_FOCUS` (2) | Hunter pet |
| 生命 (Health) | N/A (percent cost) | Warlock life tap |
| 自定义 | Custom enum | Game-specific resources |

## Mapping Priority Rule

When decomposing a skill:

1. **First**: Try to map every component to an existing WoW mechanism
2. **Second**: If WoW has a similar mechanism that needs adaptation, use it with documented modifications
3. **Last resort**: Create a custom mechanism, with clear explanation of why WoW's framework cannot cover it
4. **Never**: Mix custom mechanisms when a WoW equivalent exists, just because the custom version seems "simpler"

The goal is framework consistency — a unified mechanism vocabulary that the whole team understands.
