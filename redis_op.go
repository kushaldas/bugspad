package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vaughan0/go-ini"
	"strconv"
	"strings"
)

type Bug map[string]interface{}

// TODO: Update this codebase to keep all search related indexs on redis.

// Generic function to delete a redis HASH
func redis_hdel(name, key string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	defer conn.Close()
	_, err = conn.Do("HDEL", name, key)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
}

//Function to update search sets for the bug.

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
	//fmt.Println("hset result")
	//fmt.Println(value)
	//fmt.Println(val)
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

//Generic function to add into a redis set
func redis_sadd(name, value string) error {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return nil
	}

	defer conn.Close()
	val, err := conn.Do("SADD", name, value)
	if err != nil || val == nil {
		// handle error
		fmt.Print(err)
		return nil
	}
	return err
}

//Generic function to remove from a redis set
func redis_srem(name, value string) error {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return nil
	}

	defer conn.Close()
	_, err = conn.Do("SREM", name, value)
	if err != nil {
		// handle error
		fmt.Print(err)
	}
	return err
}

//Generic function to check if an element exist in a redis set
//Time complexity: O(1)
func redis_sismember(name, value string) int {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return -1
	}
	val, err := conn.Do("SISMEMBER", name, value)
	if err != nil {
		fmt.Println(err)
		return -1
	}
	return val.(int)

}

//Function for deleting a DataStructure in redis
func redis_del(name string) int {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return -1
	}
	val, err := conn.Do("DEL", name)
	if err != nil {
		// handle error
		fmt.Print(err)
		return -1
	}
	return val.(int)
}

//Generic function to get all entries from a redis set
func redis_smembers(name string) interface{} {
	conn, err := redis.Dial("tcp", ":6379")
	m := make([]interface{}, 0)
	if err != nil {
		// handle error
		fmt.Println(err)
		return m
	}

	defer conn.Close()
	val, err := conn.Do("SMEMBERS", name)
	if err != nil || val == nil {
		// handle error
		fmt.Print(err)
		return m
	}
	return val
	//m=append(val,m)
	/*	for i,_ := range(val) {
		    m = append(m,val[i].([]uint8))
		}
	*/
	//return m
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
Searching bugs.
Currently returns the union of bugs
*/
func search_redis_bugs(components []string, products []string, statuses []string, versions []string, fixedinvers []string) map[int][2]string {
	ans := make(map[int][2]string)

	//conn, err := redis.Dial("tcp", ":6379")
	for index, _ := range components {
		bugids := redis_smembers("componentbug:" + components[index])
		bugidlist := bugids.([]interface{})
		for j, _ := range bugidlist {
			tmp := get_redis_bug(string(bugidlist[j].([]uint8)))
			bugid := int(tmp["id"].(float64))
			ans[bugid] = [2]string{tmp["summary"].(string), tmp["status"].(string)}
		}
	}
	for index, _ := range products {
		bugids := redis_smembers("productbug:" + products[index])
		//fmt.Println(bugids)
		bugidlist := bugids.([]interface{})
		for j, _ := range bugidlist {
			tmp := get_redis_bug(string(bugidlist[j].([]uint8)))
			//fmt.Println(string(bugidlist[j].([]uint8)))
			bugid := int(tmp["id"].(float64))
			ans[bugid] = [2]string{tmp["summary"].(string), tmp["status"].(string)}
		}
	}
	for index, _ := range statuses {
		bugids := redis_smembers("statusbug:" + statuses[index])
		bugidlist := bugids.([]interface{})
		for j, _ := range bugidlist {
			tmp := get_redis_bug(string(bugidlist[j].([]uint8)))
			bugid := int(tmp["id"].(float64))
			ans[bugid] = [2]string{tmp["summary"].(string), tmp["status"].(string)}
		}
	}
	for index, _ := range versions {
		bugids := redis_smembers("versionbug:" + versions[index])
		bugidlist := bugids.([]interface{})
		for j, _ := range bugidlist {
			tmp := get_redis_bug(string(bugidlist[j].([]uint8)))
			bugid := int(tmp["id"].(float64))
			ans[bugid] = [2]string{tmp["summary"].(string), tmp["status"].(string)}
		}
	}
	for index, _ := range fixedinvers {
		bugids := redis_smembers("fixedinverbug:" + fixedinvers[index])
		bugidlist := bugids.([]interface{})
		for j, _ := range bugidlist {
			tmp := get_redis_bug(string(bugidlist[j].([]uint8)))
			bugid := int(tmp["id"].(float64))
			ans[bugid] = [2]string{tmp["summary"].(string), tmp["status"].(string)}
		}
	}
	return ans
}

/*
Loads all user details from the database into redis. This needs to called before
the server starts up.
*/
func load_users() {
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, email, password, type from users")
	if err == nil {
		var email, password, utype string
		var id int64
		c := make(chan int)
		for rows.Next() {
			err = rows.Scan(&id, &email, &password, &utype)
			go update_redis(id, email, password, utype, c)
		}
	}
	defer rows.Close()
	fmt.Println("Users loaded.")
}

//We are not using a set here and instead a hash, since we do not need
//frequent insertion deletions, and we just need to compare from a standard
//list.
func load_bugtags() {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}
	file, _ := ini.LoadFile("config/static.ini")
	statuses, _ := file.Get("bugspad", "statuses")
	severities, _ := file.Get("bugspad", "severities")
	priorities, _ := file.Get("bugspad", "priorities")
	_, err = conn.Do("HSET", "tags", "statuses", statuses)
	_, err = conn.Do("HSET", "tags", "severities", severities)
	_, err = conn.Do("HSET", "tags", "priorities", priorities)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}

	return
}

func get_redis_bugtags() ([]string, []string, []string) {
	m1 := make([]string, 0)
	m2 := make([]string, 0)
	m3 := make([]string, 0)
	dec := json.NewDecoder(strings.NewReader(string(redis_hget("tags", "statuses"))))
	dec.Decode(&m1)
	dec = json.NewDecoder(strings.NewReader(string(redis_hget("tags", "severities"))))
	dec.Decode(&m2)
	dec = json.NewDecoder(strings.NewReader(string(redis_hget("tags", "priorities"))))
	dec.Decode(&m3)
	return m1, m2, m3
}

func update_redis(id int64, email string, password string, utype string, channel chan int) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		channel <- 1
		return
	}
	fmt.Println(email)
	defer conn.Close()
	_, err = conn.Do("HSET", "users", email, password)
	_, err = conn.Do("HSET", "userstype", email, utype)
	_, err = conn.Do("HSET", "userids", email, id)
	if err != nil {
		// handle error

		fmt.Print(err)
		channel <- 1
		return
	}
	channel <- 1
}

func add_latest_created(bug_id int64) {
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

func set_redis_bug(bug Bug) {
	data, _ := json.Marshal(bug)
	fmt.Println(data)
	sdata := string(data)
	bugidint, _ := bug["id"].(int)
	sid := strconv.FormatInt(int64(bugidint), 10)
	//fmt.Println(sid)
	//fmt.Println(sdata)
	redis_hset("bugs", sid, sdata)

	componentstring := strconv.Itoa(bug["component_id"].(int))
	qastring := strconv.Itoa(bug["qa"].(int))
	docsstring := ""
	versionstring := ""
	fixedinverstring := ""
	if bug["docs"] != nil {
		docsstring = strconv.Itoa(bug["docs"].(int))
	}
	reporterstring := strconv.Itoa(bug["reporter"].(int))
	assignedtostring := strconv.Itoa(bug["assigned_to"].(int))
	compid := bug["component_id"].(int)
	productstring := strconv.Itoa(get_product_of_component(compid))
	bugstring := strconv.Itoa(int(bug["id"].(int64)))
	if bug["version"] != nil {

		versionstring = strconv.Itoa(bug["version"].(int))
	}
	if bug["fixedinver"] != nil {
		fixedinverstring = strconv.Itoa(bug["fixedinver"].(int))
	}
	redis_sadd("componentbug:"+componentstring, bugstring)
	redis_sadd("productbug:"+productstring, bugstring)
	redis_sadd("versionbug:"+versionstring, bugstring)

	_, ok := bug["fixedinver"]
	if ok {
		redis_sadd("fixedinverbug:"+fixedinverstring, bugstring)
	}
	_, ok = bug["severity"]
	if ok {
		redis_sadd("severitybug:"+bug["severity"].(string), bugstring)
	}
	_, ok = bug["status"]
	if ok {
		redis_sadd("statusbug:"+bug["status"].(string), bugstring)
	}
	_, ok = bug["priority"]
	if ok {
		redis_sadd("prioritybug:"+bug["priority"].(string), bugstring)
	}
	_, ok = bug["qa"]
	if ok {
		redis_sadd("qabug:"+qastring, bugstring)
		//redis_sadd("userbug:"+qastring, bugstring)
	}
	_, ok = bug["docs"]
	if ok {
		redis_sadd("docsbug:"+docsstring, bugstring)
		//redis_sadd("userbug:"+docsstring, bugstring)
	}
	_, ok = bug["assigned_to"]
	if ok {
		redis_sadd("assigned_tobug:"+assignedtostring, bugstring)
		//redis_sadd("userbug:"+assignedtostring, bugstring)

	}
	_, ok = bug["reporter"]
	if ok {
		redis_sadd("reporterbug:"+reporterstring, bugstring)
		//redis_sadd("userbug:"+reporterstring, bugstring)
	}
}

func update_redis_bug(oldbug Bug, newbug Bug) {
	newdata, _ := json.Marshal(newbug)

	//setting new data for bug.
	sdata := string(newdata)
	bugstring := strconv.Itoa(oldbug["id"].(int))
	sid := strconv.FormatInt(int64(oldbug["id"].(int)), 10)
	redis_hset("bugs", sid, sdata)

	if oldbug["component_id"] != newbug["component_id"] {
		if oldbug["component_id"] != nil {
			tmp := strconv.Itoa(oldbug["component_id"].(int))
			redis_srem("componentbug:"+tmp, bugstring)
		} else {
			if newbug["component_id"] != nil {
				tmp := strconv.Itoa(newbug["component_id"].(int))
				redis_sadd("componentbug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["qa"] != newbug["qa"] {
		if oldbug["qa"] != nil {
			tmp := strconv.Itoa(oldbug["qa"].(int))
			redis_srem("qabug:"+tmp, bugstring)
		} else {
			if newbug["qa"] != nil {
				tmp := strconv.Itoa(newbug["qa"].(int))
				redis_sadd("qabug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["version"] != newbug["version"] {
		if oldbug["version"] != nil {
			tmp := strconv.Itoa(oldbug["version"].(int))
			redis_srem("versionbug:"+tmp, bugstring)
		} else {
			if newbug["version"] != nil {
				tmp := strconv.Itoa(newbug["version"].(int))
				redis_sadd("versionbug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["fixedinver"] != newbug["fixedinver"] {
		if oldbug["fixedinver"] != nil {
			tmp := strconv.Itoa(oldbug["fixedinver"].(int))
			redis_srem("fixedinverbug:"+tmp, bugstring)
		} else {
			if newbug["fixedinver"] != nil {
				tmp := strconv.Itoa(newbug["fixedinver"].(int))
				redis_sadd("fixedinverbug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["severity"] != newbug["severity"] {
		if oldbug["severity"] != nil {
			tmp := strconv.Itoa(oldbug["severity"].(int))
			redis_srem("severitybug:"+tmp, bugstring)
		} else {
			if newbug["severity"] != nil {
				tmp := strconv.Itoa(newbug["severity"].(int))
				redis_sadd("severitybug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["status"] != newbug["status"] {
		if oldbug["status"] != nil {
			tmp := strconv.Itoa(oldbug["status"].(int))
			redis_srem("statusbug:"+tmp, bugstring)
		} else {
			if newbug["status"] != nil {
				tmp := strconv.Itoa(newbug["status"].(int))
				redis_sadd("statusbug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["priority"] != newbug["priority"] {
		if oldbug["priority"] != nil {
			tmp := strconv.Itoa(oldbug["priority"].(int))
			redis_srem("prioritybug:"+tmp, bugstring)
		} else {
			if newbug["priority"] != nil {
				tmp := strconv.Itoa(newbug["priority"].(int))
				redis_sadd("prioritybug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["docs"] != newbug["docs"] {
		if oldbug["docs"] != nil {
			tmp := strconv.Itoa(oldbug["docs"].(int))
			redis_srem("docsbug:"+tmp, bugstring)
		} else {
			if newbug["docs"] != nil {
				tmp := strconv.Itoa(newbug["docs"].(int))
				redis_sadd("docsbug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["assigned_to"] != newbug["assigned_to"] {
		if oldbug["assigned_to"] != nil {
			tmp := strconv.Itoa(oldbug["assigned_to"].(int))
			redis_srem("assigned_tobug:"+tmp, bugstring)
		} else {
			if newbug["assigned_to"] != nil {
				tmp := strconv.Itoa(newbug["assigned_to"].(int))
				redis_sadd("assigned_tobug:"+tmp, bugstring)
			}
		}
	}

	if oldbug["reporter"] != newbug["reporter"] {
		if oldbug["reporter"] != nil {
			tmp := strconv.Itoa(oldbug["reporter"].(int))
			redis_srem("reporterbug:"+tmp, bugstring)
		} else {
			if newbug["reporter"] != nil {
				tmp := strconv.Itoa(newbug["reporter"].(int))
				redis_sadd("reporterbug:"+tmp, bugstring)
			}
		}
	}
}

func add_redis_release(name string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
	defer conn.Close()
	_, err = conn.Do("LPUSH", "releases", name)

}

func get_redis_release_list() interface{} {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
		return nil
	}
	val, err := conn.Do("LRANGE", "releases", 0, -1)
	if err != nil {
		// handle error
		fmt.Println(err)
		return nil
	}
	return val
}

func clear_redis_releases() {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Print(err)
	}
	conn.Do("DEL", "releases")

}

/* To find out if an user already exists or not. */
func find_redis_user(email string) (bool, string) {
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
