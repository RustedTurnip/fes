package schema

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

//go:embed store.tmpl
var fesTmpl string

type tmplComponent struct {
	// UpperName is the user-provided name of the component, but with the first
	// character capitalised.
	UpperName string

	// UpperName is the user-provided name of the component, but with the first
	// character lower-case.
	LowerName string

	// Type is the type of the component as a string (prefixed with "<package>."
	// where the type is imported from elsewhere. This is therefore ready to use
	// as-is in the template.
	Type string
}

type tmplComposition struct {
	// UpperName is the user-provided name of the composition, but with the
	// first character capitalised.
	UpperName string

	// UpperName is the user-provided name of the composition, but with the
	// first character lower-case.
	LowerName string

	// Components is a list of all the components that come together to make an
	// instance of the composition.
	Components []tmplComponent

	// TODO comment
	Compatibles []*tmplComposition
}

type tmplData struct {
	Version      string
	Package      string
	Imports      []string
	Components   []tmplComponent
	Compositions []tmplComposition
}

func schemaToTemplData(s *Schema) (tmplData, error) {
	dir, _ := path.Split(s.destination)

	dst, err := destinationPackage(dir)
	if err != nil {
		return tmplData{}, fmt.Errorf(
			"failed to determine output package: %w",
			err,
		)
	}

	components := buildTmplComponents(s, dst)

	return tmplData{
		Version:      version,
		Package:      dst.Name,
		Imports:      buildTmplImports(s, dst),
		Components:   components,
		Compositions: buildTmplCompositions(s, components),
	}, nil
}

func buildTmplImports(s *Schema, dst pkg) []string {
	imports := make([]string, 0, len(s.packages))

	for _, p := range s.packages {
		// below handles primitive types with no import (int etc.)
		if dst.Path == "" {
			continue
		}

		if dst.Path == p.Path {
			continue
		}

		v := `"` + p.Path + `"`
		if p.Name != path.Base(p.Path) {
			v = p.Name + " " + v
		}

		imports = append(
			imports,
			v,
		)
	}

	return imports
}

func buildTmplComponents(s *Schema, dst pkg) []tmplComponent {
	cmps := make([]tmplComponent, 0, len(s.components))

	for _, c := range s.components {
		var t string

		switch s.packages[c.pkgID].Path {
		case "", dst.Path:
			// no package name required before type in this case
			t = c.typeName

		default:
			// package must be specified
			t = s.packages[c.pkgID].Name + "." + c.typeName
		}

		cmps = append(
			cmps,
			tmplComponent{
				UpperName: c.name.upper,
				LowerName: c.name.lower,
				Type:      t,
			},
		)
	}

	return cmps
}

func buildTmplCompositions(s *Schema, cmps []tmplComponent) []tmplComposition {
	cs := make([]tmplComposition, 0, len(s.compositions))

	for _, c := range s.compositions {
		cnts := make([]tmplComponent, 0, len(c.components))

		for _, id := range c.components {
			cnts = append(cnts, cmps[id])
		}

		cs = append(
			cs,
			tmplComposition{
				UpperName:   c.name.upper,
				LowerName:   c.name.lower,
				Components:  cnts,
				Compatibles: nil, // built below
			},
		)
	}

	for i := range cs {
		for _, id := range s.compositionGraph[i] {
			cs[i].Compatibles = append(
				cs[i].Compatibles,
				&cs[id],
			)
		}
	}

	return cs
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
