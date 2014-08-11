package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"strconv"
	"time"
	"flag"
)

func main() {
	load_config("config/bugspad.ini")
	flag.Parse()
	pool = newPool(*redisServer, *redisPassword)
	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
	}
	defer db.Close()
	//loading bugs and related searchable items into redis

	rows, err := db.Query("SELECT id, status, description, version, severity, hardware, priority, whiteboard, reported, component_id, reporter, summary, fixedinver, qa, docs, assigned_to from bugs")
	//rows, err := db.Query("SELECT id, status, summary FROM bugs")
	m := make(Bug)
	if err == nil {
		var bug_id int
		var status, description, severity, hardware, priority, whiteboard, summary []byte
		var reporter, component_id, assigned_to, version int
		var qa, docs, fixedinver sql.NullInt64
		var reported time.Time
		for rows.Next() {
			err = rows.Scan(&bug_id, &status, &description, &version, &severity, &hardware, &priority, &whiteboard, &reported, &component_id, &reporter, &summary, &fixedinver, &qa, &docs, &assigned_to)
			if err == nil {
				//fmt.Println("Bug inserted",bug_id)
				qaint := -1
				docsint := -1
//				subcint := -1
				fixedinverint := -1
				if qa.Valid {
					qaint = int(qa.Int64)
				}
				if fixedinver.Valid {
					fixedinverint = int(fixedinver.Int64)
				}
				if docs.Valid {
					docsint = int(docs.Int64)
				}
/*				if subcomponent_id.Valid {
					subcint = int(subcomponent_id.Int64)
				}
*/				qa_email := ""
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
				m["id"] = bug_id
				m["status"] = string(status)
				m["summary"] = string(summary)
				m["severity"] = string(severity)
				m["description"] = string(description)
				m["hardware"] = string(hardware)
				m["priority"] = string(priority)
				m["whiteboard"] = string(whiteboard)
				m["reported"] = reported.String()
				m["reporter"] = reporter
				m["assigned_to"] = assigned_to
				if qaint != -1 {
					m["qa"] = qaint
				}
				if docsint != -1 {
					m["docs"] = docsint
				}
				m["component_id"] = component_id
//				m["subcomponent_id"] = subcint
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
//				m["subcomponent"] = get_subcomponent_name_by_id(subcint)
				m["fixedinvername"] = get_version_text(fixedinverint)
				//m["cclist"] = get_bugcc_list(bug_id)
				//bugs_idint, _ := strconv.Atoi(bug_id)
				//sid := strconv.FormatInt(int64(bug_id), 10)
				set_redis_bug(m)
				//redis_add_search_sets(m)
				//update_redis_bug_status(sid, status)
			} else {
				fmt.Println(err.Error() + strconv.Itoa(bug_id))
			}
		}
		fmt.Println("All bug indexes loaded in Redis.")

		// Now load the releases.
		clear_redis_releases()
		releases := get_releases()
		for i := range releases {
			add_redis_release(releases[i])
		}
		fmt.Println("All releases loaded in Redis.")
	} else {
		fmt.Println("err in loading data")
		fmt.Println(err.Error())
	}

	//loading the user-related bug ids
	rows, err = db.Query("SELECT bug_id, who FROM cc")
	if err == nil {
		var bug_id, user_id int
		for rows.Next() {
			err = rows.Scan(&bug_id, &user_id)
			user_idstr := strconv.Itoa(user_id)
			//fmt.Println("userbug"+user_idstr)
			redis_sadd("userbug"+user_idstr, strconv.Itoa(bug_id))
		}
		fmt.Println("All user related bugs loaded in Redis.")
	} else {
		fmt.Println("err in loading data")
		fmt.Println(err.Error())
	}

	//load bugs comments written separately for clarity, complexity remains same.
	var bugid int
	rowsout, _ := db.Query("SELECT id from bugs")
	for rowsout.Next() {
		_ = rowsout.Scan(&bugid)
		rows, _ = db.Query("SELECT users.email as useremail, users.name as username, comments.id as com_id, description, datec, bug FROM comments JOIN users WHERE bug=? and users.id=comments.user;", bugid)
		//fmt.Println(rows)
		var description, username, useremail, bug string
		var datec time.Time
		var com_id int64
		m := make(map[string]interface{})
		for rows.Next() {
			err = rows.Scan(&useremail, &username, &com_id, &description, &datec, &bug)
			//fmt.Println(description)
			//fmt.Println(com_id)
			m["id"] = com_id
			m["useremail"] = template.HTML(useremail)
			m["username"] = template.HTML(username)
			m["description"] = template.HTML(description)
			m["datec"] = template.HTML(time.Time.String(datec))
			commentdata, _ := json.Marshal(m)
			sdata := string(commentdata)
			redis_sadd("bugcomments"+bug, sdata)
		}
		defer rows.Close()
	}
	defer rowsout.Close()
	fmt.Println("All Bug Comments loaded.")

	//load components and the products associated with it
	var productid int
	rowsout, _ = db.Query("SELECT id from products")
	for rowsout.Next() {
		_ = rowsout.Scan(&productid)
		rows, _ = db.Query("SELECT id, name, description, owner, qa FROM components WHERE product_id=?", productid)
		//fmt.Println(rows)
		var description, name string
		var component_id, owner int
		var qa sql.NullInt64
		m := make(map[string]interface{})
		for rows.Next() {
			err = rows.Scan(&component_id, &name, &description, &owner, &qa)
			qaint := -1
			if qa.Valid {
				qaint = int(qa.Int64)
			}
			//fmt.Println(description)
			m["id"] = component_id
			m["name"] = name
			m["owner"] = owner
			m["description"] = description
			m["qa"] = qaint
			m["product_id"] = productid
			componentdata, _ := json.Marshal(m)
			sdata := string(componentdata)
			redis_sadd("productcomponents"+strconv.Itoa(productid), strconv.Itoa(component_id))
			redis_hset("components", strconv.Itoa(component_id), sdata)
		}
		defer rows.Close()
	}
	fmt.Println("All Components and Products associated loaded.")

}
