package generator

import (
	_ "embed"
)

//go:embed generator.tmpl
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
	Compatibles []tmplComposition
}

type newTemplatePayload struct {
	Version      string
	Package      string
	Imports      []string
	Components   []tmplComponent
	Compositions []tmplComposition
}
