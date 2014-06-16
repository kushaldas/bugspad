import random
import requests
import json

url = "http://127.0.0.1:9998/latestcreated/"
r = requests.post(url)
data = json.loads(r.text)
print data

