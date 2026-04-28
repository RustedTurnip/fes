package entities

import "golang.org/x/exp/constraints"

type Number interface {
	constraints.Integer | constraints.Float
}

type Vector[T Number] struct {
	X T
	Y T
}
