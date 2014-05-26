bugspad: bugspad.go backend.go redis_op.go
	go build bugspad.go backend.go redis_op.go webcore.go
	go build load_all_bugs_redis.go redis_op.go backend.go

web:	web4.go webcore.go
	go build web4.go webcore.go backend.go redis_op.go
