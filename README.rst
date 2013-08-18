Bugspad
========

Backend depends on golang. Download the latest from `here <http://code.google.com/p/go/downloads/list>`_.

After installation, get the dependent libraries. 
::

	$ go get github.com/garyburd/redigo/redis
	$ go get github.com/go-sql-driver/mysql
	$ go get github.com/vaughan0/go-ini

Update the `config/bugspad.ini` file with database details.