import os

from db import get_db, DATABASE

def use_initdb(app):
    CREATE_TRANSACTIONS_TABLE_SQL = '''
    CREATE TABLE IF NOT EXISTS transactions (
        id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
        date date NOT NULL,
        amount integer NOT NULL,
        note varchar(64) NOT NULL,
        start date NULL,
        end date NULL,
        credit_id integer NOT NULL REFERENCES accounts (id) DEFERRABLE INITIALLY DEFERRED,
        debit_id integer NOT NULL REFERENCES accounts (id) DEFERRABLE INITIALLY DEFERRED
    )
    '''

    CREATE_ACCOUNTS_TABLE_SQL = '''
    CREATE TABLE IF NOT EXISTS accounts (
        id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
        type integer NOT NULL,
        name varchar(16) NOT NULL
    )
    '''

    @app.cli.command()
    def initdb():
        if os.path.exists(DATABASE):
            print('データベースファイルがすでに存在します。file = ' + DATABASE)
            return

        db = get_db()

        db.execute(CREATE_TRANSACTIONS_TABLE_SQL)
        db.execute(CREATE_ACCOUNTS_TABLE_SQL)
