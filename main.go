// Copyright 2019 The Collatz Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"sort"
)

var (
	one   = big.NewInt(1)
	two   = big.NewInt(2)
	three = big.NewInt(3)
	a     = &big.Int{}
	b     = &big.Int{}
)

var (
	number     = flag.String("number", "13", "starting number")
	brute      = flag.Bool("brute", false, "try a bunch of numbers")
	aa         = flag.String("a", "2", "number series parameter")
	bb         = flag.String("b", "3", "number series parameter")
	arithmetic = flag.Bool("arithmetic", false, "use arithmetic integers for series")
	geometric  = flag.Bool("geometric", false, "use geometric integers for series")
	atomic     = flag.Bool("atomic", false, "use atomic neutron counts for series")
)

func collatz(i *big.Int) []big.Int {
	series := make([]big.Int, 0, 256)
	cp := func() (z big.Int) {
		z.Set(i)
		return z
	}
	series = append(series, cp())
	for one.Cmp(i) != 0 {
		if i.Bit(0) == 0 {
			i.Rsh(i, 1)
		} else {
			z := cp()
			i.Lsh(i, 1).SetBit(i, 0, 1).Add(i, &z)
		}
		series = append(series, cp())
	}

	return series
}

func arithmeticSeries() []big.Int {
	series := make([]big.Int, 256)
	for i := range series {
		x := &series[i]
		x.SetInt64(int64(i)).Mul(b, x).Add(a, x)
	}
	return series
}

func geometricSeries() []big.Int {
	series := make([]big.Int, 256)
	for i := range series {
		x := &series[i]
		x.SetInt64(int64(i)).Exp(b, x, nil).Mul(a, x)
	}
	return series
}

type Element struct {
	Name         string  `json:"name"`
	Appearance   string  `json:"appearance"`
	AtomicMass   float64 `json:"atomic_mass"`
	Boil         float64 `json:"boil"`
	Category     string  `json:"category"`
	Color        string  `json:"color"`
	Density      float64 `json:"density"`
	DiscoveredBy string  `json:"discovered_by"`
	Melt         float64 `json:"melt"`
	MolarHeat    float64 `json:"molar_heat"`
	NamedBy      string  `json:"named_by"`
	Number       int     `json:"number"`
	Period       int     `json:"period"`
	Phase        string  `json:"phase"`
	Source       string  `json:"source"`
	SpectralImg  string  `json:"spectral_img"`
	Summary      string  `json:"summary"`
	Symbol       string  `json:"symbol"`
	XPos         int     `json:"xpos"`
	YPos         int     `json:"ypos"`
	Shells       []int   `json:"shells"`
}

type Elements struct {
	Elements []Element `json:"elements"`
}

func atomicSeries() []big.Int {
	series, elements, neutrons := make([]big.Int, 0, 256), Elements{}, make(map[int]bool)
	data, err := ioutil.ReadFile("./PeriodicTableJSON.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &elements)
	if err != nil {
		panic(err)
	}
	for _, element := range elements.Elements {
		n := int(math.Round(element.AtomicMass)) - element.Number
		neutrons[n] = true
	}
	sorted := make([]int, 0, len(neutrons))
	for n := range neutrons {
		sorted = append(sorted, n)
	}
	sort.Ints(sorted)
	for _, n := range sorted {
		series = append(series, *big.NewInt(int64(n)))
	}
	return series
}

func sumProductTest(series []big.Int) {
	length := len(series)
	sums, products := make(map[string]int, length*length), make(map[string]int, length*length)
	for _, x := range series {
		for _, y := range series {
			sum, product := big.Int{}, big.Int{}
			sum.Add(&x, &y)
			sums[sum.Text(2)]++
			product.Mul(&x, &y)
			products[product.Text(2)]++
		}
	}
	max := (length * (length + 1)) / 2
	fmt.Println(max, float64(len(sums))/float64(max), float64(len(products))/float64(max))
}

func main() {
	flag.Parse()

	_, ok := a.SetString(*aa, 10)
	if !ok {
		panic("invalid string for parameter a")
	}
	_, ok = b.SetString(*bb, 10)
	if !ok {
		panic("invalid string for parameter b")
	}

	if *brute {
		for i := 1; i < 1024; i++ {
			series := collatz(big.NewInt(int64(i)))
			sumProductTest(series)
		}
		return
	}

	if *arithmetic {
		series := arithmeticSeries()
		for _, item := range series {
			fmt.Println(&item)
		}
		sumProductTest(series)
		return
	}
	if *geometric {
		series := geometricSeries()
		for _, item := range series {
			fmt.Println(&item)
		}
		sumProductTest(series)
		return
	}
	if *atomic {
		series := atomicSeries()
		for _, item := range series {
			fmt.Println(&item)
		}
		sumProductTest(series)
		return
	}

	i := big.Int{}
	_, ok = i.SetString(*number, 10)
	if !ok {
		panic("invalid number")
	}
	series := collatz(&i)
	for _, item := range series {
		fmt.Println(&item)
	}
	sumProductTest(series)
}
