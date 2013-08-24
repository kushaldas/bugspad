import flask
from flask.ext import wtf
from datetime import time
from datetime import datetime

from wtforms import ValidationError

class LoginForm(wtf.Form):
    """ Form to log in the application. """
    username = wtf.TextField('Username', [wtf.validators.Required()])
    password = wtf.PasswordField('Password', [wtf.validators.Required()])