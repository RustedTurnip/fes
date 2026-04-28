# TODO

## Must do (first release)

- [x] Create the shape the schema should take.
- [x] Move to a Structure of Arrays (SoA) approach later on (where each
  component of each archetype is in its own array to make the most of the
  CPU cache).
- [x] Find a way to prevent adding imports for types declared in the same
  package as the output file.
    - [x] Use Pkg.Name rather than path.Base(Pkg.Path) when building packages.
- [x] Add version number.
- [x] Add auto-generated header to output (including fes version).
- [x] Make Build() method return an error rather than rely on panics throughout
  code.
- [x] Address remaining TODOs.
- [x] Comment types, functions, and methods that would benefit.
- [x] General refactor, moving template related stuff into template.go.
- [x] When checking if the name of a Component or Composition is a valid
  identifier, we should check the Upper and Lower versions of the name
  instead of the name itself.
- [x] Add config to Schema constructor so non-breaking options can be added
  later.
- [x] Add README.md.
- [ ] Allow Compositions or Components to be added to RegisterComposition
  for convenience.
- [ ] Add documentation to the output template for users when using the
  resulting store.
- [ ] Consider allowing querying of specifically only that Composition, i.e. 
  when querying for Foos, if requested only Foos are guaranteed to be returned
  (rather than Foos and FooBars).
- [ ] Unit tests.
- [ ] Reevaluate internal Component/Composition naming.

## Should do

- [ ] Have a less performance-centric output option for convenience (using
  structs instead of many return params).
- [ ] Create a decision log to track memory/performance-based incentives for
  design.
- [ ] Later on, add archetype namespacing to allow groupings to improve
  intellisense usability - e.g. Physics archetypes could be aggregated in
  such a group so that you can do: `store.Physics.PutFoo()` instead of
  `store.PutFoo()`.
- [ ] Investigate using generic methods when they are available (see
  [here](https://github.com/golang/go/issues/77273))