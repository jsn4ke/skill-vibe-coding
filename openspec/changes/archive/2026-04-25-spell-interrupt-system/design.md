## Context

当前引擎已完成四技能迁移，所有法术通过 Engine.CastSpell + Unit.Update 驱动。但 Spell.Update() 中的中断检查几乎是空的——只有 IsAlive() 检查和一个永远不触发的 IsMoving() 检查。

参考 TC 的 `spell-interrupt-cancel.md` 分析，TC 有三层中断：Spell 级持续验证、Aura 级事件驱动、cancel 状态特定清理。我们需要在现有引擎架构中构建等价系统。

当前关键文件：
- `pkg/spell/spell.go`: Spell.Update() 第 167-208 行，中断检查框架在
- `pkg/spell/info.go`: SpellAttribute 有 AttrBreakOnMove / AttrChanneled，但缺少 InterruptFlags
- `pkg/unit/unit.go`: IsMoving() 第 81 行硬编码 return false
- `pkg/aura/aura.go`: Aura 缺少 InterruptFlags
- TC reference: `tc-references/spell-interrupt-cancel.md`

## Goals / Non-Goals

**Goals:**
- Spell.Update() 中实现完整的中断检查：目标消失、移动打断、范围检查
- Channel 法术的持续目标验证（目标离范围、目标死亡、全部目标消失则终止）
- SpellInterruptFlags 标志系统替代当前简单的 AttrBreakOnMove
- Aura 级中断标志系统 + RemoveAurasWithInterruptFlags() 事件驱动移除
- Unit 基础移动追踪（SetPosition + isMoving）
- 所有四个技能的中断测试覆盖

**Non-Goals:**
- 完整的移动系统（路径、速度、spline）—— 只做位置变更 + isMoving 标记
- 推回（pushback）机制 —— 受伤时施法时间推回，属于另一层复杂性
- 朝向（facing）检查 —— TC 有 SPELL_FACING_FLAG_INFRONT，暂不实现
- LOS（视线）检查 —— 需要碰撞/地图系统支持
- GCD 系统 —— PREPARING 中断需要退 GCD，但我们没有 GCD

## Decisions

### Decision 1: 两套 Flag 系统，复用 SpellAttribute 位置

TC 有 SpellInterruptFlags（uint32）和 SpellAuraInterruptFlags（uint32）两套。我们目前 SpellAttribute 也是 bitmask。

**选择**: 新增两个独立类型 `SpellInterruptFlags` 和 `SpellAuraInterruptFlags`，不扩 SpellAttribute。原因：语义不同，SpellAttribute 是法术固有属性（被动、瞬发、引导），InterruptFlags 是中断条件，混在一起职责不清。

现有的 AttrBreakOnMove 保留为兼容别名，底层映射到 SpellInterruptFlags.Movement。

### Decision 2: isMoving 基于位置变化检测

TC 用 `_positionUpdateInfo.Relocated` 标记。我们在 Unit 维护 `prevPos`，每次 Advance 末尾对比。如果位置与上一次不同，isMoving=true；相同则 isMoving=false。SetPosition() 只更新位置，isMoving 的判定在 Advance() 做一次。

**替代方案**: 每次 SetPosition() 设 isMoving=true。问题：可能在同一个 Update 内多次设位置。用 Advance 末尾统一判定更干净。

### Decision 3: Channel 目标验证放在 Spell.Update() 里

TC 在 Spell::update() 的 CHANNELING 分支调用 UpdateChanneledTargetList()。我们也在 Spell.Update() 的 StateChanneling 分支做。每 tick 检查：
1. 目标是否还存在（engine.GetUnit != nil）
2. 目标是否还活着
3. 目标是否在范围内（距离 > RangeMax + tolerance）

不满足则从 TargetInfos 移除并移除对应 aura。全部移除则 cancel()。

### Decision 4: RemoveAurasWithInterruptFlags 在 Unit.Update 中调用

移动触发的中断：Engine.Advance() → Unit.Update() → detect movement → RemoveAurasWithInterruptFlags(Moving)。

不把 RemoveAurasWithInterruptFlags 放在 Engine 层，因为 TC 的设计是 Unit 自己管理自己的 aura 中断。Engine 只负责驱动 Update。

### Decision 5: cancel() 采用 TC 的 oldState 恢复模式

cancel() 记录 oldState，做状态特定清理后恢复 oldState 给 finish()。这样 finish() 可以根据原始状态做不同收尾。当前实现已经是 StateFinished 然后直接 Finish，不够精细。

### Decision 6: 范围容差

参考 TC，非严格模式给 10% 容差（但不超过固定最大值）。Spell.Update() 中的范围检查使用 strict=false（给容差），因为目标可能在移动中。

## Risks / Trade-offs

- **[风险] 移动中断标志与 AttrBreakOnMove 重叠** → AttrBreakOnMove 保留但标记 deprecated，新增 SpellInterruptFlags.Movement。迁移期两者都检查，后续移除 AttrBreakOnMove。
- **[风险] Channel 目标验证每 tick 做距离计算** → 性能可接受（单位数少）。如果单位数增长，可用缓存优化。
- **[风险] AuraInterruptFlags 可能触发级联移除** → TC 用 `m_removedAurasCount` 检测级联。我们简化：RemoveAurasWithInterruptFlags 遍历 ownedAuras 一次，不做递归。
- **[取舍] 不做 pushback** → 受伤推回（增加施法时间）需要额外的 timer 管理逻辑，增加复杂度。先做 cancel 级别的中断，pushback 后续独立 proposal。
