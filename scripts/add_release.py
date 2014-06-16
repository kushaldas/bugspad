import json
import requests


url = "http://127.0.0.1:9998/releases/"

d = {'user': 'kushaldas@gmail.com', "password": 'asdf', 'name':"F23"}
r = requests.post(url, data=json.dumps(d))
print r.text