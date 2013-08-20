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

type Bug struct {
	id                                                                                                                            int
	reporter, qa, docs, component_id, status, version, severity, hardware, priority, whiteboard, summary, description, fixedinver interface{}
}

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

func update_redis(email string, password string, utype string, channel chan int) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		channel <- 1
		return
	}
	defer conn.Close()
	_, err = conn.Do("HSET", "users", email, password)
	_, err = conn.Do("HSET", "userstype", email, utype)

	if err != nil {
		// handle error
		fmt.Print(err)
		channel <- 1
		return
	}
	channel <- 1
}

/*
Loads all user details from the database into redis. This needs to called before
the server starts up.
*/
func load_users() {
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT email, password, type from users")
	if err == nil {
		var email, password, utype string
		c := make(chan int)
		for rows.Next() {
			err = rows.Scan(&email, &password, &utype)
			go update_redis(email, password, utype, c)
		}
	}
	defer rows.Close()
	fmt.Println("Users loaded.")
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
	update_mysql(name, email, user_type, mdstr)
	_ = <-c
	return "User added."

}

/*
Adds a new user into the database.
*/
func update_mysql(name string, email string, user_type string, password string) {
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
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	defer conn.Close()
	ret, err := conn.Do("HGET", "users", email)

	//val, err := redis.Values(ret, err)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}

	if string(ret.([]uint8)) == mdstr {
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

func new_bug(reporter int, summary string, description string, component_id int) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	defer db.Close()

	ret, err := db.Exec("INSERT INTO bugs (reporter, summary, description, component_id, reported) VALUES (?,?,?,?, NOW())", reporter, summary, description, component_id)
	if err != nil {
		fmt.Println(err.Error())
		return "Error in entering a new bug", err
	}
	rid, err := ret.LastInsertId()
	return strconv.FormatInt(rid, 10), err
}

/*
Updates a given bug based on input
*/
func update_bug(data map[string]interface{}) {
	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE bugs SET ")

	val, ok := data["status"]
	if ok {
		buffer.WriteString("status=?")
		vals = append(vals, val)
	}

	val, ok = data["version"]
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
	}
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
