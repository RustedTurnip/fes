# TODO

- [ ] Tidy up archetype naming in storage_blueprint.go (to differentiate
      between components and archetypes that hold only one component, i.e.
      between the Foo component and the Foo archetype).
- [ ] Create a decision log to track memory/performance-based incentives for
      design.
- [x] Create the shape the schema should take.
- [ ] Later on, add archetype namespacing to allow groupings to improve
      intellisense usability - e.g. Physics archetypes could be aggregated in
      such a group so that you can do: `store.Physics.PutFoo()` instead of
      `store.PutFoo()`.
- [x] Move to a Structure of Arrays (SoA) approach later on (where each
      component of each archetype is in its own array to make the most of the
      CPU cache).
- [ ] Investigate using generic methods when they are available (see
      [here](https://github.com/golang/go/issues/77273))