package schema

import (
	"bytes"
	"errors"
	"fmt"
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

type ComponentID = int

// Component is the private interface to encourage user component types to be
// explicitly defined (rather than inlined).
type Component interface {
	componentType() reflect.Type
}

type Composition interface {
	isComposition()
}

var (
	rtComponent   = reflect.TypeFor[Component]()
	rtComposition = reflect.TypeFor[Composition]()
)

// ComponentBase implements component and is intended to be embedded anonymously
// within user-defined components to provide compatibility with the component
// interface.
//
// ComponentBase also implements ComponentGroup to allow components to be built
// into compositions.
type ComponentBase[T any] struct{}

func (c ComponentBase[T]) componentType() reflect.Type {
	return reflect.TypeFor[T]()
}

// CompositionBase implements composition and is intended to be embedded
// anonymously within user-defined compositions to provide compatibility with
// the composition interface.
//
// Unlike ComponentBase, types composed of CompositionBase do not implement
// CompositionGroup.
type CompositionBase struct{}

func (c CompositionBase) isComposition() {}

type pkg struct {
	Path string
	Name string
}

type schemaComponent struct {
	pkgID    int
	typeName string
	name     string
}

func (c schemaComponent) component() schemaComponent {
	return c
}

type schemaComposition struct {
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
// RegisterComposition). A Schema is used to build the desired store compatible
// with said Components and Compositions (via Build).
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

	typeComponents   map[reflect.Type]int // TODO new field
	typeCompositions map[reflect.Type]int // TODO new field

	// components is a list of unique components provided to the Schema by the
	// user. These components are used to construct compositions.
	components []schemaComponent

	// compositions is a list of unique compositions provided by the user.
	compositions []schemaComposition

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

// registerPackage will attempt to register the provided import (imp) as new
// package or return the ID of the matching, already registered import in the
// provided Schema.
//
// registerPackage attempts to visit the package to learn what it's native name
// is (as it may be different to the base of the path). As such, an error may
// occur if the package can't be found (as it uses the destination as the
// "location" context which may be a different project to where the tool is
// being executed which may lead to such an error).
func (s *Schema) registerPackage(imp string) (int, error) {
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

// registerComponent registers a new Component to the provided Schema for use in
// the Composition it was provided via. The Component's name is the concrete
// type of the Component implementation, and it's value type is the type
// returned by Component.componentType().
func (s *Schema) registerComponent(c Component) (ComponentID, error) {
	nt := reflect.TypeOf(c)

	id, ok := s.typeComponents[nt]
	if ok {
		return id, nil
	}

	exists := slices.ContainsFunc(
		s.components,
		func(c schemaComponent) bool {
			return strings.EqualFold(nt.Name(), c.name)
		},
	)
	if exists {
		return 0, fmt.Errorf(
			"component with the name %s already exists",
			nt.Name(),
		)
	}

	pID, err := s.registerPackage(c.componentType().PkgPath())
	if err != nil {
		return 0, fmt.Errorf(
			"failed to register component's package: %w",
			err,
		)
	}

	tp, ts, _ := strings.Cut(c.componentType().String(), ".")
	if ts == "" {
		ts = tp
	}

	rc := schemaComponent{
		pkgID:    pID,
		typeName: ts,
		name:     nt.Name(),
	}

	s.components = append(s.components, rc)

	return len(s.components) - 1, nil
}

// registerComposition registers the provided Composition to the Schema. visited
// is a list of the Compositions already processed within the provided
// Composition and is used for cycle detection.
func (s *Schema) registerComposition(
	c Composition,
	visited []reflect.Type,
) (int, error) {
	t := reflect.TypeOf(c)

	for _, v := range visited {
		if t == v {
			return 0, fmt.Errorf(
				"cyclic compositions are not allowed (%s repeats)",
				t.String(),
			)
		}
	}

	// if composition already registered, return its ID
	ec, ok := s.typeCompositions[t]
	if ok {
		return ec, nil
	}

	exists := slices.ContainsFunc(
		s.compositions,
		func(c schemaComposition) bool {
			return c.name == t.Name()
		},
	)
	if exists {
		return 0, fmt.Errorf(
			"composition with name %s already registered",
			t.Name(),
		)
	}

	at := schemaComposition{
		name:       t.Name(),
		components: nil,
	}

	for i := range t.NumField() {
		f := t.Field(i)

		if !f.Anonymous {
			continue
		}

		fi := reflect.New(f.Type).Elem()

		if f.Type.Implements(rtComponent) {
			id, err := s.registerComponent(
				fi.Interface().(Component),
			)
			if err != nil {
				return 0, fmt.Errorf("failed to register component: %w", err)
			}

			at.components = append(at.components, id)

			continue
		}

		if !f.Type.Implements(rtComposition) {
			continue
		}

		subID, err := s.registerComposition(
			fi.Interface().(Composition),
			append(visited, t),
		)
		if err != nil {
			// TODO handle err
		}

		at.components = append(
			at.components,
			s.compositions[subID].components...,
		)
	}

	if !set.IsSet(at.components) {
		// TODO consider documenting which components are duplicated in error below
		return 0, errors.New("composition contains duplicate components")
	}

	i := len(s.compositions)
	s.compositions = append(s.compositions, at)
	s.typeCompositions[t] = i
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

	return i, nil
}

// RegisterComposition registers to the provided Schema a Composition which
// is a set of Components that make up an "entity type".
//
// The composition's name will be set to the concrete type of the provided
// Composition, and its components will be any anonymous field of the provided
// Composition that implements Composition or Component.
//
// Cyclic Compositions are not permitted, and the Components must be unique by
// name (not type).
func (s *Schema) RegisterComposition(c Composition) error {
	_, err := s.registerComposition(c, []reflect.Type{})
	if err != nil {
		return err
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
func (s *Schema) MustRegisterComposition(
	c Composition,
) {
	err := s.RegisterComposition(c)
	if err != nil {
		panic(
			fmt.Errorf(
				"failed to register composition %s: %w",
				reflect.TypeOf(c).String(),
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
