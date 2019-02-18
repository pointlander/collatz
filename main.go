// Copyright 2019 The Collatz Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	oeis       = flag.Bool("oeis", false, "search through oeis")
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

func fetch(url, name string) {
	head, err := http.Head(url)
	if err != nil {
		panic(err)
	}
	size, err := strconv.Atoi(head.Header.Get("Content-Length"))
	if err != nil {
		panic(err)
	}
	last, err := http.ParseTime(head.Header.Get("Last-Modified"))
	if err != nil {
		panic(err)
	}
	head.Body.Close()
	stat, err := os.Stat("./" + name)
	if err != nil || stat.ModTime().Before(last) || stat.Size() != int64(size) {
		fmt.Println("downloading", url, "->", name, size, last)
		response, err := http.Get(url)
		if err != nil {
			panic(err)
		}

		out, err := os.Create("./" + name)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(out, response.Body)
		if err != nil {
			panic(err)
		}
		response.Body.Close()
		out.Close()

		err = os.Chtimes("./"+name, last, last)
		if err != nil {
			panic(err)
		}
		fmt.Println("done downloading", url, "->", name, size, last)

		return
	}

	fmt.Println("skipping", url, "->", name, size, last)
}

func oeisSearch() {
	fetch("https://oeis.org/stripped.gz", "stripped.gz")

	type Series struct {
		Name                string
		Series              []string
		Score, Sum, Product float64
	}
	var sorted [256]Series
	for i := range sorted {
		sorted[i].Score = math.Sqrt2
	}
	add := func(series Series) {
		for i, a := range sorted {
			if series.Score < a.Score {
				sorted[i] = series
				for j, b := range sorted[i+1:] {
					a, sorted[j+i+1] = b, a
				}
				break
			}
		}
	}

	size := runtime.NumCPU() * 2
	results := make(chan Series, size)
	test := func(series Series) {
		unique := make(map[string]bool, len(series.Series))
		for _, number := range series.Series {
			unique[number] = true
		}

		integers, i := make([]big.Int, len(unique)), 0
		for number := range unique {
			_, ok := integers[i].SetString(number, 10)
			if !ok {
				panic("invalid number: " + number)
			}
			i++
		}
		sumScore, productScore := sumProductTest(integers)
		series.Score = math.Sqrt(sumScore*sumScore + productScore*productScore)
		series.Sum = sumScore
		series.Product = productScore
		results <- series
	}

	file, err := os.Open("./stripped.gz")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	decoder, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()
	reader := bufio.NewReader(decoder)
	getSeries := func() (series Series, err error) {
		s, err := reader.ReadString('\n')
		if err != nil || strings.HasPrefix(s, "#") {
			return
		}
		s = strings.TrimRight(s, "\n")
		line := strings.Split(s, " ")
		if len(line) != 2 {
			panic("invalid file format")
		}
		series.Name = line[0]
		csv := strings.TrimRight(strings.TrimLeft(line[1], ","), ",")
		series.Series = strings.Split(csv, ",")
		return
	}

	i, series := 0, Series{}
	for i < size {
		series, err = getSeries()
		if series.Name == "" {
			continue
		}
		if err != nil {
			break
		}
		go test(series)
		i++
	}

	for err == nil {
		series = <-results
		i--
		add(series)
		series, err = getSeries()
		if err == nil {
			i++
			go test(series)
		}
	}

	for j := 0; j < i; j++ {
		series = <-results
		add(series)
	}

	out, err := os.Create("README.md")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	fmt.Fprintf(out, "| Name | Score | Sum | Product | Numbers |\n")
	fmt.Fprintf(out, "| ---- | ----- | --- | ------- | ------- |\n")
	for _, series := range sorted {
		fmt.Fprintf(out, "| [%s](https://oeis.org/%s) | %f | %f | %f | %v |\n",
			series.Name, series.Name, series.Score, series.Sum, series.Product, series.Series)
	}
}

func sumProductTest(series []big.Int) (float64, float64) {
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
	sumScore, productScore := float64(len(sums))/float64(max), float64(len(products))/float64(max)
	if !*oeis {
		fmt.Println(max, sumScore, productScore)
	}
	return sumScore, productScore
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
	if *oeis {
		oeisSearch()
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
