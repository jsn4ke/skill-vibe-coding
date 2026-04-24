package main

import (
	"fmt"

	"skill-go/pkg/combat"
	"skill-go/pkg/cooldown"
	"skill-go/pkg/diminishing"
	"skill-go/pkg/entity"
	"skill-go/pkg/proc"
	"skill-go/pkg/script"
	"skill-go/pkg/stat"
	"skill-go/pkg/timer"
)

type Unit struct {
	Entity  *entity.Entity
	Stats   *stat.StatSet
	History *cooldown.History
	Targets map[uint64]*Unit
}

func main() {
	fmt.Println("=== Skill System Demo ===")

	cdMgr := cooldown.NewHistory()
	dimMgr := diminishing.NewManager()
	procMgr := proc.NewManager()
	scriptReg := script.NewRegistry()
	combatMgr := combat.NewCombatManager()
	sched := timer.NewScheduler()
	sched.Start()
	defer sched.Stop()

	_ = cdMgr
	_ = dimMgr
	_ = procMgr
	_ = scriptReg
	_ = combatMgr

	dimMgr.RegisterLevel(diminishing.Level{Group: diminishing.GroupStun, ReturnType: diminishing.ReturnStandard, MaxLevel: 3})
	dimMgr.RegisterLevel(diminishing.Level{Group: diminishing.GroupFear, ReturnType: diminishing.ReturnStandard, MaxLevel: 3})

	caster := &Unit{
		Entity:  entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0, Y: 0, Z: 0, Facing: 0}),
		Stats:   stat.NewStatSet(),
		Targets: make(map[uint64]*Unit),
	}
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.Mana, 500)
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.CritChance, 0.15)

	_ = caster

	// Skill simulations and timeline verification live in skill tests:
	//   skills/fireball/fireball_timeline_test.go
	//   skills/blizzard/blizzard_timeline_test.go

	fmt.Println()
	fmt.Println("Run tests to see skill timelines:")
	fmt.Println("  go test ./skills/... -run Timeline -v")
	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}
