package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"
	"strings"
	"crypto/rand"
    "encoding/base64" 
)

type User struct {
	Email string
}


func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}



func generate_hash() []byte {
    b := make([]byte, 16)
    rand.Read(b)
    en := base64.StdEncoding // or URLEncoding
    d := make([]byte, en.EncodedLen(len(b)))
    en.Encode(d, b) 
    return b
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tml, err := template.ParseFiles("login.html")
		if err != nil {
			checkError(err)
		}
		tml.Execute(w, nil)
		return
	} else {
		user := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		if authenticate_redis(user, password) {
			hash := generate_hash()
			new_hash := get_hex(string(hash))
			expire := time.Now().AddDate(0, 0, 1)
			final_hash := user + ":" + new_hash
			cookie := http.Cookie{Name: "bugsuser", Value: final_hash, Path: "/", Expires: expire, MaxAge: 86400}
			http.SetCookie(w, &cookie)
			redis_hset("sessions", user, final_hash)
			tml, err := template.ParseFiles("logout.html")
			if err != nil {
				checkError(err)
			}
			user_type := User{Email: user}
			
			tml.Execute(w, user_type)

		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

func logout(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("bugsuser")
	if err == nil {
		hash := cookie.Value
		words := strings.Split(hash, ":")
		if len(words) == 2 {
			pss := string(redis_hget("sessions", words[0]))
			if pss == hash {
				// Now we have a proper session matched, We can logout now.
				fmt.Println("Logout man!")
				http.Redirect(w, r, "/login", http.StatusFound)
			}
		}
		

	}
	return
}

func main() {
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout/", logout)
	http.ListenAndServe(":9999", nil)
}
