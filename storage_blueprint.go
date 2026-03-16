package fes

// types -- this is auto generated
type archetypeID uint16

const (
	archetypeIDFooer archetypeID = iota
	archetypeIDBarer
	archetypeIDFooBarer
)

var (
	deleteFunctions = []func(store *Store, key entityKey){
		// Foos
		func(store *Store, key entityKey) {
			deleted := store.foos[key.index]

			store.foos[key.index] = store.foos[len(store.foos)-1]
			store.foos[len(store.foos)-1] = entityFoo{} // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.foos = store.foos[:len(store.foos)-1]

			store.ids[deleted.id] = nil
			store.ids[store.foos[key.index].id].index = key.index
		},
		// Bars
		func(store *Store, key entityKey) {
			deleted := store.bars[key.index]

			store.bars[key.index] = store.bars[len(store.bars)-1]
			store.bars[len(store.bars)-1] = entityBar{} // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.bars = store.bars[:len(store.bars)-1]

			store.ids[deleted.id] = nil
			store.ids[store.bars[key.index].id].index = key.index
		},
		// FooBars
		func(store *Store, key entityKey) {
			deleted := store.fooBars[key.index]

			store.fooBars[key.index] = store.fooBars[len(store.fooBars)-1]
			store.fooBars[len(store.fooBars)-1] = entityFooBar{} // this line prevents memory leak of any underlying pointer values in Fooer (until next append)
			store.fooBars = store.fooBars[:len(store.fooBars)-1]

			store.ids[deleted.id] = nil
			store.ids[store.fooBars[key.index].id].index = key.index
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
type entityFoo struct {
	id  int
	foo Foo
}

type entityBar struct {
	id  int
	bar Bar
}

type entityFooBar struct {
	id  int
	foo Foo
	bar Bar
}

// storage -- boilerplate
type Store struct {
	ids []*entityKey

	// entity storage
	foos    []entityFoo
	bars    []entityBar
	fooBars []entityFooBar
}

// storage funcs -- boilerplate
func (s *Store) GetFoo(id int) *Foo {
	key := s.ids[id]

	return &s.foos[key.index].foo
}

func (s *Store) GetBar(id int) *Bar {
	key := s.ids[id]

	return &s.bars[key.index].bar
}

func (s *Store) GetFooBar(id int) (*Foo, *Bar) {
	key := s.ids[id]

	return &s.fooBars[key.index].foo, &s.fooBars[key.index].bar
}

func (s *Store) ForEachFoo(fn func(id int, foo *Foo)) {
	for i := range s.foos {
		fn(
			s.foos[i].id,
			&s.foos[i].foo,
		)
	}

	for i := range s.fooBars {
		fn(
			s.fooBars[i].id,
			&s.fooBars[i].foo,
		)
	}
}

func (s *Store) ForEachBar(fn func(id int, bar *Bar)) {
	for i := range s.bars {
		fn(
			s.bars[i].id,
			&s.bars[i].bar,
		)
	}

	for i := range s.fooBars {
		fn(
			s.fooBars[i].id,
			&s.fooBars[i].bar,
		)
	}
}

func (s *Store) ForEachFooBar(fn func(id int, foo *Foo, bar *Bar)) {
	for i := range s.fooBars {
		fn(
			s.fooBars[i].id,
			&s.fooBars[i].foo,
			&s.fooBars[i].bar,
		)
	}
}

func (s *Store) PutFoo(foo Foo) int {
	id := len(s.ids)

	s.foos = append(s.foos, entityFoo{
		id:  id,
		foo: foo,
	})

	s.ids = append(s.ids, &entityKey{
		archetype: archetypeIDFooer,
		index:     len(s.foos),
	})

	return id
}

func (s *Store) PutBar(bar Bar) int {
	id := len(s.ids)

	s.bars = append(s.bars, entityBar{
		id:  id,
		bar: bar,
	})

	s.ids = append(s.ids, &entityKey{
		archetype: archetypeIDBarer,
		index:     len(s.bars),
	})

	return id
}

func (s *Store) PutFooBarer(foo Foo, bar Bar) int {
	id := len(s.ids)

	s.fooBars = append(s.fooBars, entityFooBar{
		id:  id,
		foo: foo,
		bar: bar,
	})

	s.ids = append(s.ids, &entityKey{
		archetype: archetypeIDFooBarer,
		index:     len(s.fooBars),
	})

	return id
}

func Delete(store *Store, id int) {
	key := store.ids[id]

	deleteFunctions[key.archetype](store, *key)
}
