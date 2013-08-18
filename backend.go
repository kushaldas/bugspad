package main

import (
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

func update_redis(email string, password string, channel chan int) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		channel <- 1
		return
	}
	defer conn.Close()
	_, err = conn.Do("HSET", "users", email, password)

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
	rows, err := db.Query("SELECT email, password from users")
	if err == nil {
		var email, password string
		c := make(chan int)
		for rows.Next() {
			err = rows.Scan(&email, &password)
			go update_redis(email, password, c)
		}
	}
	fmt.Println("Users loaded.")
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
func insert_component(name string, desc string, product_id string) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	defer db.Close()

	ret, err := db.Exec("INSERT INTO components (name, description, product_id) VALUES (?, ?, ?)", name, desc, product_id)
	if err != nil {
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

/*
Adds a new user to the system. user_type can be 0,1,2. First it updates the
redis server and then saves the details to the MySQL db.
*/
func add_user(name string, email string, user_type string, password string) {
	mdstr := get_hex(password)
	c := make(chan int)
	go update_redis(email, mdstr, c)
	fmt.Println(mdstr)
	update_mysql(name, email, user_type, mdstr)
	_ = <-c

}

func get_components_by_id(product_id string) map[string][3]string {
	m := make(map[string][3]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, description from components where product_id=?", product_id)
	if err != nil {
		return m
	}
	var name, description, c_id string
	for rows.Next() {
		err = rows.Scan(&c_id, &name, &description)
		//fmt.Println(c_id, name, description)
		m[name] = [3]string{c_id, name, description}
	}
	return m

}
