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
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/MaxHalford/eaopt"
	"github.com/VividCortex/gohistogram"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

var (
	zero  = big.NewInt(0)
	one   = big.NewInt(1)
	two   = big.NewInt(2)
	three = big.NewInt(3)
	fOne  = big.NewFloat(1)
	fTwo  = big.NewFloat(2)
	fFive = big.NewFloat(5)
	a     = &big.Int{}
	b     = &big.Int{}
)

var (
	number      = flag.String("number", "13", "starting number")
	brute       = flag.Bool("brute", false, "try a bunch of numbers")
	aa          = flag.String("a", "2", "number series parameter")
	bb          = flag.String("b", "3", "number series parameter")
	arithmetic  = flag.Bool("arithmetic", false, "use arithmetic integers for series")
	geometric   = flag.Bool("geometric", false, "use geometric integers for series")
	atomic      = flag.Bool("atomic", false, "use atomic neutron counts for series")
	random      = flag.Bool("random", false, "use random numbers for series")
	seven       = flag.Bool("seven", false, "use seven smooth series")
	sevenComp   = flag.Bool("sevenComp", false, "use seven smooth complement series")
	oeis        = flag.Bool("oeis", false, "search through oeis")
	fibonacci   = flag.Bool("fibonacci", false, "fibonacci search")
	printPrimes = flag.Uint64("primes", 0, "print the prime number out")
	search      = flag.Bool("search", false, "search for series")
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

func randomSeries() []big.Int {
	series, rnd, dupe := make([]big.Int, 256), rand.New(rand.NewSource(1)), make(map[uint64]bool)
	for i := range series {
		number := rnd.Uint64()
		for dupe[number] {
			number = rnd.Uint64()
		}
		dupe[number] = true
		series[i].SetUint64(number)
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
			if series.Score < a.Score || (series.Score == a.Score && strings.Compare(series.Name, a.Name) < 0) {
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
	fmt.Fprintf(out, "Score for seven smooth series, A002473, of different sizes:\n")
	fmt.Fprintf(out, "![seven smooth scores](sevenSmooth.png?raw=true)\n\n")
	fmt.Fprintf(out, "| Name | Score | Sum | Product | Numbers |\n")
	fmt.Fprintf(out, "| ---- | ----- | --- | ------- | ------- |\n")
	for _, series := range sorted {
		fmt.Fprintf(out, "| [%s](https://oeis.org/%s) | %f | %f | %f | %v |\n",
			series.Name, series.Name, series.Score, series.Sum, series.Product, series.Series)
	}
}

var primes = [...]int{2, 3, 5, 7}

func sevenSmoothSeries(size int) []big.Int {
	series := make([]big.Int, 0, size)
	isSmooth := func(number int) bool {
		for _, p := range primes {
			for number%p == 0 {
				number /= p
			}
		}
		return number == 1
	}
	i := 1
	for len(series) < size {
		if isSmooth(i) {
			smooth := big.Int{}
			smooth.SetInt64(int64(i))
			series = append(series, smooth)
		}
		i++
	}
	return series
}

func sevenSmoothComplementSeries(size int) []big.Int {
	series := make([]big.Int, 0, size)
	isSmooth := func(number int) bool {
		for _, p := range primes {
			for number%p == 0 {
				number /= p
			}
		}
		return number != 1
	}
	i := 1
	for len(series) < size {
		if isSmooth(i) {
			smooth := big.Int{}
			smooth.SetInt64(int64(i))
			series = append(series, smooth)
		}
		i++
	}
	return series
}

type Source struct {
	Generate  func(size int) []big.Int
	Key, Nice string
}

var Registry = map[string]Source{
	"sevenSmooth": {
		Generate: sevenSmoothSeries,
		Key:      "sevenSmooth",
		Nice:     "seven smooth",
	},
	"sevenSmoothComplement": {
		Generate: sevenSmoothComplementSeries,
		Key:      "sevenSmoothComplement",
		Nice:     "seven smooth complement",
	},
}

func (s Source) graph(max int) {
	type Result struct {
		Score, Sum, Product float64
		Size                int
	}
	cores := runtime.NumCPU() * 2
	results := make(chan Result, cores)
	sample := func(size int) {
		series := s.Generate(size)
		sum, product := sumProductTest(series)
		results <- Result{
			Score:   math.Sqrt(sum*sum + product*product),
			Sum:     sum,
			Product: product,
			Size:    size,
		}
	}

	points, minSize, minScore := make(plotter.XYs, 0, max), 0, math.Sqrt2
	i, j := 1, 0
	for j < cores && i < max {
		go sample(i)
		j++
		i++
	}

	data := make([]Result, 0, max)
	for i < max {
		result := <-results
		j--
		if result.Score < minScore {
			minSize, minScore = result.Size, result.Score
		}
		points = append(points, plotter.XY{X: float64(result.Size), Y: result.Score})
		fmt.Println(result.Size, result.Sum, result.Product, result.Score)
		data = append(data, result)
		go sample(i)
		j++
		i++
	}

	for j > 0 {
		result := <-results
		j--
		if result.Score < minScore {
			minSize, minScore = result.Size, result.Score
		}
		points = append(points, plotter.XY{X: float64(result.Size), Y: result.Score})
		fmt.Println(result.Size, result.Sum, result.Product, result.Score)
		data = append(data, result)
	}
	fmt.Println(minSize, minScore)

	sort.Slice(data, func(i, j int) bool {
		return data[i].Size < data[j].Size
	})
	out, err := os.Create(fmt.Sprintf("%s.csv.gz", s.Key))
	if err != nil {
		panic(err)
	}
	defer out.Close()
	csv, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	defer csv.Close()
	fmt.Fprintf(csv, "size, sum, product, score\n")
	for _, item := range data {
		fmt.Fprintf(csv, "%d, %g, %g, %g\n", item.Size, item.Sum, item.Product, item.Score)
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = fmt.Sprintf("score vs size for %s numbers", s.Nice)
	p.X.Label.Text = "size"
	p.Y.Label.Text = "score"

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		panic(err)
	}
	scatter.GlyphStyle.Radius = vg.Length(1)
	scatter.GlyphStyle.Shape = draw.CircleGlyph{}
	p.Add(scatter)

	err = p.Save(8*vg.Inch, 8*vg.Inch, fmt.Sprintf("%s.png", s.Key))
	if err != nil {
		panic(err)
	}
}

func searchSeries() {
	ga, err := eaopt.NewDefaultGAConfig().NewGA()
	if err != nil {
		panic(err)
	}

	ga.NGenerations = 100
	ga.RNG = rand.New(rand.NewSource(1))
	ga.ParallelEval = true
	ga.PopSize = 100

	ga.Callback = func(ga *eaopt.GA) {
		fmt.Printf("Best fitness at generation %d: %f\n", ga.Generations, ga.HallOfFame[0].Fitness)
		fmt.Println(ga.HallOfFame[0].Genome.(BoolSlice).String())
	}

	err = ga.Minimize(BoolSliceFactory)
	if err != nil {
		panic(err)
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
	if !*oeis && !*seven && !*search {
		fmt.Println(max, sumScore, productScore)
	}
	return sumScore, productScore
}

func factor(a big.Int) []big.Int {
	number, primes, x := big.Int{}, make([]big.Int, 0, 256), big.Int{}
	number.Set(&a)

	for x.Mod(&number, two).Cmp(zero) == 0 {
		primes = append(primes, *two)
		number.Div(&number, two)
	}

	i := big.NewInt(3)
	for x.Mul(i, i).Cmp(&number) <= 0 {
		for x.Mod(&number, i).Cmp(zero) == 0 {
			y := big.Int{}
			y.Set(i)
			primes = append(primes, y)
			number.Div(&number, i)
		}
		i.Add(i, two)
	}

	if number.Cmp(two) > 0 {
		y := big.Int{}
		y.Set(&number)
		primes = append(primes, y)
	}

	return primes
}

type Searcher func(x, y uint64) (int, *big.Int)

func fibonacciSearch(i0, i1 int64) Searcher {
	return func(x, y uint64) (int, *big.Int) {
		base := big.NewInt(0)
		base.SetUint64(x * y)
		test := func(offset *big.Int) (bool, *big.Int) {
			gcd, sum := big.Int{}, big.Int{}
			sum.Add(base, offset)
			if gcd.GCD(nil, nil, base, &sum).Cmp(one) > 0 {
				return true, &gcd
			}
			return false, nil
		}

		a, b, i := big.NewInt(i0), big.NewInt(i1), 0
		//fmt.Println(a)
		//fmt.Println(b)
		if ok, gcd := test(b); ok {
			return i, gcd
		}
		i++
		for {
			c := big.NewInt(0)
			c.Add(a, b)
			a, b = b, c
			//fmt.Println(c)
			if ok, gcd := test(b); ok {
				return i, gcd
			}
			i++
		}
	}
}

type PrimeSource interface {
	Next() (x, y uint64)
	More() bool
}

type SequentialSource struct {
	Primes []uint64
	I      int
}

func NewSequentialSource(max uint64) *SequentialSource {
	primes := sieveOfEratosthenes(max)
	return &SequentialSource{
		Primes: primes,
	}
}

func (s *SequentialSource) Next() (x, y uint64) {
	x, y = s.Primes[s.I], s.Primes[s.I+1]
	s.I++
	return
}

func (s *SequentialSource) More() bool {
	return s.I < len(s.Primes)-1
}

type RandomSource struct {
	Primes []uint64
	I      int
}

func NewRandomSource(max uint64) *RandomSource {
	primes := sieveOfEratosthenes(max)
	return &RandomSource{
		Primes: primes,
	}
}

func (r *RandomSource) Next() (x, y uint64) {
	length := len(r.Primes)
	x, y = r.Primes[rand.Intn(length)], r.Primes[rand.Intn(length)]
	r.I++
	return
}

func (r *RandomSource) More() bool {
	return r.I < len(r.Primes)-1
}

func fibonacciGraph(name string, source PrimeSource, searchers []Searcher) {
	type Result struct {
		X, Y, Index uint64
		GCD         *big.Int
	}
	cores := runtime.NumCPU() * 2
	results := make(chan Result, cores)
	factor := func(x, y uint64) {
		var index uint64 = math.MaxUint64
		var gcd *big.Int
		for _, searcher := range searchers {
			i, g := searcher(x, y)
			if uint64(i) < index {
				index, gcd = uint64(i), g
			}
		}
		results <- Result{
			X:     x,
			Y:     y,
			Index: index,
			GCD:   gcd,
		}
	}

	data, routines := make([]Result, 0, len(primes)-1), 0
	for source.More() {
		if routines < cores {
			go factor(source.Next())
			routines++
			continue
		}
		result := <-results
		data = append(data, result)
		routines--
		fmt.Printf("%d %d %d %v\n", result.X, result.Y, result.Index, result.GCD)
	}
	for routines > 0 {
		result := <-results
		data = append(data, result)
		routines--
		fmt.Printf("%d %d %d %v\n", result.X, result.Y, result.Index, result.GCD)
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].X < data[j].X
	})

	out, err := os.Create(fmt.Sprintf("%s.csv.gz", name))
	if err != nil {
		panic(err)
	}
	defer out.Close()
	csv, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	defer csv.Close()
	fmt.Fprintf(csv, "x, y, index, gcd\n")
	for _, item := range data {
		fmt.Fprintf(csv, "%d, %d, %d, %v\n", item.X, item.Y, item.Index, item.GCD)
	}

	points := make(plotter.XYs, 0, len(primes)-1)
	for _, item := range data {
		points = append(points, plotter.XY{X: float64(item.GCD.Uint64()), Y: float64(item.Index)})
	}

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = fmt.Sprintf("factor vs index for %s", name)
	p.X.Label.Text = "factor"
	p.Y.Label.Text = "index"

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		panic(err)
	}
	scatter.GlyphStyle.Radius = vg.Length(1)
	scatter.GlyphStyle.Shape = draw.CircleGlyph{}
	p.Add(scatter)

	err = p.Save(8*vg.Inch, 8*vg.Inch, fmt.Sprintf("%s.png", name))
	if err != nil {
		panic(err)
	}

	sort.Slice(data, func(i, j int) bool {
		return float64(data[i].GCD.Uint64())/float64(data[i].Index) < float64(data[j].GCD.Uint64())/float64(data[j].Index)
	})

	outCluster, err := os.Create(fmt.Sprintf("%s_cluster.csv.gz", name))
	if err != nil {
		panic(err)
	}
	defer outCluster.Close()
	csvCluster, err := gzip.NewWriterLevel(outCluster, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	defer csvCluster.Close()
	fmt.Fprintf(csvCluster, "x, y, index, gcd, slope\n")
	for _, item := range data {
		fmt.Fprintf(csvCluster, "%d, %d, %d, %v, %f\n",
			item.X, item.Y, item.Index, item.GCD,
			float64(item.GCD.Uint64())/float64(item.Index))
	}
}

func binet(nn *big.Int) {
	prec := uint(1024)
	n := big.Int{}
	n.Set(nn)

	f1, p1 := big.Float{}, big.Float{}
	f1.SetPrec(prec).Sqrt(fFive)
	f1.Add(fOne, &f1)
	p1.SetPrec(prec).Set(fOne)

	f2, p2 := big.Float{}, big.Float{}
	f2.SetPrec(prec).Sqrt(fFive)
	f2.Sub(fOne, &f2)
	p2.SetPrec(prec).Set(fOne)

	f3, p3 := big.Float{}, big.Float{}
	f3.SetPrec(prec).Set(fTwo)
	p3.SetPrec(prec).Set(fOne)

	for n.Cmp(zero) > 0 {
		if n.Bit(0) == 1 {
			p1.Mul(&p1, &f1)
			p2.Mul(&p2, &f2)
			p3.Mul(&p3, &f3)
		}
		f1.Mul(&f1, &f1)
		f2.Mul(&f2, &f2)
		f3.Mul(&f3, &f3)
		n.Rsh(&n, 1)
	}

	f := big.Float{}
	f.SetPrec(prec).Sub(&p1, &p2)
	d := big.Float{}
	d.SetPrec(prec).Sqrt(fFive)
	d.Mul(&p3, &d)
	f.Quo(&f, &d)
	output := big.Int{}
	f.Int(&output)
	fmt.Println(&output)
}

func sieveOfEratosthenes(n uint64) (primes []uint64) {
	b := make([]bool, n)
	for i := uint64(2); i < n; i++ {
		if b[i] {
			continue
		}
		primes = append(primes, i)
		for j := i * i; j < n; j += i {
			b[j] = true
		}
	}
	return
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
	if *random {
		series := randomSeries()
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
	if *seven {
		series := sevenSmoothSeries(100)
		for _, number := range series {
			fmt.Printf(" %s", number.String())
		}
		fmt.Printf("\n")

		Registry["sevenSmooth"].graph(256)
		return
	}
	if *sevenComp {
		series := sevenSmoothComplementSeries(1024)
		for _, number := range series {
			fmt.Printf(" %s", number.String())
		}
		fmt.Printf("\n")
		sum, product := sumProductTest(series)
		fmt.Println(math.Sqrt(sum*sum + product*product))

		Registry["sevenSmoothComplement"].graph(2048)
		return
	}
	if *search {
		searchSeries()
		return
	}
	if *fibonacci {
		//i, gcd := fibonacciSearch(99989, 99991)
		//fmt.Println("found", gcd, i)
		source := NewSequentialSource(50000)
		fibonacciGraph("fibonacci", source, []Searcher{fibonacciSearch(0, 1)})
		//source := NewRandomSource(50000)
		//fibonacciGraph("random", source, []Searcher{fibonacciSearch(0, 1)})
		//fibonacciGraph("lucas", source, []Searcher{fibonacciSearch(2, 1)})
		//fibonacciGraph("combined", source, []Searcher{fibonacciSearch(0, 1), fibonacciSearch(2, 1)})

		//n := big.Int{}
		//n.SetString(*number, 10)
		//binet(&n)
		return
	}
	if *printPrimes > 0 {
		p := sieveOfEratosthenes(*printPrimes)
		for _, i := range p {
			fmt.Printf("%d ", i)
		}
		fmt.Printf("\n")
		return
	}

	i := big.Int{}
	_, ok = i.SetString(*number, 10)
	if !ok {
		panic("invalid number")
	}
	series := collatz(&i)
	for _, item := range series {
		fmt.Printf("%v [", &item)
		factors := factor(item)
		for _, f := range factors {
			fmt.Printf("%v, ", &f)
		}
		fmt.Printf("]\n")
	}
	sumProductTest(series)

	found := make(map[string]bool)
	j := big.Int{}
	for i := 1; i < 1e3; i++ {
		j.SetInt64(int64(i))
		series := collatz(&j)
		for _, item := range series {
			found[item.String()] = true
		}
		count := 0
		for k := 1; k < i; k++ {
			j.SetInt64(int64(k))
			if !found[j.String()] {
				count++
			}
		}
		if count > 0 {
			fmt.Println(i, count)
		}
	}

	histogram := make(map[int]int)
	h := gohistogram.NewHistogram(100)
	for i := 2; i < 1024*1024*1024; i += 2 {
		count, number := 0, i
		for number&1 == 0 {
			count++
			number >>= 1
		}
		histogram[count]++
		h.Add(float64(count))
	}
	fmt.Println(h.String())
	max := 0
	for key := range histogram {
		if key > max {
			max = key
		}
	}
	values := make([]int, max+1)
	for key, value := range histogram {
		values[key] = value
	}
	for key, value := range values {
		fmt.Println(key, value)
	}
}
