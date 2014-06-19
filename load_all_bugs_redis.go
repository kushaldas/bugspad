package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
)

func main() {
	load_config("config/bugspad.ini")

	db, err := sql.Open("mysql", conn_str)
	if err != nil {
		// handle error
		fmt.Print(err)
	}
	defer db.Close()
	// TODO: Update all searchable columns from bugs
	rows, err := db.Query("SELECT id, status, summary FROM bugs")
	if err == nil {
		var id int64
		var status, summary string
		for rows.Next() {
			err = rows.Scan(&id, &status, &summary)
			sid := strconv.FormatInt(id, 10)
			set_redis_bug(id, status, summary)
			update_redis_bug_status(sid, status)
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
	defer rows.Close()

}
