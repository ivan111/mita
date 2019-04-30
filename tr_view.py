from flask import render_template, request, redirect, abort
from psycopg2.extras import RealDictCursor

import db
from transaction import Transaction
from pagination import Pagination

PER_PAGE = 20


def use_transactions(app):

    # ------------------------------------------------------------------------
    # Detail
    # ------------------------------------------------------------------------

    TRANSACTION_SQL = '''
    SELECT *
    FROM transactions_view
    WHERE transaction_id = %s
    '''

    HISTORY_SQL = '''
    SELECT *
    FROM history_view
    WHERE transaction_id = %s
    ORDER BY operate_time DESC
    '''

    RELATIONS_SQL = '''
    SELECT *
    FROM transactions_view AS tr
         LEFT JOIN relations AS rel ON tr.transaction_id = rel.related_transaction_id
    WHERE rel.transaction_id = %s
    ORDER BY tr.date DESC, tr.transaction_id DESC
    '''

    SUMMARY_SQL = '''
    SELECT ac.name, tm.month,
           SUM(accrual_debit_amount) as accrual_debit_amount,
           SUM(accrual_credit_amount) as accrual_credit_amount,
           SUM(cash_debit_amount) as cash_debit_amount,
           SUM(cash_credit_amount) as cash_credit_amount
    FROM transactions_month AS tm
         LEFT JOIN accounts AS ac ON tm.account_id = ac.account_id
    WHERE tm.transaction_id = %s
    GROUP BY ac.name, tm.account_id, tm.month
    ORDER BY tm.month, tm.account_id
    '''

    @app.route('/transactions/<int:id>')
    def tr_detail(id):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(TRANSACTION_SQL, (id,))
                transaction = cur.fetchone()

                cur.execute(HISTORY_SQL, (id,))
                history = cur.fetchall()

                cur.execute(RELATIONS_SQL, (id,))
                relations = cur.fetchall()

                cur.execute(SUMMARY_SQL, (id,))
                summary = cur.fetchall()

        if not transaction:
            abort(404)

        relations_sum = 0

        for d in relations:
            relations_sum += d['amount']

        return render_template('tr_detail.html',
                               transaction=transaction,
                               history=history,
                               relations=relations,
                               relations_sum=relations_sum,
                               summary=summary)

    # ------------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------------

    TRANSACTIONS_SQL = '''
    SELECT *
    FROM transactions_view
    ORDER BY date DESC, transaction_id DESC
    OFFSET %s
    LIMIT %s
    '''

    @app.route('/transactions/', defaults={'page': 1})
    @app.route('/transactions/page/<int:page>')
    def tr_list(page):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute('SELECT COUNT(*) FROM transactions_view')
                count = cur.fetchone()['count']

                cur.execute(TRANSACTIONS_SQL, ((page-1) * PER_PAGE, PER_PAGE))
                transactions = cur.fetchall()

        if not transactions and page != 1:
            abort(404)

        pagination = Pagination(page, PER_PAGE, count)

        return render_template('tr_list.html',
                               pagination=pagination,
                               transactions=transactions)

    # ------------------------------------------------------------------------
    # Create, Update
    # ------------------------------------------------------------------------

    @app.route('/transactions/create', methods=['GET', 'POST'])
    def tr_create():
        return tr_update(None)

    ACCOUNTS_SQL = '''
    SELECT account_id, account_type, name
    FROM accounts
    ORDER BY account_type, account_id
    '''

    @app.route('/transactions/update/<int:transaction_id>',
               methods=['GET', 'POST'])
    def tr_update(transaction_id):
        if request.method == 'POST':
            transaction = Transaction(form=request.form)

            if transaction.validate():
                save(transaction)
                return redirect(request.args.get('next', '/transactions'))
        else:  # request.method == 'GET'
            if transaction_id is not None:
                transaction = get_tr(transaction_id)
            else:
                transaction = Transaction()

        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(ACCOUNTS_SQL)
                accounts = cur.fetchall()

        return render_template('tr_edit.html',
                               accounts=accounts,
                               transaction=transaction)

    UPDATE_TRANSACTION_SQL = '''
    UPDATE transactions
    SET date = %s, debit_id = %s, credit_id = %s, amount = %s,
        description = %s, start_month = %s, end_month = %s
    WHERE transaction_id = %s
    '''

    INSERT_TRANSACTION_SQL = '''
    INSERT INTO transactions(date, debit_id, credit_id, amount,
                             description, start_month, end_month)
    VALUES(%s, %s, %s, %s, %s, %s, %s)
    '''

    def save(transaction):
        with db.connect() as conn:
            with conn.cursor() as cur:
                d = transaction

                if d.transaction_id:
                    cur.execute(UPDATE_TRANSACTION_SQL,
                                (d.date, d.debit_id, d.credit_id, d.amount,
                                 d.description, d.start_month, d.end_month,
                                 d.transaction_id))
                else:
                    cur.execute(INSERT_TRANSACTION_SQL,
                                (d.date, d.debit_id, d.credit_id, d.amount,
                                 d.description, d.start_month, d.end_month))

    def get_tr(transaction_id):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(TRANSACTION_SQL, (transaction_id,))
                data = cur.fetchone()

        return Transaction(data=data)

    # ------------------------------------------------------------------------
    # Delete
    # ------------------------------------------------------------------------

    @app.route('/transactions/confirm_delete/<int:transaction_id>')
    def tr_confirm_delete(transaction_id):
        transaction = get_tr(transaction_id)
        return render_template('tr_delete.html', transaction=transaction)

    DELETE_TRANSACTION_SQL = '''
    DELETE FROM transactions
    WHERE transaction_id = %s
    '''

    @app.route('/transactions/delete/<int:transaction_id>')
    def tr_delete(transaction_id):
        with db.connect() as conn:
            with conn.cursor() as cur:
                cur.execute(DELETE_TRANSACTION_SQL, (transaction_id,))

        return redirect(request.args.get('next', '/transactions'))
