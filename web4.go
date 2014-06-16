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
	"io"
	//"io/ioutil"
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
		    return
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
		if interface_data["id"] == nil {
		    	fmt.Fprintln(w,"Bug does not exist!")
			return
		}
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
		interface_data["dependencylist"] = bugs_dependent_on(bug_id)
		interface_data["blockedlist"] = bugs_blocked_by(bug_id)
		//fmt.Println(bug_data["reporter"])
		if il{
		    bug_product_id := get_product_of_component(interface_data["component_id"].(int))
		    if bug_product_id==-1{
			fmt.Fprintln(w,"Please specify the product!")
			return
		    }
		    allcomponents := get_components_by_id(bug_product_id)
		    allsubcomponents := get_subcomponents_by_component(interface_data["component_id"].(int))
		    interface_data["allcomponents"] = allcomponents
		    interface_data["attachments"] = get_bug_attachments(bug_id)
		    interface_data["versions"] = get_product_versions(bug_product_id)
		    interface_data["allsubcomponents"] = allsubcomponents
		}
		err=tml.ExecuteTemplate(w,"base", interface_data)

		fmt.Println(err)
		

		//fmt.Println(comment_data)
		return
	    
	} else if r.Method == "POST"{
	    //fmt.Println(r.FormValue("com_content"))
	    if r.Method == "POST"{
			    	interface_data["id"]=r.FormValue("bug_id")
				interface_data["status"]=r.FormValue("bug_status")
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
				assignid := get_user_id(r.FormValue("bug_assigned_to"))
				if (qaid ==-1 && (r.FormValue("bug_qa")!="")){
				    fmt.Fprintln(w,"Please enter correct QA entry.")
				    return
				}
				if (docsid==-1 && (r.FormValue("bug_docs")!="")) {
				    fmt.Fprintln(w,"Please enter correct Docs entry.")
				    return
				}
				if (assignid==-1 && (r.FormValue("bug_assigned_to")!="")){
				    fmt.Fprintln(w,"Please enter correct Assigned to entry.")
				    return
				}
				fixversion_id,_:=strconv.Atoi(r.FormValue("bug_fixedinver"))
				version_id,_:=strconv.Atoi(r.FormValue("bug_version"))
				if qaid!= -1 {
				    interface_data["qa"]=qaid
				}
				if docsid!= -1 {
				    interface_data["docs"]=docsid
				}
				interface_data["assigned_to"]=assignid
				interface_data["version"]=version_id
				interface_data["fixedinver"]=fixversion_id
				err := update_bug(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,"Bug could not be updated!")
				    return
				}
				
				//update dependencies
				currentbug_idint, _ := strconv.Atoi(r.FormValue("bug_id"))
				
				olddependencies:=""
				dep:=bugs_dependent_on(r.FormValue("bug_id"))
				for i,_ := range(dep) {
				    tmp:=strconv.Itoa(dep[i])
				    olddependencies=olddependencies+tmp+","
				}
				oldblocked:=""
				bloc:=bugs_blocked_by(r.FormValue("bug_id"))
				for i,_ := range(bloc) {
				    tmp:=strconv.Itoa(bloc[i])
				    oldblocked=oldblocked+tmp+","
				}
				clear_dependencies(currentbug_idint,"blocked")
				dependbugs:=strings.SplitAfter(r.FormValue("dependencylist"),",")
				//fmt.Println(dependbugs)
				for index,_ := range(dependbugs) {
					//fmt.Println(dependbug)
					if dependbugs[index]!=""{
					    dependbug_idint, _ := strconv.Atoi(strings.Trim(dependbugs[index],","))
					   // fmt.Println(dependbug_idint)
					    valid,val:=is_valid_bugdependency(currentbug_idint, dependbug_idint)
					    fmt.Println(val)
					    if valid {
						err:=add_bug_dependency(currentbug_idint, dependbug_idint)
						if err!=nil{
						    fmt.Fprintln(w,err)
						    return
						}
					    } else {
						fmt.Fprintln(w,val)
						return
					    }
					}
					
				}

				clear_dependencies(currentbug_idint,"dependson")
				blockedbugs:=strings.SplitAfter(r.FormValue("blockedlist"),",")
				//fmt.Println(blockedbugs)
				for index,_ := range(blockedbugs) {
					if blockedbugs[index]!="" {
					    blockedbug_idint, _ := strconv.Atoi(strings.Trim(blockedbugs[index],","))
					    valid,val:=is_valid_bugdependency(blockedbug_idint, currentbug_idint)
					    if valid {
						err:=add_bug_dependency(blockedbug_idint, currentbug_idint)
						if err!=nil{
						    fmt.Fprintln(w,err)
						    return
						}

					    } else {
						fmt.Fprintln(w,val)
						return
					    }
					}
					
				}
				
				newdependencies:=""
				dep=bugs_dependent_on(r.FormValue("bug_id"))
				for i,_ := range(dep) {
				    tmp:=strconv.Itoa(dep[i])
				    newdependencies=newdependencies+tmp+","
				}
				newblocked:=""
				bloc=bugs_blocked_by(r.FormValue("bug_id"))
				for i,_ := range(bloc) {
				    tmp:=strconv.Itoa(bloc[i])
				    newblocked=newblocked+tmp+","
				}
				net_comment:=""
				if olddependencies!=newdependencies {
				    net_comment=net_comment+htmlify(olddependencies,newdependencies,"depends on")
				}
				fmt.Println(oldblocked)
				fmt.Println(newblocked)
				if oldblocked!=newblocked {
				    net_comment=net_comment+htmlify(oldblocked,newblocked,"blocks")
				}//fmt.Fprintln(w,"Bug successfully updated!")
				if net_comment!=""{    
				    new_comment(interface_data["post_user"].(int),currentbug_idint,net_comment)
				}
				http.Redirect(w,r,"/bugs/"+r.FormValue("bug_id"),http.StatusFound)
			
			}
	    
	}
 }

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
		interface_data["versions"]=get_product_versions(prod_idint)
		fmt.Println(get_product_versions(prod_idint))
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
		newbug["description"]=r.FormValue("bug_description")
		newbug["priority"]=r.FormValue("bug_priority")
		newbug["component_id"]=r.FormValue("bug_component")
		newbug["reporter"]=get_user_id(useremail)
		docsint:=get_user_id(r.FormValue("bug_docs"))
		if docsint!=-1{
			newbug["docs"]=docsint
		} else {
			    fmt.Fprintln(w,"Please enter a valid user as Docs Maintainer.")
			    return
		}
		comp_idint,_ := strconv.Atoi(r.FormValue("bug_component"))
		version_int,_ := strconv.Atoi(r.FormValue("bug_version"))
		fmt.Println(version_int)
		if r.FormValue("bug_assignee")!="" {
			assignid := get_user_id(r.FormValue("bug_assignee"))
			if assignid==-1{
			    fmt.Fprintln(w,"Please enter a valid user as assignee")
			    return
			}
			newbug["assigned_to"]=assignid
		} else {//simply add the component owner as the assignee.
		    newbug["assigned_to"]=get_component_owner(comp_idint)
		}
		
 		newbug["version"]=version_int
		id,err := new_bug(newbug)
		
		if err != nil {
			fmt.Fprintln(w,err)
			return
		}
		//CC Addition
		bugccemails:=make([]interface{},0)
		ccs:=strings.SplitAfter(r.FormValue("bug_cc"),",")
		for index,_ := range(ccs) {
					//fmt.Println(dependbug)
					if ccs[index]!=""{
					    bugccemails=append(bugccemails,strings.Trim(ccs[index],","))
					}
		}
		bugid, ok := strconv.ParseInt(id, 10, 32)
		if ok == nil {
			if bugccemails!= nil {
			    if !add_bug_cc(bugid, bugccemails) {
				fmt.Fprintln(w,"Bug CC could not be updated, please check")
				return
			    }
			}
			http.Redirect(w,r,"/bugs/"+id,http.StatusFound)


		} else {
		        fmt.Fprintln(w, "The Bug creation had errors, unable to fetch Bug ID.")
			return
		}
		bug_id:=int(bugid)
		//Adding dependencies and blocked.
		    dependbugs:=strings.SplitAfter(r.FormValue("bug_depends_on"),",")
			    //fmt.Println(dependbugs)
		    for index,_ := range(dependbugs) {
		    //fmt.Println(dependbug)
			    if dependbugs[index]!=""{
				dependbug_idint, _ := strconv.Atoi(strings.Trim(dependbugs[index],","))
			    // fmt.Println(dependbug_idint)
				valid,val:=is_valid_bugdependency(bug_id, dependbug_idint)
				fmt.Println(val)
				if valid {
				    err:=add_bug_dependency(bug_id, dependbug_idint)
				    if err!=nil{
					fmt.Fprintln(w,err)
					return
				    }
				} else {
				    fmt.Fprintln(w,val)
				    return
				}
			    }
					    
		    }

		    blockedbugs:=strings.SplitAfter(r.FormValue("bug_blocks")," ")
				//fmt.Println(blockedbugs)
		    for index,_ := range(blockedbugs) {
			    if blockedbugs[index]!="" {
				blockedbug_idint, _ := strconv.Atoi(strings.Trim(blockedbugs[index],","))
				valid,val:=is_valid_bugdependency(blockedbug_idint, bug_id)
				if valid {
				    err:=add_bug_dependency(blockedbug_idint, bug_id)
				    if err!=nil{
					fmt.Fprintln(w,err)
					return
				    }
				} else {
				    fmt.Fprintln(w,val)
				    return
				}
			}
					
		    }
		
	    }
	}
    
}

/*
An editing page for bug.

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
			    

			}if r.Method == "POST"{
			    	interface_data["id"]=r.FormValue("bug_id")
				interface_data["status"]=r.FormValue("bug_status")
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
				assignid := get_user_id(r.FormValue("bug_assigned_to"))
				if (qaid ==-1 || docsid==-1 || assignid==-1) && (r.FormValue("bug_qa")!="") && (r.FormValue("bug_docs")!=""  && (r.FormValue("bug_assigned_to")!="")){
				    fmt.Fprintln(w,"Bug could not be updated!")
				    return
				}
				fixversion_id,_:=strconv.Atoi(r.FormValue("bug_fixedinver"))
				version_id,_:=strconv.Atoi(r.FormValue("bug_version"))
				if qaid!= -1 {
				    interface_data["qa"]=qaid
				}
				if docsid!= -1 {
				    interface_data["docs"]=docsid
				}
				interface_data["assigned_to"]=assignid
				interface_data["version"]=version_id
				interface_data["fixedinver"]=fixversion_id
				err := update_bug(interface_data)
				if err!=nil{
				    fmt.Fprintln(w,"Bug could not be updated!")
				    return
				}
				//update dependencies
				currentbug_idint, _ := strconv.Atoi(r.FormValue("bug_id"))

				clear_dependencies(currentbug_idint,"blocked")
				dependbugs:=strings.SplitAfter(r.FormValue("dependencylist"),",")
				//fmt.Println(dependbugs)
				for index,_ := range(dependbugs) {
					//fmt.Println(dependbug)
					if dependbugs[index]!=""{
					    dependbug_idint, _ := strconv.Atoi(strings.Trim(dependbugs[index],","))
					   // fmt.Println(dependbug_idint)
					    valid,val:=is_valid_bugdependency(currentbug_idint, dependbug_idint)
					    fmt.Println(val)
					    if valid {
						err:=add_bug_dependency(currentbug_idint, dependbug_idint)
						if err!=nil{
						    fmt.Fprintln(w,err)
						    return
						}
					    } else {
						fmt.Fprintln(w,val)
						return
					    }
					}
					
				}

				clear_dependencies(currentbug_idint,"dependson")
				blockedbugs:=strings.SplitAfter(r.FormValue("blockedlist")," ")
				//fmt.Println(blockedbugs)
				for index,_ := range(blockedbugs) {
					if blockedbugs[index]!="" {
					    blockedbug_idint, _ := strconv.Atoi(strings.Trim(blockedbugs[index],","))
					    valid,val:=is_valid_bugdependency(blockedbug_idint, currentbug_idint)
					    if valid {
						err:=add_bug_dependency(blockedbug_idint, currentbug_idint)
						if err!=nil{
						    fmt.Fprintln(w,err)
						    return
						}
					    } else {
						fmt.Fprintln(w,val)
						return
					    }
					}
					
				}
				interface_data["blockedlist"]=r.FormValue
				//fmt.Fprintln(w,"Bug successfully updated!")
				http.Redirect(w,r,"/bugs/"+r.FormValue("bug_id"),http.StatusFound)
			
			}
	} else {
		http.Redirect(w,r,"/login",http.StatusFound)

	}
}
*/
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
Admin:: Listing product versions
*/
func listproductversions(w http.ResponseWriter, r *http.Request) {
    	product_id := r.URL.Path[len("/listproductversions/"):]
    	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
		if (r.Method == "GET" && product_id!="") {
			    tml, err := template.ParseFiles("./templates/listproductversions.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    prod_idint,_:=strconv.Atoi(product_id)
			    interface_data=get_product_by_id(product_id)
			    if interface_data["id"] ==nil {
				fmt.Fprintln(w,"Product does not exist!")
				return
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Bug "+product_id+" CC"
			    //fmt.Println(get_product_versions(prod_idint))
			    interface_data["versions"]=get_product_versions(prod_idint)
			    interface_data["id"]=product_id
			    //productcomponents := 
			    err=tml.ExecuteTemplate(w,"base",interface_data)
			    fmt.Println(err)
			    

		}
	} else {
		    http.Redirect(w,r,"/login",http.StatusFound)
	}
}

/*
Admin:: Editing version of a product
*/
func editproductversion(w http.ResponseWriter, r *http.Request) {
    	version_id := r.URL.Path[len("/editproductversion/"):]
    	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
		if (r.Method == "GET" && version_id!="") {
			    tml, err := template.ParseFiles("./templates/editproductversion.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    ver_idint,_:=strconv.Atoi(version_id)
			    vertxt:=get_version_text(ver_idint)
			    if vertxt=="" {
				fmt.Fprintln(w,"Version does not exist!")
				return
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Version "+vertxt
			    interface_data["id"]=version_id
			    interface_data["value"]=vertxt
			    isact,_:=is_version_active(version_id)
			    if isact {
				interface_data["isactive"]=1
			    } else {
				interface_data["isactive"]=0
			    }
			    //productcomponents := 
			    err=tml.ExecuteTemplate(w,"base",interface_data)
			    fmt.Println(err)

		}
		if r.Method == "POST"{
				veractive:=0
				if r.FormValue("version_isactive")=="on" {
				    veractive=1
				}
				fmt.Println(veractive)
				err:=update_product_version(r.FormValue("version_value"), veractive, r.FormValue("version_id"))
				if err!=nil{
				    fmt.Fprintln(w,"Version could not be updated!")
				    return
				}
				    http.Redirect(w,r,"/editproductversion/"+r.FormValue("version_id"), http.StatusFound)
				//fmt.Fprintln(w,"Bug successfully updated!")
				//http.Redirect(w,r,"/editbugcc/"+bug_id, http.StatusFound)
			
		}
	} else {
		http.Redirect(w,r,"/login",http.StatusFound)

	}
}

/*
Admin:: Add product version
*/
func addproductversion (w http.ResponseWriter, r *http.Request) {
        //product_id := r.URL.Path[len("/addproductversion/"):]
    	il, _ := is_logged(r)
	//interface_data := make(map[string]interface{})    	
	if il{
		if r.Method == "POST"{
			err:=add_product_version(r.FormValue("product_id"),r.FormValue("newversionentry"))
			if err!=nil{
			    fmt.Fprintln(w,err)
			    return
			}
			http.Redirect(w,r,"/listproductversions/"+r.FormValue("product_id"), http.StatusFound)			
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
			    if productdata["id"] == nil {
				fmt.Fprintln(w, "Product does not exist!")
				return
			    }
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
				if userdata["id"] == nil {
				    fmt.Fprintln(w, "User does not exist!")
				    return
				}
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
			    if cdata["id"] == nil {
				fmt.Fprintln(w,"Component does not exist!")
				return 
			    }
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
			    tmp:=get_bug(bug_id)
			    if tmp["id"] ==nil {
				fmt.Fprintln(w,"Bug does not exist!")
				return
			    }
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
				    //fmt.Println(len(r.Form["ccem"]))
				    //fmt.Println(len(r.Form["ccentry"]))
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

func addattachment (w http.ResponseWriter, r *http.Request) {

    	bug_id := r.URL.Path[len("/addattachment/"):]
    	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})    	
	if il{
		if (r.Method == "GET" && bug_id!="") {
			    tml, err := template.ParseFiles("./templates/addattachment.html","./templates/base.html")
			    if err != nil {
				checkError(err)
			    }
			    tmp:=get_bug(bug_id)
			    if tmp["id"] ==nil {
				fmt.Fprintln(w,"Bug does not exist!")
				return
			    }
			    interface_data["islogged"]=il
			    interface_data["useremail"]=useremail
			    interface_data["pagetitle"]="Edit Bug Attachments"+bug_id+" CC"
			    interface_data["attachments"]=get_bug_attachments(bug_id)
			    interface_data["id"]=bug_id
			    //productcomponents := 
			    tml.ExecuteTemplate(w,"base",interface_data)
			    

		}
		if r.Method == "POST"{
				r.ParseMultipartForm(32 << 20)
				file, handler, err := r.FormFile("uploadfile")
				//reading the file and saving it
				    if err != nil {
					if err != http.ErrMissingFile {
					    fmt.Println(err)
					    fmt.Fprintln(w,err)
					    //file.Close()
					    return
					}
				    } else {
					//file.Close()
					defer file.Close()
					//fmt.Fprintf(w, "%v", handler.Header)
					cnt := count_entries("attachments")
					if cnt == -1{
					    fmt.Fprintln(w,"Error occured while counting entries of attachment table.")
					    return
					}
					count_str:=strconv.Itoa(cnt+1)
					filename := "attach_"+count_str+"_"+handler.Filename
					systempath := "resources/attachments/"+filename
					f, err := os.OpenFile(systempath, os.O_WRONLY|os.O_CREATE, 0666)
					if err != nil {
					    fmt.Fprintln(w,err)
					    return
					}
					defer f.Close()
					io.Copy(f, file)
					//now make an entry in the attachments table 
					interface_data["description"] = r.Form["attachment_desc"][0]
					interface_data["systempath"] = "/"+systempath
					interface_data["filename"] = filename 
					interface_data["submitter"] = get_user_id(useremail)
					interface_data["bug_id"] = r.Form["bug_id"][0]
					err = add_attachment(interface_data)
					
					if err!=nil {
					    fmt.Fprintln(w,err)
					    return
					}
					
				    }
				attach_obs :=make([]string,0)
				for index,_ := range(r.Form["attachments_obsolete"]) {
				    attach_obs = append(attach_obs, r.Form["attachments_obsolete"][index])
				}
				err=make_attachments_obsolete(attach_obs)
				if err!=nil {
				    fmt.Fprintln(w,err)
				    return
				}
				http.Redirect(w,r,"/addattachment/"+r.Form["bug_id"][0], http.StatusFound)
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
	http.HandleFunc("/addattachment/", addattachment)
	http.HandleFunc("/filebug_product/", before_createbug)
	http.HandleFunc("/admin/", admin)
	http.HandleFunc("/editusers/", editusers)
	http.HandleFunc("/edituserpage/", edituserpage)
	http.HandleFunc("/editproductpage/", editproductpage)
	http.HandleFunc("/editproducts/", editproducts)
	//http.HandleFunc("/editbugpage/", editbugpage)
	http.HandleFunc("/editbugcc/", editbugcc)
	http.HandleFunc("/editcomponentpage/", editcomponentpage)
	http.HandleFunc("/addcomponent/", addcomponentpage)
	http.HandleFunc("/addproductversion/", addproductversion)
	http.HandleFunc("/editproductversion/", editproductversion)
	http.HandleFunc("/listproductversions/", listproductversions)
	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))
	//http.Handle("/css/", http.FileServer(http.Dir("css/style.css")))
	//http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css")))) 
	http.ListenAndServe(":9955", nil)
}
