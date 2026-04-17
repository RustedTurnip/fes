package generator

import (
	"fmt"
	"go/token"
	"os"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template"

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

	// compositionGraph tracks the subtypes of each composition. It can be
	// thought of as a map, where the index is the ID of the composition, and
	// the slice value contains a list of that compositions subtypes.
	compositionGraph [][]int
}

func RegisterComponent[T any](g *Generator, name string) ComponentID {
	exists := slices.ContainsFunc(
		g.components,
		func(c component) bool {
			return strings.EqualFold(name, c.Name)
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
			return strings.EqualFold(name, c.Name)
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

	i := len(g.compositions)
	g.compositions = append(g.compositions, at)
	g.compositionGraph = append(g.compositionGraph, nil)

	// subtypes
	for j := range len(g.compositions) - 1 {
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

func (g *Generator) Build(loc, pkgName string) {
	// TODO make output.tmpl built into lib (maybe as variable)
	tmpl, err := template.
		New("output.tmpl").
		ParseFiles("output.tmpl") // TODO better way to provide templ
	if err != nil {
		panic(err) // TODO wrap error
	}

	fo, err := os.Create(loc)
	if err != nil {
		panic(err) // TODO wrap error
	}

	payload := newTemplatePayload{
		Package: pkgName,
	}

	for _, p := range g.packages {
		v := `"` + p.Path + `"`
		if p.Alias != "" {
			v = p.Alias + " " + v
		}

		payload.Imports = append(
			payload.Imports,
			v,
		)
	}

	for _, c := range g.components {
		pkgAlias := g.packages[c.PkgID].Alias

		if pkgAlias == "" {
			pkgAlias = path.Base(g.packages[c.PkgID].Path)
		}

		payload.Components = append(
			payload.Components,
			tmplComponent{
				UpperName: toUpper(c.Name),
				LowerName: toLower(c.Name),
				Type:      pkgAlias + "." + c.Name,
			},
		)
	}

	// TODO tody up this - maybe refactor nested tmplCompositions in
	//  tmplCompositions
	for i, c := range g.compositions {
		cnts := make([]tmplComponent, 0, len(c.Components))

		for _, id := range c.Components {
			cnts = append(cnts, payload.Components[id])
		}

		compatibles := make([]tmplComposition, 0, len(g.compositionGraph[i]))

		for _, id := range g.compositionGraph[i] {
			csn := g.compositions[id]

			compatibles = append(
				compatibles,
				tmplComposition{
					UpperName:  toUpper(csn.Name),
					LowerName:  toLower(csn.Name),
					Components: cnts,
				},
			)
		}

		payload.Compositions = append(
			payload.Compositions,
			tmplComposition{
				UpperName:   toUpper(c.Name),
				LowerName:   toLower(c.Name),
				Components:  cnts,
				Compatibles: compatibles,
			},
		)
	}

	err = tmpl.Execute(
		fo,
		payload,
	)
	if err != nil {
		panic(err) // TODO wrap error
	}
}

func toUpper(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

func toLower(s string) string {
	if s == "" {
		return s
	}

	pos := 0

	for i := range s {
		if s[i] < 66 || s[i] > 90 {
			break
		}

		pos = i
	}

	if pos == len(s)-1 {
		return strings.ToLower(s)
	}

	// if first character is last sequential uppercase, then the first char must
	// be made to be lower (Foo -> Foo) so artificially shift pos to account for
	// this
	if pos == 0 {
		pos++
	}

	return strings.ToLower(s[:pos]) + s[pos:]
}
