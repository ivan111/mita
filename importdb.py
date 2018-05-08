import datetime
import os
import re
import sys

from dateutil.relativedelta import relativedelta
import click

from db import get_db, DATABASE
from account import TYPE_ASSETS, TYPE_LIABILITIES, TYPE_INCOME, TYPE_EXPENSE, TYPE_OTHER, STR2TYPE


def use_importdb(app):
    @app.cli.command()
    @click.argument('filename')
    def import_accounts(filename):
        check_database_exist()
        check_file_exist(filename)

        db = get_db()

        accounts = read_accounts_tsv(filename)
        insert_accounts(db, accounts)

        db.commit()

    @app.cli.command()
    @click.argument('filename')
    def import_transactions(filename):
        check_database_exist()
        check_file_exist(filename)

        db = get_db()

        transactions = read_transactions_tsv(db, filename)
        insert_transactions(db, transactions)

        db.commit()

    def check_database_exist():
        if not os.path.exists(DATABASE):
            print('データベースファイルが存在しません。\nflask initdb コマンドを実行してください。')

            sys.exit(1)

    def check_file_exist(filename):
        if not os.path.exists(filename):
            print('ファイル {} が存在しません。'.format(filename))

            sys.exit(1)

    def perror(filename, line, lineno, msg):
        print('{} ファイルの形式が正しくありません。行番号 = {}\n{}\n\n{}'
              .format(filename, lineno, line, msg))

        sys.exit(1)

    def read_accounts_tsv(filename):
        accounts = []

        for i, line in enumerate(open(filename, encoding='utf-8')):
            lineno = i + 1
            line = line.rstrip()

            if len(line) == 0:
                continue

            if line[0] == '#':
                continue

            row = line.split('\t')

            if len(row) != 2:
                perror(filename, line, lineno,
                       '列の数が正しくありません。すべての列がタブで区切られているか確認してください。必要な列数 = 2, 読み込まれたの列数 = {}'.format(len(row)))

            type_str, name = row

            if type_str in STR2TYPE:
                account_type = STR2TYPE[type_str]
            else:
                perror(filename, line, lineno,
                    '不明な区分 = {}。区分は "資産", "負債", "収入", "費用", "その他" のいずれかにしてください。{}'
                    .format(type_str))

            accounts.append({
                'type': account_type,
                'name': name,
            })

        print('{} 読み込み件数 = {}'.format(filename, len(accounts)))

        return accounts

    INSERT_ACCOUNT_SQL = 'INSERT INTO accounts(type, name) VALUES (?,?)'

    def insert_accounts(db, accounts):
        for d in accounts:
            db.execute(INSERT_ACCOUNT_SQL, [d['type'], d['name']])

    SELECT_ACCOUNTS = '''
    SELECT id, name
    FROM accounts
    '''

    def create_name2id(db):
        name2id = {}

        rows = db.execute(SELECT_ACCOUNTS).fetchall()

        for row in rows:
            name2id[row['name']] = row['id']

        return name2id

    RE_DATE = re.compile(r'^[12]\d{3}-[01]\d-[0123]\d$')
    RE_YM = re.compile(r'^[12]\d{3}-[01]\d$')

    def read_transactions_tsv(db, filename):
        name2id = create_name2id(db)

        transactions = []

        for i, line in enumerate(open(filename, encoding='utf-8')):
            lineno = i + 1
            line = line.rstrip('\r\n')

            if len(line) == 0:
                continue

            if line[0] == '#':
                continue

            row = line.split('\t')

            if len(row) != 7:
                perror(filename, line, lineno,
                    '列の数が正しくありません。すべての列がタブで区切られているか確認してください。必要な列数 = 7, 読み込まれたの列数 = {}'.format(len(row)))

            date, debit, credit, amount_str, note, start, end = row

            if not RE_DATE.match(date):
                perror(filename, line, lineno,
                    '日付の形式(YYYY-MM-DD)が正しくありません。{}'.format(date))

            if debit not in name2id:
                perror(filename, line, lineno,
                    '存在しない勘定科目です。{}'.format(debit))

            debit_id = name2id[debit]

            if credit not in name2id:
                perror(filename, line, lineno,
                    '存在しない勘定科目です。{}'.format(credit))

            credit_id = name2id[credit]

            try:
                amount = int(amount_str.replace(',', ''))
            except:
                perror(filename, line, lineno,
                    '金額が数値ではありません。{}'.format(amount_str))

            if start == '':
                start = None
            elif RE_YM.match(start):
                start = start + '-01'
            else:
                perror(filename, line, lineno,
                    '開始日付の形式(YYYY-MM)が正しくありません。{}'.format(start))

            if end == '':
                end = None
            elif RE_YM.match(end):
                end_dt = datetime.datetime.strptime(end + '-01', '%Y-%m-%d')
                end_dt = end_dt + relativedelta(months=+1, days=-1)
                end = end_dt.strftime('%Y-%m-%d')
            else:
                perror(filename, line, lineno,
                    '終了日付の形式(YYYY-MM)が正しくありません。{}'.format(end))

            transactions.append({
                'date': date,
                'debit_id': debit_id,
                'credit_id': credit_id,
                'amount': amount,
                'note': note,
                'start': start,
                'end': end,
            })

        print('{} 読み込み件数 = {}'.format(filename, len(transactions)))

        return transactions

    INSERT_TRANSACTION_SQL = 'INSERT INTO transactions(date, debit_id, credit_id, amount, note, start, end) VALUES (?,?,?,?,?,?,?)'

    def insert_transactions(db, transactions):
        for d in transactions:
            db.execute(INSERT_TRANSACTION_SQL, [d['date'], d['debit_id'], d['credit_id'], d['amount'], d['note'], d['start'], d['end']])
