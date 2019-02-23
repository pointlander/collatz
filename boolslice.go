package main

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"

	"github.com/MaxHalford/eaopt"
)

type BoolSlice []bool

func (s BoolSlice) At(i int) interface{} {
	return s[i]
}

func (s BoolSlice) Set(i int, v interface{}) {
	fmt.Println(v)
	s[i] = v.(bool)
}

func (s BoolSlice) Len() int {
	return len(s)
}

func (s BoolSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s BoolSlice) Slice(a, b int) eaopt.Slice {
	return s[a:b]
}

func (s BoolSlice) Split(k int) (eaopt.Slice, eaopt.Slice) {
	return s[:k], s[k:]
}

func (s BoolSlice) Append(t eaopt.Slice) eaopt.Slice {
	return append(s, t.(BoolSlice)...)
}

func (s BoolSlice) Replace(t eaopt.Slice) {
	copy(s, t.(BoolSlice))
}

func (s BoolSlice) Copy() eaopt.Slice {
	t := make(BoolSlice, len(s))
	copy(t, s)
	return t
}

func (s BoolSlice) String() string {
	series, space := "", ""
	for i, value := range s {
		if value {
			series += fmt.Sprintf("%s%d", space, i+1)
			space = " "
		}
	}
	return series
}

func (s BoolSlice) Evaluate() (float64, error) {
	series := make([]big.Int, 0, len(s))
	for i, value := range s {
		if value {
			number := big.Int{}
			number.SetInt64(int64(i+1))
			series = append(series, number)
		}
	}
	sum, product := sumProductTest(series)
	score := math.Sqrt(sum * sum + product * product)
	return score, nil
}

func (s BoolSlice) Mutate(rng *rand.Rand) {
	eaopt.MutPermute(s, 1, rng)
}

func (s BoolSlice) Crossover(r eaopt.Genome, rng *rand.Rand) {
	eaopt.CrossGNX(s, r.(BoolSlice), 1, rng)
}

func (s BoolSlice) Clone() eaopt.Genome {
    r := make(BoolSlice, len(s))
    copy(r, s)
    return r
}

func BoolSliceFactory(rng *rand.Rand) eaopt.Genome {
	s := make(BoolSlice, 128)
	for i := range s {
		s[i] = rng.Intn(2) == 0
	}
	return s
}