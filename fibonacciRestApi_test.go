package main

import (
	"math"
	"testing"
)

func TestFibonacciNegative(t *testing.T) {

	var test_values = []int{-1, -2, -3, -10, -1000, -1000000, math.MinInt64}
	for _, i := range test_values {
		if _, err := NewFibonacciGenerator(i); err == nil {
			t.Errorf("Expected FibonacciGenerator to return error when asked for %d iterations", i)

		}
	}
}

func TestFibonacciNumberOfValuesGenerated(t *testing.T) {
	var test_values = []int{1, 2, 3, 10, 20, 100, 1000, 100000}

	for _, i := range test_values {
		fg, err := NewFibonacciGenerator(i)
		if err != nil {
			t.Errorf("Expected FibonacciGenerator to return error when asked for %d iterations", i)
		}

		result_chan := make(chan FibNum)
		go fg.fibonacci(result_chan)

		cnt := 0
		for dont_care := range result_chan {
			_ = dont_care
			cnt = cnt + 1
		}

		if cnt != i {
			t.Errorf("Generated %d numbers when asked to generate %d, maxIterations is %d", cnt, i, fg.maxIterations)
		}
	}
}

func TestFibonacciVerifyCorrectOutput(t *testing.T) {
	var test_values = []int{1, 2, 3, 10, 20, 1000, 100000}

	for _, i := range test_values {
		fg, err := NewFibonacciGenerator(i)
		if err != nil {
			t.Errorf("Expected FibonacciGenerator to return error when asked for %d iterations", i)
		}

		result_chan := make(chan FibNum)
		go fg.fibonacci(result_chan)

		cnt := 0
		x, y := 0, 1
		for val := range result_chan {

			if x != val.value {
				t.Errorf("Incorrect value on %d iteration: expected %d got %d", cnt, x, val)
			}
			x, y = y, x+y
			if x < 0 {
				t.Fatalf("Test problem on %d iteration: expected value went negative %d", cnt, x)
			}
			cnt = cnt + 1
		}
	}
}
