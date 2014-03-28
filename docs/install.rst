Installation
==================================

Requirements
-------------

* golang
* python-requests module (to add sample components to play with)
* mariadb server
* redis
* git (to get the sources)

Install golang
---------------

Download golang from `here <http://go.googlecode.com/files/go1.1.2.linux-amd64.tar.gz>`_ , extract go directory
under your home directory.

::
	
	$ mkdir ~/gocode

Now write the following lines in your ~/.bashrc file.
::

	export PATH=$PATH:~/go/bin
	export GOPATH=~/gocode/
 	export GOROOT=~/go/

and then ::

 	$ source ~/.bashrc

Install the dependencies
------------------------- 	

After golang installation, get the dependent libraries. 
::

	$ go get github.com/garyburd/redigo/redis
	$ go get github.com/go-sql-driver/mysql
	$ go get github.com/vaughan0/go-ini
	$ go get github.com/gorilla/securecookie


Setup Mariadb (or MySQL)
-------------------------
::

	$ mysql -u root
	> CREATE USER 'bugspad'@'localhost' IDENTIFIED BY 'mypass';
	> CREATE DATABASE bugzilla;
	> GRANT ALL PRIVILEGES ON bugzilla.* TO 'bugspad'@'localhost';

Clone the git repo
-------------------

Now clone the source repo somewhere in your home directory.
::

	$ git clone https://github.com/kushaldas/bugspad.git

Create the tables
------------------------
First edit `scripts/bootstrap.sql` line 2 with your username and email id.

::
	
	$ mysql -u bugspad -pmypass bugzilla < createdb.sql
	$ mysql -u bugspad -pmypass bugzilla < bootstrap.sql

Build bugspad
-------------
::
	
	$ make

After this you have to build the helper tools also.
::

	$ go build load_all_bugs_redis.go redis_op.go backend.go

This should create a binary called `bugspad` in the directory.

Install and run redis server
----------------------------
::

	# yum install redis
	# service redis start

Customize config file
---------------------
First, copy the sample config file ``config/bugspad.ini-dist`` to ``config/bugspad.ini``.
::

    $ cp config/bugspad.ini-dist config/bugspad.ini

Now, edit ``config/bugspad.ini`` and add proper credentials(``user`` and
``password``) to access your ``bugzilla`` database.

Start the backend server
-------------------------
First run the loader to load all index data in redis.
::
	
	$ ./load_all_bugs_redis

::

	$ ./bugspad



Populate database with components
----------------------------------
So, we will put some (16k+) components in the database so that we can test.
::

	$ cd scripts
	$ wget http://kushal.fedorapeople.org/comps.json.tar.gz
	$ tar -xzvf comps.json.tar.gz

Then update `addcomponents.py` with your email id as username and execute it.
::

	$ python addcomponents.py
