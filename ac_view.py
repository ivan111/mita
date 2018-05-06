from flask import render_template, request, redirect, g

from db import get_db
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
    # List
    # ------------------------------------------------------------------------

    ACCOUNTS_SQL = '''
    SELECT id, type, name
    FROM accounts
    ORDER BY type, id
    '''

    @app.route('/accounts')
    def ac_list_all():
        db = get_db()

        accounts = db.execute(ACCOUNTS_SQL).fetchall()

        return render_template('ac_list.html', accounts=accounts)

    # ------------------------------------------------------------------------
    # Create, Update
    # ------------------------------------------------------------------------

    @app.route('/accounts/create', methods=['GET', 'POST'])
    def ac_create():
        return ac_update(None)

    @app.route('/accounts/update/<account_id>', methods=['GET', 'POST'])
    def ac_update(account_id):
        db = get_db()

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
    SET type = ?, name = ?
    WHERE id = ?
    '''

    INSERT_ACCOUNT_SQL = '''
    INSERT INTO accounts(type, name)
    VALUES(?, ?)
    '''

    def save(account):
        db = get_db()

        d = account

        if d.id:
            db.execute(UPDATE_ACCOUNT_SQL, (d.type, d.name, d.id))
        else:
            db.execute(INSERT_ACCOUNT_SQL, (d.type, d.name))

        db.commit()

    ACCOUNT_SQL = '''
    SELECT id, type, name
    FROM accounts
    WHERE id = ?
    '''

    def get_ac(account_id):
        db = get_db()
        data = db.execute(ACCOUNT_SQL, (account_id,)).fetchone()
        return Account(data=data)

    # ------------------------------------------------------------------------
    # Delete
    # ------------------------------------------------------------------------

    REF_SQL = '''
    SELECT count(*)
    FROM transactions
    WHERE debit_id = ? OR credit_id = ?
    '''

    @app.route('/accounts/confirm_delete/<account_id>')
    def ac_confirm_delete(account_id):
        db = get_db()

        account = get_ac(account_id)

        ref_count = db.execute(REF_SQL, (account_id, account_id)).fetchone()[0]

        return render_template('ac_delete.html', account=account, ref_count=ref_count)

    DELETE_ACCOUNT_SQL = '''
    DELETE FROM accounts
    WHERE id = ?
    '''

    @app.route('/accounts/delete/<account_id>')
    def ac_delete(account_id):
        db = get_db()

        db.execute(DELETE_ACCOUNT_SQL, (account_id,))
        db.commit()

        return redirect('/accounts')
