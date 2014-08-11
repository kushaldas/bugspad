import random
import requests
import json

text = """
New submission from Vajrasky Kok:

According to:

http://oald8.oxfordlearnersdictionaries.com/dictionary/alphanumeric
http://en.wikipedia.org/wiki/Alphanumeric

Alphanumeric is defined as [A-Za-z0-9]. Underscore (_) is not one of them. One of the documentation in Python (Doc/tutorial/stdlib2.rst) differentiates them very clearly:

"The format uses placeholder names formed by ``$`` with valid Python identifiers
(alphanumeric characters and underscores).  Surrounding the placeholder with
braces allows it to be followed by more alphanumeric letters with no intervening
spaces.  Writing ``$$`` creates a single escaped ``$``::"

Yet, in documentations as well as comments in regex, we implicitely assumes underscore belongs to alphanumeric.

Explicit is better than implicit!

Attached the patch to differentiate alphanumeric and underscore in documentations and comments in regex.

This is important in case someone is confused with this code:
>>> import re
>>> re.split('\W', 'haha$hihi*huhu_hehe hoho')
['haha', 'hihi', 'huhu_hehe', 'hoho']

On the side note:
In Python code base, sometimes we write "alphanumerics" and "underscores", yet sometimes we write "alphanumeric characters" and "underscore characters". Which one again is the true way?
"""
url = "http://127.0.0.1:9998/bug/"
for x in range(10): # 1079020
	component_id = random.randint(522,530)
	summary = "Test bug with %s for %s" % (x, component_id)
	d = {'user': 'kushaldas@gmail.com', "password": 'asdf', "summary": summary, "description": text, "component_id": component_id, "version":1 }
	r = requests.post(url, data=json.dumps(d))
	# print r.text

