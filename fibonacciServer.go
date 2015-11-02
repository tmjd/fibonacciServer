package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/tmjd/fibonacci"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// Parses a variable n out of a POST form or query or a GET query value, all other
// methods will result in an error being returned
func getIterationCount(req *http.Request) (iterations int, err error) {
	if req.Method == "POST" {
		if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
			if err := req.ParseMultipartForm(1024); err != nil {
				return 0, fmt.Errorf("Bad multipart form parse: %s", err)
			}
		}
		if err := req.ParseForm(); err != nil {
			return 0, fmt.Errorf("Bad form parse: %s", err)
		}

		n, err := strconv.Atoi(req.FormValue("n"))
		if err != nil {
			return 0, fmt.Errorf("Bad value(%s) in form: %s", req.FormValue("n"), err)
		}

		return n, nil
	} else if req.Method == "GET" {
		values := req.URL.Query()
		n, err := strconv.Atoi(values.Get("n"))
		if err != nil {
			return 0, fmt.Errorf("Bad value(%s) in form: %s", values.Get("n"), err)
		}
		return n, nil
	} else {
		return 0, fmt.Errorf("Method %s not valid", req.Method)
	}
}

// Pulls FibNum(s) out of the passed in channel until it is closed and returns
// the byte slice. The output is wrapped in [] and has a comma between each element
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

// Request handler that will serve up fibonacci numbers. Also comes with a stats
// monitor that must be ran or the channels for collecting stats will fill
// and cause the handler to become blocked
type FibonacciRequestHandler struct {
	activeReq chan int
	reqStats  chan reqStat
	url_path  string
}

// These is our dependency injection for testing
var timeTriggerDelay = time.After
var statSelectDone = func() {}
var writeLogMsg = log.Printf

func clearInjectionPoints() {
	timeTriggerDelay = time.After
	statSelectDone = func() {}
	writeLogMsg = log.Printf
}

// Periodically prints out the stats over the last 2 seconds if there are or have
// been any requests handled
func (frh *FibonacciRequestHandler) statsMonitor() {
	var state statState
	state.clear()
	cur_req := 0

	printDelay, _ := time.ParseDuration("2s")
	timeTrigger := timeTriggerDelay(printDelay)
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
				writeLogMsg("Fibonacci stats: %s", state)
			}
			state.clear()
			// Immediately set the max to the cur_req
			state.max_concurrent_requests = cur_req

			// Reset the timeTrigger
			timeTrigger = timeTriggerDelay(printDelay)
		}
		statSelectDone() //Injection point for testing
	}
}

// Create new fibonacci request handler and setup the channels used for stats collection
func NewFibonacciRequestHandler(url_path string) *FibonacciRequestHandler {
	var frh FibonacciRequestHandler
	frh.activeReq = make(chan int, 100)
	frh.reqStats = make(chan reqStat, 100)
	frh.url_path = path.Clean("/" + url_path)
	return &frh
}

func respondToUnsupportedMethod(res http.ResponseWriter, req *http.Request) {
	http.Error(res, fmt.Sprintf("%q unsupported", req.Method), http.StatusMethodNotAllowed)
	writeLogMsg("%q", req)
}

// Handler for generating fibonacci numbers, expects a variable n to be set through
// a POST form or query or a GET query value
func (frh *FibonacciRequestHandler) FibonacciRequestHandleFunc(res http.ResponseWriter, req *http.Request) {
	frh.activeReq <- 1
	start := time.Now()
	var stat reqStat
	defer func() {
		frh.activeReq <- -1
		stat.duration = time.Since(start)
		frh.reqStats <- stat
	}()

	// If the path does not match exactly then response with error
	if req.URL.Path != frh.url_path {
		msg := fmt.Sprintf("Request path (%s) does not match %s", req.URL.Path, frh.url_path)
		writeLogMsg("%s, respond with code StatusNotFound", msg)
		http.Error(res, msg, http.StatusNotFound)
		return
	}
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
		writeLogMsg("FibonacciGenerator reported %q from request %q", err, req)
		return
	}

	nums := make(chan fibonacci.FibNum)
	go fg.Produce(nums)
	output := buildOutput(nums)

	_, err = res.Write(output)
	if err != nil {
		writeLogMsg("Error (%s) while writing response for %q", err, req.Host)
	}
}

var serve_path string
var port int

func init() {
	flag.StringVar(&serve_path, "serve_path", "fibonacci",
		"the path from root that will access the RestAPI")
	flag.IntVar(&port, "port", 8080, "port where the server will listen")
}

func main() {
	flag.Parse()

	serve_path = path.Clean(fmt.Sprintf("/%s", serve_path))

	writeLogMsg("FibonacciServer listening on port %d at path %s", port, serve_path)

	sm := http.NewServeMux()
	frh := NewFibonacciRequestHandler(serve_path)
	sm.HandleFunc(serve_path, frh.FibonacciRequestHandleFunc)

	// Must run the stats monitor or the stats channels will fill and block requests
	go frh.statsMonitor()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), sm))
}
