import datetime

from dateutil.relativedelta import relativedelta
from flask import render_template, make_response

from db import get_db
from account import TYPE_ASSETS, TYPE_LIABILITIES


def use_chart(app):

    @app.route('/chart')
    def chart():
        return render_template('chart.html')

    ACCOUNTS_SQL = '''
    SELECT id, type, name
    FROM accounts
    ORDER BY type, id
    '''

    @app.route('/accounts.tsv')
    def accounts_tsv():
        db = get_db()
        accounts = db.execute(ACCOUNTS_SQL).fetchall()

        rows = []

        rows.append('\t'.join(['id', 'type', 'name']))

        for d in accounts:
            rows.append('\t'.join([str(d['id']), str(d['type']), d['name']]))

        data = '\n'.join(rows)

        response = make_response(data)
        response.headers['Content-Type'] = 'text/tab-separated-values'

        return response

    @app.route('/monthly_amount.tsv')
    def monthly_amount_tsv():
        # dataset: { ym + '_' + account_id: amount }
        dataset = {}

        sum_month_amount(dataset)

        if len(dataset) == 0:
            response = make_response()
            response.headers['Content-Type'] = 'text/tab-separated-values'

            return response

        db = get_db()
        accounts = db.execute(ACCOUNTS_SQL).fetchall()

        start_str = min([key[:7] for key in dataset.keys()]) + '-01'
        start_dt = datetime.datetime.strptime(start_str, '%Y-%m-%d')
        start = datetime.date(start_dt.year, start_dt.month, start_dt.day)

        accum_bs(dataset, accounts, start)

        rows = dataset2rows(dataset, accounts, start)
        data = ['\t'.join(row) for row in rows]
        data = '\n'.join(data)

        response = make_response(data)
        response.headers['Content-Type'] = 'text/tab-separated-values'

        return response

    def get_key(ym, account_id):
        return '_'.join([ym, str(account_id)])

    def add_amount(dataset, ym, account_id, amount):
        key = get_key(ym, account_id)

        if key not in dataset:
            dataset[key] = 0

        dataset[key] += amount

    DEBIT_SQL = '''
    SELECT strftime('%Y-%m', date) AS ym, debit_id AS account_id, sum(amount) AS sum_amount
    FROM transactions
    WHERE start IS NULL AND end IS NULL
    GROUP BY ym, debit_id
    '''

    CREDIT_SQL = '''
    SELECT strftime('%Y-%m', date) AS ym, credit_id AS account_id, -sum(amount) AS sum_amount
    FROM transactions
    WHERE start IS NULL AND end IS NULL
    GROUP BY ym, credit_id
    '''

    DEBIT_SPAN_SQL = '''
    SELECT strftime('%Y-%m', date) AS ym, debit_id AS account_id, type AS account_type, amount AS sum_amount, start, end
    FROM transactions AS tr
    LEFT JOIN accounts AS ac ON tr.debit_id = ac.id
    WHERE start IS NOT NULL AND end IS NOT NULL
    '''

    CREDIT_SPAN_SQL = '''
    SELECT strftime('%Y-%m', date) AS ym, credit_id AS account_id, type AS account_type, -amount AS sum_amount, start, end
    FROM transactions AS tr
    LEFT JOIN accounts AS ac ON tr.credit_id = ac.id
    WHERE start IS NOT NULL AND end IS NOT NULL
    '''

    def sum_month_amount(dataset):
        sum_month_amount_sub(dataset, DEBIT_SQL)
        sum_month_amount_sub(dataset, CREDIT_SQL)
        sum_month_amount_sub_span(dataset, DEBIT_SPAN_SQL)
        sum_month_amount_sub_span(dataset, CREDIT_SPAN_SQL)

    def sum_month_amount_sub(dataset, sql):
        db = get_db()

        data = db.execute(sql).fetchall()

        for d in data:
            add_amount(dataset, d['ym'], d['account_id'], d['sum_amount'])

    def sum_month_amount_sub_span(dataset, sql):
        db = get_db()

        data = db.execute(sql).fetchall()

        for d in data:
            # 収入・負債のときのみ金額を期間で分割する
            if d['account_type'] in [TYPE_ASSETS, TYPE_LIABILITIES]:
                add_amount(dataset, d['ym'], d['account_id'], d['sum_amount'])
            else:
                add_amount_span(dataset, d['start'], d['end'], d['account_id'], d['sum_amount'])

    def add_amount_span(dataset, start, end, account_id, sum_amount):
        months = calc_months(end, start)

        # ここで小さな誤差がでることがあるのを、ご理解いただきたい
        amount = int(sum_amount / months)

        for ym in iter_between(start, end):
            add_amount(dataset, ym, account_id, amount)

    def iter_between(start, end):
        cur = start

        while cur < end:
            ym = cur.strftime('%Y-%m')

            yield ym

            cur += relativedelta(months=1)

    def calc_months(d1, d2):
        return (d1.year - d2.year) * 12 + d1.month - d2.month + 1

    def accum_bs(dataset, accounts, start):
        end = datetime.date.today() + datetime.timedelta(days=1)

        for account in accounts:
            if account['type'] not in (TYPE_ASSETS, TYPE_LIABILITIES):
                continue

            key_prev = ''

            for ym in iter_between(start, end):
                key = get_key(ym, account['id'])

                if key_prev in dataset:
                    add_amount(dataset, ym, account['id'], dataset[key_prev])

                key_prev = key

    def dataset2rows(dataset, accounts, start):
        rows = []

        title_row = ['ym']

        for account in accounts:
            title_row.append(str(account['id']))

        rows.append(title_row)

        end = datetime.date.today() + datetime.timedelta(days=1)

        for ym in iter_between(start, end):
            row = [ym]

            for account in accounts:
                key = get_key(ym, account['id'])
                val = dataset.get(key, 0)
                row.append(str(val))

            rows.append(row)

        return rows
