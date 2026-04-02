package generator

import (
	"fmt"
	"go/token"
	"path"
	"reflect"
	"slices"
	"strconv"

	"github.com/rustedturnip/fes/set"
)

func isValidIdentifier(id string) bool {
	if !token.IsIdentifier(id) {
		return false
	}

	if token.IsKeyword(id) {
		return false
	}

	return true
}

type ComponentID = int

type pkg struct {
	Path  string
	Name  string
	Alias string
}

type component struct {
	PkgID int
	Type  string
	Name  string
}

type composition struct {
	Name       string
	Components []int
}

type compositionAlias struct {
	Name          string
	CompositionID int

	// Components is the list of components provided to the alias. This is
	// tracked separately to the components in the composition aliased by this
	// in case the components were provided in a different order.
	Components []int
}

func (c component) component() component {
	return c
}

type Generator struct {
	packages []pkg

	// TODO comment
	components []component

	// TODO comment
	// example: FooBar: Foo{}:struct{}{}, Bar{}:struct{}{},
	compositions []composition

	// TODO comment
	compositionAliases []compositionAlias

	// compositionGraph tracks the subtypes of each composition. It can be
	// thought of as a map, where the index is the ID of the composition, and
	// the slice value contains a list of that compositions subtypes.
	compositionGraph [][]int
}

func RegisterComponent[T any](g *Generator, name string) ComponentID {
	exists := slices.ContainsFunc(
		g.components,
		func(c component) bool {
			return name == c.Name
		},
	)
	if exists {
		panic(
			fmt.Errorf(
				"component with the name %s already exists",
				name,
			),
		)
	}

	var t T
	rt := reflect.TypeOf(t)

	pID := g.trackPackage(rt.PkgPath())

	c := component{
		PkgID: pID,
		Type:  rt.String(),
		Name:  name,
	}

	g.components = append(g.components, c)

	return len(g.components) - 1
}

func (g *Generator) trackPackage(p string) int {
	base := path.Base(p)

	count := 0

	for i := range g.packages {
		if g.packages[i].Path == p {
			return i
		}

		if path.Base(g.packages[i].Path) != base {
			continue
		}

		count++
	}

	alias := ""
	if count > 0 {
		alias = path.Base(base) + strconv.Itoa(count)
	}

	g.packages = append(
		g.packages,
		pkg{
			Path:  p,
			Name:  base,
			Alias: alias,
		},
	)

	return len(g.packages) - 1
}

func RegisterComposition(g *Generator, name string, components ...ComponentID) {
	if !isValidIdentifier(name) {
		panic(
			fmt.Errorf(
				"invalid name provided for composition (must be valid "+
					"Go identifier): %s",
				name,
			),
		)
	}

	if !set.IsSet(components) {
		panic("duplicate components provided")
	}

	exists := slices.ContainsFunc(
		g.compositions,
		func(c composition) bool {
			return name == c.Name
		},
	)

	exists = exists || slices.ContainsFunc(
		g.compositionAliases,
		func(a compositionAlias) bool {
			return name == a.Name
		},
	)
	if exists {
		panic(
			fmt.Errorf(
				"composition with name %s already registered",
				name,
			),
		)
	}

	at := composition{
		Name:       name,
		Components: components,
	}

	// aliases
	for id, a := range g.compositions {
		if !set.AreEqual(components, a.Components) {
			continue
		}

		g.compositionAliases = append(g.compositionAliases, compositionAlias{
			Name:          name,
			CompositionID: id,
			Components:    components,
		})

		// if it's an alias, then it as a subtype is handled by the composition
		// it is an alias of, so return early
		return
	}

	i := len(g.compositions)
	g.compositions = append(g.compositions, at)
	g.compositionGraph = append(g.compositionGraph, nil)

	// subtypes
	for j := range len(g.compositions) - 1 {
		// as aliases are already handled, if same length here then one cannot
		// be subtype of the other
		if len(g.compositions[i].Components) ==
			len(g.compositions[j].Components) {
			continue
		}

		if set.IsSubset(
			g.compositions[i].Components,
			g.compositions[j].Components,
		) {
			g.compositionGraph[j] = append(g.compositionGraph[j], i)

			continue
		}

		if set.IsSubset(
			g.compositions[j].Components,
			g.compositions[i].Components,
		) {
			g.compositionGraph[i] = append(g.compositionGraph[i], j)
		}
	}
}

type templatePayload struct {
	Package            string
	Packages           map[string]pkg
	Components         map[component]interface{}
	Compositions       map[string]composition
	CompositionAliases map[string]composition
	CompositionGraph   map[string]map[string]any
}

func (g *Generator) Build(loc, pkgName string) {
	// TODO implement
}
