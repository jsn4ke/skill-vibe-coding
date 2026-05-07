# TC References

TrinityCore 机制分析知识库。由 `tc-mechanism-ref` skill 自动生成。

文件命名规则：`<topic-slug>.md`（英文文件名，中文内容）

## 索引

| 文件 | 主题 |
|------|------|
| spell-decomposition-philosophy.md | 法术拆解设计哲学：5 种组合模式 + 决策流程图 + WoW 实例 |
| effect-pipeline.md | 效果管线：Launch/Hit 四阶段、Handler mode guard |
| effect-trigger-spell.md | EffectTriggerSpell 机制：数据驱动的子法术触发 |
| spell-cast-flow.md | 法术施放流程：Prepare → Cast → Launch → Hit |
| spell-script.md | 法术脚本系统：Dummy+Script hook、HookList、PreventDefault |
| spell-interrupt-cancel.md | 施法打断/取消：InterruptFlags、Channel 取消 |
| spell-timing-fields.md | 法术时序字段：LaunchDelay、Speed、TimeDelay |
| damage-dealing-settlement.md | 伤害结算：计算→衰减→吸收→护盾→最终结算 |
| target-selection.md | 目标选择：ImplicitTarget 类型、AoE 衰减 |
| unit-update-architecture.md | Unit 更新架构：Engine→Unit→Spell/Aura 驱动链 |
