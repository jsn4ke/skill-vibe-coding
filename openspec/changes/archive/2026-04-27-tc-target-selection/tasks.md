## Implementation Tasks

### Phase 1: Targeting 包重写 — 数据驱动查表

- [x] T1: 定义 5 维枚举类型和 StaticData 结构体
  - 在 `pkg/targeting/` 中定义 `ObjectTypes`（ObjNone/ObjUnit/ObjDest/ObjSrc/ObjUnitAndDest）、`ReferenceTypes`（RefNone/RefCaster/RefTarget/RefLast/RefSrc/RefDest）、`SelectionCategory`（SelectNYI/SelectDefault/SelectNearby/SelectCone/SelectArea/SelectLine/SelectChannel/SelectTraj）、`CheckTypes`（CheckDefault/CheckEnemy/CheckAlly/CheckParty/CheckRaid/CheckSummoned/CheckEntry）、`DirectionTypes`（DirNone/DirFront/DirBack/DirRight/DirLeft/DirFrontRight/DirBackRight/DirBackLeft/DirFrontLeft/DirRandom/DirEntry）
  - 定义 `StaticData` 结构体包含上述 5 个字段
  - 定义 `ImplicitTargetInfo` 类型（包装 `ImplicitTarget` 值）提供 `GetObjectType()`/`GetReferenceType()`/`GetSelectionCategory()`/`GetCheckType()`/`GetDirectionType()`/`CalcDirectionAngle()`/`IsArea()` 方法
  - 文件: `pkg/targeting/types.go`

- [x] T2: 扩展 ImplicitTarget 枚举至 ~50 个值
  - 在 `pkg/spell/info.go` 中扩展 `ImplicitTarget` 常量，使用 TC 原始编号（显式赋值，不用 iota）
  - 覆盖: Group A (1,5,6,21,25,27,92), Group B (2,3,4,38), Group C (15,16,20,30,31,33,34,37,56,118,120,122), Group D (24,54,59,104,128,129,136), Group E (133,134,135), Group F (76,77), Group H (18,28,29,47-50,53,62,63,87,88,91,131,132,138)
  - 保留 `TargetNone = 0`
  - 文件: `pkg/spell/info.go`

- [x] T3: 实现 targetData 查表数组
  - 在 `pkg/targeting/` 中实现 `var targetData [MaxImplicitTarget]StaticData`，每个 ImplicitTarget 值映射到对应 StaticData
  - 未实现的索引填充 `{SelectNYI}` 默认值
  - 实现 `NewImplicitTargetInfo(target ImplicitTarget) ImplicitTargetInfo` 构造函数
  - 实现 `CalcDirectionAngle()` 方向转弧度转换
  - 文件: `pkg/targeting/lookup.go`

- [x] T4: SpellEffectInfo 新增 Radius 和 ChainTargets 字段
  - 在 `SpellEffectInfo` 结构体中新增 `Radius float64` 和 `ChainTargets int32`
  - 更新所有现有 SpellInfo 定义，为需要半径的 effect 设置 Radius（Blizzard=8.0, Living Bomb explosion=10.0）
  - 文件: `pkg/spell/info.go`, `skills/*/`

### Phase 2: 目标解析引擎

- [x] T5: 实现 resolveCenter() 和 resolveReferer()
  - `resolveCenter(ref ReferenceTypes, spell *Spell) [3]float64` — 根据 ReferenceType 解析搜索中心位置
  - `resolveReferer(ref ReferenceTypes, spell *Spell) *unit.Unit` — 根据 ReferenceType 解析参考单位
  - RefCaster → spell.Caster, RefTarget → spell.Targets.UnitTarget, RefLast → TargetInfos 最后一个, RefSrc → DestPos, RefDest → DestPos
  - 文件: `pkg/targeting/resolve.go`

- [x] T6: 实现 passesCheck() 友敌过滤
  - `passesCheck(check CheckTypes, caster *unit.Unit, candidate *unit.Unit) bool`
  - CheckDefault → true, CheckEnemy → 不同 EntityType, CheckAlly → 相同 EntityType, CheckParty/CheckRaid → 相同 EntityType (fallback), CheckSummoned → 召唤关系
  - 文件: `pkg/targeting/check.go`

- [x] T7: 实现 SelectDefault 分支
  - 处理 ObjectType=Unit 的 DEFAULT 目标：根据 ReferenceType 添加 Caster/Target 到 TargetInfos
  - 处理 ObjectType=Dest/Src 的 DEFAULT 目标：设置 Spell.Targets 的 SrcPos/DestPos
  - 处理 DirectionType：根据方向在参考位置偏移生成 DestPos
  - 文件: `pkg/targeting/select_default.go`

- [x] T8: 实现 SelectArea 分支 — SearchAreaTargets
  - `searchAreaTargets(center [3]float64, radius float64, check CheckTypes, caster *unit.Unit, excludeID uint64) []*unit.Unit`
  - 调用 engine 的 GetUnitsInRadius 获取候选，passesCheck 过滤，排除 excludeID
  - 文件: `pkg/targeting/select_area.go`

- [x] T9: 实现 SelectCone 分支 — SearchConeTargets
  - `searchConeTargets(center [3]float64, direction float64, arcAngle float64, radius float64, check CheckTypes, caster *unit.Unit, excludeID uint64) []*unit.Unit`
  - 在 Area 基础上增加角度过滤：计算候选相对 caster 朝向的角度，在 [direction - arcAngle/2, direction + arcAngle/2] 范围内
  - 文件: `pkg/targeting/select_cone.go`

- [x] T10: 实现 SelectLine 分支 — SearchLineTargets
  - `searchLineTargets(from [3]float64, to [3]float64, width float64, check CheckTypes, caster *unit.Unit, excludeID uint64) []*unit.Unit`
  - 计算候选到线段的距离，在 width/2 范围内
  - 文件: `pkg/targeting/select_line.go`

- [x] T11: 实现 SelectNearby 分支 — SearchNearbyTarget
  - `searchNearbyTarget(center [3]float64, radius float64, check CheckTypes, caster *unit.Unit, excludeID uint64) *unit.Unit`
  - 在范围内找最近的一个合法目标
  - 文件: `pkg/targeting/select_nearby.go`

- [x] T12: 实现 SelectChannel 分支
  - 从 spell 的 channel spell 获取目标
  - 文件: `pkg/targeting/select_channel.go`

### Phase 3: Chain 跳跃模型

- [x] T13: 实现 searchChainTargets 跳跃搜索
  - `searchChainTargets(initial *unit.Unit, maxJumps int, jumpRadius float64, check CheckTypes, caster *unit.Unit, excludeIDs []uint64) []*unit.Unit`
  - 从 initial 开始，每次在 jumpRadius 内找最近/最优目标，更新 chainSource 继续跳跃
  - 支持 ChainHeal 模式：选最大 HP deficit 的目标
  - 排除已选目标
  - 文件: `pkg/targeting/select_chain.go`

### Phase 4: Spell 集成

- [x] T14: 重写 Spell.SelectTargets() 为 SelectEffectTargets()
  - 对每个 effect 的 TargetA 和 TargetB 分别调用 selectEffectImplicitTargets()
  - 实现 processedEffectMask 共享搜索优化
  - 实现 TargetA 先、TargetB 后的解析顺序
  - 移除旧的 SelectTargets() 逻辑
  - 文件: `pkg/spell/spell.go`

- [x] T15: 实现 selectEffectImplicitTargets() 分发函数
  - 根据 SelectionCategory 分发到 T7-T12 的各分支
  - 将搜索结果添加到 Spell.TargetInfos
  - ChainTargets > 0 时调用 searchChainTargets
  - 文件: `pkg/spell/spell.go`

- [x] T16: 移除 AoESelector 接口和相关字段
  - 删除 `AoESelector` 接口定义
  - 删除 `WithAoE()` CastOption
  - 删除 `Spell.AoESelector`/`Spell.AoECenter`/`Spell.AoEExcludeID` 字段
  - 删除 engine 中 AoESelector 相关的初始化逻辑
  - 文件: `pkg/engine/engine.go`, `pkg/spell/spell.go`

- [x] T17: 新增 HookOnTargetSelect Script Hook
  - 在 `script.Registry` 中新增 `HookOnTargetSelect` hook 类型
  - 在 selectEffectImplicitTargets 搜索完成后、AddTarget 前调用 hook
  - 文件: `pkg/script/registry.go`, `pkg/spell/spell.go`

### Phase 5: 迁移现有技能

- [x] T18: 迁移 Fireball
  - 确认 TargetA=TargetUnitTargetEnemy, TargetB=TargetNone
  - 移除任何 WithAoE 调用（如有）
  - 更新测试
  - 文件: `skills/fireball/`

- [x] T19: 迁移 Blizzard
  - 设置 TargetA=TargetDestCasterGround, TargetB=TargetUnitAreaEnemy, Radius=8.0
  - 移除 WithAoE 调用
  - 确保 tickAreaAura 使用 CheckEnemy 过滤
  - 更新测试
  - 文件: `skills/blizzard/`

- [x] T20: 迁移 Arcane Missiles
  - 确认 TargetA=TargetUnitTargetEnemy, TargetB=TargetNone
  - 移除 WithAoE 调用（如有）
  - 更新测试
  - 文件: `skills/arcane-missiles/`

- [x] T21: 迁移 Living Bomb
  - 主技能: TargetA=TargetUnitTargetEnemy, TargetB=TargetNone
  - 爆炸技能: TargetA=TargetUnitAreaEnemy, Radius=10.0
  - 移除 AoESelector 注入和 WithAoE 调用
  - 爆炸目标选择改为数据驱动（TargetA+Radius 自动解析）
  - 更新 RegisterScripts 中的爆炸逻辑
  - 更新测试：移除 WithAoE，验证 AoE 由引擎自动解析
  - 文件: `skills/living-bomb/`

### Phase 6: 验证

- [x] T22: 运行全量测试 + race 检测
  - `go test -race ./...`
  - 所有技能测试通过
  - 无 race warning
