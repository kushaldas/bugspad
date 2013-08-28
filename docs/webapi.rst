Web API
========

The following document explains the current Web API, remember this project
is under heavy development, so the API inputs might change a lot.


Creating a new component
-------------------------

- Request type: *POST*
- URL:          */component/*

Post data:
::

	{
	   "description":"description of the component",
	   "name":"Name",
	   "product_id":1,
	   "user":"user@example.com",
	   "password":"asdf",
	   "owner_id":1
	}

Get component list for a product
---------------------------------

- Request type: *GET*
- URL:          */components/<int: product_id>*

Output:
::

	{
	   "0ad":[
	      "522",
	      "0ad",
	      "Cross-Platform RTS Game of Ancient Warfare"
	   ],
	   "0ad-data":[
	      "523",
	      "0ad-data",
	      "The Data Files for 0 AD"
	   ],
	   "0xFFFF":[
	      "524",
	      "0xFFFF",
	      "The Open Free Fiasco Firmware Flasher"
	   ],
	   "389-admin":[
	      "525",
	      "389-admin",
	      "Admin Server for 389 Directory Server"
	   ]
	}


Create a new bug
-----------------

- Request type: *POST*
- URL:          */bug/*

Post data:
::

	{
	   "user":"username@example.com",
	   "password":"asdf",
	   "summary":"summary text of the bug",
	   "description":"description of the bug",
	   "component_id":1,
	   "status":"status of the bug",
	   "version":"version",
	   "severity":"severity",
	   "hardware":"hardware",
	   "priority":"priority",
	   "whiteboard":"whiteboard",
	   "fixedinver":"fixedinver"
	}

Output:
::

	bug_id

Default values (optional arguments)
####################################
*priority*, *severity* has a default value of "medium". *status* is "new" by default.
*hardware*, *whiteboard*, *fixedinver* is optional.

Update a bug
-----------------

- Request type: *POST*
- URL:          */updatebug/*

Post data:
::

	{
	   "user":"username@example.com",
	   "password":"asdf",
	   "bug_id":1,
	   "component_id":1,
	   "status":"status of e bug",
	   "version":"version",
	   "severity":"severity",
	   "hardware":"hardware",
	   "priority":"priority",
	   "whiteboard":"whiteboard",
	   "fixedinver":"fixedinver"
	}


Adding a comment to a bug
-------------------------

- Request type: *POST*
- URL:          */comment/*

Post data:
::
	
	{
	   "user":"username@example.com",
	   "password":"asdf",
	   "bug_id":1,
	   "desc":"comment text",
	}
