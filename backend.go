package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vaughan0/go-ini"
	"strconv"
)

var conn_str string

func load_config(filepath string) {
	file, _ := ini.LoadFile(filepath)
	db_user, _ := file.Get("bugspad", "user")
	db_pass, _ := file.Get("bugspad", "password")
	db_host, _ := file.Get("bugspad", "host")
	db_name, _ := file.Get("bugspad", "database")
	conn_str = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", db_user, db_pass, db_host, db_name)

}

func get_hex(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	md := hash.Sum(nil)
	mdstr := hex.EncodeToString(md)
	return mdstr
}

/* To find out if an user already exists or not. */
func find_user(email string) (bool, string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return true, "Redis error."
	}
	defer conn.Close()
	ret, err := conn.Do("HGET", "users", email)
	if ret != nil {
		return true, "User exists."
	}
	return false, "No such user."
}

/*
Adds a new user to the system. user_type can be 0,1,2. First it updates the
redis server and then saves the details to the MySQL db.
*/
func add_user(name string, email string, user_type string, password string) string {
	answer, res := find_user(email)
	if answer == true {
		return res
	}
	mdstr := get_hex(password)
	c := make(chan int)
	go update_redis(email, mdstr, user_type, c)
	fmt.Println(mdstr)
	add_user_mysql(name, email, user_type, mdstr)
	_ = <-c
	return "User added."

}

/*
Adds a new user into the database.
*/
func add_user_mysql(name string, email string, user_type string, password string) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
	defer db.Close()

	db.Exec("INSERT INTO users (name, email, type, password) VALUES (?, ?, ?, ?)", name, email, user_type, password)

}

/*
Inserts a new product into the database.
*/
func insert_product(name string, desc string) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	defer db.Close()

	ret, err := db.Exec("INSERT INTO products (name, description) VALUES (?, ?)", name, desc)
	if err != nil {
		return "", err
	}
	rid, err := ret.LastInsertId()
	return strconv.FormatInt(rid, 10), err
}

/*
Inserts a new component in the database for a given product_id.
*/
func insert_component(name string, desc string, product_id int, owner int) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	defer db.Close()

	ret, err := db.Exec("INSERT INTO components (name, description, product_id, owner) VALUES (?, ?, ?, ?)", name, desc, product_id, owner)
	if err != nil {
		fmt.Println(err.Error())
		return "No such product.", err
	}
	rid, err := ret.LastInsertId()
	return strconv.FormatInt(rid, 10), err
}

/*
Authenticate a given username (email) and password against the redis
db. The details are stored in a hashmap "users" in the redis db.
*/
func authenticate_redis(email string, password string) bool {
	mdstr := get_hex(password)
	ret := redis_hget("users", email)

	if string(ret) == mdstr {
		return true
	} else {
		return false
	}
}

func get_components_by_id(product_id string) map[string][3]string {
	m := make(map[string][3]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, description from components where product_id=?", product_id)
	if err != nil {
		return m
	}
	defer rows.Close()
	var name, description, c_id string
	for rows.Next() {
		err = rows.Scan(&c_id, &name, &description)
		//fmt.Println(c_id, name, description)
		m[name] = [3]string{c_id, name, description}
	}
	return m

}

func new_bug(data map[string]interface{}) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	defer db.Close()

	var status, summary string
	var buffer, buffer2 bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("INSERT INTO bugs (")

	val, ok := data["status"]
	if ok {
		buffer.WriteString("status")
		buffer2.WriteString("?")
		vals = append(vals, val)
		status = val.(string)
	} else {
		buffer.WriteString("status")
		buffer2.WriteString("'new'")
		status = "new"
	}

	val, ok = data["version"]
	if ok {
		buffer.WriteString(", version")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["severity"]
	if ok {
		buffer.WriteString(", severity")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["hardware"]
	if ok {
		buffer.WriteString(", hardware")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["priority"]
	if ok {
		buffer.WriteString(", priority")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["whiteboard"]
	if ok {
		buffer.WriteString(", whiteboard")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["subcomponent_id"]
	if ok {
		buffer.WriteString(", subcomponent_id")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["reporter"]
	if ok {
		buffer.WriteString(", reporter")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	} else {
		return "Missing input: reporter", nil
	}

	val, ok = data["summary"]
	if ok {
		buffer.WriteString(", summary")
		buffer2.WriteString(",?")
		vals = append(vals, val)
		summary = val.(string)
	} else {
		return "Missing input: summary", nil
	}

	val, ok = data["description"]
	if ok {
		buffer.WriteString(", description")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	} else {
		return "Missing input: description", nil
	}

	val, ok = data["component_id"]
	if ok {
		buffer.WriteString(", component_id")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	} else {
		return "Missing input: component_id", nil
	}

	buffer.WriteString(", reported) VALUES (")
	buffer2.WriteString(",NOW()")

	buffer.WriteString(buffer2.String())
	buffer.WriteString(")")

	ret, err := db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err.Error())
		return "Error in entering a new bug", err
	}

	rid, err := ret.LastInsertId()
	bug_id := strconv.FormatInt(rid, 10)
	// Now update redis cache for status
	update_redis_bug_status(bug_id, status)
	set_redis_bug(rid, status, summary)
	add_latest_created(bug_id)
	return bug_id, err
}

/*
Updates a given bug based on input
*/
func update_bug(data map[string]interface{}) {
	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE bugs SET ")
	bug_id := data["bug_id"].(float64)

	val1, ok1 := data["status"]
	if ok1 {
		buffer.WriteString("status=?")
		vals = append(vals, val1)
		// TODO
		// In case of status change we have to update the redis index for the bug
	} else {

		bug := get_redis_bug(strconv.FormatInt(int64(bug_id), 10))
		buffer.WriteString("status='")
		buffer.WriteString(bug["status"].(string))
		buffer.WriteString("'")
	}

	val, ok := data["version"]
	if ok {
		buffer.WriteString(", version=?")
		vals = append(vals, val)
	}

	val, ok = data["severity"]
	if ok {
		buffer.WriteString(", severity=?")
		vals = append(vals, val)
	}

	val, ok = data["hardware"]
	if ok {
		buffer.WriteString(", hardware=?")
		vals = append(vals, val)
	}

	val, ok = data["priority"]
	if ok {
		buffer.WriteString(", priority=?")
		vals = append(vals, val)
	}

	val, ok = data["reporter"]
	if ok {
		buffer.WriteString(", reporter=?")
		vals = append(vals, val)
	}

	val, ok = data["qa"]
	if ok {
		buffer.WriteString(", qa=?")
		vals = append(vals, val)
	}

	val, ok = data["docs"]
	if ok {
		buffer.WriteString(", docs=?")
		vals = append(vals, val)
	}

	val, ok = data["whiteboard"]
	if ok {
		buffer.WriteString(", whiteboard=?")
		vals = append(vals, val)
	}

	val, ok = data["subcomponent_id"]
	if ok {
		buffer.WriteString(", subcomponent_id=?")
		vals = append(vals, val)
	}

	val, ok = data["fixedinver"]
	if ok {
		buffer.WriteString(", fixedinver=?")
		vals = append(vals, val)
	}

	val, ok = data["component_id"]
	if ok {
		buffer.WriteString(", component_id=?")
		vals = append(vals, val)
	}

	buffer.WriteString(" WHERE id=?")
	fmt.Println(buffer.String())

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}
	defer db.Close()

	vals = append(vals, data["bug_id"])
	_, err = db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err)
		return
	}
	if ok1 {
		bug := get_redis_bug(strconv.FormatInt(int64(bug_id), 10))
		old_status := bug["status"].(string)
		delete_redis_bug_status(strconv.FormatInt(int64(bug_id), 10), old_status)
		update_redis_bug_status(strconv.FormatInt(int64(bug_id), 10), val1.(string))
	}
	add_latest_updated(strconv.FormatInt(int64(bug_id), 10))
}

/*
Adds new CC users to a bug
*/
func add_bug_cc(bug_id int64, emails interface{}) bool {
	email_list := emails.([]interface{})
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	defer db.Close()
	for i := range email_list {
		email := email_list[i].(string)
		user_id := get_user_id(email)
		// If user_id is -1 means no such user
		if user_id != -1 {
			db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, user_id)
		}
	}
	return true
}

/*
Removes CC users from a bug
*/
func remove_bug_cc(bug_id int64, emails interface{}) bool {
	email_list := emails.([]interface{})
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	defer db.Close()
	for i := range email_list {
		email := email_list[i].(string)
		user_id := get_user_id(email)
		// If user_id is -1 means no such user
		if user_id != -1 {
			_, err = db.Exec("DELETE FROM cc WHERE bug_id=? and who=?", bug_id, user_id)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return true
}

/*
Adds a new comment to a given bug.
*/
func new_comment(reporter int, bug_id int, desc string) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	defer db.Close()

	ret, err := db.Exec("INSERT INTO comments (description, user, datec, bug) VALUES (?,?,NOW(),?)", desc, reporter, bug_id)
	if err != nil {
		fmt.Println(err.Error())
		return "Error in entering a new comment", err
	}
	rid, err := ret.LastInsertId()
	return strconv.FormatInt(rid, 10), err
}

func get_user_id(email string) int {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return -1
	}
	defer db.Close()
	rows, err := db.Query("SELECT id from users where email = ?", email)
	if err == nil {
		var id int
		for rows.Next() {
			err = rows.Scan(&id)
			if id != 0 {
				return id
			}
		}
	}
	return -1
}

func add_release(name string) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO releases (name) VALUES (?)", name)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}

}

func get_releases() []string {
	m := make([]string, 0)
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return m
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM releases")
	if err != nil {
		return m
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		err = rows.Scan(&name)
		//fmt.Println(c_id, name, description)
		m = append(m, name)
	}
	return m
}
