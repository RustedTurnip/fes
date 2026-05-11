package schema

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path"
	"reflect"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/rustedturnip/fes/set"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

const module = "github.com/rustedturnip/fes"

var version string

func init() {
	setVersion()
}

func setVersion() {
	bi, _ := debug.ReadBuildInfo()

	for i := range bi.Deps {
		if bi.Deps[i].Path != module {
			continue
		}

		version = bi.Deps[i].Version

		return
	}

	if bi.Main.Path == module {
		version = bi.Main.Version

		return
	}

	version = "unknown"
}

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
	pkgID    int
	typeName string
	name     string
}

func (c component) component() component {
	return c
}

type composition struct {
	name       string
	components []int
}

type graph [][]int

// sort returns a topologically ordered list of the graph's nodes (indices).
// sort assumes that the graph is acyclic.
func (g graph) sort() []int {
	resolved := make([]bool, len(g))
	result := make([]int, 0, len(g))

	var visit func(i int)

	visit = func(i int) {
		for j := range g[i] {
			if resolved[g[i][j]] {
				continue
			}

			visit(g[i][j])
		}

		resolved[i] = true
		result = append(result, i)
	}

	for i := range g {
		if resolved[i] {
			continue
		}

		visit(i)
	}

	return result
}

// Schema holds the configured Components and Compositions (provided by
// RegisterComponent and RegisterComposition). A Schema is used to build the
// desired store compatible with said Components and Compositions (via Build).
type Schema struct {
	// destination is the path to the output file as specified by the user.
	destination string

	// packageOverride is the name to override the output package name with.
	// This is provided to the Schema via Config and can will be ignored when
	// not set.
	packageOverride string

	// packages is a registry of unique imports that are used by the provided
	// components.
	packages []pkg

	// components is a list of unique components provided to the Schema by the
	// user. These components are used to construct compositions.
	components []component

	// compositions is a list of unique compositions provided by the user.
	compositions []composition

	// compositionGraph tracks the subtypes of each composition. It can be
	// thought of as a map, where the index is the ID of the composition, and
	// the slice value contains a list of that compositions subtypes.
	compositionGraph graph
}

type Config struct {
	// Output is the desired output file location. Where not provided,
	// "./fes_gen.go" will be used.
	Output string

	// Package is the name of the package that the output will be generated to
	// (not including the path). When left blank, fes will attempt to calculate
	// what the package should be, either from other Go files already in the
	// Output location, or from the Output file path.
	//
	// Package should typically left empty, and only used when a specific
	// package name is desired that doesn't match the values that would be
	// defaulted to.
	Package string
}

func (c Config) applyDefaults() Config {
	if c.Output == "" {
		c.Output = "./fes_gen.go"
	}

	return c
}

// New instantiates a new Schema configured with the desired output location
// provided as dst which should be the path to the desired output file.
func New(cfg Config) *Schema {
	cfg = cfg.applyDefaults()

	return &Schema{
		destination: cfg.Output,
	}
}

// RegisterComponent registers a new Component to the provided Schema for later
// use in the definition of a Composition. The Component is of type T and the
// provided name.
func RegisterComponent[T any](s *Schema, name string) (ComponentID, error) {
	if !isValidIdentifier(name) {
		return 0, fmt.Errorf(
			"invalid component name provided: %s",
			name,
		)
	}

	exists := slices.ContainsFunc(
		s.components,
		func(c component) bool {
			return strings.EqualFold(name, c.name)
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

	pID, err := registerPackage(s, rt.PkgPath())
	if err != nil {
		return 0, fmt.Errorf(
			"failed to register component's package: %w",
			err,
		)
	}

	tp, ts, _ := strings.Cut(rt.String(), ".")
	if ts == "" {
		ts = tp
	}

	c := component{
		pkgID:    pID,
		typeName: ts,
		name:     name,
	}

	s.components = append(s.components, c)

	return len(s.components) - 1, nil
}

// MustRegisterComponent registers a new Component (for later use in the
// definition of a Composition) of type T and the provided name. The Component
// is registered to the provided Schema s.
//
// If an error is encountered, a panic occurs rather than an error being
// returned. See RegisterComponent if this isn't desired.
func MustRegisterComponent[T any](s *Schema, name string) ComponentID {
	id, err := RegisterComponent[T](s, name)
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
// package or return the ID of the matching, already registered import in the
// provided Schema.
//
// registerPackage attempts to visit the package to learn what it's native name
// is (as it may be different to the base of the path). As such, an error may
// occur if the package can't be found (as it uses the destination as the
// "location" context which may be a different project to where the tool is
// being executed which may lead to such an error).
func registerPackage(s *Schema, imp string) (int, error) {
	for i := range s.packages {
		if s.packages[i].Path != imp {
			continue
		}

		return i, nil
	}

	if imp == "" {
		// empty pkg represents stdlib
		s.packages = append(s.packages, pkg{})

		return len(s.packages) - 1, nil
	}

	dst, _ := path.Split(s.destination)

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

	for i := range s.packages {
		if n != s.packages[i].Name {
			continue
		}

		c++

		n = rp[0].Name + strconv.Itoa(c)
	}

	id := len(s.packages)

	s.packages = append(s.packages, pkg{
		Path: imp,
		Name: n,
	})

	return id, nil
}

// RegisterComposition registers to the provided Schema a Composition which
// is a set of Components that make up an "entity type".
//
// The provided components must be unique to each other, and name must be unique
// to the other Components. The name must also be a valid Go identifier.
func RegisterComposition(
	s *Schema,
	name string,
	components ...ComponentID,
) error {
	if !isValidIdentifier(name) {
		return fmt.Errorf(
			"invalid composition name provided: %s",
			name,
		)
	}

	if !set.IsSet(components) {
		return errors.New("duplicate components provided")
	}

	exists := slices.ContainsFunc(
		s.compositions,
		func(c composition) bool {
			return strings.EqualFold(name, c.name)
		},
	)
	if exists {
		return fmt.Errorf(
			"composition with name %s already registered",
			name,
		)
	}

	at := composition{
		name:       name,
		components: components,
	}

	i := len(s.compositions)
	s.compositions = append(s.compositions, at)
	s.compositionGraph = append(s.compositionGraph, nil)

	// subtypes
	for j := range len(s.compositions) - 1 {
		if set.IsSubset(
			s.compositions[i].components,
			s.compositions[j].components,
		) {
			s.compositionGraph[i] = append(s.compositionGraph[i], j)

			continue
		}

		if set.IsSubset(
			s.compositions[j].components,
			s.compositions[i].components,
		) {
			s.compositionGraph[j] = append(s.compositionGraph[j], i)
		}
	}

	return nil
}

// MustRegisterComposition registers to the provided Schema a Composition which
// is a set of Components that make up an "entity type".
//
// The provided Components must be unique to each other, and name must be unique
// to the other Components in a case-insensitive way. The name must also be a
// valid Go identifier.
//
// If an error is encountered, a panic occurs rather than an error being
// returned. See RegisterComposition if this isn't desired.
func MustRegisterComposition(
	s *Schema,
	name string,
	components ...ComponentID,
) {
	err := RegisterComposition(s, name, components...)
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

// Build compiles the provided schema into an in-memory Store compatible with
// the Schema's Components and Compositions at the preconfigured destination.
func Build(s *Schema) error {
	tmpl, err := template.
		New("generator").
		Funcs(template.FuncMap{
			"inc": func(i int) int {
				return i + 1
			},
		}).
		Parse(fesTmpl)
	if err != nil {
		return fmt.Errorf(
			"failed to parse template: %w",
			err,
		)
	}

	data, err := schemaToTmplData(s)
	if err != nil {
		return fmt.Errorf(
			"failed to build from schema: %w",
			err,
		)
	}

	fo, err := os.Create(s.destination)
	if err != nil {
		return fmt.Errorf(
			"failed to open output file: %w",
			err,
		)
	}
	defer func() {
		_ = fo.Close()
	}()

	buf := &bytes.Buffer{}

	err = tmpl.Execute(
		buf,
		data,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to execute template: %w",
			err,
		)
	}

	// imports.Process implicitly runs gofmt, but has better import formatting
	// rules so has been favoured here
	result, err := imports.Process(s.destination, buf.Bytes(), nil)
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
