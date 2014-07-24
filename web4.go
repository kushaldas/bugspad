package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"
)

type Page struct {
	Email string
	Flag_login bool
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

func getCookie (user string) (http.Cookie,string){
	hash := generate_hash()
	new_hash := get_hex(string(hash))
	expire := time.Now().AddDate(0, 0, 1)
	final_hash := user + ":" + new_hash
	cookie := http.Cookie{Name: "bugsuser", Value: final_hash, Path: "/", Expires: expire, MaxAge: 86400}
	return cookie, final_hash
}

func get_template(name string) (*template.Template, error) {
	tml, err := template.ParseFiles(name, "./templates/navbar.html", "./templates/base.html")
	return tml, err
}

/*
This function is the starting point for user authentication.
*/

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var tml *template.Template
		var err error
		login_flag, useremail := is_logged(r)
		page := Page{Flag_login: login_flag, Email: useremail}
		if !(login_flag) {
		//One style of template parsing.
			tml, err = get_template("./templates/login.html")
			fmt.Println("login page")
			if err != nil {
				checkError(err)
			}
			
		} else {
			fmt.Println(page)
			tml, err = get_template("./templates/home.html")
			if err != nil {
				checkError(err)
			}	
		}
		tml.ExecuteTemplate(w, "base", page)
		return
	} else {
		user := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		if authenticate_redis(user, password) {
			
			cookie,final_hash := getCookie(user)
			http.SetCookie(w, &cookie)
			redis_hset("sessions", user, final_hash)			
			tml, _ := get_template("templates/logout.html") 

			user_type := Page{Email: user, Flag_login: true}

			tml.ExecuteTemplate(w, "base", user_type)

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
				redis_hdel("sessions",words[0])
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.ListenAndServe(":9999", nil)
}
