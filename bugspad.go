package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Result1 map[string]string

func product(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
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
			fmt.Fprintln(w, "\"Authentication failure.\"")
		}

	}
}

func component(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
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
		product_id := pdata["product_id"]
		if authenticate_redis(user, password) {
			fmt.Println(user, password, name, desc, product_id)
			id, _ := insert_component(name, desc, product_id)
			res := Result1{"id": id, "name": name, "description": desc}
			res_json, _ := json.Marshal(res)
			fmt.Fprintln(w, string(res_json))

		} else {
			fmt.Fprintln(w, "\"Authentication failure.\"")
		}

	}
}

func components(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		pdata := make(map[string]string)
		err := decoder.Decode(&pdata)
		if err != nil {
			panic(err)
		}
		// name := pdata["name"].(string)
		product_id := pdata["product_id"]
		if product_id != "" {
			m := get_components_by_id(product_id)
			res_json, _ := json.Marshal(m)
			fmt.Fprintln(w, string(res_json))
		}

	}
}

func main() {
	load_config("config/bugspad.ini")
	load_users()
	fmt.Println(conn_str)
	http.HandleFunc("/component/", component)
	http.HandleFunc("/components/", components)
	http.HandleFunc("/product/", product)
	http.ListenAndServe(":9998", nil)
}
