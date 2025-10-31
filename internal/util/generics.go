package util

import (
	"reflect"
)

func Ptr[T any](v T) *T {
	return &v
}

func MergeMapsShallow[K comparable, T any](vals ...map[K]T) map[K]T {
	result := make(map[K]T)
	for _, val := range vals {
		for k, v := range val {
			result[k] = v
		}
	}

	return result
}

func CoalesceZero[T any](vals ...T) T {
	for _, val := range vals {
		if !reflect.ValueOf(val).IsZero() {
			return val
		}
	}

	var zero T
	return zero
}

func CoalesceSlices[T any](vals ...[]T) []T {
	for _, val := range vals {
		if len(val) > 0 {
			return val
		}
	}

	return nil
}

func CoalesceMaps[K comparable, T any](vals ...map[K]T) map[K]T {
	for _, val := range vals {
		if len(val) > 0 {
			return val
		}
	}

	return nil
}
