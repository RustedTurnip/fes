package generator

import (
	"bytes"
	"errors"
	"fmt"
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
	"golang.org/x/tools/imports"
)

const version = "v0.1.0-dev"

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

// Generator is used to generate the desired entity storage object. It should be
// instantiated using New.
type Generator struct {
	// destination is the path to the output file as specified by the user.
	destination string

	// packages is a registry of unique imports that are used by the provided
	// components.
	packages []pkg

	// components is a list of unique components provided to the Generator by
	// the user. These components are used to construct compositions.
	components []component

	// compositions is a list of unique compositions provided by the user.
	compositions []composition

	// compositionGraph tracks the subtypes of each composition. It can be
	// thought of as a map, where the index is the ID of the composition, and
	// the slice value contains a list of that compositions subtypes.
	compositionGraph [][]int
}

// New instantiates a new Generator configured with the desired output location
// provided as dst which should be the path to the desired output file.
func New(dst string) *Generator {
	return &Generator{
		destination: dst,
	}
}

// RegisterComponent registers a new component (for later use in the definition
// of a composition) of type T and the provided name. The component is
// registered to the provided Generator g.
func RegisterComponent[T any](g *Generator, name string) (ComponentID, error) {
	exists := slices.ContainsFunc(
		g.components,
		func(c component) bool {
			return strings.EqualFold(name, c.Name)
		},
	)
	if exists {
		return 0, fmt.Errorf(
			"component with the name %s already exists",
			name,
		)
	}

	var t T
	rt := reflect.TypeOf(t)

	pID, err := g.registerPackage(rt.PkgPath())
	if err != nil {
		return 0, fmt.Errorf(
			"failed to register component's package: %w",
			err,
		)
	}

	c := component{
		PkgID: pID,
		Type:  rt.String(),
		Name:  name,
	}

	g.components = append(g.components, c)

	return len(g.components) - 1, nil
}

// MustRegisterComponent registers a new component (for later use in the
// definition of a composition) of type T and the provided name. The component
// is registered to the provided Generator g.
//
// If an error is encountered, a panic occurs rather than an error being
// returned. See RegisterComponent if this isn't desired.
func MustRegisterComponent[T any](g *Generator, name string) ComponentID {
	id, err := RegisterComponent[T](g, name)
	if err != nil {
		panic(
			fmt.Errorf(
				`failed to register component "%s": %w`,
				name,
				err,
			),
		)
	}

	return id
}

// registerPackage will attempt to register the provided import (imp) as new
// package in Generator, or return the ID of the matching, already registered,
// import.
//
// registerPackage attempts to visit the package to learn what it's native name
// is (as it may be different to the base of the path). As such, an error may
// occur if the package can't be found (as it uses the destination as the
// "location" context which may be a different project to where the tool is
// being executed which may lead to such an error).
func (g *Generator) registerPackage(imp string) (int, error) {
	for i := range g.packages {
		if g.packages[i].Path != imp {
			continue
		}

		return i, nil
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
		return 0, fmt.Errorf(
			"failed to load package %s: %w",
			imp,
			err,
		)
	}
	if len(rp) != 1 {
		return 0, fmt.Errorf(
			"unexpected number of packages returned for %s (%d)",
			imp,
			len(rp),
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

	return id, nil
}

// RegisterComposition registers to the provided Generator a Composition which
// is a set of components that make up an "entity type".
//
// The provided components must be unique to each other, and name must be unique
// to the other components in a case-insensitive way. The name must also be a
// valid Go identifier.
func RegisterComposition(
	g *Generator,
	name string,
	components ...ComponentID,
) error {
	if !isValidIdentifier(name) {
		return fmt.Errorf(
			"invalid name provided for composition (must be valid Go "+
				"identifier): %s",
			name,
		)
	}

	if !set.IsSet(components) {
		return errors.New("duplicate components provided")
	}

	exists := slices.ContainsFunc(
		g.compositions,
		func(c composition) bool {
			return strings.EqualFold(name, c.Name)
		},
	)
	if exists {
		return fmt.Errorf(
			"composition with name %s already registered",
			name,
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
			g.compositionGraph[i] = append(g.compositionGraph[i], j)

			continue
		}

		if set.IsSubset(
			g.compositions[j].Components,
			g.compositions[i].Components,
		) {
			g.compositionGraph[j] = append(g.compositionGraph[j], i)
		}
	}

	return nil
}

// RegisterComposition registers to the provided Generator a Composition which
// is a set of components that make up an "entity type".
//
// The provided components must be unique to each other, and name must be unique
// to the other components in a case-insensitive way. The name must also be a
// valid Go identifier.
//
// If an error is encountered, a panic occurs rather than an error being
// returned. See RegisterComposition if this isn't desired.
func MustRegisterComposition(
	g *Generator,
	name string,
	components ...ComponentID,
) {
	err := RegisterComposition(g, name, components...)
	if err != nil {
		panic(
			fmt.Errorf(
				"failed to register composition %s: %w",
				name,
				err,
			),
		)
	}
}

// Build generates the entity store at the preconfigured destination.
func (g *Generator) Build() error {
	tmpl, err := template.
		New("generator").
		Parse(fesTmpl)
	if err != nil {
		return fmt.Errorf(
			"failed to parse template: %w",
			err,
		)
	}

	fo, err := os.Create(g.destination)
	if err != nil {
		return fmt.Errorf(
			"failed to open output file: %w",
			err,
		)
	}
	defer func() {
		_ = fo.Close()
	}()

	dir, _ := path.Split(g.destination)

	dst, err := destinationPackage(dir)
	if err != nil {
		return fmt.Errorf(
			"failed to determine output package: %w",
			err,
		)
	}

	payload := newTemplatePayload{
		Version: version,
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

	for _, c := range g.compositions {
		cnts := make([]tmplComponent, 0, len(c.Components))

		for _, id := range c.Components {
			cnts = append(cnts, payload.Components[id])
		}

		payload.Compositions = append(
			payload.Compositions,
			tmplComposition{
				UpperName:   toUpper(c.Name),
				LowerName:   toLower(c.Name),
				Components:  cnts,
				Compatibles: nil, // built below
			},
		)
	}

	for i := range payload.Compositions {
		for _, id := range g.compositionGraph[i] {
			payload.Compositions[i].Compatibles = append(
				payload.Compositions[i].Compatibles,
				&payload.Compositions[id],
			)
		}
	}

	buf := &bytes.Buffer{}

	err = tmpl.Execute(
		buf,
		payload,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to execute template: %w",
			err,
		)
	}

	// imports.Process implicitly runs gofmt, but has better import formatting
	// rules so has been favoured here
	result, err := imports.Process(g.destination, buf.Bytes(), nil)
	if err != nil {
		return fmt.Errorf(
			"failed to format output: %w",
			err,
		)
	}

	_, err = fo.Write(result)
	if err != nil {
		return fmt.Errorf(
			"failed to write output: %w",
			err,
		)
	}

	return nil
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
