package fes

// types -- this is auto generated
type archetypeID uint16

const (
	archetypeIDFooer archetypeID = iota
	archetypeIDBarer
	archetypeIDFooBarer
)

var (
	zeroFoo Foo
	zeroBar Bar
)

var (
	deleteFunctions = []func(store *Store, key entityKey){
		// Foos
		func(store *Store, key entityKey) {
			deleted := store.foos.ids[key.index]

			store.foos.ids[key.index] = store.foos.ids[len(store.foos.ids)-1]
			store.foos.ids = store.foos.ids[:len(store.foos.ids)-1]

			store.foos.foos[key.index] = store.foos.foos[len(store.foos.foos)-1]
			store.foos.foos[len(store.foos.foos)-1] = zeroFoo // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.foos.foos = store.foos.foos[:len(store.foos.foos)-1]

			store.ids[deleted] = nil
			store.ids[store.foos.ids[key.index]].index = key.index
		},
		// Bars
		func(store *Store, key entityKey) {
			deleted := store.bars.ids[key.index]

			store.bars.ids[key.index] = store.bars.ids[len(store.bars.ids)-1]
			store.bars.ids = store.bars.ids[:len(store.bars.ids)-1]

			store.bars.bars[key.index] = store.bars.bars[len(store.bars.bars)-1]
			store.bars.bars[len(store.bars.bars)-1] = zeroBar // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.bars.bars = store.bars.bars[:len(store.bars.bars)-1]

			store.ids[deleted] = nil
			store.ids[store.bars.ids[key.index]].index = key.index
		},
		// FooBars
		func(store *Store, key entityKey) {
			deleted := store.fooBars.ids[key.index]

			store.fooBars.ids[key.index] = store.fooBars.ids[len(store.fooBars.ids)-1]
			store.fooBars.ids = store.fooBars.ids[:len(store.fooBars.ids)-1]

			store.fooBars.foos[key.index] = store.fooBars.foos[len(store.fooBars.foos)-1]
			store.fooBars.foos[len(store.fooBars.foos)-1] = zeroFoo // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.fooBars.foos = store.fooBars.foos[:len(store.fooBars.foos)-1]

			store.fooBars.bars[key.index] = store.fooBars.bars[len(store.fooBars.bars)-1]
			store.fooBars.bars[len(store.fooBars.bars)-1] = zeroBar // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.fooBars.bars = store.fooBars.bars[:len(store.fooBars.bars)-1]

			store.ids[deleted] = nil
			store.ids[store.fooBars.ids[key.index]].index = key.index
		},
	}
)

type entityKey struct {
	archetype archetypeID
	index     int
}

// components -- these are referred to from schema
type Foo int
type Bar string

// archetypes -- these are auto generated
type storeFoo struct {
	ids  []int
	foos []Foo
}

type storeBar struct {
	ids  []int
	bars []Bar
}

type storeFooBar struct {
	ids  []int
	foos []Foo
	bars []Bar
}

// storage -- boilerplate
type Store struct {
	ids []*entityKey

	// entity storage
	foos    storeFoo
	bars    storeBar
	fooBars storeFooBar
}

// storage funcs -- boilerplate
type FooAccessor struct {
	current   int
	ids       [][]int
	slicesFoo [][]Foo
}

func (a *FooAccessor) Next() bool {
	if a.current < len(a.ids)-1 {
		return false
	}

	a.current++

	return true
}

type FooResult struct {
	IDs  []int
	Foos []Foo
}

func (a *FooAccessor) Foos() FooResult {
	return FooResult{
		IDs:  a.ids[a.current],
		Foos: a.slicesFoo[a.current],
	}
}

type BarResult struct {
	IDs  []int
	Bars []Bar
}

type BarAccessor struct {
	current   int
	ids       [][]int
	slicesBar [][]Bar
}

func (a *BarAccessor) Next() bool {
	if a.current < len(a.ids)-1 {
		return false
	}

	a.current++

	return true
}

func (a *BarAccessor) Bars() BarResult {
	return BarResult{
		IDs:  a.ids[a.current],
		Bars: a.slicesBar[a.current],
	}
}

type FooBarResult struct {
	IDs  []int
	Foos []Foo
	Bars []Bar
}

type FooBarAccessor struct {
	current   int
	ids       [][]int
	slicesFoo [][]Foo
	slicesBar [][]Bar
}

func (a *FooBarAccessor) Next() bool {
	if a.current < len(a.ids)-1 {
		return false
	}

	a.current++

	return true
}

func (a *FooBarAccessor) FooBars() FooBarResult {
	return FooBarResult{
		IDs:  a.ids[a.current],
		Foos: a.slicesFoo[a.current],
		Bars: a.slicesBar[a.current],
	}
}

func (s *Store) FooByID(id int) *Foo {
	index := s.ids[id].index

	return &s.foos.foos[index]
}

func (s *Store) Foos() *FooAccessor {
	return &FooAccessor{
		current: 0,
		ids: [][]int{
			s.foos.ids,
			s.fooBars.ids,
		},
		slicesFoo: [][]Foo{
			s.foos.foos,
			s.fooBars.foos,
		},
	}
}

func (s *Store) BarByID(id int) *Bar {
	index := s.ids[id].index

	return &s.bars.bars[index]
}

func (s *Store) Bars() *BarAccessor {
	return &BarAccessor{
		current: 0,
		ids: [][]int{
			s.bars.ids,
			s.fooBars.ids,
		},
		slicesBar: [][]Bar{
			s.bars.bars,
			s.bars.bars,
		},
	}
}

func (s *Store) FooBarByID(id int) (*Foo, *Bar) {
	index := s.ids[id].index

	return &s.fooBars.foos[index], &s.fooBars.bars[index]
}

func (s *Store) FooBars() *FooBarAccessor {
	return &FooBarAccessor{
		current: 0,
		ids: [][]int{
			s.fooBars.ids,
		},
		slicesFoo: [][]Foo{
			s.fooBars.foos,
		},
		slicesBar: [][]Bar{
			s.fooBars.bars,
		},
	}
}

func (s *Store) PutFoo(foo Foo) int {
	id := len(s.ids)

	s.ids = append(s.ids, &entityKey{
		archetype: archetypeIDFooer,
		index:     len(s.foos.ids),
	})

	s.foos.ids = append(s.foos.ids, id)
	s.foos.foos = append(s.foos.foos, foo)

	return id
}

func (s *Store) PutBar(bar Bar) int {
	id := len(s.ids)

	s.ids = append(s.ids, &entityKey{
		archetype: archetypeIDBarer,
		index:     len(s.bars.ids),
	})

	s.bars.ids = append(s.bars.ids, id)
	s.bars.bars = append(s.bars.bars, bar)

	return id
}

func (s *Store) PutFooBar(foo Foo, bar Bar) int {
	id := len(s.ids)

	s.ids = append(s.ids, &entityKey{
		archetype: archetypeIDFooBarer,
		index:     len(s.fooBars.ids),
	})

	s.fooBars.ids = append(s.fooBars.ids, id)
	s.fooBars.foos = append(s.fooBars.foos, foo)
	s.fooBars.bars = append(s.fooBars.bars, bar)

	return id
}

func Delete(store *Store, id int) {
	key := store.ids[id]

	deleteFunctions[key.archetype](store, *key)
}
