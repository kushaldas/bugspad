package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
)

// TODO: Update this codebase to keep all search related indexs on redis.

func update_redis_bug_status(bug_id string, status string) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	defer conn.Close()
	_, err = conn.Do("HSET", "b_status:"+status, bug_id, 1)
	if err != nil {
		// handle error
		fmt.Print(err)
		return
	}
}
