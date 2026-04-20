package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	"io"
	"os"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/rustedturnip/fes/set"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
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
	Path string
	Name string
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
	destination string

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

func New(d string) *Generator {
	return &Generator{
		destination: d,
	}
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

	pID := g.registerPackage(rt.PkgPath())

	c := component{
		PkgID: pID,
		Type:  rt.String(),
		Name:  name,
	}

	g.components = append(g.components, c)

	return len(g.components) - 1
}

// TODO comment
func (g *Generator) registerPackage(imp string) int {
	for i := range g.packages {
		if g.packages[i].Path != imp {
			continue
		}

		return i
	}

	dst, _ := path.Split(g.destination)

	rp, err := packages.Load(
		&packages.Config{
			Mode: packages.NeedName,
			Dir:  dst,
		},
		imp,
	)
	if err != nil {
		panic(
			fmt.Errorf(
				"failed to load package %s: %w",
				imp,
				err,
			),
		)
	}
	if len(rp) != 1 {
		panic(
			fmt.Errorf(
				"unexpected number of packages returned for %s (%d)",
				imp,
				len(rp),
			),
		)
	}

	n := rp[0].Name
	c := 0

	for i := range g.packages {
		if n != g.packages[i].Name {
			continue
		}

		c++

		n = rp[0].Name + strconv.Itoa(c)
	}

	id := len(g.packages)

	g.packages = append(g.packages, pkg{
		Path: imp,
		Name: n,
	})

	return id
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

func (g *Generator) Build() {
	tmpl, err := template.
		New("generator").
		Parse(fesTmpl)
	if err != nil {
		panic(err) // TODO wrap error
	}

	fo, err := os.Create(g.destination)
	if err != nil {
		panic(err) // TODO wrap error
	}
	defer func() {
		_ = fo.Close()
	}()

	dir, _ := path.Split(g.destination)

	dst, err := destinationPackage(dir)
	if err != nil {
		panic(err) // TODO handle/wrap error?
	}

	payload := newTemplatePayload{
		Package: dst.Name,
	}

	for _, p := range g.packages {
		if dst.Path == p.Path {
			continue
		}

		v := `"` + p.Path + `"`
		if p.Name != path.Base(p.Path) {
			v = p.Name + " " + v
		}

		payload.Imports = append(
			payload.Imports,
			v,
		)
	}

	for _, c := range g.components {
		t := c.Name

		if g.packages[c.PkgID].Path != dst.Path {
			t = g.packages[c.PkgID].Name + "." + c.Name
		}

		payload.Components = append(
			payload.Components,
			tmplComponent{
				UpperName: toUpper(c.Name),
				LowerName: toLower(c.Name),
				Type:      t,
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

	buf := &bytes.Buffer{}

	err = tmpl.Execute(
		buf,
		payload,
	)
	if err != nil {
		panic(err) // TODO wrap error
	}

	result, err := format.Source(buf.Bytes())
	if err != nil {
		panic(
			fmt.Errorf(
				"failed to format output: %w",
				err,
			),
		)
	}

	_, err = fo.Write(result)
	if err != nil {
		panic(
			fmt.Errorf(
				"failed to write output: %w",
				err,
			),
		)
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

func destinationPackage(dir string) (pkg, error) {
	pkgs, err := packages.Load(
		&packages.Config{
			Mode: packages.NeedName | packages.NeedFiles,
			Dir:  dir,
		},
		dir,
	)
	if err != nil {
		return pkg{}, fmt.Errorf(
			"failed to load package info: %w",
			err,
		)
	}

	if len(pkgs) != 1 {
		return pkg{}, fmt.Errorf(
			"unexpected number of packages found (%d)",
			len(pkgs),
		)
	}

	if pkgs[0].PkgPath != "" {
		name := pkgs[0].Name

		if name == "" {
			name = path.Base(pkgs[0].PkgPath)
		}

		return pkg{
			Path: pkgs[0].PkgPath,
			Name: name,
		}, nil
	}

	mod := dir

	for {
		_, err = os.Stat(path.Join(mod, "go.mod"))
		if err == os.ErrNotExist {
			if mod == "" {
				return pkg{}, errors.New(
					"unable to locate package info for destination",
				)
			}

			mod, _ = path.Split(mod)

			continue
		}
		if err != nil {
			return pkg{}, fmt.Errorf(
				"unexpected error when locating go.mod: %s",
				err,
			)
		}

		break
	}

	rel := strings.TrimPrefix(dir, mod)
	mod = path.Join(mod, "go.mod")

	fi, err := os.Open(mod)
	if err != nil {
		return pkg{}, fmt.Errorf(
			"failed to open go.mod to determine package: %w",
			err,
		)
	}

	defer func() {
		_ = fi.Close()
	}()

	b, err := io.ReadAll(fi)
	if err != nil {
		return pkg{}, fmt.Errorf(
			"failed to read go.mod when determining package info: %w",
			err,
		)
	}

	module := modfile.ModulePath(b)
	if module == "" {
		return pkg{}, fmt.Errorf(
			"failed to parse go.mod (%s) when determining package info",
			mod,
		)
	}

	return pkg{
		Path: path.Join(module, rel),
		Name: path.Base(rel),
	}, nil
}
