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
	"strconv"
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

/*
Function for creating a new bug.
*/
func createbug(w http.ResponseWriter, r *http.Request) {
	//perform any preliminary check
	//backend_bug(w,r)
	//to_be_rendered by the template
	interface_data := make(Bug)
	il, useremail := is_logged(r)
	page := Page{Flag_login: il, Email: useremail}
	if r.Method == "GET" {
		product_id := r.URL.Path[len("/filebug/"):]
		_, err := strconv.ParseInt(product_id, 10, 32)
		if err != nil {
			fmt.Fprintln(w, "You need to give a valid product for filing a bug!")
			return
		}
		tml, err := get_template("./templates/createbug.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		if il {
			prod_idint, _ := strconv.Atoi(product_id)
			allcomponents := get_components_by_product(prod_idint)
			interface_data["useremail"] = useremail
			interface_data["islogged"] = il
			interface_data["is_user_admin"] = is_user_admin(useremail)
			interface_data["components"] = allcomponents
			interface_data["pagetitle"] = "File Bug"
			interface_data["versions"] = get_product_versions(prod_idint)
			page.Bug = interface_data
			tml.ExecuteTemplate(w, "base", page)
			return
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	} else if r.Method == "POST" {
		if il {
			_, severs, priors := get_redis_bugtags()
			if !isvalueinlist(r.FormValue("bug_priority"), priors) {
				fmt.Fprintln(w, "Bug Priority Invalid.")
				return
			}
			if !isvalueinlist(r.FormValue("bug_severity"), severs) {
				fmt.Fprintln(w, "Bug Severity Invalid.")
				return
			}
			newbug := make(Bug)
			newbug["summary"] = r.FormValue("mybugsummary")
			newbug["whiteboard"] = r.FormValue("bug_whiteboard")
			newbug["severity"] = r.FormValue("bug_severity")
			newbug["hardware"] = r.FormValue("bug_hardware")
			newbug["description"] = r.FormValue("bug_description")
			newbug["priority"] = r.FormValue("bug_priority")
			compid, _ := strconv.Atoi(r.FormValue("bug_component"))
			newbug["component_id"] = compid
			newbug["reporter"] = get_user_id(useremail)
			//fmt.Println(reflect.TypeOf(newbug["component_id"]))
			//fmt.Println(reflect.TypeOf(newbug["reporter"]))
			docsint := get_user_id(r.FormValue("bug_docs"))
			if r.FormValue("bug_docs") != "" {
				if docsint != -1 {
					newbug["docs"] = docsint
				} else {
					fmt.Fprintln(w, "Please enter a valid user as Docs Maintainer.")
					return
				}
			}
			comp_idint, _ := strconv.Atoi(r.FormValue("bug_component"))
			version_int, _ := strconv.Atoi(r.FormValue("bug_version"))
			fmt.Println(version_int)
			if r.FormValue("bug_assignee") != "" {
				assignid := get_user_id(r.FormValue("bug_assignee"))
				if assignid == -1 {
					fmt.Fprintln(w, "Please enter a valid user as assignee")
					return
				}
				newbug["assigned_to"] = assignid
			} else { //simply add the component owner as the assignee.
				newbug["assigned_to"] = get_component_owner(comp_idint)
			}

			newbug["version"] = version_int
			bugid, err := new_bug(newbug)

			if bugid == -1 {
				fmt.Fprintln(w, err)
				return
			}
			//CC Addition
			bugccemails := make([]interface{}, 0)
			ccs := strings.SplitAfter(r.FormValue("bug_cc"), ",")
			for index, _ := range ccs {
				//fmt.Println(dependbug)
				if ccs[index] != "" {
					bugccemails = append(bugccemails, strings.Trim(ccs[index], ","))
				}
			}
			if bugccemails != nil {
				if !add_bug_cc(int64(bugid), bugccemails) {
					fmt.Fprintln(w, "Bug CC could not be updated, please check")
					return
				}
				http.Redirect(w, r, "/bugs/"+strconv.Itoa(bugid), http.StatusFound)

			} else {
				fmt.Fprintln(w, "The Bug creation had errors, unable to fetch Bug ID.")
				return
			}
			//Adding dependencies and blocked.
			dependbugs := strings.SplitAfter(r.FormValue("bug_depends_on"), ",")
			//fmt.Println(dependbugs)
			for index, _ := range dependbugs {
				//fmt.Println(dependbug)
				if dependbugs[index] != "" {
					dependbug_idint, _ := strconv.Atoi(strings.Trim(dependbugs[index], ","))
					// fmt.Println(dependbug_idint)
					valid, val := is_valid_bugdependency(bugid, dependbug_idint)
					fmt.Println(val)
					if valid {
						err := add_bug_dependency(bugid, dependbug_idint)
						if err != nil {
							fmt.Fprintln(w, err)
							return
						}
					} else {
						fmt.Fprintln(w, val)
						return
					}
				}

			}

			blockedbugs := strings.SplitAfter(r.FormValue("bug_blocks"), " ")
			//fmt.Println(blockedbugs)
			for index, _ := range blockedbugs {
				if blockedbugs[index] != "" {
					blockedbug_idint, _ := strconv.Atoi(strings.Trim(blockedbugs[index], ","))
					valid, val := is_valid_bugdependency(blockedbug_idint, bugid)
					if valid {
						err := add_bug_dependency(blockedbug_idint, bugid)
						if err != nil {
							fmt.Fprintln(w, err)
							return
						}
					} else {
						fmt.Fprintln(w, val)
						return
					}
				}

			}

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
	http.HandleFunc("/filebug/", createbug)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.ListenAndServe(":9999", nil)
}
