from flask import render_template

from db import get_db
from account import TYPE_ASSETS, TYPE_LIABILITIES


def use_balance(app):

    ACCOUNTS_SQL = '''
    SELECT id, type, name
    FROM accounts
    ORDER BY type, id
    '''

    DEBIT_SQL = '''
    SELECT debit_id, sum(amount) AS sum_amount
    FROM transactions
    GROUP BY debit_id
    '''

    CREDIT_SQL = '''
    SELECT credit_id, sum(amount) AS sum_amount
    FROM transactions
    GROUP BY credit_id
    '''

    @app.route('/balance')
    def balance():
        db = get_db()

        accounts = db.execute(ACCOUNTS_SQL).fetchall()

        debit_list = db.execute(DEBIT_SQL).fetchall()
        debit_dict = {str(d['debit_id']): d for d in debit_list}

        credit_list = db.execute(CREDIT_SQL).fetchall()
        credit_dict = {str(d['credit_id']): d for d in credit_list}

        assets = []
        liabilities = []

        sum_assets = 0
        sum_liabilities = 0

        default_val = {'sum_amount': 0}

        for account in accounts:
            debit = debit_dict.get(str(account['id']), default_val)['sum_amount']
            credit = credit_dict.get(str(account['id']), default_val)['sum_amount']
            balance = debit - credit

            if account['type'] == TYPE_ASSETS:
                sum_assets += balance
                assets.append({'name': account['name'], 'amount': balance})
            elif account['type'] == TYPE_LIABILITIES:
                sum_liabilities += balance
                liabilities.append({'name': account['name'], 'amount': balance})

        return render_template('balance.html', assets=assets, liabilities=liabilities,
                               sum_assets=sum_assets, sum_liabilities=sum_liabilities)
