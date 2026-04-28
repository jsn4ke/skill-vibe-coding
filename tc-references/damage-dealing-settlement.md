# Damage Dealing & Settlement（伤害结算机制）

> Source: TrinityCore | Generated: 2026-04-28 | Topic: damage settlement, DoDamageAndTriggers, DealDamage, damage pipeline, proc trigger

## 1. 核心数据结构

### SpellNonMeleeDamage — 法术伤害信息包

| 字段 | 用途 |
|------|------|
| `attacker` | 攻击者 Unit |
| `target` | 受击者 Unit |
| `damage` | 最终伤害值（经护甲/吸收/抗性修正后） |
| `originalDamage` | 修正前的原始伤害 |
| `absorb` | 被吸收的伤害量 |
| `resist` | 被抗性减免的伤害量 |
| `blocked` | 被格挡的伤害量 |
| `schoolMask` | 法术学校掩码 |
| `Spell` | 关联的 SpellInfo |
| `HitInfo` | 命中类型标记（暴击、全吸收、部分吸收等） |

### TargetInfo — 每目标结算容器

| 字段 | 用途 |
|------|------|
| `Damage` | LaunchTarget 阶段累加的伤害值 |
| `Healing` | LaunchTarget 阶段累加的治疗值 |
| `MissCondition` | 命中判定结果（NONE/MISS/RESIST/IMMUNE等） |
| `IsCrit` | 是否暴击 |
| `ProcHitMask` | Proc 命中掩码（NORMAL/CRIT/IMMUNE/ABSORB等） |
| `EffectMask` | 该目标受哪些 effect 影响 |
| `HitAura` | 命中阶段创建的 Aura 指针 |

### DamageInfo — Proc 用的伤害信息

| 字段 | 用途 |
|------|------|
| 继承 SpellNonMeleDamage | 包含攻击者/目标/伤害值 |
| `DamageEffectType` | SPELL_DIRECT_DAMAGE / DOT / NODAMAGE |
| `AttackType` | BASE_ATTACK / OFF_ATTACK / RANGED_ATTACK |
| `ProcHitMask` | Proc 命中类型 |

### HealInfo — 治疗信息包

| 字段 | 用途 |
|------|------|
| `heal` | 原始治疗量 |
| `effectiveHeal` | 实际生效的治疗量（扣除过量治疗） |
| `absorb` | 被吸收的治疗量 |

## 2. 流程图

### 完整伤害结算流程

```
Spell::cast()
│
├── HandleLaunchPhase()
│   ├─ LAUNCH (无目标)
│   └─ LAUNCH_TARGET (每目标)
│      │  EffectSchoolDMG → 计算 base+variance+scaling
│      │  → 存入 spell.m_damage
│      │  → 累加到 targetInfo.Damage
│      │
│      └─ PreprocessSpellLaunch()
│         → 对每个目标计算暴击/免疫判定
│         → 设置 targetInfo.IsCrit, MissCondition
│
├── handle_immediate() / handle_delayed()
│   │
│   ├─ _handle_immediate_phase()
│   │  └─ HIT (无目标)
│   │
│   └─ DoProcessTargetContainer(m_UniqueTargetInfo)
│      │
│      ├─ Step 1: PreprocessTarget (每目标)
│      │  → 设置 DRGroup, AuraDuration, AuraBasePoints
│      │
│      ├─ Step 2: DoTargetSpellHit (每目标 × 每 effect)
│      │  └─ HIT_TARGET 处理
│      │     EffectApplyAura → 创建 Aura
│      │     EffectDummy → 脚本钩子
│      │     EffectEnergize → 恢复能量
│      │     ...
│      │
│      └─ Step 3: DoDamageAndTriggers (每目标) ← 伤害结算核心
│         │
│         ├─ 恢复 m_damage/m_healing 从 targetInfo
│         │  spell.m_damage = targetInfo.Damage
│         │  spell.m_healing = targetInfo.Healing
│         │
│         ├─ 确定 ProcFlags
│         │  → procAttacker / procVictim / procSpellType
│         │
│         ├─ 治疗结算（m_healing > 0）
│         │  ├─ 暴击判定 → SpellCriticalHealingBonus
│         │  ├─ HealBySpell() → target.ModifyHealth(+heal)
│         │  ├─ 威胁值转移
│         │  └─ procSpellType |= PROC_SPELL_TYPE_HEAL
│         │
│         ├─ 伤害结算（m_damage > 0）
│         │  ├─ 免疫检查 → IsImmunedToDamage
│         │  │  └─ 免疫: ProcHitMask = IMMUNE, m_damage = 0
│         │  ├─ CalculateSpellDamageTaken() ← 伤害修正管线
│         │  │  ├─ 护甲减伤 → CalcArmorReducedDamage
│         │  │  ├─ 暴击加成 → SpellCriticalDamageBonus
│         │  │  ├─ 格挡计算 → blocked amount
│         │  │  ├─ 韧性减伤 → ApplyResilience
│         │  │  ├─ 脚本修正 → ModifySpellDamageTaken hook
│         │  │  └─ 吸收/抗性 → CalcAbsorbResist
│         │  ├─ DealDamageMods() → 伤害乘数修正
│         │  ├─ DealSpellDamage() → 实际扣血
│         │  │  └─ DealDamage() → victim.ModifyHealth(-damage)
│         │  │     ├─ AI DamageTaken/DamageDealt 回调
│         │  │     ├─ 打断受伤中断光环
│         │  │     ├─ 分摊伤害（SHARE_DAMAGE_PCT 光环）
│         │  │     ├─ 决斗限制（血量不低于1）
│         │  │     ├─ 不杀标记（CANNOT_KILL_TARGET）
│         │  │     └─ 击杀处理 → Kill() / OnHealthDepleted
│         │  └─ SendSpellNonMeleeDamageLog → 发送日志包
│         │
│         ├─ 无伤害无治疗（miss/immune）
│         │  └─ 仍触发 Proc（PROC_SPELL_TYPE_NO_DMG_HEAL）
│         │
│         └─ Proc 触发
│            ProcSkillsAndAuras(caster, target, procAttacker, procVictim, ...)
│            ├─ actor.ProcSkillsAndReactives → 技能触发
│            ├─ actionTarget.ProcSkillsAndReactives → 受击触发
│            └─ TriggerAurasProcOnEvent → 光环 Proc 触发
│
└── finish(SPELL_CAST_OK)
```

### CalculateSpellDamageTaken 修正管线（法术伤害）

```
原始 damage (来自 LaunchTarget 计算)
│
├─ 护甲减伤 (仅 MELEE/RANGED 类法术)
│  CalcArmorReducedDamage()
│
├─ 暴击加成 (如果 IsCrit)
│  ├─ MELEE/RANGED: damage += crit_bonus, ApplyCritDamageMod
│  └─ MAGIC: SpellCriticalDamageBonus()
│
├─ 格挡 (仅 MELEE/RANGED + blocked)
│  CalculatePct(damage, blockPercent) → damage -= blocked
│
├─ 韧性减伤 (仅 PvP)
│  ApplyResilience()
│
├─ 脚本修正
│  ModifySpellDamageTaken hook
│
├─ 吸收 + 抗性计算
│  CalcAbsorbResist()
│  ├─ damageInfo.absorb = 被吸收量
│  ├─ damageInfo.resist = 被抗性减免量
│  └─ damageInfo.damage = damage - absorb - resist
│
└─ 最终 damageInfo.damage → 传入 DealDamage
```

### DealDamage 核心扣血流程

```
DealDamage(attacker, victim, damage, ...)
│
├─ AI 回调
│  victim.AI->DamageTaken(attacker, damage)
│  attacker.AI->DamageDealt(victim, damage)
│
├─ 脚本回调
│  OnDamage hook
│
├─ 受伤中断光环
│  victim.RemoveAurasWithInterruptFlags(Damage)
│
├─ 分摊伤害光环
│  SHARE_DAMAGE_PCT → 将部分伤害分摊给光环施法者
│
├─ 决斗限制
│  PvP 决斗中血量不低于1
│
├─ 不杀标记
│  CANNOT_KILL_TARGET → damageTaken = health - 1
│
├─ 实际扣血
│  victim.ModifyHealth(-damageTaken)
│
├─ 击杀处理
│  if (damageTaken >= health)
│  ├─ Unit::Kill() → 死亡流程
│  └─ OnHealthDepleted() → 未死但血量耗尽回调
│
└─ 统计更新
   DamageDealt/DamageTaken criteria
```

## 3. 关键设计决策

### 决策: LaunchTarget 计算 + HitTarget 后 DoDamageAndTriggers 施加

**原因**: 弹道法术需要在发射时确定伤害值（服务端计算），但等到命中时才实际施加效果。两阶段分离支持弹道延迟。`DoProcessTargetContainer` 的三步结构（Preprocess → DoTargetSpellHit → DoDamageAndTriggers）确保所有 HIT_TARGET effect 先完成，再统一结算伤害。

**含义**: `EffectSchoolDMG` 在 `LAUNCH_TARGET` 只计算数值并存入 `targetInfo.Damage`，不扣血。扣血发生在 `DoDamageAndTriggers` 中。

### 决策: DoDamageAndTriggers 是每个目标独立执行

**原因**: 每个目标有自己的命中判定（MissCondition）、暴击状态（IsCrit）、免疫状态。伤害结算必须逐目标执行，不能批量处理。这也允许每个目标独立触发 Proc。

### 决策: 免疫检查在 DoDamageAndTriggers 而非 LaunchTarget

**原因**: 目标可能在弹道飞行期间获得免疫（如开启冰霜免疫光环）。LaunchTarget 时目标没有免疫，但 HitTarget 时有了。免疫检查必须在结算时做，而非计算时。

### 决策: 伤害修正管线（CalculateSpellDamageTaken）在结算时执行

**原因**: 护甲减伤、吸收、抗性都依赖目标的实时状态。这些在弹道飞行期间可能变化。修正必须在命中时刻基于目标当前状态计算。

### 决策: Proc 触发在伤害结算之后

**原因**: Proc 需要知道伤害结果（暴击/普通/吸收/免疫）才能决定是否触发和触发什么。`ProcHitMask` 在伤害结算过程中逐步构建，最终传给 `ProcSkillsAndAuras`。

### 决策: 无伤害无治疗仍触发 Proc

**原因**: 即使法术 miss 或被免疫，攻击者和受击者仍可能需要触发 proc（如"被攻击时触发"光环不关心是否命中）。`PROC_SPELL_TYPE_NO_DMG_HEAL` 覆盖这种情况。

### 决策: DealDamage 是 static 方法

**原因**: 伤害扣减是全局行为，不限于法术。近战攻击、光环周期伤害、环境伤害都走同一个 `DealDamage`。统一入口确保所有扣血路径都经过相同的 AI 回调、光环中断、分摊、决斗限制等处理。

## 4. 可借鉴的通用模式

### 模式: 计算-修正-施加三阶段分离

```
Phase 1: 计算 (LaunchTarget) → 基础数值存入中间容器
Phase 2: 修正 (DoDamageAndTriggers) → 基于实时状态修正数值
Phase 3: 施加 (DealDamage) → 实际扣血 + 后处理
```

**适用场景**: 有延迟的投射物系统、目标状态可能在计算和施加之间变化、需要全局修正（如 AoE 伤害上限）。

**不适用**: 纯即时无延迟系统可以合并计算和施加。

### 模式: 统一扣血入口

所有伤害来源（法术、近战、光环 tick、环境）走同一个 `DealDamage` 函数。统一处理：
- AI 回调
- 光环中断（受伤打断隐身等）
- 分摊伤害
- PvP 限制
- 击杀处理

**适用场景**: 多种伤害来源的游戏系统，需要一致的扣血后处理。

### 模式: DamageInfo 信息包传递

伤害不是简单的数字，而是包含攻击者、目标、学校、命中类型、吸收、抗性等完整信息的信息包。下游系统（Proc、日志、AI）需要这些上下文。

**适用场景**: Proc 系统需要命中类型信息；日志系统需要完整伤害分解；AI 需要知道攻击者。

### 模式: Proc 延迟到结算后触发

Proc 不在 effect handler 中触发，而是在 `DoDamageAndTriggers` 结束后统一触发。这确保 Proc 能拿到完整的命中结果。

**适用场景**: Proc 系统需要依赖伤害结算结果（暴击、吸收、免疫）来决定触发行为。

### 模式: 免疫检查在施加时而非计算时

免疫检查在 `DoDamageAndTriggers` 中做，不在 `EffectSchoolDMG` 中做。这处理了弹道飞行期间目标获得免疫的情况。

**适用场景**: 有延迟的投射物系统，目标状态可能在飞行期间变化。