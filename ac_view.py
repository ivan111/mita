from flask import render_template, request, redirect
from psycopg2.extras import RealDictCursor
import datetime

import db
from account import Account, TYPE2STR


def use_accounts(app):

    @app.template_filter('ac_type2name')
    def ac_type2name(value):
        name = ''

        if str(value) in TYPE2STR:
            name = TYPE2STR[str(value)]
        else:
            name = '不明(' + str(value) + ')'

        return name

    # ------------------------------------------------------------------------
    # Detail
    # ------------------------------------------------------------------------

    TRANSACTIONS_SQL = '''
    SELECT *
    FROM transactions_view
    WHERE debit_id = %s OR credit_id = %s
    ORDER BY date DESC
    LIMIT 20
    '''

    SUMMARY_SQL = '''
    SELECT month,
           SUM(accrual_debit_amount) as accrual_debit_amount,
           SUM(accrual_credit_amount) as accrual_credit_amount,
           SUM(cash_debit_amount) as cash_debit_amount,
           SUM(cash_credit_amount) as cash_credit_amount
    FROM transactions_month
    WHERE account_id = %s AND month <= %s
    GROUP BY account_id, month
    ORDER BY month DESC
    LIMIT 12
    '''

    @app.route('/accounts/<int:id>')
    def ac_detail(id):
        account = get_ac(id)

        now = datetime.datetime.now()
        month = now.year * 100 + now.month

        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(TRANSACTIONS_SQL, (id, id))
                transactions = cur.fetchall()

                cur.execute(SUMMARY_SQL, (id, month))
                summary = cur.fetchall()

        return render_template('ac_detail.html',
                               account=account,
                               transactions=transactions,
                               summary=summary)

    # ------------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------------

    ACCOUNTS_SQL = '''
    SELECT account_id, account_type, name
    FROM accounts
    ORDER BY account_type, account_id
    '''

    @app.route('/accounts')
    def ac_list():
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(ACCOUNTS_SQL)
                accounts = cur.fetchall()

        return render_template('ac_list.html', accounts=accounts)

    # ------------------------------------------------------------------------
    # Create, Update
    # ------------------------------------------------------------------------

    @app.route('/accounts/create', methods=['GET', 'POST'])
    def ac_create():
        return ac_update(None)

    @app.route('/accounts/update/<int:account_id>', methods=['GET', 'POST'])
    def ac_update(account_id):
        if request.method == 'POST':
            account = Account(form=request.form)

            if account.validate():
                save(account)
                return redirect('/accounts')
        else:  # request.method == 'GET'
            if account_id is not None:
                account = get_ac(account_id)
            else:
                account = Account()

        return render_template('ac_edit.html', account=account)

    UPDATE_ACCOUNT_SQL = '''
    UPDATE accounts
    SET name = %s
    WHERE account_id = %s
    '''

    INSERT_ACCOUNT_SQL = '''
    INSERT INTO accounts(account_type, name)
    VALUES(%s, %s)
    '''

    def save(account):
        with db.connect() as conn:
            with conn.cursor() as cur:
                d = account

                if d.account_id:
                    cur.execute(UPDATE_ACCOUNT_SQL, (d.name, d.account_id))
                else:
                    cur.execute(INSERT_ACCOUNT_SQL, (d.account_type, d.name))

    ACCOUNT_SQL = '''
    SELECT account_id, account_type, name
    FROM accounts
    WHERE account_id = %s
    '''

    def get_ac(account_id):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(ACCOUNT_SQL, (account_id,))
                data = cur.fetchone()

        return Account(data=data)

    # ------------------------------------------------------------------------
    # Delete
    # ------------------------------------------------------------------------

    REF_SQL = '''
    SELECT count(*)
    FROM transactions
    WHERE debit_id = %s OR credit_id = %s
    '''

    @app.route('/accounts/confirm_delete/<int:account_id>')
    def ac_confirm_delete(account_id):
        account = get_ac(account_id)

        with db.connect() as conn:
            with conn.cursor() as cur:
                cur.execute(REF_SQL, (account_id, account_id))
                ref_count = cur.fetchone()[0]

        return render_template('ac_delete.html',
                               account=account,
                               ref_count=ref_count)

    DELETE_ACCOUNT_SQL = '''
    DELETE FROM accounts
    WHERE account_id = %s
    '''

    @app.route('/accounts/delete/<int:account_id>')
    def ac_delete(account_id):
        with db.connect() as conn:
            with conn.cursor() as cur:
                cur.execute(DELETE_ACCOUNT_SQL, (account_id,))

        return redirect('/accounts')
