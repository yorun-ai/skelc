package sliceutil

import "slices"

// Filter returns the elements for which predicate returns true.
func Filter[E any](items []E, predicate func(E) bool) []E {
	result := make([]E, 0)
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Find returns the first element for which predicate returns true.
func Find[E any](items []E, predicate func(E) bool) (E, bool) {
	index := slices.IndexFunc(items, predicate)
	if index >= 0 {
		return items[index], true
	}
	return *new(E), false
}

// ForEach invokes action for every element in order.
func ForEach[E any](items []E, action func(E)) {
	for _, item := range items {
		action(item)
	}
}

// Map transforms every element while preserving order.
func Map[E, R any](items []E, transform func(E) R) []R {
	result := make([]R, len(items))
	for index, item := range items {
		result[index] = transform(item)
	}
	return result
}

// MapToMap transforms every element into a key-value pair.
// Later values replace earlier values with the same key.
func MapToMap[E any, K comparable, V any](items []E, transform func(E) (K, V)) map[K]V {
	result := make(map[K]V, len(items))
	for _, item := range items {
		key, value := transform(item)
		result[key] = value
	}
	return result
}
