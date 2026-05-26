# ⚠️ This project has moved to Codeberg

**Development on GitHub has been permanently halted.** This repository is now
archived and serves as a read-only historical record for existing dependencies.

### New Home

`fes` is still being maintained and updated. It can now be found at
https://codeberg.org/rustedturnip/fes.

### Why the move?
My confidence in GitHub has been repeatedly knocked of late; given growing
downtime, the recent
[merge queue issue](https://www.githubstatus.com/incidents/zsg1lk7w13cf) causing
silent reverts, and most recently the
[source code breach](https://github.blog/security/investigating-unauthorized-access-to-githubs-internal-repositories/)
and the risk of future vulnerability discovery that it brings, I have decided to
migrate my projects to [Codeberg](codeberg.org).

Codeberg is a non-profit, privacy-respecting, and open-source platform powered
by Forgejo.
[In their own words](https://github.blog/security/investigating-unauthorized-access-to-githubs-internal-repositories/):

> The platform we choose for hosting says a lot about the values of our
> ecosystem. Codeberg's open-source roots and non-profit status help assure us
> that their interests are in the collaboration. Further, Codeberg's tooling is
> moving toward federated interoperability which will make it even easier to
> work across services in the future.

### What you need to do:

To make use of the latest version of `fes`, update your imports from
```go
import "github.com/rustedturnip/fes"
```

to

```go
  import "codeberg.org/rustedturnip/fes"
```

followed by running

```shell
go mod tidy
```

to remove the now-redundant dependency.

---

# `f`ast `e`ntity `s`torage

`fes` is a library that is used to build custom entity storage in Go for 
ECS-based projects. It is intended to be used within a pre-compile script, where
a "schema" that accommodates the project's entity types is defined and built
using `fes`.

## Why `fes`?

- Gives you compile-time type-safety when handling entity storage.
- Provides the performance of sequential memory access.
- Avoids interfaces and the expensive lookups they incur.
- Although it adds a pre-compile build stage to your project, `fes` provides a
  simple interface and can be easily slotted into a project with `go generate`.

## Concepts

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
  schema.MustRegisterComposition(s, "Positionable", cPosition)
  schema.MustRegisterComposition(s, "Travellable", cPosition, cDirection, cVelocity)
  schema.MustRegisterComposition(s, "Trader", cIncome, cGold)
  schema.MustRegisterComposition(s, "Ship", cPosition, cDirection, cVelocity, cIncome, cGold)

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
high-level, the resulting  `Store` contains a "namespace" (method) for each
Composition. Each "namespace" contains the following interface:

### General
- `Put`: allows the insertion of a single entity of the desired Composition.
- `ByID`: query for a specific Composition using its ID.
- `All`: query for all entities that match the desired Composition's
  interface. Access to the resulting entities is provided via an `Accessor`.
- `Delete`: allows the deletion of an entity by its ID.

#### Composition Accessor

Each registered Composition has an "Accessor". This can be thought of as the 
query result when querying for all of a type of Composition. Below is an example
of how to use an Accessor to visit all entities that implement that Composition:

```go
func (s *PhysicsSystem) Run() {
    // retrieve the MovableAccessor from the Store instance associated with
    // PhysicsSystem
    acc := s.store.Movable().All()
	
    // for each group of Movables
    for {
		// retrieve entity group
		entities := acc.Movables()
		
        // for each entity within group
        for i := range len(entities.IDs) {
			// perform logic on entity
            pos := calculatePosition(
                entities.Position[0],
                entities.Direction[0],
                entities.Velocity[0],
            )
			
            // update entity
			entities.Position[0] = pos
        }

		// if no more groups, cease system execution
	    if !pos.Next() {
            break
        }
    }
}
```