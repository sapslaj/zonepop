package utils

// Reversed returns a reversed copy of a slice.
func Reversed[V any](source []V) []V {
	result := make([]V, len(source))
	for i, j := 0, len(source)-1; i < len(source); i, j = i+1, j-1 {
		result[j] = source[i]
	}
	return result
}

// Filter returns a new slice with f() applied to each element to determine
// whether that element should be included in the new slice.
func Filter[V any](f func(V) bool, source []V) []V {
	result := make([]V, 0)
	for i := 0; i < len(source); i++ {
		if f(source[i]) {
			result = append(result, source[i])
		}
	}
	return result
}

// Map calls function f for each item in source slice.
func Map[A any, B any](f func(A) B, source []A) []B {
	result := make([]B, 0)
	for i := 0; i < len(source); i++ {
		result = append(result, f(source[i]))
	}
	return result
}
