package main

import (
	"fmt"

	"skill-go/pkg/engine"
	"skill-go/pkg/entity"
	"skill-go/pkg/stat"
)

func main() {
	fmt.Println("=== Skill System Demo ===")

	eng := engine.New()

	caster := eng.AddUnitWithID(1, entity.NewEntity(1, entity.TypePlayer, entity.Position{X: 0}), stat.NewStatSet())
	caster.Stats.SetBase(stat.Health, 1000)
	caster.Stats.SetBase(stat.Mana, 500)
	caster.Stats.SetBase(stat.SpellPower, 100)
	caster.Stats.SetBase(stat.CritChance, 0.15)

	// Skill simulations and timeline verification live in skill engine tests:
	//   skills/fireball/fireball_engine_test.go
	//   skills/blizzard/blizzard_engine_test.go
	//   skills/arcane-missiles/arcane_missiles_engine_test.go
	//   skills/living-bomb/living_bomb_engine_test.go

	fmt.Println()
	fmt.Println("Run engine-driven tests to see skill timelines:")
	fmt.Println("  go test ./skills/... -run Engine -v")
	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}
