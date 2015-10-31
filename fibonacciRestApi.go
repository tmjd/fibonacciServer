package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type FibNum struct {
	value int
}

func (fn *FibNum) Accumlate(in *FibNum) {
	fn.value = fn.value + in.value
}

func (fn FibNum) String() string {
	return strconv.Itoa(fn.value)
}

type FibonacciGenerator struct {
	maxIterations int
}

func NewFibonacciGenerator(iterations int) (fg *FibonacciGenerator, err error) {
	if iterations < 0 {
		return nil, errors.New("Number of iterations cannot be negative")
	} else if iterations > 100000 {
		return nil, errors.New("Number of iterations cannot be greater than 100000")
	}
	fg = &FibonacciGenerator{}
	fg.maxIterations = iterations
	return fg, nil
}

func (fg *FibonacciGenerator) fibonacci(out chan<- FibNum) {
	if fg.maxIterations == 0 {
		return
	}
	var v [2]int
	v[0] = 0
	v[1] = 1
	idx := 0

	for i := 0; i < fg.maxIterations; i = i + 1 {
		out <- FibNum{v[idx]}
		v[idx] = v[0] + v[1]
		if idx == 0 {
			idx = 1
		} else {
			idx = 0
		}
	}

	close(out)
}

func handleUnsupportedMethod(res http.ResponseWriter, req *http.Request) {
	http.Error(res, fmt.Sprintf("%q unsupported", req.Method), http.StatusMethodNotAllowed)
	log.Print(req)
}

func handleFibonacciRequest(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		fg, err := NewFibonacciGenerator(5)
		if err != nil {
			http.Error(res, err.Error(), http.StatusMethodNotAllowed)
			log.Printf("FibonacciGenerator reported %q from request %q", err, req)
		}

		nums := make(chan FibNum)
		go fg.fibonacci(nums)
		var output bytes.Buffer
		for num := range nums {
			output.WriteString(num.String())
		}

		_, err = res.Write(output.Bytes())
		if err != nil {
			log.Printf("Error (%q) while writing response for %q", err, req.Host)
		}
	} else {
		handleUnsupportedMethod(res, req)
	}
}

func main() {

	sm := http.NewServeMux()
	sm.HandleFunc("/fibonacci", handleFibonacciRequest)

	log.Fatal(http.ListenAndServe(":8080", sm))
}
