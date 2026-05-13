package main

import (
	"fmt"

	"github.com/rustedturnip/fes/example/entities"
	"github.com/rustedturnip/fes/schema"
)

//go:generate go run .
func main() {
	s := schema.New(
		schema.Config{
			Output: "../store_gen.go",
		},
	)

	cPosition := schema.MustRegisterComponent[entities.Vector[float32]](s, "position")
	cDirection := schema.MustRegisterComponent[float64](s, "direction")
	cVelocity := schema.MustRegisterComponent[float64](s, "velocity")
	cIncome := schema.MustRegisterComponent[int](s, "income")
	cGold := schema.MustRegisterComponent[int](s, "gold")

	schema.MustRegisterComposition(s, "Positionable", cPosition)
	schema.MustRegisterComposition(s, "Travellable", cPosition, cDirection, cVelocity)
	schema.MustRegisterComposition(s, "Trader", cIncome, cGold)
	schema.MustRegisterComposition(s, "Ship", cPosition, cDirection, cVelocity, cIncome, cGold)

	err := schema.Build(s)
	if err != nil {
		panic(
			fmt.Errorf(
				"failed to generate store from schema: %w",
				err,
			),
		)
	}
}
