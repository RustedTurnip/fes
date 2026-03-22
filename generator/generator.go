package generator

import (
	"fmt"
	"go/token"
	"path"
	"reflect"
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

type component struct {
	pkg  string
	typ  string
	name string
}

type archetype struct {
	name       string
	components map[component]any
}

func (c component) component() component {
	return c
}

type Component interface {
	component() component
}

type Generator struct {
	packages map[string]int

	// TODO comment
	componentsByName map[string]any

	// TODO comment
	components map[component]any

	// TODO comment
	archetypeAliases map[string]map[string]any

	// TODO comment
	// example: FooBar: Foo{}:struct{}{}, Bar{}:struct{}{},
	archetypes map[string]archetype

	// archetypeGraph contains each archetype as a key, and then the set of
	// archetypes that it is a subset of as the value.
	archetypeGraph map[string]map[string]any
}

func RegisterComponent[T any](g *Generator, name string) Component {
	_, ok := g.componentsByName[name]
	if ok {
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
		pkg:  rt.PkgPath(),
		typ:  rt.String(),
		name: name,
	}

	g.trackPackage(c.pkg)
	g.components[c] = struct{}{}

	return c
}

func (g *Generator) trackPackage(pkg string) {
	_, ok := g.packages[pkg]
	if ok {
		return
	}

	pb := path.Base(pkg)

	count := 0

	for p := range g.packages {
		if path.Base(p) != pb {
			continue
		}

		count++
	}

	g.packages[pkg] = count
}

func RegisterArchetype(g *Generator, name string, components ...Component) {
	if !isValidIdentifier(name) {
		panic(
			fmt.Errorf(
				"invalid name provided for archetype (must be valid Go "+
					"identifier): %s",
				name,
			),
		)
	}

	_, asArchetype := g.archetypes[name]
	_, asArchetypeAlias := g.archetypeAliases[name]
	if asArchetype || asArchetypeAlias {
		panic(
			fmt.Errorf(
				"archetype with name %s already registered",
				name,
			),
		)
	}

	at := archetype{
		name:       name,
		components: map[component]any{},
	}

	for _, c := range components {
		at.components[c.component()] = struct{}{}
	}

	if len(at.components) != len(components) {
		panic(
			fmt.Errorf(
				"duplicate component provided to %s archetype",
				name,
			),
		)
	}

	// track archetype (or alias if archetype already exists under a different
	// name)
aliases:
	for _, atype := range g.archetypes {
		if len(at.components) != len(atype.components) {
			continue
		}
		for cmp, _ := range at.components {
			_, ok := atype.components[cmp]
			if !ok {
				continue aliases
			}
		}

		// found to be an alias of existing archetype
		g.archetypeAliases[atype.name][name] = struct{}{}

		// exit early as the archetype that this one is an alias of has already
		// been processed so no further steps required
		return
	}

	// check which (if any) archetypes are a subtype of this one (or vice versa)
subtypes:
	for _, atype := range g.archetypes {
		// as already handled, if same length here, then one cannot be subtype
		// of the other
		if len(at.components) == len(atype.components) {
			continue
		}

		a, b := atype, at

		if len(a.components) > len(components) {
			a, b = b, a
		}

		for k := range a.components {
			_, ok := b.components[k]
			if !ok {
				continue subtypes
			}
		}

		g.archetypeGraph[b.name][a.name] = struct{}{}
	}
}

func (g *Generator) Build(loc, pkgName string) {
	// TODO implement
}

func (g *Generator) buildImports() {
	// package path: package alias
	pkgs := map[string]string{}

	for cmp := range g.components {
		_, ok := pkgs[cmp.pkg]
		if ok {
			continue
		}

		for _, alias := range pkgs {
			if path.Base(cmp.pkg) != alias {
				pkgs[cmp.pkg] = alias
			}
		}
	}
}
