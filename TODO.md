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
- [ ] Make Build() method return an error rather than rely on panics throughout
      code.
- [ ] Address remaining TODOs.
- [ ] Add README.md.
- [ ] Comment types, functions, and methods that would benefit.
- [ ] Unit tests.

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