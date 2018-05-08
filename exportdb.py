import os
import sys

from db import get_db, DATABASE
from account import TYPE_ASSETS, TYPE_LIABILITIES, TYPE_INCOME, TYPE_EXPENSE, TYPE_OTHER, TYPE2STR

ACCOUNTS_TSV_FILE = 'export_accounts.tsv'
TRANSACTIONS_TSV_FILE = 'export_transactions.tsv'


def use_exportdb(app):
    @app.cli.command()
    def exportdb():
        if not os.path.exists(DATABASE):
            print('データベースファイルが存在しません。')
            return

        db = get_db()

        export_accounts(db)
        export_transactions(db)

    SELECT_ACCOUNTS_SQL = '''
    SELECT id, type, name
    FROM accounts
    ORDER BY type, id
    '''

    def export_accounts(db):
        f = open(ACCOUNTS_TSV_FILE, 'w', encoding='utf-8')

        rows = db.execute(SELECT_ACCOUNTS_SQL).fetchall()

        for d in rows:
            type_str = ''

            if str(d['type']) in TYPE2STR:
                type_str = TYPE2STR[str(d['type'])]

            f.write('\t'.join([type_str, d['name']]))
            f.write('\n')

        f.close()

        print('{} 書き込み件数 = {}'.format(ACCOUNTS_TSV_FILE, len(rows)))

    SELECT_TRANSACTIONS_SQL = '''
    SELECT tr.id, date, debit_id, da.name AS debit, credit_id, ca.name AS credit, amount, note, start, end
    FROM transactions AS tr
    LEFT JOIN accounts AS da ON tr.debit_id = da.id
    LEFT JOIN accounts AS ca ON tr.credit_id = ca.id
    ORDER BY date, tr.id
    '''

    def export_transactions(db):
        f = open(TRANSACTIONS_TSV_FILE, 'w', encoding='utf-8')

        rows = db.execute(SELECT_TRANSACTIONS_SQL).fetchall()

        for d in rows:
            date = d['date'].strftime('%Y-%m-%d')

            if d['start']:
                start = d['start'].strftime('%Y-%m')
            else:
                start = ''

            if d['end']:
                end = d['end'].strftime('%Y-%m')
            else:
                end = ''

            f.write('\t'.join([date, d['debit'], d['credit'], str(d['amount']), d['note'], start, end]))
            f.write('\n')

        f.close()

        print('{} 書き込み件数 = {}'.format(TRANSACTIONS_TSV_FILE, len(rows)))
