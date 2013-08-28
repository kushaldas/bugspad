package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
)

type Bug map[string]interface{}

// TODO: Update this codebase to keep all search related indexs on redis.

// Generic function to update a redis HASH
func redis_hset(name, key, value string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	defer conn.Close()
	_, err = conn.Do("HSET", name, key, value)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
}

// Generic function to get a redis HASH
func redis_hget(name, key string) []byte {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return nil
	}

	defer conn.Close()
	val, err := conn.Do("HGET", name, key)
	if err != nil || val == nil {
		// handle error
		fmt.Print(err)
		return nil
	}
	return val.([]uint8)
}

func update_redis_bug_status(bug_id string, status string) {
	redis_hset("b_status:"+status, bug_id, "1")
}

func delete_redis_bug_status(bug_id string, status string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	defer conn.Close()
	conn.Do("HDEL", "b_status:"+status, bug_id)
}

func get_redis_bug(bug_id string) Bug {
	m := make(Bug)
	data := redis_hget("bugs", bug_id)
	if data == nil {
		fmt.Println("sorry no data")
		return nil
	}
	err := json.Unmarshal(data, &m)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return m

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

func add_latest_created(bug_id string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
	defer conn.Close()
	_, err = conn.Do("LPUSH", "latest_created", bug_id)
	_, err = conn.Do("LTRIM", "latest_created", 0, 9)

}

/*
This function returns a slice of latest created bugs (last 10)
*/
func get_latest_created_list() interface{} {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return nil
	}
	val, err := conn.Do("LRANGE", "latest_created", 0, 9)
	if err != nil {
		// handle error
		fmt.Println(err)
		return nil
	}
	return val
}

func add_latest_updated(bug_id string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
	defer conn.Close()
	_, err = conn.Do("LPUSH", "latest_updated", bug_id)
	_, err = conn.Do("LTRIM", "latest_updated", 0, 9)

}

func get_latest_updated_list() interface{} {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return nil
	}
	val, err := conn.Do("LRANGE", "latest_updated", 0, 9)
	if err != nil {
		// handle error
		fmt.Println(err)
		return nil
	}
	return val
}

func set_redis_bug(id int64, status, summary string) {
	m := make(Bug)
	m["id"] = id
	m["status"] = status
	m["summary"] = summary
	data, _ := json.Marshal(m)
	sdata := string(data)
	sid := strconv.FormatInt(id, 10)
	redis_hset("bugs", sid, sdata)
}
