## Context

将 skill-go 从 flat global Manager + 测试循环各自驱动 改为 TC 对齐的 Unit 中心架构。核心参考：`tc-references/unit-update-architecture.md`。

## Goals / Non-Goals

**Goals:**
- Unit 持有 activeSpells、ownedAuras、appliedAuras — 与 TC 的 `m_currentSpells`、`m_ownedAuras`、`m_appliedAuras` 对齐
- Engine.Tick(diff) 是唯一的时间驱动入口 — 与 TC 的 Map.Update → Unit.Update 对齐
- 三条法术执行路径 — triggered instant 同帧、normal instant 同帧带注册、delayed 走 Update 链
- owned/applied aura 分离 — owner 驱动 tick、target 感受效果
- 所有技能测试统一用 engine.Tick 驱动 — 消除手动模拟循环

**Non-Goals:**
- 不实现 SpellEvent 事件队列（初版用简单的 timer 倒数，不做亚 tick 精度调度）
- 不实现 AuraApplication 类（初版 Aura 同时承担 owned 和 applied 职责，用 container 区分）
- 不实现网络/多线程（单线程模拟）
- 不实现 CurrentSpellTypes 4 槽位（初版用 []*Spell 列表）
- 不实现 SpellHistory 在 Unit 上（cooldown.History 保持独立）

## Decisions

### Decision 1: Unit 结构 — 四容器对齐 TC

**选择**: Unit 持有 `activeSpells []*Spell`、`ownedAuras []*aura.Aura`、`appliedAuras []*aura.Aura`、`history *cooldown.History`

**理由**: 参考 TC 的 Unit.h:1918-1922。TC 用 `m_currentSpells[4]` 固定数组、`m_ownedAuras multimap`、`m_appliedAuras multimap`。初版简化为 slice，但持有关系不变。

**实现**:
```go
type Unit struct {
    Entity        *entity.Entity
    Stats         *stat.StatSet
    History       *cooldown.History
    activeSpells  []*spell.Spell
    ownedAuras    []*aura.Aura
    appliedAuras  []*aura.Aura
    // engine back-reference
    engine        *Engine
}
```

### Decision 2: Engine.Tick — 统一驱动链

**选择**: Engine.Tick(diff) 按固定顺序遍历所有 Unit，每个 Unit 内按 TC 的 _UpdateSpells 顺序执行

**理由**: TC 的顺序是 (Unit.cpp:425-433): `WorldObject::Update → _UpdateSpells`。_UpdateSpells 内部顺序 (Unit.cpp:2952-2986): `清理 finished spells → 遍历 ownedAuras.UpdateOwner → 清理 expired auras`。这个顺序保证了 finished spell 在 aura tick 之前被清理，避免悬挂指针。

**实现**:
```go
func (e *Engine) Tick(diff int32) {
    e.advanceTime(diff)  // renderer.SetTime
    for _, unit := range e.units {
        unit.Update(diff)
    }
}

func (u *Unit) Update(diff int32) {
    u.updateSpells(diff)     // Spell.Update for each activeSpell; clean finished
    u.updateAuras(diff)      // Aura.UpdateOwner for each ownedAura; clean expired
}
```

### Decision 3: ownedAuras vs appliedAuras — 职责分离

**选择**: ownedAuras 由 caster 的 Unit.Update 驱动 tick 和过期；appliedAuras 记录在 target 的 Unit 上供查询

**理由**: TC 中 Aura 存储在 caster.m_ownedAuras，AuraApplication 存储在 target.m_appliedAuras (参考 tc-references/unit-update-architecture.md 2.3 节)。owner 负责驱动 tick，target 负责"感受"效果。初版不引入 AuraApplication 类，Aura 本身在两边注册。

**数据流**:
```
Caster Unit                        Target Unit
├── ownedAuras: [{LB aura}]        ├── appliedAuras: [{LB aura}]
│   updateAuras():                 │   (查询用：我身上有什么 debuff)
│     aura.Tick(diff)              │
│     if expired → RemoveFromBoth  │
```

### Decision 4: CastSpell — 注册而非自驱动

**选择**: 新的 CastSpell(caster, target, spellInfo) 创建 Spell，注册到 caster.activeSpells，根据类型决定执行路径

**理由**: TC 的 WorldObject::CastSpell (Object.cpp:2253) 创建 Spell 并调用 prepare()。prepare() 内部分三条路径。当前代码的 CastXxx 函数在内部调用 `s.Update(CastTime)` 来"预跑完"施法阶段，这违反了 Engine 是唯一驱动者的原则。

**三条路径实现**:
```go
func (e *Engine) CastSpell(caster *Unit, spellInfo *spell.SpellInfo, opts ...CastOption) *spell.Spell {
    s := spell.NewSpell(...)
    s.Prepare()

    if s.IsTriggeredInstant() {
        // 路径 A: 同帧同步完成
        s.Cast(true)
        s.HandleImmediate()
        s.Finish()
    } else if s.CastTime == 0 {
        // 路径 B: 同帧，但注册到 activeSpells
        caster.AddActiveSpell(s)
        s.Cast(true)
        s.HandleImmediate()
        s.Finish()
    } else {
        // 路径 C: 注册，等 Update 驱动
        caster.AddActiveSpell(s)
        s.SetTimer(s.CastTime)
    }
    return s
}
```

### Decision 5: Aura Manager 简化 — 工厂 + 查询

**选择**: aura.Manager 保留为 Aura 创建工厂和全局查询入口，但不再负责 Tick。Tick 由 Unit.updateAuras 驱动。

**理由**: 当前 Manager.TickPeriodic/TickPeriodicArea 的 tick 逻辑本质上是在做 owner 的工作。迁移到 Unit 后，Manager 只需要：CreateAura (工厂)、FindAura (查询)、RemoveAura (双向移除)。

**Manager 新职责**:
```go
type Manager struct {
    bus      *event.Bus
    registry *script.Registry
    // 不再持有 auras map — auras 分别存在 Unit.ownedAuras 和 Unit.appliedAuras
}

func (m *Manager) CreateAura(...) *Aura          // 工厂
func (m *Manager) ApplyAura(owner, target *Unit, ...)  // 注册到 owner.owned + target.applied
func (m *Manager) RemoveAura(aura, mode)          // 双向移除
func (m *Manager) FindAura(target *Unit, ...)     // 查询 target.appliedAuras
```

### Decision 6: 测试改造 — Engine.Tick 替代手动循环

**选择**: 所有技能测试改为创建 Engine，注册 Unit，调用 engine.Tick(step) 驱动模拟

**理由**: 消除每个测试手写的 `for simMs := 0; simMs < total; simMs += step` 循环。统一入口意味着统一的执行顺序，不会出现 fireball 先 spell 后 aura、arcane missiles 先 aura 后 spell 的不一致。

**测试模式**:
```go
func TestFireballTimeline(t *testing.T) {
    engine := engine.New()
    caster := engine.AddUnit(...)
    target := engine.AddUnit(...)

    engine.CastSpell(caster, &fireball.Info, spell.WithTarget(target))
    // Engine 内部驱动：instant 部分同帧完成，cast time 部分由 Tick 驱动

    for i := 0; i < totalSteps; i++ {
        engine.Tick(100)  // 100ms 步长
    }

    output := engine.Renderer.Render()
    t.Log("\n" + output)
}
```

### Decision 7: Aura tick 在 Unit 内的粒度

**选择**: Unit.updateAuras 对 ownedAuras 执行 tick。Area aura 需要额外的 target resolve 逻辑。

**理由**: TC 中 Aura::UpdateOwner 在 owner 的 _UpdateSpells 中被调用。对于单目标 aura，tick 作用在 aura.TargetID 上。对于 area aura（如 Blizzard），tick 时需要 resolve 当前区域内的 targets，然后对每个 target 触发效果。这个 resolve 需要访问 Engine 的 Unit 列表。

**实现**: Unit 持有 engine 反向引用，area aura tick 时通过 `unit.engine.GetUnitsInRadius(center, radius)` 获取目标列表。

## Risks / Trade-offs

**[Breaking change — 所有测试改写]** → 4 个技能的所有测试文件需要重写。这是不可避免的代价，但一次性完成。缓解：先完成 engine + unit，用一个技能（Fireball）验证，再逐个迁移其他技能。

**[AuraManager 双重职责过渡]** → 从"持有所有 aura + 驱动 tick"改为"工厂 + 查询"是最大的接口变更。缓解：保留 Manager 上的 FindAura/RemoveAura API 签名不变，内部改为查询 Unit.appliedAuras。

**[owned/applied 双注册一致性]** → 每个 aura 必须同时出现在 owner.ownedAuras 和 target.appliedAuras 中。如果一边移除另一边没移除，会导致悬挂引用。缓解：Manager.ApplyAura 和 Manager.RemoveAura 保证原子性双注册/双移除。

**[Area aura 的 owner-driven tick]** → 当 area aura 的 owner 不是 target 时，owner 的 updateAuras 需要 resolve target 列表。这要求 Unit 能访问 Engine 的全局 Unit 列表。缓解：Unit 持有 engine 反向引用。
