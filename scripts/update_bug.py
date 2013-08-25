import random
import requests
import json


url = "http://127.0.0.1:9998/updatebug/"

d = {'user': 'kushaldas@gmail.com', "password": 'asdf', "bug_id": 1, "status": "new", "hardware": "x86_64" }
r = requests.post(url, data=json.dumps(d))
print r.text

