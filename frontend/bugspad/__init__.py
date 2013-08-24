
import os

import flask
import vobject
from flask_fas_openid import FAS
from functools import wraps
from sqlalchemy.exc import SQLAlchemyError

from bsession import RedisSessionInterface

# import forms as forms


# Create the application.
APP = flask.Flask(__name__)
# set up FAS
# APP.config.from_object('bugspad.default_config')
APP.session_interface = RedisSessionInterface()


FAS = FAS(APP)


@APP.route('/')
def index():
    """ Displays the index page accessible at '/'
    """
    return flask.render_template('index.html')

@APP.route('/login/', methods=('GET', 'POST'))
def auth_login():
    """ Method to log into the application using FAS OpenID. """
    
    return_point = flask.url_for('index')
    if 'next' in flask.request.args:
        return_point = flask.request.args['next'] 
    
    if flask.g.fas_user:
        return flask.redirect(return_point)

    return FAS.login(return_url=return_point)


@APP.route('/logout/')
def auth_logout():
    """ Method to log out from the application. """
    if not flask.g.fas_user:
        return flask.redirect(flask.url_for('index'))
    FAS.logout()
    flask.flash('You have been logged out')
    return flask.redirect(flask.url_for('index'))