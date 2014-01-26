import json
import requests

f = open('comps.json')
comps = json.load(f)
f.close()

url = "http://127.0.0.1:9998/component/"
for comp in comps:
    d = {'user': 'kushaldas@gmail.com', "password": 'asdf', 'name':comp['name'], 'description': comp['description'], "product_id": 1, "owner": "kushaldas@gmail.com"}
    r = requests.post(url, data=json.dumps(d))
