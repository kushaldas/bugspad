package main

import (
	"fmt"
	"net/http"
	"time"
)


const (
	AUTH_ERROR string = "\"Authentication failure.\""
	SUCCESS    string = "\"Success\""
)

func myrecover(w http.ResponseWriter) {
	if r := recover(); r != nil {
		fmt.Fprintln(w, "\"Wrong input.\"")
	}
}

func log(r *http.Request, tm time.Time) {
	duration := time.Now().Sub(tm)
	fmt.Printf("%s %s %s %v\n", r.RemoteAddr, r.Method, r.URL, duration)
}