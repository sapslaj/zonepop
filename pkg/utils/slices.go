package utils

// Reversed returns a reversed copy of a slice.
func Reversed[V any](source []V) []V {
	result := make([]V, len(source))
	for i, j := 0, len(source)-1; i < len(source); i, j = i+1, j-1 {
		result[j] = source[i]
	}
	return result
}

// FilterErr returns a new slice with f() applied to each element to determine
// whether that element should be included in the new slice. Unlike Filter, the
// function should return (bool, error) and if an error is returned, FilterErr
// immediately bubbles the error up instead of continuing with the loop.
func FilterErr[V any](f func(V) (bool, error), source []V) ([]V, error) {
	result := make([]V, 0)
	for i := 0; i < len(source); i++ {
		pass, err := f(source[i])
		if err != nil {
			return result, err
		}
		if pass {
			result = append(result, source[i])
		}
	}
	return result, nil
}

// Filter returns a new slice with f() applied to each element to determine
// whether that element should be included in the new slice. Unlike FilterErr,
// the function should only return a bool depending on whether it should be
// filtered through or not.
func Filter[V any](f func(V) bool, source []V) []V {
	result := make([]V, 0)
	for i := 0; i < len(source); i++ {
		if f(source[i]) {
			result = append(result, source[i])
		}
	}
	return result
}

// MapErr calls function f for each item in source slice. Unlike Map, the
// function should return (B, error) and if an error is returned, MapErr
// immatately bubbles the error up instead of continuing the loop.
func MapErr[A any, B any](f func(A) (B, error), source []A) ([]B, error) {
	result := make([]B, 0)
	for i := 0; i < len(source); i++ {
		mapped, err := f(source[i])
		if err != nil {
			return result, err
		}
		result = append(result, mapped)
	}
	return result, nil
}

// Map calls function f for each item in source slice. Unlike MapErr, the
// function should only return the B type.
func Map[A any, B any](f func(A) B, source []A) []B {
	result := make([]B, 0)
	for i := 0; i < len(source); i++ {
		result = append(result, f(source[i]))
	}
	return result
}

// All takes a []bool and returns true if all of the values are true, else
// false.
func All(a []bool) bool {
	for _, v := range a {
		if !v {
			return false
		}
	}
	return true
}

// Any takes a []bool and returns true if any of the values are true. If none
// are true, returns false.
func Any(a []bool) bool {
	for _, v := range a {
		if v {
			return true
		}
	}
	return false
}
