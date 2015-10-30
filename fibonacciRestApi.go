package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
)

type ctrl struct {
	cmd string
}

func fibonacci(out chan int, in chan ctrl) {
	var i [2]int
	i[0] = 0
	i[1] = 1
	idx := 0
	for {
		select {
		case out <- i[idx]:
			i[idx] = i[0] + i[1]
			if idx == 0 {
				idx = 1
			} else {
				idx = 0
			}
		case <-in:
			i[0] = 0
			i[1] = 1
			idx = 0
		}
	}
}

type handleFibUrl struct {
	fib_data chan int
}

func (hfu handleFibUrl) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		fmt.Fprintf(res, "Hello %q. Your num is: %d",
			html.EscapeString(req.Host), <-hfu.fib_data)
	} else {
		http.Error(res, "POST unsupported", http.StatusMethodNotAllowed)
		log.Print(req)
		return
	}
}

type handleResetUrl struct {
	ctrl_chan chan ctrl
}

func (hru handleResetUrl) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		hru.ctrl_chan <- ctrl{"Reset"}
		fmt.Fprintf(res, "Hello %q. Sequence has been reset", html.EscapeString(req.Host))
	} else {
		http.Error(res, "POST unsupported", http.StatusMethodNotAllowed)
		log.Print(req)
	}
}

func main() {
	fib_chan := make(chan int)
	ctrl_chan := make(chan ctrl)

	go fibonacci(fib_chan, ctrl_chan)

	sm := http.NewServeMux()
	var fibHandleFunc handleFibUrl
	fibHandleFunc.fib_data = fib_chan
	sm.Handle("/fibonacci", fibHandleFunc)
	var resetHandleFunc handleResetUrl
	resetHandleFunc.ctrl_chan = ctrl_chan
	sm.Handle("/reset", resetHandleFunc)

	log.Fatal(http.ListenAndServe(":8080", sm))
}
