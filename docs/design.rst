Design decisions
=================

We are using redis to store all search related indexes on memory. This means any
search term must be indexed in redis.

File structures
----------------

- bugspad.go 
	contains all web code
- backend.go
	contains all logic code
- redis_op.go
	contains all redis operation functions
- load_all_bugs_redis.go
	contains helper code to create index on redis