package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vaughan0/go-ini"
	"html/template"
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

/*
Adds a new user to the system. user_type can be 0,1,2. First it updates the
redis server and then saves the details to the MySQL db.
*/
func add_user(name string, email string, user_type string, password string) string {
	answer, res := find_redis_user(email)
	fmt.Println(res)
	if answer == true {
		return res
	}
	mdstr := get_hex(password)
	c := make(chan int)
	id, err := add_user_mysql(name, email, user_type, mdstr)
	fmt.Println(err)
	go update_redis(id, email, mdstr, user_type, c)
	//fmt.Println(mdstr)
	_ = <-c
	return "User added."

}

/*
Adds a new user into the database.
*/
func add_user_mysql(name string, email string, user_type string, password string) (int64, error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return -1, err
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
func insert_component(name string, desc string, product_id int, owner int, qa int) (id string, err error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return "", err
	}
	var ret sql.Result
	defer db.Close()
	if qa != -1 {
		ret, err = db.Exec("INSERT INTO components (name, description, product_id, owner, qa) VALUES (?, ?, ?, ?, ?)", name, desc, product_id, owner, qa)
	} else {
		ret, err = db.Exec("INSERT INTO components (name, description, product_id, owner) VALUES (?, ?, ?, ?)", name, desc, product_id, owner)
	}
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
		ret := string(redis_hget("userstype", email))
		if ret != "-1" {
			return true
		}
	}
	return false
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

/* Finds all subcomponents for a given component*/
func get_subcomponents_by_component(component_id int) map[string][3]string {
	m := make(map[string][3]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, description from subcomponent where component_id=?", component_id)
	if err != nil {
		return m
	}
	defer rows.Close()
	var name, description, sc_id string
	for rows.Next() {
		err = rows.Scan(&sc_id, &name, &description)
		//fmt.Println(c_id, name, description)
		m[name] = [3]string{sc_id, name, description}
	}
	return m

}

/* Get bug cc list*/
func get_bugcc_list(bug_id int) map[int][2]string {
	ans := make(map[int][2]string)
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
			name := get_user_name(user_id)
			if email != "" {
				ans[user_id] = [2]string{name, email}
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
			name := get_user_name(user_id)
			if email != "" {
				ans[user_id] = [2]string{name, email}
			}
		}
	}
	/*Leaving subcomponents for the time
	//from subcomponent_owners table
	rows, err = db.Query("select subcomponent_owners.subcomponent_owner as user from bugs join subcomponent_owners where bugs.id=? and subcomponent_owners.subcomponent_id=bugs.subcomponent_id", bug_id)
	if err == nil {
		var user_id int
		for rows.Next() {
			err = rows.Scan(&user_id)
			email := get_user_email(user_id)
			if email != "" {
				ans = append(ans, email)
			}
		}
	}*/
	//fmt.Println(ans)
	return ans

}

/* Files a new bug based on input*/
func new_bug(data map[string]interface{}) (int, string) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return -1, err.Error()
	}
	defer db.Close()

	//var status, summary string

	var buffer, buffer2 bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("INSERT INTO bugs (")
	//Since the default status has to be "new"
	data["status"]="new"
	buffer.WriteString("status")
	buffer2.WriteString("?")
	vals = append(vals, data["status"].(string))

	/*
	 * While filing a bug status cannot be anything other than new.
	 * val, ok := data["status"]
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

	val, ok = data["status"]
	if ok {
		buffer.WriteString(", status")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}
	*/
	val, ok := data["version"]
	if ok {
		buffer.WriteString(", version")
		buffer2.WriteString(", ?")
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

	val, ok = data["assigned_to"]
	if ok {
		buffer.WriteString(", assigned_to")
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
		return -1, "Missing input: reporter"
	}

	val, ok = data["summary"]
	if ok {
		buffer.WriteString(", summary")
		buffer2.WriteString(",?")
		vals = append(vals, val)
		//		summary = val.(string)
	} else {
		return -1, "Missing input: summary"
	}

	val, ok = data["description"]
	if ok {
		buffer.WriteString(", description")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	} else {
		return -1, "Missing input: description"
	}
	//since docs is an optional fields
	val, ok = data["docs"]
	if ok {
		buffer.WriteString(", docs")
		buffer2.WriteString(",?")
		vals = append(vals, val)
	}

	val, ok = data["component_id"]
	if ok {
		buffer.WriteString(", component_id")
		buffer2.WriteString(",?")
		vals = append(vals, val)

		buffer.WriteString(", qa")
		buffer2.WriteString(",?")
		vals = append(vals, strconv.Itoa(get_component_owner(val.(int))))
	} else {
		return -1, "Missing input: component_id"
	}

	buffer.WriteString(", reported) VALUES (")
	buffer2.WriteString(",NOW()")

	buffer.WriteString(buffer2.String())
	buffer.WriteString(")")
	//fmt.Println(buffer.String())
	ret, err := db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("dsadasd")
		return -1, "Error in entering a new bug"
	}

	rid, err := ret.LastInsertId()
	fmt.Println(err)
	bug_id := rid
	// Now update redis cache for status
	//update_redis_bug_status(bug_id, status)

	//adding the reporter,assignee,qa,docs to the cclist

	reporter, _ := data["reporter"]
	_, present := db.Query("SELECT * from cc where bug_id=? and who=?", bug_id, reporter)
	if present == nil {
		db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, reporter)
	}
	assignee, _ := data["assigned_to"]
	_, present = db.Query("SELECT * from cc where bug_id=? and who=?", bug_id, assignee)
	if present == nil {
		db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, assignee)
	}
	//comp_id,_:= strconv.Atoi(data["component_id"].(string))
	qa := get_component_owner(data["component_id"].(int))
	_, present = db.Query("SELECT * from cc where bug_id=? and who=?", bug_id, qa)
	if present == nil {
		db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, qa)
	}

	docs, ok := data["docs"]
	if ok {
		_, present = db.Query("SELECT * from cc where bug_id=? and who=?", bug_id, docs)
		if present == nil {
			db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, docs)
		}
	}

	//_, err = db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, docs)
	//redis_sadd("userbug"+val.(string), bug_id)
	data["id"] = int(bug_id)
	//comp_id,_=strconv.Atoi(data["component_id"].(string))
	data["qa"] = get_component_owner(data["component_id"].(int))
	set_redis_bug(data)
	add_latest_created(bug_id)

	fmt.Println(err)
	return int(bug_id), ""
}

/*
HTMLify the string
*/
func htmlify(olddata string, newdata string, datatype string) string {
	return "<p class=\"bugstatchange\"><label class=\"updatetype\">" + datatype + "</label>" + "<label class=\"oldstat\">" + olddata + "</label>" + "<label class=\"newstat\">" + newdata + "</label></p>"
}

/*
Adds comments for each change made to the bug.
*/
func add_bug_comments(olddata map[string]interface{}, newdata map[string]interface{}) {
	changes_comments := ""
	if olddata["summary"] != newdata["summary"] {
		changes_comments = changes_comments + htmlify(olddata["summary"].(string), newdata["summary"].(string), "summary")
	}
	if olddata["status"] != newdata["status"] {
		changes_comments = changes_comments + htmlify(olddata["status"].(string), newdata["status"].(string), "status")
	}
	if olddata["severity"] != newdata["severity"] {
		changes_comments = changes_comments + htmlify(olddata["severity"].(string), newdata["severity"].(string), "severity")
	}
	if olddata["whiteboard"] != newdata["whiteboard"] {
		changes_comments = changes_comments + htmlify(olddata["whiteboard"].(string), newdata["whiteboard"].(string), "whiteboard")
	}
	if olddata["hardware"] != newdata["hardware"] {
		changes_comments = changes_comments + htmlify(olddata["hardware"].(string), newdata["hardware"].(string), "hardware")
	}
	if olddata["priority"] != newdata["priority"] {
		changes_comments = changes_comments + htmlify(olddata["priority"].(string), newdata["priority"].(string), "priority")
	}
	if olddata["component_id"] != newdata["component_id"] {
		changes_comments = changes_comments + htmlify(olddata["component"].(string), newdata["component"].(string), "component")

	}
	if olddata["subcomponent_id"] != newdata["subcomponent_id"] {
		changes_comments = changes_comments + htmlify(olddata["subcomponent"].(string), newdata["subcomponent"].(string), "subcomponent")
	}
	fmt.Println("fiv")
	fmt.Println(olddata["fixedinver"].(int))
	fmt.Println(newdata["fixedinver"].(int))
	fmt.Println("fiv")
	if olddata["fixedinver"] != newdata["fixedinver"] {
		changes_comments = changes_comments + htmlify(get_version_text(olddata["fixedinver"].(int)), get_version_text(newdata["fixedinvername"].(int)), "fixedinverame")
	}
	if olddata["version"] != newdata["version"] {
		changes_comments = changes_comments + htmlify(get_version_text(olddata["version"].(int)), get_version_text(newdata["version"].(int)), "version")
	}
	//fmt.Println(olddata["qaint"].(int))
	//fmt.Println(newdata["qa"].(int))
	if newdata["qa"] != nil {
		if olddata["qa"] != newdata["qa"] && newdata["qa"] != -1 {
			changes_comments = changes_comments + htmlify(get_user_email(olddata["qa"].(int)), get_user_email(newdata["qa"].(int)), "qa")
		}
	}

	if olddata["assigned_to"] != newdata["assigned_to"] && newdata["assigned_to"] != -1 {
		changes_comments = changes_comments + htmlify(get_user_email(olddata["assigned_to"].(int)), get_user_email(newdata["assigned_to"].(int)), "assigned_to")
	}

	if newdata["docs"] != nil {
		if olddata["docs"] != newdata["docs"] && newdata["docs"].(int) != -1 {
			changes_comments = changes_comments + htmlify(get_user_email(olddata["docs"].(int)), get_user_email(newdata["docs"].(int)), "docs")
		}
	}
	bug_idint, _ := strconv.Atoi(newdata["id"].(string))
	if changes_comments != "" {
		//user_idint,_ := strconv.Atoi(newdata["post_user"].(int))
		new_comment(newdata["post_user"].(int), bug_idint, changes_comments)
	}
	if newdata["com_content"].(string) != "" {
		new_comment(newdata["post_user"].(int), bug_idint, newdata["com_content"].(string))
	}

}

/*
Adds attachment to a bug
*/
func add_attachment(data map[string]interface{}) error {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO attachments (bug_id, datec, description, filename, systempath, submitter) VALUES (?,NOW(),?,?,?,?)", data["bug_id"].(string), data["description"].(string), data["filename"].(string), data["systempath"].(string), data["submitter"].(int))
	if err != nil {
		fmt.Println(err)
	}
	return err

}

/*
Updates user type.
*/
func update_user_type(userid int, newtype string) bool {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	_, err = db.Exec("UPDATE users set type=? where id=?", newtype, userid)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true

}

/*
Gets the total number of attachments.
*/
func count_entries(table_name string) int {
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	if err != nil {
		return -1
	}
	row := db.QueryRow("SELECT count(*) from " + table_name)
	var entries int
	err = row.Scan(&entries)
	fmt.Println(err)
	if err == nil {
		return entries
	}
	return -1

}

/*
Fetches all the attachments of a bug.
*/
func get_bug_attachments(bug_id string) map[int][6]string {
	m := make(map[int][6]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	if err != nil {
		fmt.Println(err)
		return m
	}
	rows, err := db.Query("SELECT id, datec, description, filename, systempath, submitter from attachments where isobsolete=0 and bug_id=?", bug_id)
	if err != nil {
		fmt.Println(err)
		return m
	}
	var description, systempath, filename string
	var id, submitter int
	var datec time.Time
	for rows.Next() {
		err = rows.Scan(&id, &datec, &description, &filename, &systempath, &submitter)
		if err == nil {
			m[id] = [6]string{systempath, description, filename, get_user_email(submitter), get_user_name(submitter), time.Time.String(datec)}
		} else {
			fmt.Println(err)
			return make(map[int][6]string)
		}
	}
	//fmt.Println("ad")
	return m

}

/*
Makes attachments obsolete
*/
func make_attachments_obsolete(attachment_ids []string) error {

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	defer db.Close()
	for _, x := range attachment_ids {
		_, err = db.Exec("UPDATE attachments set isobsolete=1 where id=?", x)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return err

}

/*
Getting all bug ids blocked by the given bug.
*/
func bugs_blocked_by(bug_id string) map[int]string {
	m := make(map[int]string)
	//fmt.Print("dgffg")
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return m
	}
	defer db.Close()
	rows, err := db.Query("SELECT distinct dependencies.blocked as blocked, bugs.status as status from dependencies join bugs where bugs.id=dependencies.dependson and dependson=?", bug_id)
	if err != nil {
		fmt.Println(err)
		return m
	}
	defer rows.Close()
	var b_id int
	var status string
	for rows.Next() {
		err = rows.Scan(&b_id, &status)
		m[b_id] = status
		//fmt.Println(c_id, name, description)
	}
	return m

}

/*
Getting all bugs which the given bug is dependent on.
*/
func bugs_dependent_on(bug_id string) map[int]string {
	m := make(map[int]string)
	//fmt.Print("dgffg")
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return m
	}
	defer db.Close()
	rows, err := db.Query("SELECT distinct dependencies.dependson as dependson,bugs.status as status from dependencies join bugs where bugs.id=dependencies.blocked and blocked=?", bug_id)
	if err != nil {
		fmt.Println(err)
		return m
	}
	defer rows.Close()
	var b_id int
	var status string
	for rows.Next() {
		err = rows.Scan(&b_id, &status)
		m[b_id] = status
		//fmt.Println(c_id, name, description)
	}
	return m

}

/*
Check is the bug dependency is valid
*/
func is_valid_bugdependency(blocked int, dependson int) (bool, string) {
	//bug cannot depend on itself.
	if blocked == dependson {
		return false, "Both blocked and depends can't be the same"
	}
	//checking for circular dependency
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		fmt.Print(err)
		return false, err.Error()
	}
	defer db.Close()
	rows, err := db.Query("SELECT * from dependencies where dependson=? and blocked=?", blocked, dependson)
	for rows.Next() {
		return false, "Circular Dependency cannot exist."
	}
	return true, "Valid"
}

func clear_dependencies(bug int, bugtype string) error {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	defer db.Close()
	_, err = db.Exec("DELETE from dependencies where "+bugtype+"=?", bug)
	return err

}

/*
Add a bug dependency
*/
func add_bug_dependency(blocked int, dependson int) error {
	//fmt.Println("blocked")
	fmt.Println(blocked)
	//fmt.Println("dependson")
	fmt.Println(dependson)
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	defer db.Close()
	_, err = db.Exec("INSERT into dependencies (blocked,dependson) VALUES (?,?)", blocked, dependson)
	fmt.Println(err)
	return err
}

/*
Checking if an entry exists in a table
*/
func entry_exists(tablename string, fieldname string, fieldvalue string) bool {

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		fmt.Print(err)
		return false
	}
	defer db.Close()
	rows, err := db.Query("SELECT * from " + tablename + " where " + fieldname + "=" + fieldvalue)
	fmt.Println(err)
	for rows.Next() {

		return false
	}
	return true

}

/*
Updates a given bug based on input
*/
func update_bug(data map[string]interface{}) error {
	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE bugs SET ")
	bug_id, _ := strconv.ParseInt(data["id"].(string), 10, 64)
	old_bug := get_bug(data["id"].(string))
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

	val, ok = data["summary"]
	if ok {
		buffer.WriteString(", summary=?")
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
		if val == -1 {
			buffer.WriteString(", qa=NULL")
		} else {
			buffer.WriteString(", qa=?")
			vals = append(vals, val)
		}
	}

	val, ok = data["docs"]
	if ok {
		if val == -1 {
			buffer.WriteString(", docs=NULL")
		} else {
			buffer.WriteString(", docs=?")
			vals = append(vals, val)
		}
	}

	val, ok = data["whiteboard"]
	if ok {
		buffer.WriteString(", whiteboard=?")
		vals = append(vals, val)
	}

	val, ok = data["assigned_to"]
	if ok {
		buffer.WriteString(", assigned_to=?")
		vals = append(vals, val)
	}
	// * Since this canot be changes.
	val, ok = data["subcomponent_id"]
	if ok && val != -1 {
		buffer.WriteString(", subcomponent_id=?")
		vals = append(vals, val)
	}
	//*/
	val, ok = data["fixedinver"]
	if ok {
		buffer.WriteString(", fixedinver=?")
		vals = append(vals, val)
	}
	///*Since this cannot be changed
	val, ok = data["component_id"]
	if ok {
		buffer.WriteString(", component_id=?")
		vals = append(vals, val)
	}
	//*/
	buffer.WriteString(" WHERE id=?")
	//fmt.Println(buffer.String())

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Println(err)
		return err
	}
	defer db.Close()
	//fmt.Println(data["id"].(string)+"lll")
	vals = append(vals, data["id"])
	_, err = db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err)
		return err
	}/*
	if ok1 {
		bug := get_redis_bug(strconv.FormatInt(int64(bug_id), 10))
		old_status := bug["status"].(string)
		delete_redis_bug_status(strconv.FormatInt(int64(bug_id), 10), old_status)
		update_redis_bug_status(strconv.FormatInt(int64(bug_id), 10), val1.(string))
	}*/
	add_latest_updated(strconv.FormatInt(int64(bug_id), 10))
	add_bug_comments(old_bug, data)
	return nil
}

/*
Fetches the version value.
*/
func get_version_text(id int) string {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return ""
	}
	defer db.Close()
	rows, err := db.Query("SELECT value from versions where id = ?", id)
	if err == nil {
		var vertext string
		for rows.Next() {
			err = rows.Scan(&vertext)
			if vertext != "" {
				return vertext
			}
		}
	}
	return ""
}

/*
Get a bug details.
*/
func get_bug(bug_id string) Bug {
	//fmt.Println(bug_id)
	m := make(Bug)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	row := db.QueryRow("SELECT status, description, version, severity, hardware, priority, whiteboard, reported, component_id, subcomponent_id, reporter, summary, fixedinver, qa, docs, assigned_to from bugs where id=?", bug_id)
	var status, description, severity, hardware, priority, whiteboard, summary []byte
	var reporter, component_id, assigned_to, version int
	var qa, docs, subcomponent_id, fixedinver sql.NullInt64
	var reported time.Time
	err = row.Scan(&status, &description, &version, &severity, &hardware, &priority, &whiteboard, &reported, &component_id, &subcomponent_id, &reporter, &summary, &fixedinver, &qa, &docs, &assigned_to)
	if err == nil {
		qaint := -1
		docsint := -1
		subcint := -1
		fixedinverint := 0
		if qa.Valid {
			qaint = int(qa.Int64)
		}
		if fixedinver.Valid {
			fixedinverint = int(fixedinver.Int64)
		}
		if docs.Valid {
			docsint = int(docs.Int64)
		}
		if subcomponent_id.Valid {
			subcint = int(subcomponent_id.Int64)
		}
		qa_email := ""
		qa_name := ""
		docs_email := ""
		docs_name := ""
		if qaint != -1 {
			qa_email = get_user_email(qaint)
			qa_name = get_user_name(qaint)
		}
		if docsint != -1 {
			docs_email = get_user_email(docsint)
			docs_name = get_user_name(docsint)
		}
		bug_idint, _ := strconv.Atoi(bug_id)
		m["id"] = bug_id
		m["status"] = string(status)
		m["summary"] = string(summary)
		m["severity"] = string(severity)
		m["description"] = string(description)
		m["hardware"] = string(hardware)
		m["priority"] = string(priority)
		m["whiteboard"] = string(whiteboard)
		m["reported"] = reported.String()
		m["reporter"] = get_user_email(reporter)
		m["assigned_to"] = assigned_to
		m["qa"] = qaint
		m["docs"] = docsint
		m["component_id"] = component_id
		m["subcomponent_id"] = subcint
		m["fixedinver"] = fixedinverint
		m["version"] = version

		//extra fields for convenience
		m["versiontext"] = get_version_text(version)
		m["qaemail"] = qa_email
		m["qaname"] = qa_name
		m["docsemail"] = docs_email
		m["docsname"] = docs_name
		m["assigned_toname"] = get_user_name(assigned_to)
		m["assigned_toemail"] = get_user_email(assigned_to)
		m["reportername"] = get_user_name(reporter)
		m["reporteremail"] = get_user_email(reporter)
		m["component"] = get_component_name_by_id(component_id)
		m["subcomponent"] = get_subcomponent_name_by_id(subcint)
		m["fixedinvername"] = get_version_text(fixedinverint)
		m["cclist"] = get_bugcc_list(bug_idint)
		fmt.Println(bug_idint)

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
Returns the user name given the user id.
*/
func get_user_name(id int) string {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return ""
	}
	defer db.Close()
	rows, err := db.Query("SELECT name from users where id = ?", id)
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
		//fmt.Println(email)
		//fmt.Println(bug_id)
		user_id := get_user_id(email)
		// If user_id is -1 means no such user
		if user_id != -1 {
			_, err := db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, user_id)
			fmt.Println(err)
			redis_sadd("userbug"+strconv.Itoa(user_id), strconv.Itoa(int(bug_id)))
		}
	}
	//fmt.Print("addcc")
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
			redis_srem("userbug"+strconv.Itoa(user_id), strconv.Itoa(int(bug_id)))
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
	//checking if reporter is in bugcc list if not add it.
	_, err = db.Query("SELECT * from cc where bug_id=?,who=?", reporter, bug_id)
	if err != nil {
		_, err = db.Exec("INSERT INTO cc (bug_id, who) VALUES (?,?)", bug_id, reporter)
		if err != nil {
			fmt.Println(err)
			return strconv.FormatInt(rid, 10), err
		}
		redis_sadd("userbug"+strconv.Itoa(reporter), strconv.Itoa(bug_id))
	}
	//check if the bug status is still new if yes then change it to open
	bug := get_redis_bug(strconv.Itoa(bug_id))
	if bug["status"].(string) == "new" && err == nil {
		bug["status"] = "open"
		update_bug(bug)
	} else {
		fmt.Println(err)
	}
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

/*
Get owner of component.
*/
func get_component_owner(id int) int {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return -1
	}
	defer db.Close()
	rows, err := db.Query("SELECT owner from components where id = ?", id)
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

/*
Get product versions.
*/
func get_product_versions(product_id int) map[string][2]int {

	m := make(map[string][2]int)
	//fmt.Print("dgffg")
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return m
	}
	defer db.Close()
	rows, err := db.Query("SELECT id, value, isactive from versions where product_id=? order by value", product_id)
	if err != nil {
		fmt.Println(err)
		return m
	}
	defer rows.Close()
	var description string
	var v_id, isactive int
	for rows.Next() {
		err = rows.Scan(&v_id, &description, &isactive)
		//fmt.Println(c_id, name, description)
		m[description] = [2]int{v_id, isactive}
	}
	return m

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

/*
Get original of a duplicate bug if it is a duplicate.
*/
func find_orig_ifdup(bug_id string) int {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return -1
	}
	defer db.Close()
	row := db.QueryRow("SELECT dup_of from duplicates where dup=?", bug_id)
	var dup_of int
	err = row.Scan(&dup_of)
	if err == nil {
		return dup_of
		//fmt.Println(m["name"])
	} else {
		return -1
	}

}

/*
Set original of a duplicate bug if it is a duplicate.
*/
func add_dup_bug(dup string, dup_of string) bool {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	defer db.Close()
	_, err = db.Exec("INSERT into duplicates (dup,dup_of) VALUES (?,?)", dup, dup_of)
	if err == nil {
		return true
	}
	fmt.Println(err)
	return false
}

/*
Remove original of a duplicate bug if it is a duplicate.
*/
func remove_dup_bug(dup string) bool {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	defer db.Close()
	_, err = db.Exec("DELETE from duplicates where dup=?", dup)
	if err == nil {
		return true
	}
	fmt.Println(err)
	return false
}

/*Add Product Version.*/
func add_product_version(product_id string, value string) error {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO versions (value,product_id) VALUES (?,?)", value, product_id)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	return err
}

/*
Update the versions
*/
func update_product_version(version_value string, version_active int, version_id string) error {

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return err
	}
	defer db.Close()
	_, err = db.Exec("UPDATE versions set value=?, isactive=? where id=?", version_value, version_active, version_id)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return err

}

/*
Get version activity.
*/
func is_version_active(version_id string) (bool, error) {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false, err
	}
	defer db.Close()
	row := db.QueryRow("SELECT isactive from versions where id=?", version_id)
	var act int
	err = row.Scan(&act)
	if err == nil {
		if act != 0 {
			return true, nil
		} else {
			return false, nil
		}
		//fmt.Println(m["name"])
	} else {

		return false, err
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
func fetch_comments_by_bug(bug_id string) map[int64][4]template.HTML {

	m := make(map[int64][4]template.HTML)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT users.email as useremail, users.name as username, comments.id as com_id, description, datec, bug FROM comments JOIN users WHERE bug=? and users.id=comments.user;", bug_id)
	//fmt.Println(rows)
	if err != nil {
		return m
	}
	defer rows.Close()
	var description, username, useremail, bug string
	var datec time.Time
	var com_id int64
	for rows.Next() {
		err = rows.Scan(&useremail, &username, &com_id, &description, &datec, &bug)
		//fmt.Println(c_id, name, description)
		//m = append(m,Comment{com_id, description, user, datec})
		//user="jj"
		//fmt.Println(datec)
		m[com_id] = [4]template.HTML{template.HTML(useremail), template.HTML(username), template.HTML(description), template.HTML(time.Time.String(datec))}
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
	rows, err := db.Query("SELECT id, name, description from components order by name")
	if err != nil {
		return m
	}
	defer rows.Close()
	var name, description, c_id string
	for rows.Next() {
		err = rows.Scan(&c_id, &name, &description)
		//fmt.Println(c_id, name, description)
		m[name] = [2]string{c_id, description}
	}
	return m
}

/*
Checks if the user is admin or not.
*/
func is_user_admin(email string) bool {
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
		return false
	}
	defer db.Close()
	rows, err := db.Query("SELECT type from users where email = ?", email)
	if err == nil {
		var t bool
		for rows.Next() {
			err = rows.Scan(&t)
			//fmt.Println(t)
			if t {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

/*Returns the details of the product with the given product id*/
func get_product_by_id(product_id string) map[string]interface{} {
	m := make(map[string]interface{})
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		m["error_msg"] = err
		return m
	}
	defer db.Close()
	row := db.QueryRow("SELECT name, description from products where id=?", product_id)
	if err != nil {
		m["error_msg"] = err
		return m
	}
	var name, description string
	err = row.Scan(&name, &description)
	if err == nil {
		m["id"] = product_id
		m["name"] = name
		m["description"] = description
		//fmt.Println(m["name"])
	}

	return m

}

/* Finds all components for a given product id*/
func get_components_by_id(product_id int) map[string][3]string {
	m := make(map[string][3]string)
	//fmt.Print("dgffg")
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, description from components where product_id=? order by name", product_id)
	if err != nil {
		fmt.Println(err)
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

/*Returns the product id of the entered component*/
func get_product_of_component(component_id int) int {
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	row := db.QueryRow("SELECT product_id from components where id=?", component_id)
	var product_id int
	err = row.Scan(&product_id)
	//fmt.Println(c_id, name, description)
	if err != nil {

		return -1
	}
	return product_id

}

//TODO : this should be done from redis
/*Returns all bugs related to a single product*/
func get_bugs_by_product(product_id string) (map[string][15]string, error) {
	m := make(map[string][15]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	if err != nil {
		return m, err
	}
	r1 := "distinct bugs.id as id, bugs.status as status, bugs.version as version, bugs.severity as severity, bugs.hardware as hardware, bugs.priority as priority,"
	r2 := " bugs.reporter as reporter, bugs.qa as qa, bugs.docs as docs, bugs.whiteboard as whiteboard, bugs.summary as summary, bugs.description as description, "
	r3 := "bugs.reported as reported, bugs.fixedinver as fixedinver, bugs.component_id as component_id, bugs.subcomponent_id as subcomponent_id "
	rows, err := db.Query("SELECT "+r1+r2+r3+" from bugs join components join products where products.id=components.product_id and products.id=?", product_id)
	if err != nil {
		return m, err
	}
	var status, description, severity, hardware, priority, whiteboard, summary []byte
	var id, reporter, component_id, version int
	var qa, docs, subcomponent_id, fixedinver sql.NullInt64
	var reported time.Time
	for rows.Next() {
		err = rows.Scan(&id, &status, &version, &severity, &hardware, &priority, &reporter, &qa, &docs, &whiteboard, &summary, &description, &reported, &fixedinver, &component_id, &subcomponent_id)
		if err == nil {
			qaint := -1
			docsint := -1
			subcint := -1
			fixedinverint := -1
			if qa.Valid {
				qaint = int(qa.Int64)
			}
			if fixedinver.Valid {
				fixedinverint = int(qa.Int64)
			}
			if docs.Valid {
				docsint = int(docs.Int64)
			}
			if subcomponent_id.Valid {
				subcint = int(subcomponent_id.Int64)
			}
			subc := get_subcomponent_by_id(strconv.Itoa(subcint))
			subcomponent_name := ""
			if subc["name"] != nil {
				subcomponent_name = subc["name"].(string)
			}
			c := get_component_by_id(strconv.Itoa(component_id))
			component_name := ""
			if c["name"] != nil {
				component_name = c["name"].(string)
			}

			//fmt.Println(string(f))
			m[strconv.Itoa(id)] = [15]string{string(status), string(get_version_text(version)), string(severity), string(hardware), string(priority), get_user_email(reporter), get_user_email(qaint), get_user_email(docsint), string(whiteboard), string(summary), string(description), reported.String(), string(get_version_text(fixedinverint)), component_name, subcomponent_name}
		} else {
			//fmt.Println("yaha hai")
			return m, err

		}
	}
	return m, err
}

func get_user_bugs(user_id string) map[string][2]string {

	m := make(map[string][2]string)

	//from userbug
	bug_ids := redis_smembers("userbug" + user_id)
	b := bug_ids.([]interface{})
	for i, _ := range b {
		//fmt.Println(string(b[i].([]uint8)))
		bug := get_redis_bug(string(b[i].([]uint8)))
		m[string(b[i].([]uint8))] = [2]string{bug["status"].(string), bug["summary"].(string)}
	}

	//from assignedtobug
	bug_ids = redis_smembers("assigned_tobug" + user_id)
	b = bug_ids.([]interface{})
	for i, _ := range b {
		//fmt.Println(string(b[i].([]uint8)))
		bug := get_redis_bug(string(b[i].([]uint8)))
		m[string(b[i].([]uint8))] = [2]string{bug["status"].(string), bug["summary"].(string)}
	}
	//from reporterbug
	bug_ids = redis_smembers("reporterbug" + user_id)
	b = bug_ids.([]interface{})
	for i, _ := range b {
		//fmt.Println(string(b[i].([]uint8)))
		bug := get_redis_bug(string(b[i].([]uint8)))
		m[string(b[i].([]uint8))] = [2]string{bug["status"].(string), bug["summary"].(string)}
	}
	//from docs
	bug_ids = redis_smembers("docsbug" + user_id)
	b = bug_ids.([]interface{})
	for i, _ := range b {
		//fmt.Println(string(b[i].([]uint8)))
		bug := get_redis_bug(string(b[i].([]uint8)))
		m[string(b[i].([]uint8))] = [2]string{bug["status"].(string), bug["summary"].(string)}
	}
	//fmt.Println(m)
	return m

}

/*Returns the details of the user with the given user id*/
func get_user_by_id(user_id string) map[string]interface{} {
	m := make(map[string]interface{})
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	row := db.QueryRow("SELECT name, email, type from users where id=?", user_id)
	var name, email, u_type string
	err = row.Scan(&name, &email, &u_type)
	//fmt.Println(c_id, name, description)
	if err != nil {
		m["error_msg"] = err
		return m
	}
	m["id"] = user_id
	m["name"] = name
	m["email"] = email
	m["type"] = u_type
	//fmt.Println(m["name"])
	return m

}

/*
Updates an existing user.
*/
func update_user(data map[string]interface{}) (msg string, err error) {

	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE users SET ")

	val, ok := data["name"]
	if ok {
		buffer.WriteString("name=?")
		vals = append(vals, val)
	}

	val, ok = data["email"]
	if ok {
		buffer.WriteString(", email=?")
		vals = append(vals, val)
	}

	val, ok = data["type"]
	if ok {
		buffer.WriteString(",type=?")
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

	vals = append(vals, data["id"])
	_, err = db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err)
		return "Error occured updating", err
	}
	return "Update Successful.", err
}

/*Get the details of all users */
func get_all_users() map[string][4]string {
	m := make(map[string][4]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, email, type from users")
	if err != nil {
		return m
	}
	defer rows.Close()
	var name, email, u_id, u_type string
	for rows.Next() {
		err = rows.Scan(&u_id, &name, &email, &u_type)
		//fmt.Println(c_id, name, description)
		m[u_id] = [4]string{u_id, name, email, u_type}
	}
	return m
}

/*
Updates an existing product.
*/
func update_product(data map[string]interface{}) (msg string, err error) {

	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE products SET ")

	val, ok := data["name"]
	if ok {
		buffer.WriteString("name=?")
		vals = append(vals, val)
	}

	val, ok = data["description"]
	if ok {
		buffer.WriteString(", description=?")
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

	vals = append(vals, data["id"])
	_, err = db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err)
		return "Error occured updating", err
	}
	return "Update Successful.", err
}

func get_subcomponent_by_id(subcomponent_id string) map[string]interface{} {
	m := make(map[string]interface{})
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	row := db.QueryRow("SELECT name, description, component_id from subcomponent where id=?", subcomponent_id)
	if err != nil {
		m["error_msg"] = err
		return m
	}
	var name, description string
	var component_id int
	err = row.Scan(&name, &description, &component_id)
	//fmt.Println(c_id, name, description)
	if err != nil {
		m["error_msg"] = err
		return m
	}
	m["id"] = subcomponent_id
	m["component_id"] = component_id
	m["description"] = description
	m["name"] = name
	return m

}

/* Finds component for a given component id*/
func get_component_by_id(component_id string) map[string]interface{} {
	m := make(map[string]interface{})
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	row := db.QueryRow("SELECT name, description, product_id, owner, qa from components where id=?", component_id)
	if err != nil {
		m["error_msg"] = err
		return m
	}
	var name, description string
	var owner, product_id int
	var qa sql.NullInt64
	err = row.Scan(&name, &description, &product_id, &owner, &qa)
	//fmt.Println(c_id, name, description)
	qaint := -1
	if qa.Valid {
		qaint = int(qa.Int64)
	}
	if err != nil {
		m["error_msg"] = err
		return m
	}
	m["id"] = component_id
	m["name"] = name
	m["description"] = description
	m["product_id"] = product_id
	m["ownername"] = get_user_email(owner)
	m["owner"] = owner
	m["qaname"] = get_user_email(qaint)
	m["qa"] = qaint
	return m

}

/*Get the details of all products */
func get_all_products() map[string][2]string {
	m := make(map[string][2]string)
	db, err := sql.Open("mysql", conn_str)
	defer db.Close()
	rows, err := db.Query("SELECT id, name, description from products order by name")
	if err != nil {
		return m
	}
	defer rows.Close()
	var name, description, p_id string
	for rows.Next() {
		err = rows.Scan(&p_id, &name, &description)
		//fmt.Println(c_id, name, description)
		m[name] = [2]string{p_id, description}
	}
	return m
}

/*
Updates an existing component.
*/
func update_component(data map[string]interface{}) (msg string, err error) {

	var buffer bytes.Buffer
	vals := make([]interface{}, 0)
	buffer.WriteString("UPDATE components SET ")

	val, ok := data["name"]
	if ok {
		buffer.WriteString("name=?")
		vals = append(vals, val)
	}

	val, ok = data["description"]
	if ok {
		buffer.WriteString(", description=?")
		vals = append(vals, val)
	}

	val, ok = data["product_id"]
	if ok {
		buffer.WriteString(", product_id=?")
		vals = append(vals, val)
	}

	val, ok = data["owner"]
	if ok {
		buffer.WriteString(", owner=?")
		vals = append(vals, val)
	}

	val, ok = data["qa"]
	if ok {
		buffer.WriteString(", qa=?")
		vals = append(vals, val)
	}

	buffer.WriteString(" WHERE id=?")
	//fmt.Println(buffer.String())

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}
	defer db.Close()

	vals = append(vals, data["id"])
	_, err = db.Exec(buffer.String(), vals...)
	if err != nil {
		fmt.Println(err)
		return "Error occured updating", err
	}
	return "Update Successful.", err
}
