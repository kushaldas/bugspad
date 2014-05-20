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

/*
The home landing page of bugspad
*/
func home(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
	    //fmt.Fprintln(w, "get")
	    il, useremail := is_logged(r)
	    fmt.Println(il)
	    fmt.Println(useremail)
		//fmt.Println(r.FormValue("username"))
		    
	    tml, err := template.ParseFiles("./templates/home.html","./templates/base.html")
	    if err != nil {
		checkError(err)
	    }
	    tml.ExecuteTemplate(w,"base", map[string]interface{}{"useremail":useremail,"islogged":il})
	    return
	}
	    
}
/*
This function is the starting point for user authentication.
*/

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {

		//One style of template parsing.
		tml, err := template.ParseFiles("./templates/login.html","./templates/base.html")
		if err != nil {
			checkError(err)
		}
		tml.ExecuteTemplate(w,"base", nil)
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
			//Second style of template parsing.
			tml := template.Must(template.ParseFiles("templates/logout.html","templates/base.html"))
			
			user_type := User{Email: user}
			
			tml.ExecuteTemplate(w,"base", user_type)

		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

/*
Logging out a user.
*/
func logout(w http.ResponseWriter, r *http.Request) {
	il, user := is_logged(r)
	if il{
		redis_hdel("sessions",user)
		fmt.Println("Logout!")
		http.Redirect(w,r,"/",http.StatusFound)
	    }
	return
}


/*
Registering a User
*/
func registeruser(w http.ResponseWriter, r *http.Request) {
    // TODO add email verification for the user. 
    // TODO add openid registration. 

	if r.Method == "GET" {
	
		tml, err := template.ParseFiles("./templates/registeruser.html","./templates/base.html")
		if err != nil {
			checkError(err)
		}
		tml.ExecuteTemplate(w,"base", nil)
		return
	
	} else if r.Method == "POST" {
		//type "0" refers to the normal user, while "1" refers to the admin
		ans := add_user(r.FormValue("username"), r.FormValue("useremail"), "0", r.FormValue("password") )
		if ans != "User added." {
		    fmt.Fprintln(w,"User could not be registered.")
		}		
		http.Redirect(w,r,"/",http.StatusFound)
	}
	
}

func main() {
        load_config("config/bugspad.ini")
        // Load the user details into redis.
	load_users()
	http.HandleFunc("/", home)
	http.HandleFunc("/register/", registeruser)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout/", logout)
	http.ListenAndServe(":9999", nil)
}
