package main

import (
	"github.com/tmjd/fibonacci"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerValidPostRequestResultsInSuccess(t *testing.T) {
	t.SkipNow()
	req, err := http.NewRequest("POST", "http://example.com/fibonacci",
		strings.NewReader("n=10"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	res := httptest.NewRecorder()
	frh := NewFibonacciRequestHandler()
	frh.FibonacciRequestHandleFunc(res, req)

	if res.Code != 200 {
		t.Errorf("Expect success from POST command with body: %q", res)
	}
}

func TestHandlerPostWithNoBodyResponseFailure(t *testing.T) {
	t.SkipNow()
	req, err := http.NewRequest("POST", "http://example.com/fibonacci", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	frh := NewFibonacciRequestHandler()
	frh.FibonacciRequestHandleFunc(res, req)

	if res.Code == 200 {
		t.Errorf("Expect failure from POST with no body")
	}
}

func TestFibonacciHandler(t *testing.T) {
	t.SkipNow()
	req, err := http.NewRequest("POST", "http://example.com/fibonacci",
		strings.NewReader("n=10"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	res := httptest.NewRecorder()
	frh := NewFibonacciRequestHandler()
	frh.FibonacciRequestHandleFunc(res, req)

	if res.Code != 200 {
		t.Errorf("Expect failure from crafted POST command")
	}
	if res.Body.String() != "[0,1,1,2,3,5,8,13,21,34]" {
		t.Errorf("Expect first 10 values. Got '%s'", res.Body)
	}
}

func TestBuildOutput(t *testing.T) {
	var test_values = []struct {
		iterations int
		expected   string
	}{
		{0, "[]"},
		{1, "[0]"},
		{2, "[0,1]"},
		{10, "[0,1,1,2,3,5,8,13,21,34]"},
	}

	for _, test_val := range test_values {
		fg, err := fibonacci.NewGenerator(test_val.iterations)
		if err != nil {
			t.Errorf("Creating fibonacci generator should not have errored with %d",
				test_val.iterations)
		}
		out_chan := make(chan fibonacci.FibNum)
		go fg.Produce(out_chan)
		output := buildOutput(out_chan)

		if string(output[:]) != test_val.expected {
			t.Errorf("Output from iteration %d did not match.\nExpected\n%s\nGot\n%s\n",
				test_val.iterations, test_val.expected, string(output[:]))
		}
	}
}
