package main

import (
	"fmt"
	"strconv"
)

type Data struct {
	Value int `json:"value"`
}

func Map[From any, To any](s []From, fn func(From) To) []To {
	to := make([]To, len(s))
	for i, v := range s {
		to[i] = fn(v)
	}
	return to
}

func Filter[T any](s []T, fn func(T) bool) []T {
	to := make([]T, 0)
	for _, v := range s {
		ok := fn(v)
		if ok {
			to = append(to, v)
		}
	}
	return to
}

func ForEach[T any](s []T, fn func(*T)) {
	for n, _ := range s {
		fn(&(s[n]))
	}
}

func Reduce[From any, To any](s []From, init To, fn func(From, To) To) To {
	acc := init
	for _, v := range s {
		acc = fn(v, acc)
	}
	return acc
}

func If[T any](yes bool, a T, b T) T {
	if yes {
		return a
	}
	return b
}

func IfNew[T any](yes bool, a, b func() T) T {
	if yes {
		return a()
	}
	return b()
}

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func Values[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))

	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func Zip[T1 any, T2 any](a []T1, b []T2) [][2]any {
	tuples := make([][2]any, 0, len(a))
	for n, v := range a {
		tuples = append(tuples, [2]any{v, b[n]})
	}
	return tuples
}

func Dict[K comparable, V any](a []K, b []V) map[K]V {
	kvs := make(map[K]V)
	for n, k := range a {
		kvs[k] = b[n]
	}
	return kvs

}

func Mean[T float64 | int64](data []T) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, d := range data {
		sum += float64(d)
	}
	return sum / float64(len(data))
}

func Sum[T float64 | int64](data []T) float64 {
	var sum float64
	for _, d := range data {
		sum += float64(d)
	}
	return sum
}

func _main() {
	d1 := []Data{
		Data{Value: 0},
		Data{Value: 1},
		Data{Value: 2},
		Data{Value: 3},
		Data{Value: 4},
	}

	d3 := Map(d1, func(from Data) string {
		return strconv.Itoa(from.Value*2) + "#"
	})
	fmt.Println(d3)
	d2 := Reduce(d1, 100, func(v Data, a int) int {
		return a + v.Value
	})
	fmt.Println("Reduce", d2)

	d4 := Filter(d1, func(v Data) bool {
		return true
	})
	fmt.Println(d4)

	ForEach(d1, func(v *Data) {
		(*v).Value += 200
	})
	fmt.Println(d1)

	m := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}
	keys := Keys(m)
	fmt.Println(keys)

	values := Values(m)
	fmt.Println(values)

	a := []int{1, 2, 3, 4}
	b := []string{"a", "b", "c", "d"}
	t := Zip(a, b)
	fmt.Println(t)
	for _, w := range t {
		fmt.Println(w[0], w[1])
	}
	ds := Dict(a, b)
	fmt.Println(ds)
}

func splitArray[T1 any](arr []T1, n int) [][]T1 {
	sliceSize := (len(arr) + n - 1) / n // Calculate the size of each slice
	result := make([][]T1, 0, n)

	for i := 0; i < len(arr); i += sliceSize {
		end := i + sliceSize

		if end > len(arr) {
			end = len(arr)
		}
		result = append(result, arr[i:end])
	}

	return result
}

func splitArrayBySliceSize[T1 any](arr []T1, sliceSize int) [][]T1 {
	var result [][]T1

	for i := 0; i < len(arr); i += sliceSize {
		end := i + sliceSize

		if end > len(arr) {
			end = len(arr)
		}

		result = append(result, arr[i:end])
	}

	return result
}
