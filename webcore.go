package main

import (
	"fmt"
	"net/http"
	"strings"
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

func log_request(r *http.Request, tm time.Time) {
	duration := time.Now().Sub(tm)
	fmt.Printf("%s %s %s %v\n", r.RemoteAddr, r.Method, r.URL, duration)
}

func is_logged(r *http.Request) (bool, string) {
	cookie, err := r.Cookie("bugsuser")
	if err != nil {
		return false, ""
	}
	hash_recieved := cookie.Value
	tup := strings.Split(hash_recieved, ":")
	hash_stored := redis_hget("sessions", tup[0])
	if string(hash_stored) == string(hash_recieved) {
		return true, tup[0]
	}
	return false, tup[0]
}
