package set

func IsSet[T comparable](s []T) bool {
	m := map[T]any{}

	for _, e := range s {
		m[e] = struct{}{}
	}

	return len(m) == len(s)
}

// IsSubset returns true if the values of a each exist within b.
//
// It is expected that a and b are provided as sets with no duplicate values.
func IsSubset[T comparable](a, b []T) bool {
	if len(a) > len(b) {
		return false
	}

	m := map[T]any{}

	for _, e := range b {
		m[e] = struct{}{}
	}

	for _, e := range a {
		_, ok := m[e]
		if !ok {
			return false
		}
	}

	return true
}

// AreEqual compares a and b to see if they are equal in terms of containing the
// same set of values (the order of values is not compared).
//
// It is expected that a and b are provided as sets with no duplicate values.
func AreEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	m := map[T]any{}

	for _, e := range b {
		m[e] = struct{}{}
	}

	for _, e := range a {
		_, ok := m[e]
		if !ok {
			return false
		}
	}

	return true
}
