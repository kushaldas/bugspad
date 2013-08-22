import random
import requests
import json
s1 = """Python was conceived in the late 1980s[19] and its implementation was started in December 1989[20] by Guido van Rossum at CWI in the Netherlands as a successor to the ABC language (itself inspired by SETL)[21] capable of exception handling and interfacing with the Amoeba operating system.[1] Van Rossum is Python's principal author, and his continuing central role in deciding the direction of Python is reflected in the title given to him by the Python community, Benevolent Dictator for Life (BDFL).

Python 2.0 was released on 16 October 2000, with many major new features including a full garbage collector and support for Unicode. With this release the development process was changed and became more transparent and community-backed.[22]

Python 3.0 (also called Python 3000 or py3k), a major, backwards-incompatible release, was released on 3 December 2008[23] after a long period of testing. Many of its major features have been backported to the backwards-compatible Python 2.6 and 2.7.[24]
"""

s2 = """Python is a multi-paradigm programming language: object-oriented programming and structured programming are fully supported, and there are a number of language features which support functional programming and aspect-oriented programming (including by metaprogramming[25] and by magic methods).[26] Many other paradigms are supported using extensions, including design by contract[27][28] and logic programming.[29]

Python uses dynamic typing and a combination of reference counting and a cycle-detecting garbage collector for memory management. An important feature of Python is dynamic name resolution (late binding), which binds method and variable names during program execution.

The design of Python offers only limited support for functional programming in the Lisp tradition. The language has map(), reduce() and filter() functions, comprehensions for lists, dictionaries, and sets, as well as generator expressions. The standard library has two modules (itertools and functools) that implement functional tools borrowed from Haskell and Standard ML.[30]
"""

s3 = """Rather than requiring all desired functionality to be built into the language's core, Python was designed to be highly extensible. Python can also be embedded in existing applications that need a programmable interface. This design of a small core language with a large standard library and an easily extensible interpreter was intended by Van Rossum from the very start because of his frustrations with ABC (which espoused the opposite mindset).[19]

Python's developers strive to avoid premature optimization, and moreover, reject patches to non-critical parts of CPython which would offer a marginal increase in speed at the cost of clarity.[33] When speed is important, Python programmers use PyPy, a just-in-time compiler, or move time-critical functions to extension modules written in languages such as C. Cython is also available which translates a Python script into C and makes direct C level API calls into the Python interpreter.

An important goal of the Python developers is making Python fun to use. This is reflected in the origin of the name which comes from Monty Python,[34] and in an occasionally playful approach to tutorials and reference materials, for example using spam and eggs instead of the standard foo and bar.[35][36]
"""
data = [s1, s2, s3]
url = "http://127.0.0.1:9998/bug/comment/"
for x in range(140000, 200000):
	for i in range(1,random.randint(1,20)):
		cid = random.randint(0,2)
		desc = data[cid]
		d = {'user': 'kushaldas@gmail.com', "password": 'asdf', "bug_id": x, "desc": desc}
		r = requests.post(url, data=json.dumps(d))
		print "comment id", r.text
	print x