package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Result1 map[string]string

func product(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tm := time.Now().UTC()
	defer log(r, tm)

	if r.Method == "POST" {
		// In case of wrong type of input we should recover.
		defer myrecover(w)
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]string)
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		user := pdata["user"]
		password := pdata["password"]
		name := pdata["name"]
		desc := pdata["description"]
		if authenticate_redis(user, password) {
			fmt.Println(user, password, name, desc)
			id, _ := insert_product(name, desc)
			res := Result1{"id": id, "name": name, "description": desc}
			res_json, _ := json.Marshal(res)
			fmt.Fprintln(w, string(res_json))

		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}

	}
}

func component(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tm := time.Now().UTC()
	defer log(r, tm)

	if r.Method == "POST" {
		// In case of wrong type of input we should recover.
		defer myrecover(w)
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]interface{})
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		user := pdata["user"].(string)
		password := pdata["password"].(string)
		name := pdata["name"].(string)
		desc := pdata["description"].(string)
		product_id := int(pdata["product_id"].(float64))
		owner := pdata["owner"].(string)
		qa := ""
		if pdata["qa"] != nil {
			qa = pdata["qa"].(string)
		}
		if authenticate_redis(user, password) {
			owner_id := get_user_id(owner)
			qa_id := get_user_id(qa)
			fmt.Println(user, password, name, desc, product_id, owner_id)
			id, _ := insert_component(name, desc, product_id, owner_id, qa_id)
			res := Result1{"id": id, "name": name, "description": desc}
			res_json, _ := json.Marshal(res)
			fmt.Fprintln(w, string(res_json))

		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}

	}
}

func components(w http.ResponseWriter, r *http.Request) {

	tm := time.Now().UTC()
	defer log(r, tm)

	product_id := ""
	if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]string)
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		// name := pdata["name"].(string)
		product_id = pdata["product_id"]

	} else if r.Method == "GET" {
		title := r.URL.Path[12:]

		if title == "" {
			return
		}

		index := strings.Index(title, "/")
		if index != -1 {
			title = title[:index]
		}
		product_id = title

	}
	if product_id != "" {
		w.Header().Set("Content-Type", "application/json")
		m := get_components_by_product(product_id)
		res_json, _ := json.Marshal(m)
		fmt.Fprintln(w, string(res_json))
	}
}

/*
Creates a new bug or gets the details of a bug.
*/
func backend_bug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tm := time.Now().UTC()
	defer log(r, tm)

	if r.Method == "POST" {
		// In case of wrong type of input we should recover.
		//defer myrecover(w)
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]interface{})
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}

		user := pdata["user"].(string)
		password := pdata["password"].(string)
		if authenticate_redis(user, password) {
			user_id := get_user_id(user)
			pdata["reporter"] = user_id
			id, err := new_bug(pdata)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			bug_id, ok := strconv.ParseInt(id, 10, 32)
			if ok == nil {
				if pdata["emails"] != nil {
					add_bug_cc(bug_id, pdata["emails"])
				}

			}

			fmt.Fprintln(w, id)
		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	} else if r.Method == "GET" {
		title := r.URL.Path[5:]

		if title == "" {
			return
		}

		index := strings.Index(title, "/")
		if index != -1 {
			title = title[:index]
		}
		data := get_bug(title)
		res_json, _ := json.Marshal(data)
		fmt.Fprintln(w, string(res_json))

	}
}

/*
Updates the content of a given bug.
*/
func updatebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tm := time.Now().UTC()
	defer log(r, tm)

	if r.Method == "POST" {
		// In case of wrong type of input we should recover.
		defer myrecover(w)
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]interface{})
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}

		user := pdata["user"].(string)
		password := pdata["password"].(string)
		if authenticate_redis(user, password) {
			update_bug(pdata)
		} else {
			fmt.Fprintln(w, AUTH_ERROR)
			return
		}
		fmt.Fprintln(w, SUCCESS)
	}
}

func comment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tm := time.Now().UTC()
	defer log(r, tm)

	if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]interface{})
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		user := pdata["user"].(string)
		password := pdata["password"].(string)
		desc := pdata["desc"].(string)
		bug_id := int(pdata["bug_id"].(float64))
		if authenticate_redis(user, password) {
			user_id := get_user_id(user)
			id, err := new_comment(user_id, bug_id, desc)
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Fprintln(w, id)
		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

/*
Adds or removes a new CC address to the bug.
*/
func bug_cc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tm := time.Now().UTC()
	defer log(r, tm)

	if r.Method == "POST" {
		// In case of wrong type of input we should recover.
		defer myrecover(w)
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]interface{})
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		user := pdata["user"].(string)
		password := pdata["password"].(string)
		if authenticate_redis(user, password) {
			bug_id := int64(pdata["bug_id"].(float64))
			emails := pdata["emails"]
			action := pdata["action"].(string)
			if action == "add" {
				add_bug_cc(bug_id, emails)
			} else if action == "remove" {
				remove_bug_cc(bug_id, emails)
			} else {
				fmt.Fprintln(w, "\"No vaild action provided.\"")
			}
		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

/*
API call to get latest 10 bugs from the server
*/
func latest_bugs(w http.ResponseWriter, r *http.Request) {

	tm := time.Now().UTC()
	defer log(r, tm)

	w.Header().Set("Content-Type", "application/json")
	vals := get_latest_created_list().([]interface{})
	m := make([]string, 0)
	if vals != nil {
		for i := range vals {
			bug_id := string(vals[i].([]uint8))
			json_data := redis_hget("bugs", bug_id)
			m = append(m, string(json_data))
		}
	}
	res_json, _ := json.Marshal(m)
	fmt.Fprintln(w, string(res_json))
}

/*
Find out the latest updated bugs.
Can be used in the frontpage of the application.
*/
func latest_updated_bugs(w http.ResponseWriter, r *http.Request) {

	tm := time.Now().UTC()
	defer log(r, tm)

	w.Header().Set("Content-Type", "application/json")
	vals := get_latest_updated_list().([]interface{})
	m := make([]string, 0)
	if vals != nil {
		for i := range vals {
			bug_id := string(vals[i].([]uint8))
			json_data := redis_hget("bugs", bug_id)
			m = append(m, string(json_data))
		}
	}
	res_json, _ := json.Marshal(m)
	fmt.Fprintln(w, string(res_json))
}

func releases(w http.ResponseWriter, r *http.Request) {

	tm := time.Now().UTC()
	defer log(r, tm)

	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		// In case of wrong type of input we should recover.
		defer myrecover(w)
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]interface{})
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		user := pdata["user"].(string)
		password := pdata["password"].(string)
		if authenticate_redis(user, password) {
			release_name := pdata["name"].(string)
			add_release(release_name)
			add_redis_release(release_name)
			fmt.Fprintln(w, SUCCESS)
		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
		return
	} else if r.Method == "GET" {
		vals := get_redis_release_list().([]interface{})
		releases := make([]string, 0)
		if vals != nil {
			for i := range vals {
				releases = append(releases, string(vals[i].([]uint8)))
			}
		}
		res_json, _ := json.Marshal(releases)
		fmt.Fprintln(w, string(res_json))
	}
}

// Main function of the application. This handles
// all entry points for the webapplication.
func main() {
	// First load the configuration details.
	load_config("config/bugspad.ini")
	// Load the user details into redis.
	load_users()
	http.HandleFunc("/component/", component)
	http.HandleFunc("/components/", components)
	http.HandleFunc("/product/", product)
	http.HandleFunc("/bug/", backend_bug)
	http.HandleFunc("/bug/cc/", bug_cc)
	http.HandleFunc("/updatebug/", updatebug)
	http.HandleFunc("/comment/", comment)
	http.HandleFunc("/latestcreated/", latest_bugs)
	http.HandleFunc("/latestupdated/", latest_updated_bugs)
	http.HandleFunc("/releases/", releases)

	http.ListenAndServe(":9998", nil)
}
