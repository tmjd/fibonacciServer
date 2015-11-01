package main

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestFibonacciVerifyCorrectOutputAgainstInt(t *testing.T) {
	//At 'iteration' 93 int rolls and cannot be used to validate the implemented algorithm
	var test_values = []int{1, 2, 3, 10, 20, 93}

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
			cnt = cnt + 1
			if x < 0 {
				t.Fatalf("Test problem on %d iteration: expected value went negative %d", cnt, x)
			}
			if strconv.Itoa(x) != val.String() {
				t.Fatalf("Incorrect value on iteration %d of test %d: expected %d got %s",
					cnt, i, x, val.String())
			}
			x, y = y, x+y
		}
	}
}

func TestValidPostRequestResultsInSuccess(t *testing.T) {
	req, err := http.NewRequest("POST", "http://example.com/fibonacci",
		strings.NewReader("n=10"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	res := httptest.NewRecorder()
	handleFibonacciRequest(res, req)

	if res.Code != 200 {
		t.Errorf("Expect success from POST command with body: %q", res)
	}
}

func TestPostWithNoBodyResponseFailure(t *testing.T) {
	req, err := http.NewRequest("POST", "http://example.com/fibonacci", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	handleFibonacciRequest(res, req)

	if res.Code == 200 {
		t.Errorf("Expect failure from POST with no body")
	}
}

func TestFibonacciHandler(t *testing.T) {
	req, err := http.NewRequest("POST", "http://example.com/fibonacci",
		strings.NewReader("n=10"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	res := httptest.NewRecorder()
	handleFibonacciRequest(res, req)

	if res.Code != 200 {
		t.Errorf("Expect failure from crafted POST command")
	}

}
