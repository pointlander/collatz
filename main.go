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
	number = flag.Int64("number", 13, "starting number")
	brute = flag.Bool("brute", false, "try a bunch of numbers")
)

func collatz(number int64) {
	i, series, z := big.NewInt(number), make([]big.Int, 0, 256), big.Int{}
	z.Set(i)
	series = append(series, z)
	fmt.Println(i)
	for one.Cmp(i) != 0 {
		if i.Bit(0) == 0 {
			i.Rsh(i, 1)
		} else {
			i.Mul(three, i).Add(one, i)
		}
		z = big.Int{}
		z.Set(i)
		series = append(series, z)
		if !*brute {
			fmt.Println(i)
		}
	}
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
			collatz(int64(i))
		}
		return
	}

	collatz(*number)
}