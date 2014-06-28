package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	//"reflect"
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

func getCookie(user string) (http.Cookie, string) {
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

/*
The home landing page of bugspad
*/
func home(w http.ResponseWriter, r *http.Request) {
	log_message(r, "Inside Home")
	interface_data := make(Bug)
	if r.Method == "GET" {
		il, useremail := is_logged(r)
		log_message(r, "Islogged"+useremail)
		interface_data["useremail"] = useremail
		interface_data["islogged"] = il
		interface_data["pagetitle"] = "Home"
		interface_data["is_user_admin"] = false
		if useremail != "" && il {
			log_message(r, "User logged in:"+useremail)
			interface_data["is_user_admin"] = is_user_admin(useremail)
			if interface_data["is_user_admin"].(bool) {
				log_message(r, "Admin logged:"+useremail)
			}
			interface_data["userbugs"] = get_user_bugs(get_user_id(useremail))
		}

		tml, err := template.ParseFiles("./templates/home.html", "./templates/base.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		tml.ExecuteTemplate(w, "base", interface_data)
		return
	}

}

/*
This function is the starting point for user authentication.
*/
func login(w http.ResponseWriter, r *http.Request) {

	interface_data := make(Bug)
	if r.Method == "GET" {

		tml, err := template.ParseFiles("./templates/login.html", "./templates/base.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		interface_data["pagetitle"] = "Login"
		err = tml.ExecuteTemplate(w, "base", interface_data)
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		return

	} else if r.Method == "POST" {

		user := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))

		if authenticate_redis(user, password) {
			cookie, final_hash := getCookie(user)
			http.SetCookie(w, &cookie)
			redis_hset("sessions", user, final_hash)
			http.Redirect(w, r, "/", http.StatusFound)

		} else {
			log_message(r, "Illegal Access:"+AUTH_ERROR)
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

/*
Logging out a user.
*/
func logout(w http.ResponseWriter, r *http.Request) {
	il, useremail := is_logged(r)
	if il {
		redis_hdel("sessions", useremail)
		log_message(r, "User logged out:"+useremail)
		http.Redirect(w, r, "/", http.StatusFound)
	}
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

		tml, err := template.ParseFiles("./templates/registeruser.html", "./templates/base.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		interface_data["pagetitle"] = "Register"
		err = tml.ExecuteTemplate(w, "base", interface_data)
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
Function for displaying the bug details.
*/
func showbug(w http.ResponseWriter, r *http.Request) {
	//perform any preliminary check if required.
	//backend_bug(w,r)
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	bugid := r.URL.Path[len("/bugs/"):]
	if r.Method == "GET" && bugid != "" {

		/*fetching the main bug content
		 ************************************************

		"id" = bug id
		"status" = bug status
		"summary" = bug summary
		"severity" = bug severity
		"description" = bug description
		"hardware" = bug hardware
		"priority" = bug priority
		"whiteboard" = bug whiteboard
		"reported" = bug reporting time UTC
		"reporter" = reporter user id
		"assigned_to" = assigned_to user id
		"qa" = quality Assurance user id
		"docs" = docs Maintainer user id
		"component_id" = bug component id
		"subcomponent_id" = bug subcomponent id ---> IGNORED
		"fixedinver" = version id of the version in which the bug was fixed (if it is fixed)
		"version" = version id of the component of the bug

		//extra fields for convenience
		"versiontext"
		"qaemail" = qa_email
		"qaname" = qa_name
		"docsemail" = docs_email
		"docsname" = docs_name
		"assigned_toname" = get_user_name(assigned_to)
		"assigned_toemail" = get_user_email(assigned_to)
		"reportername" = get_user_name(reporter)
		"reporteremail" = get_user_email(reporter)
		"component" = get_component_name_by_id(component_id)
		"subcomponent" = get_subcomponent_name_by_id(subcint)
		"fixedinvername" = get_version_text(fixedinverint)
		"cclist" = List of (emails,name) of CC members

		 ************************************************/
		bug_id, _ := strconv.Atoi(bugid)
		interface_data = get_bug(bug_id)

		//adding generic data
		interface_data["islogged"] = il
		interface_data["useremail"] = useremail
		interface_data["is_user_admin"] = is_user_admin(useremail)
		interface_data["pagetitle"] = "Bug - " + bugid + " details"
		//checking if the "id" interface{} is nil, ie the bug exists or not
		if interface_data["id"] == nil {
			fmt.Fprintln(w, "Bug does not exist!")
			return
		}
		tml, err := template.ParseFiles("./templates/showbug.html", "./templates/base.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		//Adding in the comments associated with the bug/
		interface_data["comment_data"] = fetch_comments_by_bug(bug_id)

		//Adding the bug dependency list for the bug.
		interface_data["dependencylist"] = bugs_dependent_on(bug_id)

		//Adding the bug blockage list for the bug.
		interface_data["blockedlist"] = bugs_blocked_by(bug_id)

		//Adding the original bug id if the bug is a duplicate
		dup := find_orig_ifdup(bug_id)
		if dup != -1 {
			interface_data["duplicateof"] = dup
		}

		//loggedin specific data
		if il {
			bug_product_id := get_product_of_component(interface_data["component_id"].(int))
			if bug_product_id == -1 {
				log_message(r, "Consistency Error:There is no product for the component "+strconv.Itoa(interface_data["component_id"].(int)))
				fmt.Fprintln(w, "No product exists for the component of the bug!")
				return
			}

			//fetching allcomponents of the currentproduct
			interface_data["allcomponents"] = get_components_by_product(bug_product_id)

			//fetching allsubcomponents of the
			interface_data["allsubcomponents"] = get_subcomponents_by_component(interface_data["component_id"].(int))

			//fetching all versions of the product
			interface_data["versions"] = get_product_versions(bug_product_id)

		}

		//Fetching the attachments related to the bug.
		interface_data["attachments"] = get_bug_attachments(bug_id)
		err = tml.ExecuteTemplate(w, "base", interface_data)
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		return

	} else if r.Method == "POST" {

		stats, severs, priors := get_redis_bugtags()

		//Checking if the Status, Severity and Priority are valid.
		if !isvalueinlist(r.FormValue("bug_status"), stats) {
			fmt.Fprintln(w, "Bug Status Invalid.")
			return
		}
		if strings.Contains(r.FormValue("bug_status"), "closed") {
			if r.FormValue("blockedlist") != "" {
				fmt.Fprintln(w, "This bug blocks other bugs and hence cannot be closed.")
				return
			}
		}
		interface_data["status"] = r.FormValue("bug_status")

		if !isvalueinlist(r.FormValue("bug_priority"), priors) {
			fmt.Fprintln(w, "Bug Priority Invalid.")
			return
		}
		interface_data["priority"] = r.FormValue("bug_priority")

		if !isvalueinlist(r.FormValue("bug_severity"), severs) {
			fmt.Fprintln(w, "Bug Severity Invalid.")
			return
		}
		interface_data["severity"] = r.FormValue("bug_severity")

		//checking if the fields are valid.
		tmp, err := strconv.Atoi(r.FormValue("bug_id"))
		if err != nil {
			fmt.Fprintln(w, "Bug id invalid.")
			return
		}
		interface_data["id"] = tmp
		/*		interface_data["subcomponent_id"],err = subcomp_idint
				if r.FormValue("bug_subcomponent") != "" && err!=nil {
					fmt.Fprintln(w, "Bug subcomponent invalid.")
					return
				}
		*/
		interface_data["component_id"], err = strconv.Atoi(r.FormValue("bug_component"))
		if err != nil {
			fmt.Fprintln(w, "Bug component invalid.")
			return
		}
		//interface_data["component_id"]=tmp

		interface_data["post_user"] = get_user_id(useremail)
		if interface_data["post_user"] == -1 {
			fmt.Fprintln(w, "Commenter is invalid.")
			return
		}
		interface_data["qa"] = get_user_id(r.FormValue("bug_qa"))
		if interface_data["qa"] == -1 && r.FormValue("bug_qa") != "" {
			fmt.Fprintln(w, "QA user is invalid.")
			return
		}
		interface_data["docs"] = get_user_id(r.FormValue("bug_docs"))
		if interface_data["docs"] == -1 && r.FormValue("bug_docs") != "" {
			fmt.Fprintln(w, "Docs user is invalid.")
			return
		}
		interface_data["assigned_to"] = get_user_id(r.FormValue("bug_assigned_to"))
		if interface_data["assigned_to"] == -1 && r.FormValue("bug_assigned_to") != "" {
			fmt.Fprintln(w, "Assignee user is invalid.")
			return
		}
		fixversion_id, _ := strconv.Atoi(r.FormValue("bug_fixedinver"))
		version_id, _ := strconv.Atoi(r.FormValue("bug_version"))
		if fixversion_id > 0 {
			interface_data["fixedinver"] = fixversion_id
		}
		if version_id > 0 {
			interface_data["version"] = version_id
		}

		interface_data["hardware"] = r.FormValue("bug_hardware")
		interface_data["summary"] = r.FormValue("bug_title")
		interface_data["whiteboard"] = r.FormValue("bug_whiteboard")
		interface_data["com_content"] = r.FormValue("com_content")

		/******update dependencies**********/
		//fetching old dependencies
		olddependencies := ""
		dep := bugs_dependent_on(interface_data["id"].(int))
		for i, _ := range dep {
			fmt.Println(i)
			tmp := strconv.Itoa(i)
			olddependencies = olddependencies + tmp + ","
		}
		//fetching old blockedby bugs
		oldblocks := ""
		bloc := bugs_blocked_by(interface_data["id"].(int))
		for i, _ := range bloc {
			tmp := strconv.Itoa(i)
			oldblocks = oldblocks + tmp + ","
		}

		clear_dependencies(interface_data["id"].(int), "blocked")
		dependbugs := strings.SplitAfter(r.FormValue("dependencylist"), ",")
		for index, _ := range dependbugs {
			if dependbugs[index] != "" {
				dependbug_idint, _ := strconv.Atoi(strings.Trim(dependbugs[index], ","))
				valid, err := is_valid_bugdependency(interface_data["id"].(int), dependbug_idint)
				if valid {
					err := add_bug_dependency(interface_data["id"].(int), dependbug_idint)
					if err != nil {
						fmt.Fprintln(w, err)
						return
					}
				} else {
					fmt.Fprintln(w, err)
					return
				}
			}

		}

		clear_dependencies(interface_data["id"].(int), "dependson")
		blockedbugs := strings.SplitAfter(r.FormValue("blockedlist"), ",")
		for index, _ := range blockedbugs {
			if blockedbugs[index] != "" {
				blockedbug_idint, _ := strconv.Atoi(strings.Trim(blockedbugs[index], ","))
				valid, val := is_valid_bugdependency(blockedbug_idint, interface_data["id"].(int))
				if valid {
					err := add_bug_dependency(blockedbug_idint, interface_data["id"].(int))
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

		newdependencies := ""
		dep = bugs_dependent_on(interface_data["id"].(int))
		for i, _ := range dep {
			tmp := strconv.Itoa(i)
			newdependencies = newdependencies + tmp + ","
		}
		newblocks := ""
		bloc = bugs_blocked_by(interface_data["id"].(int))
		for i, _ := range bloc {
			tmp := strconv.Itoa(i)
			newblocks = newblocks + tmp + ","
		}
		net_comment := ""
		if olddependencies != newdependencies {
			net_comment = net_comment + htmlify(olddependencies, newdependencies, "depends on")
		}

		if oldblocks != newblocks {
			net_comment = net_comment + htmlify(oldblocks, newblocks, "blocks")
		} //fmt.Fprintln(w,"Bug successfully updated!")

		/********dependencies updated**********/

		/*******duplicate changes if any***********/
		dupof, _ := strconv.Atoi(r.FormValue("bug_duplicate"))
		orig := find_orig_ifdup(dupof)
		if interface_data["status"].(string) == "duplicate" {
			if dupof != 0 {
				if orig != -1 {
					fmt.Fprintln(w, "Cant be duplicated as a duplicate of a duplicate.")
					return
				}
				//remove previous entry
				if remove_dup_bug(interface_data["id"].(int)) {
					//add new entry
					if !add_dup_bug(interface_data["id"].(int), dupof) {
						fmt.Fprintln(w, "Some error occured while updating duplicates.")
						return
					}
				} else {
					fmt.Fprintln(w, "Some error occured while updating duplicates.")
					return
				}
				sorg := strconv.Itoa(orig)
				net_comment = net_comment + htmlify(sorg, strconv.Itoa(dupof), "duplicateof")
			}
		} else {
			if !remove_dup_bug(interface_data["id"].(int)) {
				fmt.Fprintln(w, "Some error occured while updating duplicates.")
				return
			}
		}
		/*******duplicates done********/

		err = update_bug(interface_data)
		if err != nil {
			fmt.Fprintln(w, "Bug could not be updated!")
			return
		}
		if net_comment != "" {
			new_comment(interface_data["post_user"].(int), interface_data["id"].(int), net_comment)
		}
		http.Redirect(w, r, "/bugs/"+r.FormValue("bug_id"), http.StatusFound)

	}

}

/*
Function to handle product selection before filing a bug.
*/
func before_createbug(w http.ResponseWriter, r *http.Request) {
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if r.Method == "GET" {
		tml, err := template.ParseFiles("./templates/filebug_product.html", "./templates/base.html")
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
			err := tml.ExecuteTemplate(w, "base", interface_data)
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
	if r.Method == "GET" {
		product_id := r.URL.Path[len("/filebug/"):]
		_, err := strconv.ParseInt(product_id, 10, 32)
		if err != nil {
			fmt.Fprintln(w, "You need to give a valid product for filing a bug!")
			return
		}
		tml, err := template.ParseFiles("./templates/createbug.html", "./templates/base.html")
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
			fmt.Println(get_product_versions(prod_idint))
			tml.ExecuteTemplate(w, "base", interface_data)
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
			newbug["summary"] = r.FormValue("bug_title")
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

/*
Search interface function bugspad
*/
func searchbugs(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		m := make(Bug)
		tml, err := template.ParseFiles("./templates/searchbug.html", "./templates/base.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		m["products"] = get_all_products()
		m["components"] = get_all_components()
		m["searchresult"] = false
		m["pagetitle"] = "Search"
		err = tml.ExecuteTemplate(w, "base", m)
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}

	} else if r.Method == "POST" {
		tml, err := template.ParseFiles("./templates/searchbug.html", "./templates/base.html")
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
		r.ParseForm()
		searchbugs := search_redis_bugs(r.Form["bug_component"], r.Form["bug_product"], r.Form["bug_status"], r.Form["bug_version"], r.Form["bug_fixedinver"])
		fmt.Println(searchbugs)
		err = tml.ExecuteTemplate(w, "base", Bug{"searchbugs": searchbugs, "searchresult": true, "pagetitle": "Search"})
		if err != nil {
			log_message(r, "System Crash:"+err.Error())
		}
	}
}

/*
Admin:: Homepage of the Admin interface.
*/
func admin(w http.ResponseWriter, r *http.Request) {

	il, useremail := is_logged(r)
	if il {
		if is_user_admin(useremail) {
			//anything should happen only if the user has admin rights
			if r.Method == "GET" {
				tml, err := template.ParseFiles("./templates/admin.html", "./templates/base.html")
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}
				interface_data := make(Bug)
				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Admin"
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {

			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

}

/*
Admin:: Product list.
*/
func editproducts(w http.ResponseWriter, r *http.Request) {

	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if is_user_admin(useremail) {
			if r.Method == "GET" {
				tml, err := template.ParseFiles("./templates/editproducts.html", "./templates/base.html")
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}
				allproducts := get_all_products()

				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Edit Products"
				interface_data["productlist"] = allproducts
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {

			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

/*
Admin:: Listing product versions
*/
func listproductversions(w http.ResponseWriter, r *http.Request) {
	product_id := r.URL.Path[len("/listproductversions/"):]
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if r.Method == "GET" && product_id != "" {
			tml, err := template.ParseFiles("./templates/listproductversions.html", "./templates/base.html")
			if err != nil {
				log_message(r, "System Crash:"+err.Error())
			}
			prod_idint, _ := strconv.Atoi(product_id)
			interface_data = get_product_by_id(product_id)
			if interface_data["id"] == nil {
				fmt.Fprintln(w, "Product does not exist!")
				return
			}
			interface_data["islogged"] = il
			interface_data["useremail"] = useremail
			interface_data["is_user_admin"] = is_user_admin(useremail)
			interface_data["pagetitle"] = "Edit Bug " + product_id + " CC"
			interface_data["versions"] = get_product_versions(prod_idint)
			interface_data["id"] = product_id

			err = tml.ExecuteTemplate(w, "base", interface_data)
			if err != nil {
				log_message(r, "System Crash:"+err.Error())
			}
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

/*
Admin:: Editing version of a product
*/
func editproductversion(w http.ResponseWriter, r *http.Request) {
	version_id := r.URL.Path[len("/editproductversion/"):]
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if r.Method == "GET" && version_id != "" {
			tml, err := template.ParseFiles("./templates/editproductversion.html", "./templates/base.html")
			if err != nil {
				log_message(r, "System Crash:"+err.Error())
			}
			ver_idint, _ := strconv.Atoi(version_id)
			vertxt := get_version_text(ver_idint)
			if vertxt == "" {
				fmt.Fprintln(w, "Version does not exist!")
				return
			}
			interface_data["islogged"] = il
			interface_data["useremail"] = useremail
			interface_data["is_user_admin"] = is_user_admin(useremail)
			interface_data["pagetitle"] = "Edit Version " + vertxt
			interface_data["id"] = version_id
			interface_data["value"] = vertxt
			isact, _ := is_version_active(version_id)
			if isact {
				interface_data["isactive"] = 1
			} else {
				interface_data["isactive"] = 0
			}
			//productcomponents :=
			err = tml.ExecuteTemplate(w, "base", interface_data)
			if err != nil {
				log_message(r, "System Crash:"+err.Error())
			}
			fmt.Println(err)

		}
		if r.Method == "POST" {
			veractive := 0
			if r.FormValue("version_isactive") == "on" {
				veractive = 1
			}

			err := update_product_version(r.FormValue("version_value"), veractive, r.FormValue("version_id"))
			if err != nil {
				fmt.Fprintln(w, "Version could not be updated!")
				return
			}
			http.Redirect(w, r, "/editproductversion/"+r.FormValue("version_id"), http.StatusFound)

		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)

	}
}

/*
Admin:: Add product version
*/
func addproductversion(w http.ResponseWriter, r *http.Request) {

	il, _ := is_logged(r)
	if il {
		if r.Method == "POST" {
			err := add_product_version(r.FormValue("product_id"), r.FormValue("newversionentry"))
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}
			http.Redirect(w, r, "/listproductversions/"+r.FormValue("product_id"), http.StatusFound)
		}

	} else {
		http.Redirect(w, r, "/login", http.StatusFound)

	}
}

/*
Admin:: A product description/editing page.
*/
func editproductpage(w http.ResponseWriter, r *http.Request) {

	product_id := r.URL.Path[len("/editproductpage/"):]
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if is_user_admin(useremail) {
			//anything should happen only if the user has admin rights
			if r.Method == "GET" && product_id != "" {
				tml, err := template.ParseFiles("./templates/editproductpage.html", "./templates/base.html")
				if err != nil {
					checkError(err)
				}
				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Edit Product Page"
				productdata := get_product_by_id(product_id)
				if productdata["id"] == nil {
					fmt.Fprintln(w, "Product does not exist!")
					return
				}
				if productdata["error_msg"] != nil {
					fmt.Fprintln(w, productdata["error_msg"])
					return
				}
				interface_data["productname"] = productdata["name"]
				interface_data["productdescription"] = productdata["description"]
				prod_idint, _ := strconv.Atoi(product_id)
				interface_data["components"] = get_components_by_product(prod_idint)
				interface_data["product_id"] = product_id
				interface_data["bugs"], err = get_bugs_by_product(product_id)
				if err != nil {
					fmt.Fprintln(w, err)
					return
				}
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {

				interface_data["name"] = r.FormValue("productname")
				interface_data["description"] = r.FormValue("productdescription")
				interface_data["id"] = r.FormValue("productid")
				_, err := update_product(interface_data)
				if err != nil {
					fmt.Fprintln(w, err)
				}
				http.Redirect(w, r, "/editproductpage/"+r.FormValue("productid"), http.StatusFound)

			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)

	}

}

/*
Admin:: User list.
*/
func editusers(w http.ResponseWriter, r *http.Request) {

	il, useremail := is_logged(r)
	if il {
		if is_user_admin(useremail) {
			//anything should happen only if the user has admin rights
			if r.Method == "GET" {
				tml, err := template.ParseFiles("./templates/editusers.html", "./templates/base.html")
				if err != nil {
					checkError(err)
				}
				allusers := get_all_users()
				interface_data := make(map[string]interface{})
				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Edit Users"
				interface_data["userlist"] = allusers
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {
				r.ParseForm()
				for index, _ := range r.Form["inactiveusers"] {
					user_idint, _ := strconv.Atoi(r.Form["inactiveusers"][index])
					if !update_user_type(user_idint, "-1") {
						fmt.Fprintln(w, "User could not be made inactive!")
						return
					}
				}
				//reload the users in redis.
				load_users()
				http.Redirect(w, r, "/editusers", http.StatusFound)

			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

}

/*
Admin:: A user description/editing page.
*/
func edituserpage(w http.ResponseWriter, r *http.Request) {

	user_id := r.URL.Path[len("/edituserpage/"):]
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if is_user_admin(useremail) {
			//anything should happen only if the user has admin rights
			if r.Method == "GET" && user_id != "" {
				tml, err := template.ParseFiles("./templates/edituserpage.html", "./templates/base.html")
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}
				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Edit User Page"
				userdata := get_user_by_id(user_id)
				if userdata["id"] == nil {
					fmt.Fprintln(w, "User does not exist!")
					return
				}
				if userdata["error_msg"] != nil {
					fmt.Fprintln(w, userdata["error_msg"])
					return
				}
				interface_data["id"] = user_id
				interface_data["name"] = userdata["name"]
				interface_data["email"] = userdata["email"]
				interface_data["type"] = userdata["type"]
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {
				interface_data["name"] = r.FormValue("username")
				interface_data["email"] = r.FormValue("useremail")
				interface_data["type"] = r.FormValue("usertype")
				interface_data["id"] = r.FormValue("userid")
				msg, err := update_user(interface_data)
				if err != nil {
					fmt.Fprintln(w, err)
				}
				fmt.Fprintln(w, msg)
			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

}

/*
Admin:: A component adding page for a product.
*/
func addcomponentpage(w http.ResponseWriter, r *http.Request) {

	product_id := r.URL.Path[len("/addcomponent/"):]
	il, useremail := is_logged(r)
	if il {
		if is_user_admin(useremail) {
			//anything should happen only if the user has admin rights
			if r.Method == "GET" && product_id != "" {

				tml, err := template.ParseFiles("./templates/addcomponent.html", "./templates/base.html")
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}
				fmt.Print("inside")
				interface_data := make(map[string]interface{})
				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Add Component Page"
				interface_data["product_id"] = product_id
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {
				qa := get_user_id(r.FormValue("qaname"))
				if qa == -1 && r.FormValue("qaname") != "" {
					fmt.Fprintln(w, "Invalid QA name")
				}
				owner := get_user_id(r.FormValue("ownername"))
				if owner == -1 {
					fmt.Fprintln(w, "Invalid Owner name")
				}
				product_id, _ := strconv.Atoi(r.FormValue("productid"))
				id, err := insert_component(r.FormValue("componentname"), r.FormValue("componentdescription"), product_id, owner, qa)
				log_message(r, "Component "+id+"added.")
				if err != nil {
					fmt.Fprintln(w, err)
				}
				http.Redirect(w, r, "/addcomponent/"+r.FormValue("productid"), http.StatusFound)
			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")

		}

	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

}

/*
Admin:: A component description/editing page.
*/
func editcomponentpage(w http.ResponseWriter, r *http.Request) {

	component_id := r.URL.Path[len("/editcomponentpage/"):]
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if is_user_admin(useremail) {
			//anything should happen only if the user has admin rights
			if r.Method == "GET" && component_id != "" {
				tml, err := template.ParseFiles("./templates/editcomponentpage.html", "./templates/base.html")
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}
				interface_data["islogged"] = il
				interface_data["useremail"] = useremail
				interface_data["is_user_admin"] = is_user_admin(useremail)
				interface_data["pagetitle"] = "Edit Component Page"
				interface_data["component_id"] = component_id
				cdata := get_component_by_id(component_id)
				if cdata["id"] == nil {
					fmt.Fprintln(w, "Component does not exist!")
					return
				}
				if cdata["error_msg"] != nil {
					fmt.Fprintln(w, cdata["error_msg"])
					return
				}
				interface_data["component_name"] = cdata["name"]
				interface_data["component_qaname"] = cdata["qaname"]
				interface_data["component_ownername"] = cdata["ownername"]
				interface_data["component_description"] = cdata["description"]
				comp_idint, err := strconv.Atoi(component_id)
				interface_data["component_subs"] = get_subcomponents_by_component(comp_idint)
				//fmt.Println(componentdata["error_msg"])
				err = tml.ExecuteTemplate(w, "base", interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}

			} else if r.Method == "POST" {
				interface_data["name"] = r.FormValue("componentname")
				interface_data["product_id"] = r.FormValue("componentproduct")
				u_id := -1
				if r.FormValue("componentqa") != "" {
					u_id = get_user_id(r.FormValue("componentqa"))
					if u_id != -1 {
						interface_data["qa"] = u_id
					} else {
						fmt.Fprintln(w, "Please specify a valid QA user!")
						return
					}
				}
				u_id = get_user_id(r.FormValue("componentowner"))
				if u_id != -1 {
					interface_data["owner"] = u_id
				} else {
					fmt.Fprintln(w, "Please specify a valid Owner!")
					return
				}
				interface_data["description"] = r.FormValue("componentdescription")
				interface_data["id"] = r.FormValue("componentid")
				msg, err := update_component(interface_data)
				if err != nil {
					log_message(r, "System Crash:"+err.Error())
				}
				http.Redirect(w, r, "/editcomponentpage/"+r.FormValue("componentid"), http.StatusFound)
				fmt.Fprintln(w, msg)
			}
		} else {
			fmt.Fprintln(w, "You do not have sufficient rights!")
		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

}

func editbugcc(w http.ResponseWriter, r *http.Request) {

	bug_id := r.URL.Path[len("/editbugcc/"):]
	il, useremail := is_logged(r)
	interface_data := make(Bug)
	if il {
		if r.Method == "GET" && bug_id != "" {
			tml, err := template.ParseFiles("./templates/editbugcc.html", "./templates/base.html")
			if err != nil {
				checkError(err)
			}
			bug_idint, _ := strconv.Atoi(bug_id)
			tmp := get_bug(bug_idint)
			if tmp["id"] == nil {
				fmt.Fprintln(w, "Bug does not exist!")
				return
			}
			interface_data["islogged"] = il
			interface_data["useremail"] = useremail
			interface_data["is_user_admin"] = is_user_admin(useremail)
			interface_data["pagetitle"] = "Edit Bug " + bug_id + " CC"
			interface_data["cclist"] = get_bugcc_list(bug_idint)
			interface_data["id"] = bug_id
			//productcomponents :=
			tml.ExecuteTemplate(w, "base", interface_data)

		}
		if r.Method == "POST" {
			r.ParseForm()
			emails_rem := make([]interface{}, 0)
			emails_add := make([]interface{}, 0)
			bug_idint64, _ := strconv.ParseInt(r.Form["bug_id"][0], 10, 64)
			//fmt.Println(len(r.Form["ccem"]))
			//fmt.Println(len(r.Form["ccentry"]))
			for index, _ := range r.Form["ccentry"] {
				emails_rem = append(emails_rem, r.Form["ccentry"][index])
			}
			emails_add = append(emails_add, r.Form["newccentry"][0])
			if add_bug_cc(bug_idint64, emails_add) && remove_bug_cc(bug_idint64, emails_rem) {
				http.Redirect(w, r, "/editbugcc/"+r.Form["bug_id"][0], http.StatusFound)
			} else {
				fmt.Fprintln(w, "Bug CC could not be updated!")
				return
			}
			//fmt.Fprintln(w,"Bug successfully updated!")
			//http.Redirect(w,r,"/editbugcc/"+bug_id, http.StatusFound)

		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)

	}
}

func addattachment(w http.ResponseWriter, r *http.Request) {

	bug_id := r.URL.Path[len("/addattachment/"):]
	il, useremail := is_logged(r)
	interface_data := make(map[string]interface{})
	if il {
		if r.Method == "GET" && bug_id != "" {
			tml, err := template.ParseFiles("./templates/addattachment.html", "./templates/base.html")
			if err != nil {
				checkError(err)
			}
			bug_idint, _ := strconv.Atoi(bug_id)
			tmp := get_bug(bug_idint)
			if tmp["id"] == nil {
				fmt.Fprintln(w, "Bug does not exist!")
				return
			}
			interface_data["islogged"] = il
			interface_data["useremail"] = useremail
			interface_data["is_user_admin"] = is_user_admin(useremail)
			interface_data["pagetitle"] = "Edit Bug Attachments" + bug_id + " CC"
			interface_data["attachments"] = get_bug_attachments(bug_idint)
			interface_data["id"] = bug_id
			//productcomponents :=
			tml.ExecuteTemplate(w, "base", interface_data)

		}
		if r.Method == "POST" {
			r.ParseMultipartForm(32 << 20)
			file, handler, err := r.FormFile("uploadfile")
			//reading the file and saving it
			if err != nil {
				if err != http.ErrMissingFile {
					fmt.Println(err)
					fmt.Fprintln(w, err)
					//file.Close()
					return
				}
			} else {
				//file.Close()
				defer file.Close()
				//fmt.Fprintf(w, "%v", handler.Header)
				cnt := count_entries("attachments")
				if cnt == -1 {
					fmt.Fprintln(w, "Error occured while counting entries of attachment table.")
					return
				}
				count_str := strconv.Itoa(cnt + 1)
				filename := "attach_" + count_str + "_" + handler.Filename
				systempath := "resources/attachments/" + filename
				f, err := os.OpenFile(systempath, os.O_WRONLY|os.O_CREATE, 0666)
				if err != nil {
					fmt.Fprintln(w, err)
					return
				}
				defer f.Close()
				io.Copy(f, file)
				//now make an entry in the attachments table
				interface_data["description"] = r.Form["attachment_desc"][0]
				interface_data["systempath"] = "/" + systempath
				interface_data["filename"] = filename
				interface_data["submitter"] = get_user_id(useremail)
				interface_data["bug_id"] = r.Form["bug_id"][0]
				err = add_attachment(interface_data)

				if err != nil {
					fmt.Fprintln(w, err)
					return
				}

			}
			attach_obs := make([]string, 0)
			for index, _ := range r.Form["attachments_obsolete"] {
				attach_obs = append(attach_obs, r.Form["attachments_obsolete"][index])
			}
			err = make_attachments_obsolete(attach_obs)
			if err != nil {
				fmt.Fprintln(w, err)
				return
			}
			http.Redirect(w, r, "/addattachment/"+r.Form["bug_id"][0], http.StatusFound)
			//fmt.Fprintln(w,"Bug successfully updated!")
			//http.Redirect(w,r,"/editbugcc/"+bug_id, http.StatusFound)

		}
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)

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

	//handling the urls and their handler functions.
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
	http.HandleFunc("/searchbugs/", searchbugs)
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
