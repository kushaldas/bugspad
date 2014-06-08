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
	interface_data := make(map[string]interface{}) 
	if r.Method == "GET" {
	    //fmt.Fprintln(w, "get")
	    il, useremail := is_logged(r)
	    //fmt.Println(il)
	    //fmt.Println(useremail)
	    interface_data["useremail"]=useremail
	    interface_data["islogged"]=il
	    interface_data["pagetitle"]="Home"
		//fmt.Println(r.FormValue("username"))
		    
	    tml, err := template.ParseFiles("./templates/home.html","./templates/base.html")
	    if err != nil {
		checkError(err)
	    }
	    tml.ExecuteTemplate(w,"base",interface_data)
	    return
	}
	    
}
/*
This function is the starting point for user authentication.
*/
func login(w http.ResponseWriter, r *http.Request) {

	interface_data := make(map[string]interface{}) 
	if r.Method == "GET" {
	
		//One style of template parsing.
		tml, err := template.ParseFiles("./templates/login.html","./templates/base.html")
		if err != nil {
			checkError(err)
		}
		interface_data["pagetitle"]="Login"
		tml.ExecuteTemplate(w,"base", interface_data)
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
	interface_data := make(map[string]interface{}) 
	if r.Method == "GET" {
	
		tml, err := template.ParseFiles("./templates/registeruser.html","./templates/base.html")
		if err != nil {
			checkError(err)
		}
		interface_data["pagetitle"]="Register"
		tml.ExecuteTemplate(w,"base", interface_data)
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
	interface_data := make(map[string]interface{}) 
	bug_id := r.URL.Path[len("/bugs/"):]
    	if (r.Method == "GET" && bug_id!="") {
	    
		interface_data = get_bug(bug_id)
		tml, err := template.ParseFiles("./templates/showbug.html","./templates/base.html")
		if err != nil {
			checkError(err)
		}
		//fmt.Println(bug_data["cclist"])
		comment_data := fetch_comments_by_bug(bug_id)
		interface_data["comment_data"]=comment_data
		interface_data["islogged"]=il
		interface_data["useremail"]=useremail
		interface_data["pagetitle"]="Bug - "+bug_id+" details"
		//fmt.Println(bug_data["reporter"])
		if il{
		    bug_product_id := get_product_of_component(interface_data["component_id"].(int))
		    if bug_product_id==-1{
			fmt.Fprintln(w,"error occurred!")
			return
		    }
		    allcomponents := get_components_by_id(bug_product_id)
		    allsubcomponents := get_subcomponents_by_component(interface_data["component_id"].(int))
		    interface_data["allcomponents"] = allcomponents
		    interface_data["allsubcomponents"] = allsubcomponents
		}
		err=tml.ExecuteTemplate(w,"base", interface_data)

		fmt.Println(err)
		

		//fmt.Println(comment_data)
		return
	    
	} else if r.Method == "POST"{
	    //fmt.Println(r.FormValue("com_content"))
	    
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

func commentonbug(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST"{
	    il, useremail := is_logged(r)
	    if il{
		user_id := get_user_id(useremail)
		bug_id,err := strconv.Atoi(r.FormValue("bug_id"))
		if err!=nil{
		    checkError(err)
		}
		_,err = new_comment(user_id, bug_id, r.FormValue("com_content"))
		if err!= nil {
		    checkError(err)
		}
		fmt.Println("hool")
		http.Redirect(w,r,"/bugs/"+r.FormValue("bug_id"),http.StatusFound)
	    //fmt.Println( r.FormValue("com_content"));
	    } else {
		http.Redirect(w,r,"/login",http.StatusFound)
		//fmt.Fprintln(w, "Illegal Operation!")
	    }
	}
	    
    
}
*/
/*
Function to handle product selection before filing a bug.
*/
func before_createbug(w http.ResponseWriter, r *http.Request) {
    	il, useremail:= is_logged(r)
	interface_data := make(map[string]interface{}) 
	if r.Method == "GET" {
	    tml, err := template.ParseFiles("./templates/filebug_product.html","./templates/base.html")
	    if err != nil {
		checkError(err)
	    }
	    if il{
	    	//fmt.Println(useremail)
		//fmt.Println(r.FormValue("username"))
		allproducts := get_all_products()
				interface_data["useremail"]=useremail
		interface_data["islogged"]=il
		interface_data["products"]=allproducts
		interface_data["pagetitle"]="Choose Product"
		//fmt.Println(allcomponents)
		tml.ExecuteTemplate(w,"base", interface_data)
		return
	    } else {
		http.Redirect(w,r,"/login",http.StatusFound)
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
	interface_data := make(map[string]interface{}) 
	il, useremail:= is_logged(r)
	if r.Method == "GET" {
	    product_id := r.URL.Path[len("/filebug/"):]
	    _,err:=strconv.ParseInt(product_id, 10, 32)
	    if err!=nil{
		    fmt.Fprintln(w,"You need to give a valid product for filing a bug!")
		    return
	    }
	    tml, err := template.ParseFiles("./templates/createbug.html","./templates/base.html")
	    if err != nil {
		checkError(err)
	    }
	    if il{
	    fmt.Println(useremail)
		//fmt.Println(r.FormValue("username"))
		prod_idint,_ := strconv.Atoi(product_id)
		allcomponents := get_components_by_id(prod_idint)
		interface_data["useremail"]=useremail
		interface_data["islogged"]=il
		interface_data["components"]=allcomponents
		interface_data["pagetitle"]="File Bug"
		
		//fmt.Println(allcomponents)
		tml.ExecuteTemplate(w,"base",interface_data)
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
			http.Redirect(w,r,"/bugs/"+id,http.StatusFound)


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

    	//bug_id := r.URL.Path[len("/editbugpage/"):]
	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
			/*if (r.Method == "GET" && bug_id!="") {
			    tml, err := template.ParseFiles("./templates/editbugpage.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Bug Page"
			    tml.ExecuteTemplate(w,"base",interface_data)
			    bugdata := get_bug(bug_id)
			    if bugdata["error_msg"]!=nil{
				fmt.Fprintln(w,bugdata["error_msg"])
				return
			    }
			    fmt.Println(bugdata["summary"])
			    //productcomponents := 
			    tml.ExecuteTemplate(w,"bugdescription",bugdata)
			    

			}*/if r.Method == "POST"{
			    	interface_data["id"]=r.FormValue("bug_id")
				interface_data["status"]=r.FormValue("bug_status")
				interface_data["version"]=r.FormValue("bug_version")
				interface_data["hardware"]=r.FormValue("bug_hardware")
				interface_data["priority"]=r.FormValue("bug_priority")
				interface_data["fixedinver"]=r.FormValue("bug_fixedinver")
				interface_data["severity"]=r.FormValue("bug_severity")
				interface_data["summary"]=r.FormValue("bug_title")
				interface_data["whiteboard"]=r.FormValue("bug_whiteboard")
				//fmt.Println(interface_data["status"])				
				//fmt.Println("dd")				
				interface_data["post_user"]=get_user_id(useremail)
				interface_data["com_content"]=r.FormValue("com_content")
				comp_idint,_:=strconv.Atoi(r.FormValue("bug_component"))
				subcomp_idint:=-1
				if r.FormValue("bug_subcomponent")!=""{
				    subcomp_idint,_=strconv.Atoi(r.FormValue("bug_subcomponent"))
				}
				fmt.Println(subcomp_idint)
				interface_data["subcomponent_id"]=subcomp_idint
				interface_data["component_id"]=comp_idint
				interface_data["component"]=get_component_name_by_id(comp_idint)
				interface_data["subcomponent"]=get_subcomponent_name_by_id(subcomp_idint)
				qaid := get_user_id(r.FormValue("bug_qa"))
				docsid := get_user_id(r.FormValue("bug_docs"))
				if (qaid ==-1 || docsid==-1) && (r.FormValue("bug_qa")!="") && (r.FormValue("bug_docs")!=""){
				    fmt.Fprintln(w,"Bug could not be updated!")
				    return
				}
				interface_data["qa"]=qaid
				interface_data["docs"]=docsid
				err := update_bug(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,"Bug could not be updated!")
				    return
				}
				
				//fmt.Fprintln(w,"Bug successfully updated!")
				http.Redirect(w,r,"/bugs/"+r.FormValue("bug_id"),http.StatusFound)
			
			}
	} else {
		http.Redirect(w,r,"/login",http.StatusFound)

	}
}

/*
Admin:: Homepage of the Admin interface.
*/
func admin(w http.ResponseWriter, r *http.Request) {

	    il, useremail := is_logged(r)
	    if il{
		    if is_user_admin(useremail){
			//anything should happen only if the user has admin rights			
			if r.Method == "GET" {
			    tml, err := template.ParseFiles("./templates/admin.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    interface_data := make(map[string]interface{})
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Admin"
			    tml.ExecuteTemplate(w,"base",interface_data)
			    
		    
			 } else if r.Method == "POST"{
				    
			 }
		    } else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
		    }
	    } else {
		    http.Redirect(w,r,"/login",http.StatusFound)
	    }
    
}

/*
Admin:: Product list.
*/
func editproducts(w http.ResponseWriter, r *http.Request) {

    	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})
	    if il{
		    if is_user_admin(useremail){
			if r.Method == "GET" {
			    tml, err := template.ParseFiles("./templates/editproducts.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    allproducts := get_all_products()
			    
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Products"
			    interface_data["productlist"]=allproducts
			    tml.ExecuteTemplate(w,"base",interface_data)
		     
			} else if r.Method == "POST"{
		
			}
		    } else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
		    }
	    } else {
		    http.Redirect(w,r,"/login",http.StatusFound)
	    }
}

/*
Admin:: A product description/editing page.
*/
func editproductpage(w http.ResponseWriter, r *http.Request) {

    	product_id := r.URL.Path[len("/editproductpage/"):]
	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
		    if is_user_admin(useremail){
			//anything should happen only if the user has admin rights
			if (r.Method == "GET" && product_id!="") {
			    tml, err := template.ParseFiles("./templates/editproductpage.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Product Page"
			    productdata := get_product_by_id(product_id)
			    if productdata["error_msg"]!=nil{
				fmt.Fprintln(w,productdata["error_msg"])
				return
			    }
			    interface_data["productname"] = productdata["name"]
			    interface_data["productdescription"] = productdata["description"]
			    //productcomponents := 
			    prod_idint,_ := strconv.Atoi(product_id)
			    interface_data["components"] = get_components_by_id(prod_idint)
			    //fmt.Println(productdata["components"])
			    interface_data["product_id"] = product_id
			    interface_data["bugs"],err = get_bugs_by_product(product_id)
			    if err!=nil{
				fmt.Fprintln(w,err)
				fmt.Println(err)
				return 
			    }
			    tml.ExecuteTemplate(w,"base",interface_data)
			    

			} else if r.Method == "POST"{
				//fmt.Println(r.FormValue("productname"))
				//fmt.Println(r.FormValue("productid"))
				//fmt.Println(r.FormValue("productdescription"))
			    	interface_data["name"]=r.FormValue("productname")   
				interface_data["description"]=r.FormValue("productdescription")
				interface_data["id"]=r.FormValue("productid")
				_,err := update_product(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,err)
				}
				http.Redirect(w,r,"/editproductpage/"+r.FormValue("productid"),http.StatusFound)
				//fmt.Fprintln(w,msg)
			}
		    } else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
		    }
	} else {
		http.Redirect(w,r,"/login",http.StatusFound)

	}
	    
    
}

/*
Admin:: User list.
*/
func editusers(w http.ResponseWriter, r *http.Request) {

    	il, useremail := is_logged(r)
	    if il{
		    if is_user_admin(useremail){
			    //anything should happen only if the user has admin rights
			    if r.Method == "GET" {
				tml, err := template.ParseFiles("./templates/editusers.html","./templates/base.html")
				if err != nil {
				    checkError(err)
				}
				allusers := get_all_users()
				interface_data := make(map[string]interface{})
				interface_data["islogged"]=il
				interface_data["useremail"]=useremail
				interface_data["pagetitle"]="Edit Users"
				interface_data["userlist"]=allusers
				tml.ExecuteTemplate(w,"base",interface_data)
		    
			    } else if r.Method == "POST"{
				    
			    }
		    } else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
		    }
	    } else {
		    http.Redirect(w,r,"/login",http.StatusFound)
	    }
    
}

/*
Admin:: A user description/editing page.
*/
func edituserpage(w http.ResponseWriter, r *http.Request) {

    	user_id := r.URL.Path[len("/edituserpage/"):]
    	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})
	    if il{
		    if is_user_admin(useremail){
			 //anything should happen only if the user has admin rights
			    if (r.Method == "GET" && user_id!="") {
				tml, err := template.ParseFiles("./templates/edituserpage.html","./templates/base.html")
				if err != nil {
				    checkError(err)
				}
				interface_data["islogged"]=il
				interface_data["useremail"]=useremail
				interface_data["pagetitle"]="Edit User Page"
				userdata := get_user_by_id(user_id)
				userdata["id"]=user_id
				if userdata["error_msg"]!=nil{
				    fmt.Fprintln(w,userdata["error_msg"])
				    return
				}
				interface_data["id"]=user_id
				interface_data["name"]=userdata["name"]
				interface_data["email"]=userdata["email"]
				interface_data["type"]=userdata["type"]
				tml.ExecuteTemplate(w,"base",interface_data)
			    
			    } else if r.Method == "POST"{
				    interface_data["name"]=r.FormValue("username")
				    interface_data["email"]=r.FormValue("useremail")
				    interface_data["type"]=r.FormValue("usertype")
				    interface_data["id"]=r.FormValue("userid")
				    msg,err := update_user(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,err)
				}
				fmt.Fprintln(w,msg)
			    }
		    } else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
		    }
	    } else {
		    http.Redirect(w,r,"/login",http.StatusFound)
	    }
    
}

/*
Admin:: A component adding page for a product.
*/
func addcomponentpage(w http.ResponseWriter, r *http.Request) {

    	product_id := r.URL.Path[len("/addcomponent/"):]
	il, useremail := is_logged(r)
	if il{
		if is_user_admin(useremail){
		//anything should happen only if the user has admin rights
		    if (r.Method == "GET" && product_id!="") {

					tml, err := template.ParseFiles("./templates/addcomponent.html","./templates/base.html")
					if err != nil {
					    checkError(err)
					}
					fmt.Print("inside")
					interface_data := make(map[string]interface{})
					interface_data["islogged"]=il
					interface_data["useremail"]=useremail
					interface_data["pagetitle"]="Add Component Page"
					interface_data["product_id"]=product_id
					err=tml.ExecuteTemplate(w,"base",interface_data)

		    } else if r.Method == "POST"{
			    qa := get_user_id(r.FormValue("qaname"))
			    if (qa==-1 && r.FormValue("qaname")!="") {
				fmt.Fprintln(w,"Invalid QA name")
			    }
			    owner := get_user_id(r.FormValue("ownername"))
			    if owner==-1{
				fmt.Fprintln(w,"Invalid Owner name")
			    }
			    product_id,_ := strconv.Atoi(r.FormValue("productid"))
			    id,err := insert_component(r.FormValue("componentname"), r.FormValue("componentdescription"), product_id, owner, qa)
			    fmt.Println("Component "+id+"added.")
			    if err!=nil {
				fmt.Fprintln(w,err)
			    }
			http.Redirect(w,r,"/addcomponent/"+r.FormValue("productid"),http.StatusFound)
		    }
		} else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
			
		}
	    
	} else {
	    http.Redirect(w,r,"/login",http.StatusFound)
	}
    
}

/*
Admin:: A component description/editing page.
*/
func editcomponentpage(w http.ResponseWriter, r *http.Request) {

    	component_id := r.URL.Path[len("/editcomponentpage/"):]
	il, useremail := is_logged(r)
    	interface_data := make(map[string]interface{})
	    if il{
		    if is_user_admin(useremail){
			//anything should happen only if the user has admin rights
			if (r.Method == "GET" && component_id!="") {
			    tml, err := template.ParseFiles("./templates/editcomponentpage.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Component Page"
			    interface_data["component_id"]=component_id
			    cdata := get_component_by_id(component_id)
			    if cdata["error_msg"]!=nil{
				fmt.Fprintln(w,cdata["error_msg"])
				return
			    }
			    interface_data["component_name"]=cdata["name"]
			    interface_data["component_qa"]=cdata["qa"]
			    interface_data["component_owner"]=cdata["owner"]
			    interface_data["component_description"]=cdata["description"]
			    comp_idint,err := strconv.Atoi(component_id)
			    interface_data["component_subs"]=get_subcomponents_by_component(comp_idint)
			    //fmt.Println(componentdata["error_msg"])
			    tml.ExecuteTemplate(w,"base",interface_data)
			    
		    
			} else if r.Method == "POST"{
				interface_data["name"]=r.FormValue("componentname")   
				interface_data["product_id"]=r.FormValue("componentproduct")
				u_id := -1
				if r.FormValue("componentqa")!=""{
				    u_id = get_user_id(r.FormValue("componentqa"))
				    if u_id != -1{
					interface_data["qa"]=u_id
				    } else {
					fmt.Fprintln(w,"Please specify a valid QA user!")
					return 
				    }
				}
				u_id = get_user_id(r.FormValue("componentowner"))
				if u_id != -1 {
				    interface_data["owner"]=u_id
				} else {
				    fmt.Fprintln(w,"Please specify a valid Owner!")
				    return
				}
				interface_data["description"]=r.FormValue("componentdescription")
				interface_data["id"]=r.FormValue("componentid")
				msg,err := update_component(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,err)
				}
				fmt.Fprintln(w,msg)
			}
		    } else {
			fmt.Fprintln(w,"You do not have sufficient rights!")
		    }
	    } else {
		    http.Redirect(w,r,"/login",http.StatusFound)
	    }
    
}

func editbugcc (w http.ResponseWriter, r *http.Request) {

    	bug_id := r.URL.Path[len("/editbugcc/"):]
    	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
		if (r.Method == "GET" && bug_id!="") {
			    tml, err := template.ParseFiles("./templates/editbugcc.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    bug_idint,_:=strconv.Atoi(bug_id)
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Bug "+bug_id+" CC"
			    interface_data["cclist"]=get_bugcc_list(bug_idint)
			    interface_data["id"]=bug_id
			    //productcomponents := 
			    tml.ExecuteTemplate(w,"base",interface_data)
			    

		}
		if r.Method == "POST"{
				r.ParseForm()
				emails_rem :=make([]interface{},0)
				emails_add :=make([]interface{},0)
				bug_idint64,_ := strconv.ParseInt(r.Form["bug_id"][0],10,64)
				for index,_ := range(r.Form["ccentry"]) {
				    emails_rem = append(emails_rem, r.Form["ccentry"][index])
				}
				emails_add = append(emails_add, r.Form["newccentry"][0])
				if add_bug_cc(bug_idint64, emails_add) && remove_bug_cc(bug_idint64, emails_rem){
				    http.Redirect(w,r,"/editbugcc/"+r.Form["bug_id"][0], http.StatusFound)
				} else {
				    fmt.Fprintln(w,"Bug CC could not be updated!")
				    return
				}
				//fmt.Fprintln(w,"Bug successfully updated!")
				//http.Redirect(w,r,"/editbugcc/"+bug_id, http.StatusFound)
			
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
	http.HandleFunc("/logout/", logout)
	http.HandleFunc("/bugs/", showbug)
//	http.HandleFunc("/commentonbug/", commentonbug)
	http.HandleFunc("/filebug/", createbug)
	http.HandleFunc("/filebug_product/", before_createbug)
	http.HandleFunc("/admin/", admin)
	http.HandleFunc("/editusers/", editusers)
	http.HandleFunc("/edituserpage/", edituserpage)
	http.HandleFunc("/editproductpage/", editproductpage)
	http.HandleFunc("/editproducts/", editproducts)
	http.HandleFunc("/editbugpage/", editbugpage)
	http.HandleFunc("/editbugcc/", editbugcc)
	http.HandleFunc("/editcomponentpage/", editcomponentpage)
	http.HandleFunc("/addcomponent/", addcomponentpage)
	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))
	//http.Handle("/css/", http.FileServer(http.Dir("css/style.css")))
	//http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css")))) 
	http.ListenAndServe(":9955", nil)
}
