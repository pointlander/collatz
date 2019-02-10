// Copyright 2019 The Collatz Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"math/big"
)

var (
	one = big.NewInt(1)
	three = big.NewInt(3)
)

var (
	number = flag.String("number", "13", "starting number")
	brute = flag.Bool("brute", false, "try a bunch of numbers")
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
	max := (length * (length + 1))/2
	fmt.Println(max, float64(len(sums)) / float64(max), float64(len(products)) / float64(max))	
}

func main() {
	flag.Parse()

	if *brute {
		for i := 1; i < 256; i++ {
			series := collatz(big.NewInt(int64(i)))
			sumProductTest(series)
		}
		return
	}

	i := big.Int{}
	_, ok := i.SetString(*number, 10)
	if !ok {
		panic("invalid number")
	}
	series := collatz(&i)
	for _, item := range series {
		fmt.Println(&item)
	}
	sumProductTest(series)
}