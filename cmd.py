import datetime

import db
import click

"""
テンプレートを使って取引の登録をするコマンドラインツール

複数行をまとめて登録することができる。
金額が0の行は挿入されない。

テンプレートファイルの例:
消耗品 / 現金
雑費 / 現金
"""

def get_account_id(cur, name):
    cur.execute("select account_id from accounts where name=%s", (name,))
    return cur.fetchone()[0]

INSERT_TRANSACTION_SQL = '''
INSERT INTO transactions(date, debit_id, credit_id, amount,
                         description, start_month, end_month)
VALUES(%s, %s, %s, %s, %s, %s, %s)
'''

def run_insert_template(date, lines, amounts):
    accounts = dict()

    with db.connect() as conn:
        with conn.cursor() as cur:
            for line, amount in zip(lines, amounts):
                a1, a2 = line.split('/')
                a1 = a1.strip()
                a2 = a2.strip()

                for ac in [a1, a2]:
                    if ac not in accounts:
                        account_id = get_account_id(cur, ac)
                        accounts[ac] = account_id

                if amount != 0:
                    cur.execute(INSERT_TRANSACTION_SQL,
                                (date, accounts[a1], accounts[a2], amount,
                                 '', None, None))

            print('Done')

def use_cmd(app):
    @app.cli.command("insert-template")
    @click.argument("name")
    def insert_template(name):
        template_file = name
        lines = []
        amounts = []

        with open(template_file) as f:
            for line in f:
                line = line.strip()

                if line != '':
                    lines.append(line)

        print('日付(yyyy-mm-dd)')
        string_date = input()
        date = datetime.datetime.strptime(string_date, '%Y-%m-%d')

        for line in lines:
            print(line)
            amounts.append(int(input()))

        while True:
            print()
            print('d: {}'.format(date.strftime('%Y-%m-%d'),))

            for i, (line, amount) in enumerate(zip(lines, amounts)):
                print('{}: {} {:,}'.format(i, line, amount))

            print()
            print('y/q/d/0-{}'.format(len(lines)-1,))
            cmd = input()

            if cmd == 'y':
                run_insert_template(date, lines, amounts)
                break
            elif cmd == 'q':
                break
            elif cmd == 'd':
                print('日付(yyyy-mm-dd)')
                string_date = input()
                date = datetime.datetime.strptime(string_date, '%Y-%m-%d')
            elif str.isdecimal(cmd) and int(cmd) < len(lines):
                line_no = int(cmd)
                print(lines[line_no])
                amounts[line_no] = int(input())
