package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Page struct {
	Email      string
	Flag_login bool
	Bug        Bug
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

			cookie, final_hash := getCookie(user)
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
				redis_hdel("sessions", words[0])
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

/*
Function for displaying the bug details.
*/
func showbug(w http.ResponseWriter, r *http.Request) {
	//perform any preliminary check if required.
	//backend_bug(w,r)
	il, useremail := is_logged(r)
	page := Page{Flag_login: il, Email: useremail}
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
		tml, err := get_template("./templates/showbug.html")
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
		page.Bug = interface_data
		err = tml.ExecuteTemplate(w, "base", page)
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
	http.HandleFunc("/bugs/", showbug)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.ListenAndServe(":9999", nil)
}
