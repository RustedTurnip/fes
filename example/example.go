package main

import (
	"fmt"

	"github.com/rustedturnip/fes/example/entities"
)

func main() {
	s := entities.NewStore()

	for i := 0; i < 100; i++ {
		_ = s.Positionable().Put(
			entities.Vector[float32]{
				X: 25,
				Y: 100,
			},
		)

		_ = s.Ship().Put(
			entities.Vector[float32]{
				X: 25,
				Y: 100,
			},
			63,
			72,
			250,
			125,
		)
	}

	pos := s.Positionable().All()
	fmt.Println("START: Positionables")

	for {
		for i := range len(pos.Positionables().IDs) {
			fmt.Println(pos.Positionables().IDs[i])
		}

		if !pos.Next() {
			break
		}
	}

	fmt.Println("END: Positionables")
	fmt.Println("START: Ships")

	ships := s.Ship().All()

	for {
		for i := range len(ships.Ships().IDs) {
			fmt.Println(ships.Ships().IDs[i])
		}

		if !ships.Next() {
			break
		}
	}

	fmt.Println("END: Ships")
}
