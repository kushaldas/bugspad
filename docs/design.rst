Design decisions
=================

We are using redis to store all search related indexes on memory. This means any
search term must be indexed in redis. Mainly we are using "bugs" hash for storing
bug json information. Apart from that we would be using redis sets, used to enhance
the searching process, which are described below

- componentbug:<id> 
	Here <id> is the component id, and this set stores all bugs of belonging to 
	the component with that id.

- productbug:<id> 
	Here <id> is the product id, and this set stores all bugs of belonging to 
	the product with that id.

- versionbug:<id>
	Here <id> is the version id from the versions table, and this set stores all 
	bugs of belonging to the version with that id.

- fixedinverbug:<id>
	Here <id> is the version id in which the bug was fixed (if fixed), and this set 
	stores all bugs which were fixed in that version.

- prioritybug:<priority-name>
	Here we store all the bugs which have same priority.

- statusbug:<status-name>
	Here we store all the bugs which have the same status.

- severitybug:<severity-name>
	Here we store all the bugs which have the same severity.

- userbug<id>
	This stores all the bugs related to a particular user with userid <id>.

- assigned_tobug:<id>
	This stores all the bugs assigned to a particular user with userid <id>.

- reporterbug:<id>
	This stores all the bugs reported by a particular user with userid <id>.

- assigned_tobug:<id>
	This stores all the bugs assigned to a particular user with userid <id>.

- docsbug:<id>
	This stores all the bugs with the document maintained by particular user with userid <id>.

- qabug:<id>
	This stores all the bugs with the QA maintainer as the user with the userid <id>.


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