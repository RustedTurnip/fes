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

type ComponentID int

type pkg struct {
	Path  string
	Name  string
	Alias string
}

type component struct {
	Pkg  string
	Typ  string
	Name string
}

type archetype struct {
	Name       string
	Components []ComponentID
}

type archetypeAlias struct {
	Name        string
	ArchetypeID int

	// Components is the list of components provided to the alias. This is
	// tracked separately to the components in the archetype aliased by this in
	// case the components were provided in a different order.
	Components []ComponentID
}

func (c component) component() component {
	return c
}

type Generator struct {
	packages map[string]pkg

	// TODO comment
	components []component

	// TODO comment
	// example: FooBar: Foo{}:struct{}{}, Bar{}:struct{}{},
	archetypes []archetype

	// TODO comment
	archetypeAliases []archetypeAlias

	// archetypeGraph tracks the subtypes of each archetype. It can be thought
	// of as a map, where the index is the ID of the archetype, and the slice
	// value contains a list of that archetypes subtypes.
	archetypeGraph [][]int
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

	c := component{
		Pkg:  rt.PkgPath(),
		Typ:  rt.String(),
		Name: name,
	}

	id := ComponentID(len(g.components))

	g.trackPackage(c.Pkg)
	g.components = append(g.components, c)

	return id
}

func (g *Generator) trackPackage(np string) {
	_, ok := g.packages[np]
	if ok {
		return
	}

	pb := path.Base(np)

	count := 0

	for p := range g.packages {
		if path.Base(p) != pb {
			continue
		}

		count++
	}

	alias := ""
	if count > 0 {
		alias = path.Base(np) + strconv.Itoa(count)
	}

	g.packages[np] = pkg{
		Path:  np,
		Name:  path.Base(np),
		Alias: alias,
	}
}

func RegisterArchetype(g *Generator, name string, components ...ComponentID) {
	if !isValidIdentifier(name) {
		panic(
			fmt.Errorf(
				"invalid name provided for archetype (must be valid Go "+
					"identifier): %s",
				name,
			),
		)
	}

	if !set.IsSet(components) {
		panic("duplicate components provided")
	}

	exists := slices.ContainsFunc(
		g.archetypes,
		func(a archetype) bool {
			return name == a.Name
		},
	)

	exists = exists || slices.ContainsFunc(
		g.archetypeAliases,
		func(alias archetypeAlias) bool {
			return name == alias.Name
		},
	)
	if exists {
		panic(
			fmt.Errorf(
				"archetype with name %s already registered",
				name,
			),
		)
	}

	at := archetype{
		Name:       name,
		Components: components,
	}

	// aliases
	for id, a := range g.archetypes {
		if !set.AreEqual(components, a.Components) {
			continue
		}

		g.archetypeAliases = append(g.archetypeAliases, archetypeAlias{
			Name:        name,
			ArchetypeID: id,
			Components:  components,
		})

		// if it's an alias, then it as a subtype is handled by the archetype
		// it is an alias of, so return early
		return
	}

	i := len(g.archetypes)
	g.archetypes = append(g.archetypes, at)
	g.archetypeGraph = append(g.archetypeGraph, nil)

	// subtypes
	for j := range len(g.archetypes) - 1 {
		// as aliases are already handled, if same length here then one cannot
		// be subtype of the other
		if len(g.archetypes[i].Components) == len(g.archetypes[j].Components) {
			continue
		}

		if set.IsSubset(
			g.archetypes[i].Components,
			g.archetypes[j].Components,
		) {
			g.archetypeGraph[j] = append(g.archetypeGraph[j], i)

			continue
		}

		if set.IsSubset(
			g.archetypes[j].Components,
			g.archetypes[i].Components,
		) {
			g.archetypeGraph[i] = append(g.archetypeGraph[i], j)
		}
	}
}

type templatePayload struct {
	Package          string
	Packages         map[string]pkg
	Components       map[component]interface{}
	Archetypes       map[string]archetype
	ArchetypeAliases map[string]archetype
	ArchetypeGraph   map[string]map[string]any
}

func (g *Generator) Build(loc, pkgName string) {
	// TODO implement
}
