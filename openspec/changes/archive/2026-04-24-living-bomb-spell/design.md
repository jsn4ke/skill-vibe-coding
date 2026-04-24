## Context

项目是一个 WoW 风格的战斗模拟框架 (skill-go)，已实现 3 个技能：Fireball（弹道投射 + DoT）、Blizzard（引导 AoE 区域伤害）、Arcane Missiles（引导周期触发法术）。

框架已有 `pkg/script/script.go` 的 Registry（spellHooks + auraHooks），但**从未被任何运行时代码调用**。已有 `pkg/targeting/targeting.go` 的 TargetSelector（含 SelectAroundPoint），但 **Spell.SelectTargets() 未使用它**。

Living Bomb 的三体结构需要把这些已有但未接入的系统全部打通，形成完整的脚本驱动架构。

参考文档：`skill-designs/living-bomb-44457.md`、`skill-designs/living-bomb-217694.md`、`skill-designs/living-bomb-44461.md`。

## Goals / Non-Goals

**Goals:**
- 打通脚本系统到运行时生命周期（effect pipeline + aura lifecycle）
- 实现 Living Bomb 三体结构（44457 → 217694 → 44461 → 传染循环）
- SpellValues 传递机制（传染标记 BASE_POINT2 在三体链中流转）
- AoE 目标选择接入 Spell（44461 爆炸选择 10yd 内敌人）
- RemoveMode 过滤（仅 EXPIRE 触发爆炸，死亡/驱散不触发）
- 时间线测试验证完整事件序列

**Non-Goals:**
- 不实现 FilterTargets hook（44461 排除载体——初版通过 excludeID 参数实现）
- 不实现 Spell 全局注册表（触发法术 SpellInfo 在技能包内定义）
- 不实现法力百分比消耗（初版用固定值替代）
- 不实现天赋/装备修正器链
- 不实现多等级 Living Bomb

## Decisions

### Decision 1: 脚本 hook 接入点选择——侵入式 vs 回调式

**选择**: 侵入式——在 `effect.ProcessAll` 和 `aura.Manager` 的关键路径中直接调用 `registry.CallSpellHook / CallAuraHook`

**替代方案**: 回调式——在 Spell/Aura 上设置回调函数

**理由**:
- TC 的做法就是在关键路径中调用 `m_script->OnEffectHit` 等 hook
- Registry 已有完整的 CallSpellHook/CallAuraHook 实现，只是从未被调用
- 侵入式更符合 TC 的"脚本系统是基础设施"的定位
- 回调式需要每个 Spell 实例都管理回调，不如全局 Registry 统一

**实现细节**:
- `effect.ProcessAll` 需要接收 `*script.Registry` 参数（通过闭包或全局变量）
- 在每个效果执行前调用 `OnEffectHit` hook，检查 `PreventDefault`
- `aura.Manager` 持有 `*script.Registry`，在 RemoveAura 中调用 `AfterRemove` hook

### Decision 2: effect.ProcessAll 如何获取 Registry

**选择**: 全局变量——在 effect 包的 init() 中设置，与现有 `spell.ProcessEffectsFn` 模式一致

**替代方案**: 通过 Spell 结构体传递 Registry

**理由**:
- 现有模式是 `init() { spell.ProcessEffectsFn = ProcessAll }`，ProcessAll 是函数而非方法
- 如果改 ProcessAll 签名为接收 Registry，需要改 spell.ProcessEffectsFn 的类型定义
- 全局变量最小化变更范围——添加 `var Registry *script.Registry` 即可

**实现细节**:
```go
// pkg/effect/effect.go
var ScriptRegistry *script.Registry

func ProcessAll(s *spell.Spell, mode spell.EffectHandleMode) {
    // ... 每个 effect 处理前检查 ScriptRegistry
}
```

### Decision 3: Aura 过期回调——在 Manager.TickPeriodic 中调用 hook

**选择**: 在 TickPeriodic 的过期处理段（expired 列表循环中），在 RemoveAura 之前调用 AuraScript.AfterRemove hook

**替代方案**: 在 RemoveAura 本身中调用 hook

**理由**:
- TickPeriodic 已经在过期时发布 OnAuraExpired 事件，hook 应在同一位置调用
- RemoveAura 被多种场景调用（过期、驱散、死亡、取消），hook 在 RemoveAura 中调用会丢失 RemoveMode 上下文
- 在 TickPeriodic 的过期段调用，可以准确传递 RemoveMode=RemoveByExpire

**实现细节**:
```go
// TickPeriodic 过期段
for _, a := range expired {
    // 调用脚本 hook (RemoveByExpire)
    if m.registry != nil {
        ctx := &script.AuraContext{
            RemoveMode: aura.RemoveByExpire,
            Aura:       a,
            ...
        }
        m.registry.CallAuraHook(a.SpellID, script.AuraHookAfterRemove, ctx)
    }
    m.RemoveAura(a, RemoveByExpire)
}
```

### Decision 4: SpellValues 存储——在 Aura 中

**选择**: Aura 结构体添加 `SpellValues map[uint8]float64`，脚本创建触发法术时从 Aura 读取

**替代方案**: Spell 结构体上的 SpellValues（当前法术有效，但需要跨 Aura 传递）

**理由**:
- 传染标记 (BASE_POINT2) 需要在 44457→217694 Aura→44461→217694 Aura 之间传递
- Aura 是跨法术实例的生命周期载体——44457 创建 Aura，44461 从 Aura 读取值
- Spell 的 SpellValues 只在当前法术实例有效，无法跨 Aura 传递

### Decision 5: AoE 目标选择——扩展 Spell.SelectTargets

**选择**: 在 SelectTargets 中，当 SpellEffectInfo.TargetA 为 `TargetUnitAreaEnemy` 时，调用 targeting.TargetSelector

**替代方案**: 在脚本中手动选择目标

**理由**:
- AoE 目标选择是通用能力，不仅 Living Bomb 需要
- targeting 包已实现完整的 SelectAroundPoint，只需接入
- SelectTargets 是目标选择的正确位置——在 Cast() 中调用，在 ProcessEffects 之前

**实现细节**:
- Spell 需要持有 `TargetSelector` 引用（或通过闭包注入）
- SelectTargets 检查效果的 TargetA，如果是 Area 类型，使用 TargetSelector 选择多个目标
- 中心点从 Targets.SourcePos/DestPos 获取，或从目标实体位置获取

### Decision 6: 44461 排除载体——excludeID 方式

**选择**: 在 Spell.SelectTargets 中，通过 Aura 的 TargetID 作为 excludeID 传递给 SelectAroundPoint

**替代方案**: 实现 FilterTargets 脚本 hook

**理由**: 初版简化——excludeID 是一个参数而非脚本 hook，实现简单且满足需求。FilterTargets hook 留作后续需求。

## Risks / Trade-offs

**[全局 Registry 的测试隔离]** → 全局 `effect.ScriptRegistry` 在测试中需要手动清理。缓解：提供 `ResetRegistry()` 或使用 `defer` 清理。

**[ProcessAll 签名变更]** → 如果改为接收 Registry 参数，需要修改 `spell.ProcessEffectsFn` 类型定义。选择全局变量避免此问题。

**[Aura 爆炸回调的时序]** → 必须在 OnAuraExpired 事件发布和 RemoveAura 调用之前触发脚本 hook，否则 Aura 已被移除，脚本无法读取 SpellValues。缓解：严格按照 事件→脚本→移除 的顺序。

**[传染递归深度]** → 理论上传染链是有限的（BASE_POINT2=0 终止），但如果实现 bug 导致 BASE_POINT2 不正确，可能无限递归。缓解：在 CastXxx 入口处硬编码传染标记。
