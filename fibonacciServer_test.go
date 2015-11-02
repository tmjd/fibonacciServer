package main

import (
	"errors"
	"fmt"
	"github.com/tmjd/fibonacci"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGettingIterationCountFromRequest(t *testing.T) {
	var test_values = []struct {
		expected    int
		expect_err  error
		method      string
		url         string
		body        string
		headerName  string
		headerValue string
	}{
		{10, nil, "POST", "http://example.com/fibonacci", "n=10", "Content-Type",
			"application/x-www-form-urlencoded; param=value"},
		{10, nil, "POST", "http://example.com/fibonacci?n=10", "", "Content-Type",
			"application/x-www-form-urlencoded; param=value"},
		{10, nil, "GET", "http://example.com/fibonacci?n=10", "", "", ""},
		{10, errors.New("DontCare"), "GET", "http://example.com/fibonacci", "", "", ""},
		{0, nil, "POST", "http://example.com/fibonacci", "n=0", "Content-Type",
			"application/x-www-form-urlencoded; param=value"},
		{21, nil, "POST", "http://example.com/fibonacci", "n=21", "Content-Type",
			"application/x-www-form-urlencoded; param=value"},
		{10, errors.New("DontCare"), "POST", "http://example.com/fibonacci",
			"n=10", "Content-Type", "application/bad; param=value"},
		{10, errors.New("DontCare"), "POST", "http://example.com/fibonacci",
			"n:10", "Content-Type", "application/x-www-form-urlencoded; param=value"},
	}

	for i, test_val := range test_values {
		req, err := http.NewRequest(test_val.method, test_val.url,
			strings.NewReader(test_val.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(test_val.headerName, test_val.headerValue)

		result, err := getIterationCount(req)
		if err != nil {
			if test_val.expect_err == nil {
				t.Errorf("Iteration %d did not meet err expectation", i)
			}
		} else if result != test_val.expected {
			t.Errorf("Iteration %d resulted in %d but expected %d", i, result, test_val.expected)
		}
	}
}

func TestGettingIterationCountFromMultipartRequest(t *testing.T) {
	var test_values = []struct {
		expected    int
		expect_err  error
		method      string
		body        *strings.Reader
		headerName  string
		headerValue string
	}{
		{10, nil, "POST",
			strings.NewReader("--foo\r\n" +
				"Content-Disposition: form-data; name='n'\r\n\r\n" +
				"10\r\n--foo--\r\n"),
			"Content-Type", "multipart/form-data; boundary=foo"},
		{10, errors.New("DontCare"), "POST",
			strings.NewReader("--foo\r\n" +
				"Content-Disposition: form-data; name='n'\r\n\r\n" +
				"10\r\n--foo--\r\n"),
			"Content-Type", "multipart/mixed; boundary=foo"},
	}

	for i, test_val := range test_values {
		req, err := http.NewRequest(test_val.method, "http://example.com/fibonacci",
			test_val.body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(test_val.headerName, test_val.headerValue)

		result, err := getIterationCount(req)
		if err != nil {
			if test_val.expect_err == nil {
				t.Errorf("Iteration %d caused getIterationCount to produce error: %s on %q", i, err, req)
			}
		} else if result != test_val.expected {
			t.Errorf("Iteration %d resulted in %d but expected %d", i, result, test_val.expected)
		}
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

func TestHandlerValidPostRequestResultsInSuccess(t *testing.T) {
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
	req, err := http.NewRequest("POST", "http://example.com/fibonacci", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	res := httptest.NewRecorder()
	frh := NewFibonacciRequestHandler()
	frh.FibonacciRequestHandleFunc(res, req)

	if res.Code == 200 {
		t.Errorf("Expect failure from POST with no body")
	}
}

func TestHandlerWithUnsupportedMethod(t *testing.T) {
	req, err := http.NewRequest("DELETE", "http://example.com/fibonacci", nil)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	frh := NewFibonacciRequestHandler()
	frh.FibonacciRequestHandleFunc(res, req)

	if res.Code == 200 {
		t.Errorf("Expect failure from DELETE method")
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
	frh := NewFibonacciRequestHandler()
	frh.FibonacciRequestHandleFunc(res, req)

	if res.Code != 200 {
		t.Errorf("Expect failure from crafted POST command")
	}
	if res.Body.String() != "[0,1,1,2,3,5,8,13,21,34]" {
		t.Errorf("Expect first 10 values. Got '%s'", res.Body)
	}
}

func TestStatsMonitor(t *testing.T) {
	defer clearInjectionPoints()

	frh := NewFibonacciRequestHandler()

	trigger_chan := make(chan time.Time)
	timeTriggerDelay = func(time.Duration) <-chan time.Time {
		return trigger_chan
	}

	selectDone_chan := make(chan bool)
	statSelectDone = func() {
		selectDone_chan <- true
	}

	log_chan := make(chan string)
	logged := false
	logStats = func(format string, v ...interface{}) {
		log_chan <- fmt.Sprintf(format, v)
		logged = true
	}

	go frh.statsMonitor()

	frh.activeReq <- 1
	<-selectDone_chan
	frh.activeReq <- 1
	<-selectDone_chan
	frh.activeReq <- 1
	<-selectDone_chan
	trigger_chan <- time.Now()
	result := <-log_chan
	<-selectDone_chan

	if !strings.Contains(result, "Requests 3 Concurrent 3") {
		t.Errorf("Result (%s) did not have expected value 3", result)
	}

	trigger_chan <- time.Now()
	result = <-log_chan
	<-selectDone_chan
	if !strings.Contains(result, "Requests 0 Concurrent 3") {
		t.Errorf("Result (%s) should still have expected value 3", result)
	}

	//Clear out all the 'active' requests
	frh.activeReq <- -1
	<-selectDone_chan
	frh.activeReq <- -1
	<-selectDone_chan
	frh.activeReq <- -1
	<-selectDone_chan
	trigger_chan <- time.Now()
	// Let log message happen but we don't care about this one
	result = <-log_chan
	<-selectDone_chan

	logged = false
	trigger_chan <- time.Now()
	<-selectDone_chan
	if logged {
		t.Errorf("Expected nothing to have been logged recieved %s", <-log_chan)
	}

	// Have an active request so all subsequent time triggers will result in a log message
	frh.activeReq <- 1
	<-selectDone_chan

	//reqState: duration, iterations
	dur, _ := time.ParseDuration("7s")
	frh.reqStats <- reqStat{dur, 14}
	<-selectDone_chan
	trigger_chan <- time.Now()
	result = <-log_chan
	<-selectDone_chan

	if strings.Contains(result, "n=14-7s") == false {
		t.Errorf("After sending a n=14, with a duration of 7s the values were not in the log message %s", result)
	}

	dur, _ = time.ParseDuration("8s")
	frh.reqStats <- reqStat{dur, 4}
	<-selectDone_chan
	dur, _ = time.ParseDuration("2s")
	frh.reqStats <- reqStat{dur, 15}
	<-selectDone_chan
	trigger_chan <- time.Now()
	result = <-log_chan
	<-selectDone_chan

	if strings.Contains(result, "MaxElapse:n=4-8s") == false || strings.Contains(result, "MaxIterations:n=15-2s") == false {
		t.Errorf("Log message %s did not have expected n=4-8s and n=15-2s", result)
	}

	dur, _ = time.ParseDuration("8s")
	frh.reqStats <- reqStat{dur, 4}
	<-selectDone_chan
	dur, _ = time.ParseDuration("12s")
	frh.reqStats <- reqStat{dur, 15}
	<-selectDone_chan
	trigger_chan <- time.Now()
	result = <-log_chan
	<-selectDone_chan

	if strings.Contains(result, "MaxElapse:n=15-12s") == false {
		t.Errorf("Log message %s did not have expected n=14-12s", result)
	}
}
