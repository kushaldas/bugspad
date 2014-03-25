bugspad: bugspad.go backend.go redis_op.go
	go build bugspad.go backend.go redis_op.go webcore.go

web:	web4.go webcore.go
	go build web4.go webcore.go backend.go redis_op.go
