import requests
import json
from pprint import pprint

d = {"product_id": "1"}
url = "http://127.0.0.1:9998/components/"
pprint(d)
r = requests.post(url, data=json.dumps(d))
data = json.loads(r.text)
print len(data.keys())
