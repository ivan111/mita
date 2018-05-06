import datetime

from flask import render_template, request, redirect, g

from db import get_db
from transaction import Transaction


def use_transactions(app):

    # ------------------------------------------------------------------------
    # List
    # ------------------------------------------------------------------------

    @app.route('/transactions/all')
    def tr_list_all():
        return tr_list_year(0)

    @app.route('/transactions/now')
    def tr_list_now():
        year = datetime.datetime.now().year
        return tr_list_year(year)

    TRANSACTIONS_SQL = '''
    SELECT tr.id, date, debit_id, da.name AS debit, credit_id, ca.name AS credit, amount, note, start, end
    FROM transactions AS tr
    LEFT JOIN accounts AS da ON tr.debit_id = da.id
    LEFT JOIN accounts AS ca ON tr.credit_id = ca.id
    {}
    ORDER BY date DESC, tr.id DESC
    '''

    YEARS_SQL = '''
    SELECT strftime('%Y', date) AS year
    FROM transactions
    GROUP BY year
    ORDER BY year DESC
    '''

    @app.route('/transactions/<int:param_year>')
    def tr_list_year(param_year):
        db = get_db()

        where_strs = []
        where_params = []

        if param_year != 0:
            where_strs.append("strftime('%Y', date) = ?")
            where_params.append(str(param_year))

        account = request.args.get('account')

        if account:
            where_strs.append('(debit_id = ? OR credit_id = ?)')
            where_params.extend([account, account])

        if len(where_strs) == 0:
            sql = TRANSACTIONS_SQL.format('')
            transactions = db.execute(sql).fetchall()
        else:
            where = 'WHERE ' + ' AND '.join(where_strs)
            sql = TRANSACTIONS_SQL.format(where)
            transactions = db.execute(sql, where_params).fetchall()

        years = [int(d['year']) for d in db.execute(YEARS_SQL).fetchall()]

        return render_template('tr_list.html', transactions=transactions, years=years, year=param_year)

    # ------------------------------------------------------------------------
    # Create, Update
    # ------------------------------------------------------------------------

    @app.route('/transactions/create', methods=['GET', 'POST'])
    def tr_create():
        return tr_update(None)

    ACCOUNTS_SQL = '''
    SELECT id, type, name
    FROM accounts
    ORDER BY type, id
    '''

    @app.route('/transactions/update/<transaction_id>', methods=['GET', 'POST'])
    def tr_update(transaction_id):
        db = get_db()

        if request.method == 'POST':
            transaction = Transaction(form=request.form)

            if transaction.validate():
                save(transaction)
                return redirect(request.args.get('next', '/transactions/now'))
        else:  # request.method == 'GET'
            if transaction_id is not None:
                transaction = get_tr(transaction_id)
            else:
                transaction = Transaction()

        accounts = db.execute(ACCOUNTS_SQL).fetchall()

        return render_template('tr_edit.html', accounts=accounts, transaction=transaction)

    UPDATE_TRANSACTION_SQL = '''
    UPDATE transactions
    SET date = ?, debit_id = ?, credit_id = ?, amount = ?, note = ?, start = ?, end = ?
    WHERE id = ?
    '''

    INSERT_TRANSACTION_SQL = '''
    INSERT INTO transactions(date, debit_id, credit_id, amount, note, start, end)
    VALUES(?, ?, ?, ?, ?, ?, ?)
    '''

    def save(transaction):
        db = get_db()

        d = transaction

        if d.id:
            db.execute(UPDATE_TRANSACTION_SQL, (d.date, d.debit_id, d.credit_id, d.amount, d.note, d.start, d.end, d.id))
        else:
            db.execute(INSERT_TRANSACTION_SQL, (d.date, d.debit_id, d.credit_id, d.amount, d.note, d.start, d.end))

        db.commit()

    TRANSACTION_SQL = '''
    SELECT tr.id, date, debit_id, da.name AS debit, credit_id, ca.name AS credit, amount, note, start, end
    FROM transactions AS tr
    LEFT JOIN accounts AS da ON tr.debit_id = da.id
    LEFT JOIN accounts AS ca ON tr.credit_id = ca.id
    WHERE tr.id = ?
    '''

    def get_tr(transaction_id):
        db = get_db()
        data = db.execute(TRANSACTION_SQL, (transaction_id,)).fetchone()
        return Transaction(data=data)

    # ------------------------------------------------------------------------
    # Delete
    # ------------------------------------------------------------------------

    @app.route('/transactions/confirm_delete/<transaction_id>')
    def tr_confirm_delete(transaction_id):
        transaction = get_tr(transaction_id)
        return render_template('tr_delete.html', transaction=transaction)

    DELETE_TRANSACTION_SQL = '''
    DELETE FROM transactions
    WHERE id = ?
    '''

    @app.route('/transactions/delete/<transaction_id>')
    def tr_delete(transaction_id):
        db = get_db()

        db.execute(DELETE_TRANSACTION_SQL, (transaction_id,))
        db.commit()

        return redirect(request.args.get('next', '/transactions/now'))
