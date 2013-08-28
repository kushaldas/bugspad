package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Result1 map[string]string

const (
	AUTH_ERROR string = "\"Authentication failure.\""
)

func myrecover(w http.ResponseWriter) {
	if r := recover(); r != nil {
		fmt.Fprintln(w, "\"Wrong input.\"")
	}
}

func product(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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
		if authenticate_redis(user, password) {
			owner_id := get_user_id(owner)
			fmt.Println(user, password, name, desc, product_id, owner_id)
			id, _ := insert_component(name, desc, product_id, owner_id)
			res := Result1{"id": id, "name": name, "description": desc}
			res_json, _ := json.Marshal(res)
			fmt.Fprintln(w, string(res_json))

		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}

	}
}

func components(w http.ResponseWriter, r *http.Request) {
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
		m := get_components_by_id(product_id)
		res_json, _ := json.Marshal(m)
		fmt.Fprintln(w, string(res_json))
	}
}

func create_bug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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
	}
}

func updatebug(w http.ResponseWriter, r *http.Request) {
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
			update_bug(pdata)
		} else {
			fmt.Fprintln(w, AUTH_ERROR)
		}
	}
}

func comment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
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

func bug_cc(w http.ResponseWriter, r *http.Request) {
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

func latest_updated_bugs(w http.ResponseWriter, r *http.Request) {
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

func main() {
	load_config("config/bugspad.ini")
	load_users()
	http.HandleFunc("/component/", component)
	http.HandleFunc("/components/", components)
	http.HandleFunc("/product/", product)
	http.HandleFunc("/bug/", create_bug)
	http.HandleFunc("/bug/cc/", bug_cc)
	http.HandleFunc("/updatebug/", updatebug)
	http.HandleFunc("/comment/", comment)
	http.HandleFunc("/latestcreated/", latest_bugs)
	http.HandleFunc("/latestupdated/", latest_updated_bugs)
	http.ListenAndServe(":9998", nil)
}
