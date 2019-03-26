from flask import render_template
from psycopg2.extras import RealDictCursor

import db
from account import TYPE_ASSET, TYPE_LIABILITY


def use_index(app):
    @app.route('/')
    def index():
        transactions = []

        PB_SQL = '''
        SELECT account_type, SUM(balance) AS balance
        FROM balance_view
        GROUP BY account_type
        ORDER BY account_type
        '''

        TRANSACTIONS_VIEW_SQL = '''
        SELECT *
        FROM transactions_view
        ORDER BY date DESC, transaction_id DESC
        LIMIT 10
        '''

        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute('SELECT * FROM balance_view')
                balances = cur.fetchall()

                asset_list = []
                liability_list = []

                for d in balances:
                    if d['account_type'] == TYPE_ASSET:
                        asset_list.append(d)
                    elif d['account_type'] == TYPE_LIABILITY:
                        liability_list.append(d)

                cur.execute(PB_SQL)
                pb = cur.fetchall()

                asset_sum = 0
                liability_sum = 0

                for d in pb:
                    if d['account_type'] == TYPE_ASSET:
                        asset_sum = d['balance']
                    elif d['account_type'] == TYPE_LIABILITY:
                        liability_sum = d['balance']

                cur.execute(TRANSACTIONS_VIEW_SQL)
                transactions = cur.fetchall()

        return render_template('index.html',
                               transactions=transactions,
                               asset_list=asset_list,
                               liability_list=liability_list,
                               asset_sum=asset_sum,
                               liability_sum=liability_sum,
                               balances=balances)
