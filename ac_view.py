from flask import render_template, request, redirect
from psycopg2.extras import RealDictCursor
import datetime

import db
from account import Account, TYPE2STR
from pagination import Pagination

PER_PAGE = 20


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

    COUNT_SQL = '''
    SELECT COUNT(*)
    FROM transactions_view
    WHERE debit_id = %s OR credit_id = %s
    '''

    TRANSACTIONS_SQL = '''
    SELECT *
    FROM transactions_view
    WHERE debit_id = %s OR credit_id = %s
    ORDER BY date DESC
    OFFSET %s
    LIMIT %s
    '''

    SUMMARY_SQL = '''
    SELECT month,
           accrual_debit_amount,
           accrual_credit_amount,
           accrual_accum_diff,
           cash_debit_amount,
           cash_credit_amount,
           cash_accum_diff
    FROM transactions_summary
    WHERE account_id = %s AND month <= %s
    ORDER BY month DESC
    LIMIT 12
    '''

    @app.route('/accounts/<int:id>', defaults={'page': 1})
    @app.route('/accounts/<int:id>/page/<int:page>')
    def ac_detail(id, page):
        account = get_ac(id)

        now = datetime.datetime.now()
        month = now.year * 100 + now.month

        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(COUNT_SQL, (id, id))
                count = cur.fetchone()['count']

                cur.execute(TRANSACTIONS_SQL, (id, id, (page-1) * PER_PAGE, PER_PAGE))
                transactions = cur.fetchall()

                cur.execute(SUMMARY_SQL, (id, month))
                summary = cur.fetchall()

        if not transactions and page != 1:
            abort(404)

        pagination = Pagination(page, PER_PAGE, count)

        return render_template('ac_detail.html',
                               pagination=pagination,
                               account=account,
                               transactions=transactions,
                               summary=summary)

    # ------------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------------

    ACCOUNTS_SQL = '''
    SELECT account_id, account_type, name, search_words
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
    SET name = %s, search_words = %s
    WHERE account_id = %s
    '''

    INSERT_ACCOUNT_SQL = '''
    INSERT INTO accounts(account_type, name, search_words)
    VALUES(%s, %s, %s)
    '''

    def save(account):
        with db.connect() as conn:
            with conn.cursor() as cur:
                d = account

                if d.account_id:
                    cur.execute(UPDATE_ACCOUNT_SQL, (d.name, d.search_words, d.account_id))
                else:
                    cur.execute(INSERT_ACCOUNT_SQL, (d.account_type, d.name, d.search_words))

    ACCOUNT_SQL = '''
    SELECT account_id, account_type, name, search_words
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
