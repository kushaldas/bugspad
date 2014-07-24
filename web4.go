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
	"log"
)

type Page struct {
	Email string
	Flag_login bool
	Bug Bug
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

func isvalueinlist(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
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

func log_message(r *http.Request, message string) {
	log.Printf("%s %s", r.Header.Get("X-Real-IP"), message)
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
			http.Redirect(w, r, "/", http.StatusFound)

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

func index_page(w http.ResponseWriter, r *http.Request) {
	var tml *template.Template
	var err error
	login_flag, useremail := is_logged(r)
	page := Page{Flag_login: login_flag, Email: useremail}
	tml, err = get_template("./templates/home.html")
	if err != nil {
		checkError(err)
	}
	tml.ExecuteTemplate(w, "base", page)
	return
}

/*
Registering a User
*/
func registeruser(w http.ResponseWriter, r *http.Request) {
	// TODO add email verification for the user.
	// TODO add openid registration.
	interface_data := make(Bug)
	if r.Method == "GET" {

		tml, err := get_template("./templates/registeruser.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		interface_data["pagetitle"] = "Register"
		page := Page{Bug: interface_data}
		err = tml.ExecuteTemplate(w, "base", page)
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		return

	} else if r.Method == "POST" {
		//type "0" refers to the normal user, while "1" refers to the admin
		if r.FormValue("password") != r.FormValue("repassword") {
			fmt.Fprintln(w, "Passwords do not match.")
			return
		}
		ans := add_user(r.FormValue("username"), r.FormValue("useremail"), "0", r.FormValue("password"))
		if ans != "User added." {
			fmt.Fprintln(w, "User could not be registered.")
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}

}

/*
Function to handle product selection before filing a bug.
*/
func before_createbug(w http.ResponseWriter, r *http.Request) {
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	page := Page{Flag_login: il, Email: useremail}
	if r.Method == "GET" {
		tml, err := get_template("./templates/filebug_product.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		if il {
			allproducts := get_all_products()
			interface_data["useremail"] = useremail
			interface_data["islogged"] = il
			interface_data["is_user_admin"] = is_user_admin(useremail)
			interface_data["products"] = allproducts
			interface_data["pagetitle"] = "Choose Product"
			page.Bug = interface_data
			err := tml.ExecuteTemplate(w, "base", page)
			if err != nil {
				log_message(r, "System Crash:"+err.Error())
			}
			return
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

func main() {
	load_config("config/bugspad.ini")
	// Load the user details into redis.
	load_users()
	//loading static bug tags
	load_bugtags()
	//to be used for logging purpose.
	logf, err := os.OpenFile("search_kd.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		log.Fatalln(err)
	}


	log.SetOutput(logf)
	http.HandleFunc("/", index_page)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout/", logout)
	http.HandleFunc("/register/", registeruser)
	http.HandleFunc("/filebug_product/", before_createbug)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.ListenAndServe(":9999", nil)
}
