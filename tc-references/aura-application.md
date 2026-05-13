# AuraApplication — 光环应用层

> Source: TrinityCore | Generated: 2026-05-13 | Topic: aura application, aura apply, per-target aura state

## 1. Core Data Structures

### AuraApplication（per-target 应用实例）

每个 target 上独立存在的光环应用对象。同一个 Aura 可能在多个 target 上各有一个 AuraApplication。

| 字段 | 类型 | 用途 |
|------|------|------|
| `_target` | `Unit*` (const) | 此应用挂载的目标 |
| `_base` | `Aura*` (const) | 所属的 Aura 实例（caster 端拥有） |
| `_removeMode` | `AuraRemoveMode` (8bit) | 移除原因（Default/Cancel/Expire/Death/Dispel/Stack/Interrupt） |
| `_slot` | `uint16` | 客户端可见的光环槽位号 |
| `_flags` | `uint16` | 正/负/自身施放标记（AFLAG_POSITIVE/NEGATIVE/NOCASTER） |
| `_effectsToApply` | `uint32` | 此 target 应该应用哪些 effect（bitmask） |
| `_effectMask` | `uint32` | 当前已应用的 effect（bitmask） |
| `_needClientUpdate` | `bool` | 脏标记，通知客户端更新 |

### Aura（caster 端拥有）

| 字段 | 类型 | 用途 |
|------|------|------|
| `m_applications` | `unordered_map<Guid, AuraApplication*>` | **所有 target 的 AuraApplication 映射**，key 为 target GUID |
| `m_owner` | `WorldObject*` | 拥有此 Aura 的对象（Unit 或 DynamicObject） |
| `m_stackAmount` | `uint8` | 堆叠层数（属于 Aura 级别，不是 per-target） |
| `m_procCharges` | `uint8` | 触发充能数 |
| `m_duration` / `m_maxDuration` | `int32` | 持续时间 |
| `_effects` | `vector<AuraEffect*>` | 效果列表（AuraEffect 的 amount 等属性是 Aura 级别的） |

### Unit 的存储容器

| 容器 | 类型 | 用途 |
|------|------|------|
| `m_ownedAuras` | `multimap<uint32, Aura*>` | 自己施放的 Aura（自己是 caster/owner） |
| `m_appliedAuras` | `multimap<uint32, AuraApplication*>` | 作用于自己的光环应用（自己是 target） |
| `m_interruptableAuras` | `forward_list<AuraApplication*>` | 可被打断的光环应用子集 |
| `m_auraStateAuras` | `multimap<AuraStateType, AuraApplication*>` | 按 aura state 索引的应用 |

## 2. Flow Diagram

### 单体光环应用流程

```
Spell Hit Phase
     │
     ▼
Aura::Create(createInfo)
     │  创建 Aura 实例，挂到 owner 的 m_ownedAuras
     │
     ▼
Unit::_CreateAuraApplication(aura, effMask)      ← target 端执行
     │
     ├─ 1. new AuraApplication(target, caster, aura, effMask)
     │     ├─ 分配 _slot（可见槽位）
     │     └─ _InitFlags：判定正/负标记
     │
     ├─ 2. target->m_appliedAuras.insert(spellId, aurApp)
     │
     ├─ 3. aura->_ApplyForTarget(target, caster, aurApp)
     │     └─ aura->m_applications[targetGUID] = aurApp    ← 注册到 Aura 的 ApplicationMap
     │
     └─ 返回 aurApp
         │
         ▼
Unit::_ApplyAura(aurApp, effMask)
     │
     ├─ _RemoveNoStackAurasDueToAura()  ← 冲突光环移除
     ├─ HandleAuraSpecificMods()
     │
     └─ 遍历每个 effectIndex:
          aurApp->_HandleEffect(effIndex, apply=true)
            └─ AuraEffect::HandleEffect(aurApp, REAL, true)
                  └─ 实际修改属性/CC状态
```

### 区域光环更新流程（每 500ms）

```
Aura::UpdateOwner(diff)
     │
     ▼
Aura::UpdateTargetMap(caster, apply=true)
     │
     ├─ FillTargetMap() → 获取范围内所有 target + effMask
     │
     ├─ 比对 m_applications 现有 target：
     │   ├─ 不在范围内 → targetsToRemove → _UnapplyAura
     │   ├─ 免疫/不可应用 → targetsToRemove
     │   └─ effMask 变化 → UpdateApplyEffectMask（动态增减效果）
     │
     ├─ 新 target → target->_CreateAuraApplication(this, effMask)
     │
     └─ 对新 target → target->_ApplyAura(aurApp, effMask)
```

### 移除流程

```
Aura::_Remove(removeMode)
     │  标记 m_isRemoved = true
     │
     └─ 遍历 m_applications（所有 target）:
          target->_UnapplyAura(aurApp, removeMode)
            │
            ├─ aurApp->SetRemoveMode(removeMode)
            ├─ target->m_appliedAuras.erase(aurApp)
            ├─ 反向执行 effect（_HandleEffect false）→ 撤销属性/CC
            ├─ aura->_UnapplyForTarget(target, caster, aurApp)
            │     └─ aura->m_applications.erase(targetGUID)
            └─ delete aurApp（延迟删除）
```

## 3. Key Design Decisions

### Decision: Aura 与 AuraApplication 分离
**Reason:** 一个 Aura 可以作用于多个 target（区域光环、队伍增益）。Aura 由 caster 拥有（驱动 tick、管理 duration/stack），AuraApplication 由每个 target 独立拥有（管理该 target 上的正/负标记、effect 应用状态、移除原因）。两者是 1:N 关系。

### Decision: Aura.m_applications 是 Guid→AuraApplication 映射
**Reason:** Aura 需要按 target GUID 快速查找其 AuraApplication，用于 UpdateTargetMap 增量更新（检查 target 是否仍在范围内、effectMask 是否变化）以及 proc/trigger 时获取特定 target 的应用状态。

### Decision: Unit.m_appliedAuras 是 multimap<uint32 spellId, AuraApplication*>
**Reason:** 同一个 spellId 可能有多个 AuraApplication（不同 caster 施放的同名光环）。multimap 允许按 spellId 范围查找，支持驱散、查询、冲突检测。

### Decision: _effectsToApply vs _effectMask 分离
**Reason:** `_effectsToApply` 是"应该应用的效果"（由 target 免疫、effect 条件等决定），`_effectMask` 是"当前已应用的效果"。两者差异支持动态增减效果（UpdateApplyEffectMask），比如 target 在光环范围内获得了新的免疫，可以只移除部分 effect 而不移除整个光环。

### Decision: 正/负标记是 per-AuraApplication 而非 per-Aura
**Reason:** 同一个光环对不同 target 可能是正面的也可能是负面的（取决于 caster 与 target 的阵营关系）。_InitFlags 在创建 AuraApplication 时根据 caster→target 关系动态判定。

### Decision: 区域光环使用增量更新而非全量重建
**Reason:** UpdateTargetMap 通过比对现有 m_applications 与新 FillTargetMap 结果，只对差集操作（新增/移除/更新 effectMask），避免每帧对所有 target 执行 apply/remove 的开销。

### Decision: StackAmount 属于 Aura 级别而非 per-target
**Reason:** TC 的 stack 是 caster→spell 粒度的叠加，不是 per-target 独立叠加。不同 caster 施放的同名光环各自维护自己的 stack。

## 4. Reusable Patterns

### Pattern: Owner-Application 分离（1:N 实体关系）

**适用场景:** 一个逻辑实体需要同时在多个上下文中生效，每个上下文有独立状态。

```
Owner (单例，驱动生命周期)
  ├─ Application A (target A, per-target 状态)
  ├─ Application B (target B, per-target 状态)
  └─ Application C (target C, per-target 状态)
```

- Owner 负责：生命周期、tick 驱动、duration/stack 管理
- Application 负责：per-target 的 effect 应用/撤销、正负判定、移除原因
- 两者通过双向引用连接

### Pattern: EffectMask 差量更新

**适用场景:** 效果集合需要在运行时动态变化（免疫获取/失去、范围变化）。

```
effectsToApply: 应该生效的 effect bitmask（期望状态）
effectMask:     当前已生效的 effect bitmask（实际状态）
差集操作:       addMask = new & ~old, removeMask = old & ~new
```

### Pattern: 增量 Target Map 更新

**适用场景:** 区域效果每帧重新解析范围内的 target，但不能全量 apply/remove。

```
1. 计算新 target 集合
2. 比对现有 application map
3. 差集: 新增 → create+apply, 移除 → unapply, 变化 → updateMask
4. 只对差集操作
```

### Pattern: 正负判定延迟到 Application 创建时

**适用场景:** 同一逻辑实体的"好坏"取决于与上下文的关系（阵营、敌友）。

不在 Aura 级别判定正负，而是在创建 Application 时根据 caster→target 关系动态计算。
