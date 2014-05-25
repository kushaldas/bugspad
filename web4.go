package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"
	"strings"
	"strconv"
	"crypto/rand"
    "encoding/base64" 
    "github.com/kushaldas/openid.go/src/openid"
)

/*TODO This needs to be changed as it consumes memory.*/
var nonceStore = &openid.SimpleNonceStore{
	Store: make(map[string][]*openid.Nonce)}
var discoveryCache = &openid.SimpleDiscoveryCache{}

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

func getCookie (user string) (http.Cookie,string){
	hash := generate_hash()
	new_hash := get_hex(string(hash))
	expire := time.Now().AddDate(0, 0, 1)
	final_hash := user + ":" + new_hash
	cookie := http.Cookie{Name: "bugsuser", Value: final_hash, Path: "/", Expires: expire, MaxAge: 86400}
	return cookie, final_hash
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
	} else if r.Method == "POST" {
		//fmt.Println(r.Method)
		user := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		if authenticate_redis(user, password) {
			/*hash := generate_hash()
			new_hash := get_hex(string(hash))
			expire := time.Now().AddDate(0, 0, 1)
			final_hash := user + ":" + new_hash
			cookie := http.Cookie{Name: "bugsuser", Value: final_hash, Path: "/", Expires: expire, MaxAge: 86400}
			*/
			cookie,final_hash := getCookie(user)
			http.SetCookie(w, &cookie)
			redis_hset("sessions", user, final_hash)
			//setUserCookie(w,user)
			
			//Second style of template parsing.
			http.Redirect(w, r, "/", http.StatusFound)
			/*tml := template.Must(template.ParseFiles("templates/logout.html","templates/base.html"))
			
			user_type := User{Email: user}
			
			tml.ExecuteTemplate(w,"base", user_type)*/

		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

/*
Discovering the URL for openid 
*/
func openiddiscover(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.FormValue("id"))
	if url, err := openid.RedirectUrl(r.FormValue("id"),
		"http://localhost:9999/openidcallback",
		""); err == nil {
		http.Redirect(w, r, url, 303)
	} else {
		fmt.Println(err)
	}
}

/*
Callback function for openid.
*/
func openidcallback(w http.ResponseWriter, r *http.Request) {
	fullUrl := "http://localhost:9999" + r.URL.String()
	fmt.Println(fullUrl)
	id, err := openid.Verify(
		fullUrl,
		discoveryCache, nonceStore)
	if err == nil {
		//setCookie corresponding to the user, and redirect to homepage
		cookie,final_hash := getCookie(id["email"])
		http.SetCookie(w, &cookie)
		redis_hset("sessions", id["email"], final_hash)
		fmt.Println(id)
		http.Redirect(w, r, "/", http.StatusFound)
		
		
	} else {
		fmt.Println("WTF2")
		fmt.Print(err)
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
/*
Function for displaying the bug details.
*/
func showbug(w http.ResponseWriter, r *http.Request) {
	//perform any preliminary check if required.
	//backend_bug(w,r)
	il, useremail:= is_logged(r)
	bug_id := r.URL.Path[len("/showbug/"):]
    	if (r.Method == "GET" && bug_id!="") {
	    
		bug_data := get_bug(bug_id)
		tml, err := template.ParseFiles("./templates/showbug.html","./templates/base.html")
		if err != nil {
			checkError(err)
		}
		fmt.Println(bug_data["cclist"])
		bug_data["islogged"]=il
		bug_data["useremail"]=useremail
		fmt.Println(bug_data["reporter"])
		tml.ExecuteTemplate(w,"base", bug_data)
		comment_data := fetch_comments_by_bug(bug_id)
		tml.ExecuteTemplate(w,"comments_on_bug",map[string]interface{}{"comment_data":comment_data,"bug_id":bug_id})
		return
	    
	} else if r.Method == "POST"{
	    fmt.Println(r.FormValue("com_content"))
	    
	}
  /*
	fmt.Fprintln(w,"resp.Body: ?",resp.Body)   
	fmt.Fprintln(w,"body: "+string(body))
	json.Marshal(string(body),&res)
	fmt.Fprintln(w,"err: ?",err)
	//convert this to json and apply to the specific template
	//to_be_rendered by the template
*/}

/*
Frontend function for handling the commenting on 
a bug.
*/
func commentonbug(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST"{
	    il, _ := is_logged(r)
	    if il{
		user_id := get_user_id(r.FormValue("useremail"))
		bug_id,err := strconv.Atoi(r.FormValue("bug_id"))
		if err!=nil{
		    checkError(err)
		}
		_,err = new_comment(user_id, bug_id, r.FormValue("com_content"))
		if err!= nil {
		    checkError(err)
		}
		fmt.Println("hool")
		http.Redirect(w,r,"/showbug/"+r.FormValue("bug_id"),http.StatusFound)
	    //fmt.Println( r.FormValue("com_content"));
	    } else {
		http.Redirect(w,r,"/login",http.StatusFound)
		//fmt.Fprintln(w, "Illegal Operation!")
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
	il, useremail:= is_logged(r)
	if r.Method == "GET" {
	    tml, err := template.ParseFiles("./templates/createbug.html","./templates/base.html")
	    if err != nil {
		checkError(err)
	    }
	    if il{
	    fmt.Println(useremail)
		//fmt.Println(r.FormValue("username"))
		    

		allcomponents := get_all_components()
		fmt.Println(allcomponents)
		tml.ExecuteTemplate(w,"base", map[string]interface{}{"useremail":useremail,"islogged":il,/*"products":products_data,*/"components":allcomponents})
		return
	    } else {
		http.Redirect(w,r,"/login",http.StatusFound)
	    }
	} else if r.Method == "POST" {
	    if il{
		newbug := make(Bug)
		newbug["summary"]=r.FormValue("bug_title")
		newbug["whiteboard"]=r.FormValue("bug_whiteboard")
		newbug["severity"]=r.FormValue("bug_severity")
		newbug["hardware"]=r.FormValue("bug_hardware")
		newbug["version"]=r.FormValue("bug_version")
		newbug["description"]=r.FormValue("bug_description")
		newbug["priority"]=r.FormValue("bug_priority")
		newbug["component_id"]=r.FormValue("bug_component")
		newbug["reporter"]=get_user_id(useremail)
		id,err := new_bug(newbug)
		
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		bug_id, ok := strconv.ParseInt(id, 10, 32)
		if ok == nil {
			if newbug["emails"] != nil {
				add_bug_cc(bug_id, newbug["emails"])
			}
			http.Redirect(w,r,"/showbug/"+id,http.StatusFound)


		} else {
		        fmt.Fprintln(w, id)
		}
	    //fmt.Println( r.FormValue("com_content"));
	    }
	}
    
}

/*
An editing page for bug.
*/
func editbugpage(w http.ResponseWriter, r *http.Request) {

    	bug_id := r.URL.Path[len("/editbugpage/"):]
	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
			if (r.Method == "GET" && bug_id!="") {
			    tml, err := template.ParseFiles("./templates/editbugpage.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    tml.ExecuteTemplate(w,"base",interface_data)
			    bugdata := get_bug(bug_id)
			    if bugdata["error_msg"]!=nil{
				fmt.Fprintln(w,bugdata["error_msg"])
				return
			    }
			    fmt.Println(bugdata["summary"])
			    //productcomponents := 
			    tml.ExecuteTemplate(w,"bugdescription",bugdata)
			    

			} else if r.Method == "POST"{
			    	interface_data["id"]=r.FormValue("bug_id")
				fmt.Println(interface_data["id"])
				interface_data["status"]=r.FormValue("bug_status")
				interface_data["version"]=r.FormValue("bug_version")
				interface_data["hardware"]=r.FormValue("bug_hardware")
				interface_data["priority"]=r.FormValue("bug_priority")
				interface_data["fixedinver"]=r.FormValue("bug_fixedinver")
				interface_data["severity"]=r.FormValue("bug_fixedinver")
				
				err := update_bug(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,"Bug could not be updated!")
				    return
				}
				fmt.Fprintln(w,"Bug successfully updated!")
			
			}
	} else {
		http.Redirect(w,r,"/login",http.StatusFound)

	}
	    
    
}

func main() {
        load_config("config/bugspad.ini")
        // Load the user details into redis.
	load_users()
	http.HandleFunc("/", home)
	http.HandleFunc("/register/", registeruser)
	http.HandleFunc("/login", login)
	http.HandleFunc("/openidlogin", openiddiscover)
	http.HandleFunc("/openidcallback", openidcallback)
	http.HandleFunc("/logout/", logout)
	http.HandleFunc("/showbug/", showbug)
	http.HandleFunc("/commentonbug/", commentonbug)
	http.HandleFunc("/filebug/", createbug)
	http.HandleFunc("/editbugpage/", editbugpage)
	http.ListenAndServe(":9999", nil)
}
