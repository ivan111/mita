import psycopg2

from flask import current_app, g


def connect():
    if 'db' not in g:
        g.db = psycopg2.connect('dbname=mita')

    return g.db


def close_db(e=None):
    db = g.pop('db', None)

    if db is not None:
        db.close()


def use_db(app):
    app.teardown_appcontext(close_db)
