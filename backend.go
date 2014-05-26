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
	"time"
)

var conn_str string
/* To read config file values.  */
func load_config(filepath string) {
	file, _ := ini.LoadFile(filepath)
	db_user, _ := file.Get("bugspad", "user")
	db_pass, _ := file.Get("bugspad", "password")
	db_host, _ := file.Get("bugspad", "host")
	db_name, _ := file.Get("bugspad", "database")
	conn_str = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", db_user, db_pass, db_host, db_name)

}

/* Gets the HEX values of the string password*/
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
	id, _ := add_user_mysql(name, email, user_type, mdstr)
	go update_redis(id, email, mdstr, user_type, c)
	fmt.Println(mdstr)
	 _ = <-c
	return "User added."

}

/*
Adds a new user into the database.
*/
func add_user_mysql(name string, email string, user_type string, password string) (err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
	defer db.Close()

	ret, err := db.Exec("INSERT INTO users (name, email, type, password) VALUES (?, ?, ?, ?)", name, email, user_type, password)
	if err != nil {
		return -1, err
	}
	rid, err := ret.LastInsertId()
	return rid, err
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

/* Finds all components for a given product id*/
func get_components_by_product(product_id string) map[string][3]string {
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

/* Gets component name using the id*/
func get_component_name_by_id(component_id int) string {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return ""
	}
	defer db.Close()
	rows, err := db.Query("SELECT name from components where id = ?", component_id)
	if err == nil {
		var name string
		for rows.Next() {
			err = rows.Scan(&name)
			if name != "" {
				return name
			}
		}
	}
	return ""

}

/* Gets component name using the id*/
func get_subcomponent_name_by_id(subcomponent_id int) string {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return ""
	}
	defer db.Close()
	rows, err := db.Query("SELECT name from subcomponent where id = ?", subcomponent_id)
	if err == nil {
		var name string
		for rows.Next() {
			err = rows.Scan(&name)
			if name != "" {
				return name
			}
		}
	}
	return ""

}

/* Get bug cc list*/
func get_bugcc_list(bug_id int) []string {
	ans:=make([]string,0)
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return ans
	}
	defer db.Close()
	//from the cc table
	rows, err := db.Query("SELECT who from cc where bug_id = ?", bug_id)
	if err == nil {
		var user_id int
		for rows.Next() {
			err = rows.Scan(&user_id)
			email := get_user_email(user_id)
			if email != "" {
				ans = append(ans,email)
			}
		}
	}
	//from component_cc table
	rows, err = db.Query("select component_cc.user_id as user from bugs join component_cc where bugs.id=? and component_cc.component_id=bugs.component_id", bug_id)
	if err == nil {
		var user_id int
		for rows.Next() {
			err = rows.Scan(&user_id)
			email := get_user_email(user_id)
			if email != "" {
				ans = append(ans,email)
			}
		}
	}
	//from subcomponent_owners table
	rows, err = db.Query("select subcomponent_owners.subcomponent_owner as user from bugs join subcomponent_owners where bugs.id=? and subcomponent_owners.subcomponent_id=bugs.subcomponent_id", bug_id)
	if err == nil {
		var user_id int
		for rows.Next() {
			err = rows.Scan(&user_id)
			email := get_user_email(user_id)
			if email != "" {
				ans = append(ans,email)
			}
		}
	}
	return ans

}

/* Files a new bug based on input*/
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
func update_bug(data map[string]interface{}) error {
	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE bugs SET ")
	bug_id,_ := strconv.ParseInt(data["id"].(string),10,64)

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
/*
 * Since this canot be changes.
	val, ok = data["subcomponent_id"]
	if ok {
		buffer.WriteString(", subcomponent_id=?")
		vals = append(vals, val)
	}
*/
	val, ok = data["fixedinver"]
	if ok {
		buffer.WriteString(", fixedinver=?")
		vals = append(vals, val)
	}
/*
 * Since this cannot be changed
	val, ok = data["component_id"]
	if ok {
		buffer.WriteString(", component_id=?")
		vals = append(vals, val)
	}
*/
	buffer.WriteString(" WHERE id=?")
	fmt.Println(buffer.String())

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Println(err)
		return err
	}
	defer db.Close()
	fmt.Println(data["id"].(string)+"lll")
	vals = append(vals, data["id"])
	_, err = db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if ok1 {
		bug := get_redis_bug(strconv.FormatInt(int64(bug_id), 10))
		old_status := bug["status"].(string)
		delete_redis_bug_status(strconv.FormatInt(int64(bug_id), 10), old_status)
		update_redis_bug_status(strconv.FormatInt(int64(bug_id), 10), val1.(string))
	}
	add_latest_updated(strconv.FormatInt(int64(bug_id), 10))
	return nil
}

/*
Get a bug details.
*/
func get_bug(bug_id string) Bug {
	fmt.Println(bug_id)
	m := make(Bug)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	row := db.QueryRow("SELECT status, description, version, severity, hardware, priority, whiteboard, reported, component_id, subcomponent_id, reporter, summary, fixedinver, qa, docs from bugs where id=?", bug_id)
	var status, description, version, severity, hardware, priority, whiteboard,  summary, fixedinver []byte
	var reporter, component_id int
	var qa, docs, subcomponent_id sql.NullInt64
	var reported time.Time
	err = row.Scan(&status, &description, &version, &severity, &hardware, &priority, &whiteboard, &reported, &component_id, &subcomponent_id, &reporter, &summary, &fixedinver, &qa, &docs)
	if err == nil {
	    	    qaint := -1
		    docsint := -1
		    subcint := -1
		    if qa.Valid{
			qaint = int(qa.Int64)
		    }
		    if docs.Valid{
			docsint = int(docs.Int64)
		    }
		    if subcomponent_id.Valid{
			subcint = int(subcomponent_id.Int64)
		    }
		qa_name := ""
		docs_name := ""
		if qaint!=-1 {
		    qa_name = get_user_email(qaint) 
		}
		if docsint!=-1 {
		    docs_name = get_user_email(docsint)
		}
		bugs_idint,_ := strconv.Atoi(bug_id)
		m["id"] = bug_id
		m["status"] = string(status)
		m["summary"] = string(summary)
		m["severity"] = string(severity)
		m["description"] = string(description)
		m["version"] = string(version)
		m["hardware"] = string(hardware)
		m["priority"] = string(priority)
		m["whiteboard"] = string(whiteboard)
		m["reported"] = reported.String()
		m["reporter"] = get_user_email(reporter)
		m["component"] = get_component_name_by_id(component_id)
		m["subcomponent"] = get_subcomponent_name_by_id(subcint)
		m["fixedinver"] = string(fixedinver)
		m["qa"] = qa_name
		m["docs"] = docs_name
		m["cclist"] = get_bugcc_list(bugs_idint)

	} else {
		m["error_msg"] = err
		fmt.Println(err)
	}
	return m
}


/*
Returns the user email given the user id.
*/ 
func get_user_email(id int) string {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return ""
	}
	defer db.Close()
	rows, err := db.Query("SELECT email from users where id = ?", id)
	if err == nil {
		var email string
		for rows.Next() {
			err = rows.Scan(&email)
			if email != "" {
				return email
			}
		}
	}
	return ""
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

/*
Get user id by email.
*/
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

/*Adds release information.*/
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

/*
Retrieving comments of a bug.
*/
func fetch_comments_by_bug(bug_id string) map[string][4]string {

	m := make(map[string][4]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT users.email as useremail, users.name as username, comments.id as com_id, description, datec, bug FROM comments JOIN users WHERE bug=? and users.id=comments.user;", bug_id)
	//fmt.Println(rows)
	if err != nil {
		return m
	}
	defer rows.Close()
	var com_id, description, username, useremail, bug string
	var datec time.Time
	for rows.Next() {
		err = rows.Scan(&useremail, &username, &com_id, &description, &datec, &bug)
		//fmt.Println(c_id, name, description)
		//m = append(m,Comment{com_id, description, user, datec})
		//user="jj"
		fmt.Println(datec)
		m[com_id] = [4]string{useremail, username, description, time.Time.String(datec)}
	}
	return m
}


/*
Retrieving all components of all products.
*/
func get_all_components() map[string][2]string {
    	m := make(map[string][2]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, description from components")
	if err != nil {
		return m
	}
	defer rows.Close()
	var name, description, c_id string
	for rows.Next() {
		err = rows.Scan(&c_id, &name, &description)
		//fmt.Println(c_id, name, description)
		m[c_id] = [2]string{name, description}
	}
	return m
}
