package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
)

// Custom Number to use generating the fibonacci numbers, can swap out
// the underlying type without affecting the algorithm
type FibNum struct {
	value big.Int
}

func newFibNum(init int64) FibNum {
	fn := FibNum{}
	fn.value = *big.NewInt(init)
	return fn
}

// Clone to get copy of a FibNum.
// With big.Int need to do a new big.NewInt so the FibNum is not using the same
// big.Int value.
func cloneFibNum(src FibNum) FibNum {
	fn := FibNum{}
	fn.value = *big.NewInt(0)
	fn.value.Set(&src.value)
	return fn
}

func (fn *FibNum) add(a FibNum, b FibNum) {
	fn.value.Add(&a.value, &b.value)
}

// Return string representation of FibNum
func (fn FibNum) String() string {
	return fn.value.String()
}

type FibonacciGenerator struct {
	maxIterations int
}

func NewFibonacciGenerator(iterations int) (fg *FibonacciGenerator, err error) {
	if iterations < 0 {
		return nil, errors.New("Number of iterations cannot be negative")
	} else if iterations > 100000 {
		// Seems like an unreasonably high number but I think there should be some
		// limit to what will be generated
		return nil, errors.New("Number of iterations cannot be greater than 100000")
	}
	fg = &FibonacciGenerator{}
	fg.maxIterations = iterations
	return fg, nil
}

func (fg *FibonacciGenerator) Fibonacci(out chan<- FibNum) {
	if fg.maxIterations == 0 {
		return
	}
	var v [2]FibNum
	v[0] = newFibNum(0)
	v[1] = newFibNum(1)
	idx := 0

	for i := 0; i < fg.maxIterations; i = i + 1 {
		out <- cloneFibNum(v[idx])
		v[idx].add(v[0], v[1])
		if idx == 0 {
			idx = 1
		} else {
			idx = 0
		}
	}

	close(out)
}

func respondToUnsupportedMethod(res http.ResponseWriter, req *http.Request) {
	http.Error(res, fmt.Sprintf("%q unsupported", req.Method), http.StatusMethodNotAllowed)
	log.Print(req)
}

func handleFibonacciRequest(res http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {

		if err := req.ParseForm(); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			log.Printf("Bad form parse from request %q", req)
			return
		}

		n, err := strconv.Atoi(req.FormValue("n"))
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			log.Printf("Invalid value in form. Expected int but received (%s) in request  %q",
				req.FormValue("n"), req)
			return
		}

		fg, err := NewFibonacciGenerator(n)
		if err != nil {
			http.Error(res, err.Error(), http.StatusMethodNotAllowed)
			log.Printf("FibonacciGenerator reported %q from request %q", err, req)
			return
		}

		nums := make(chan FibNum)
		go fg.Fibonacci(nums)
		var output bytes.Buffer
		output.WriteString("[")
		first := true
		for num := range nums {
			if first {
				first = false
			} else {
				output.WriteString(",")
			}
			output.WriteString(num.String())
		}
		output.WriteString("]")

		_, err = res.Write(output.Bytes())
		if err != nil {
			log.Printf("Error (%q) while writing response for %q", err, req.Host)
		}
	} else {
		respondToUnsupportedMethod(res, req)
	}
}

func main() {

	sm := http.NewServeMux()
	sm.HandleFunc("/fibonacci", handleFibonacciRequest)

	log.Fatal(http.ListenAndServe(":8080", sm))
}
