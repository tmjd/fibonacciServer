package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tmjd/fibonacci"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func respondToUnsupportedMethod(res http.ResponseWriter, req *http.Request) {
	http.Error(res, fmt.Sprintf("%q unsupported", req.Method), http.StatusMethodNotAllowed)
	log.Print(req)
}

func getIterationCount(req *http.Request) (iterations int, err error) {
	if req.Method == "POST" {
		if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
			if err := req.ParseMultipartForm(1024); err != nil {
				log.Printf("Bad multipart form parse from request %q", req)
				return 0, err
			}
		}
		if err := req.ParseForm(); err != nil {
			log.Printf("Bad form parse from request %q", req)
			return 0, err
		}

		n, err := strconv.Atoi(req.FormValue("n"))
		if err != nil {
			log.Printf("Invalid value in form. Expected int but received (%s) in request  %q",
				req.FormValue("n"), req)
			return 0, err
		}

		return n, nil
	} else if req.Method == "GET" {
		//TODO: fill this out
		return 0, errors.New("GET is not impemented yet")
	} else {
		return 0, errors.New(fmt.Sprintf("Method %s not valid", req.Method))
	}
}

func buildOutput(in <-chan fibonacci.FibNum) []byte {
	var output bytes.Buffer
	output.WriteString("[")
	first := true
	for num := range in {
		if first {
			first = false
		} else {
			output.WriteString(",")
		}
		output.WriteString(num.String())
	}
	output.WriteString("]")

	return output.Bytes()
}

type reqStat struct {
	duration   time.Duration
	iterations int
}

func (rs reqStat) String() string {
	return fmt.Sprintf("n=%d-%s", rs.iterations, rs.duration)
}

type FibonacciRequestHandler struct {
	activeReq chan int
	reqStats  chan reqStat
}

type statState struct {
	max_concurrent_requests int
	requests_since_trigger  int
	max_iterations          reqStat
	max_duration            reqStat
	min_duration            reqStat
}

func (ss *statState) clear() {
	ss.max_concurrent_requests = 0
	ss.requests_since_trigger = 0
	ss.max_iterations.iterations = 0
	ss.max_iterations.duration = 0
	ss.max_duration.iterations = 0
	ss.max_duration.duration = time.Since(time.Now())
	ss.min_duration.iterations = 0
	ss.min_duration.duration = time.Since(time.Now().AddDate(-1, -1, -1))
}

func (ss statState) String() string {
	return fmt.Sprintf("Requests %d Concurrent %d; MaxIterations:%s MinElapse:%s MaxElapse:%s",
		ss.requests_since_trigger, ss.max_concurrent_requests, ss.max_iterations,
		ss.min_duration, ss.max_duration)
}

func (frh *FibonacciRequestHandler) statsMonitor() {
	var state statState
	state.clear()
	cur_req := 0

	printDelay, _ := time.ParseDuration("2s")
	timeTrigger := time.After(printDelay)
	for {
		select {
		case req := <-frh.activeReq:
			cur_req = cur_req + req

			if req == 1 {
				state.requests_since_trigger = state.requests_since_trigger + 1
			}

			if cur_req > state.max_concurrent_requests {
				state.max_concurrent_requests = cur_req
			}
		case stat := <-frh.reqStats:
			if state.max_duration.duration.Nanoseconds() < stat.duration.Nanoseconds() {
				state.max_duration = stat
			}
			if state.min_duration.duration.Nanoseconds() > stat.duration.Nanoseconds() {
				state.min_duration = stat
			}
			if state.max_iterations.iterations < stat.iterations {
				state.max_iterations = stat
			}
		case <-timeTrigger:
			if state.max_concurrent_requests != 0 {
				log.Printf("Fibonacci stats: %s", state)
			}
			state.clear()

			timeTrigger = time.After(printDelay)
		}
	}
}

func NewFibonacciRequestHandler() *FibonacciRequestHandler {
	var frh FibonacciRequestHandler
	frh.activeReq = make(chan int, 100)
	frh.reqStats = make(chan reqStat, 100)
	return &frh
}

func (frh *FibonacciRequestHandler) FibonacciRequestHandleFunc(res http.ResponseWriter, req *http.Request) {
	frh.activeReq <- 1
	start := time.Now()
	var stat reqStat
	defer func() {
		frh.activeReq <- -1
		stat.duration = time.Since(start)
		frh.reqStats <- stat
	}()
	if req.Method != "POST" && req.Method != "GET" {
		respondToUnsupportedMethod(res, req)
		return
	}

	n, err := getIterationCount(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	stat.iterations = n
	fg, err := fibonacci.NewGenerator(n)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		log.Printf("FibonacciGenerator reported %q from request %q", err, req)
		return
	}

	nums := make(chan fibonacci.FibNum)
	go fg.Produce(nums)
	output := buildOutput(nums)

	_, err = res.Write(output)
	if err != nil {
		log.Printf("Error (%q) while writing response for %q", err, req.Host)
	}
}

func main() {

	sm := http.NewServeMux()
	frh := NewFibonacciRequestHandler()
	sm.HandleFunc("/fibonacci", frh.FibonacciRequestHandleFunc)

	go frh.statsMonitor()

	log.Fatal(http.ListenAndServe(":8080", sm))
}
