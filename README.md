# `f`ast `e`ntity `s`torage

`fes` is a library that is used to build custom entity storage in Go for 
ECS-based projects. It is intended to be used within a pre-compile script, where
a "schema" that accommodates the project's entity types is defined and built
using `fes`.

`fes` makes use of two concepts:
- **Components**: these are the building blocks of the types of entity you 
  define using `fes`. These can be thought of as the fields an entity may 
  require e.g. `Velocity float32`, where the Component is called `Velocity` 
  and is of type `float32`.
- **Compositions**: these are the types of entity. Each Composition is a set 
  of Components, and it is the Compositions that are queried for from the 
  resulting `Store`.


## Composition Interfacing
As explained, each Composition is a set of Components. If Composition `foo` 
happens to be made up of a subset of the Components that make up Composition 
`bar`, then when `foo` is queried for, both `foo`s and `bar`s are returned.

```go
var (
    compositionFoo = []string{
        "Velocity",
        "Direction"
    }

    compositionBar = []string{
        "Velocity",
        "Direction",
        "Health"
    }
)
```

In the arbitrary example above, `compositionBar` happens to contain all the 
Components that make up `compositionFoo`, therefore `compositionBar` can be 
thought of as an instance of `compositionFoo`, just with some additional fields.

This implicit design allows the systems of your ECS to query for entities 
comprised of the fields that the system is interested in, although it should be
noted that only defined Compositions can be queried for (rather than an ad-hoc
set of Components after the `Store` has been built).

tl;dr - in the above example, `compositionBar` happens to implement 
`compositionFoo`, and as such entities of the `compositionBar` type are returned
when querying for `compositionFoo`.


## Usage

A working example for how `fes` can be introduced into your project can be found
under [`./example`](./example), with that project's `fes` build script located 
under [`./example/entities/build/build.go`](./example/entities/build/build.go).

There are four distinct steps to building the bespoke `Store` in one's build 
script:
1. Create a Schema instance.
2. Register the Components to that Schema.
3. Register the Compositions to that Schema.
4. Build the Schema into the custom `Store`.

Below is that build script, demonstrating how a `Schema` can be built and 
executed to output a bespoke entity `Store` following the four steps 
detailed above.

```go
package main

import (
  "fmt"

  "github.com/rustedturnip/fes/example/entities"
  "github.com/rustedturnip/fes/schema"
)

//go:generate go run .
func main() { 
  // 1. Create a Schema instance with your desired Config.
  s := schema.New(
    schema.Config{
      Output: "../store_gen.go",
    },
  )

  // 2. Register the Components that will be required by the later-defined
  //    Compositions. These are the "fields" of the entity types.
  cPosition := schema.MustRegisterComponent[entities.Vector[float32]](s, "position")
  cDirection := schema.MustRegisterComponent[float64](s, "direction")
  cVelocity := schema.MustRegisterComponent[float64](s, "velocity")
  cIncome := schema.MustRegisterComponent[int](s, "income")
  cGold := schema.MustRegisterComponent[int](s, "gold")

  // 3. Register your desired Compositions. These are sets of Components that
  //    can be later queried from the Store.
  schema.MustRegisterComposition(s, "positionable", cPosition)
  schema.MustRegisterComposition(s, "travellable", cPosition, cDirection, cVelocity)
  schema.MustRegisterComposition(s, "trader", cIncome, cGold)
  schema.MustRegisterComposition(s, "ship", cPosition, cDirection, cVelocity, cIncome, cGold)

  // 4. Build the Schema into the custom Store. The Store will be outputted 
  //    in the Output location provided to the Schema's Config.
  err := schema.Build(s)
  if err != nil {
    panic(
      fmt.Errorf(
        "failed to generate store from schema: %w",
        err,
      ),
    )
  }
}
```

## Store Usage

For detailed documentation, you can consult the outputted `Store`, but at a 
high-level, the resulting  `Store` provides the following methods:

### General
- `Delete`: allows the deletion of an entity by its ID.

### For Each Composition (e.g. Foo)
- `PutFoo`: allows the insertion of a single entity of the desired Composition.
- `FooByID`: query for a specific Composition using its ID.
- `Foos`: query for all entities that match the desired Composition's
  interface. Access to the resulting entities is provided via an `Accessor`.

#### Composition Accessor

Each registered Composition has a Composition "Accessor". This can be thought of
as the query result when querying for all of a type of Composition. Below is 
an example Accessor for the arbitrary `FooBar` composition:

```go
// FooBarAccessor is the Accessor type for the Composition FooBar. It can be
// queried for all entities that fit the FooBar Composition.
type FooBarAccessor struct {
    current   int
    ids       [][]int
    slicesFoo [][]Foo
    slicesBar [][]Bar
}

// Next moves through internal groupings of FooBars. You should keep calling 
// Next until false is returned to ensure you have visited all FooBars.
func (a *FooBarAccessor) Next() bool {
    if a.current < len(a.ids)-1 {
        return false
    }

    a.current++

    return true
}

// FooBars provides access to the current group of FooBars. In the FooBar 
// result, ids, slicesFoo and slicesBar are all the same length, and to refer to
// each component of a single entity, each must be visited with that  entity's
// index.
func (a *FooBarAccessor) FooBars() FooBarResult {
    return FooBarResult{
        IDs:  a.ids[a.current],
        Foos: a.slicesFoo[a.current],
        Bars: a.slicesBar[a.current],
    }
}
```