from flask import jsonify
from psycopg2.extras import RealDictCursor

import db


def use_api(app):

    BP_SQL = '''
    SELECT *
    FROM bp_view
    WHERE month BETWEEN %s AND %s
    ORDER BY month
    '''

    @app.route('/api/bp/<int:start>/<int:end>')
    def bp(start, end):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(BP_SQL, (start, end))
                data = cur.fetchall()

        return jsonify(data)

    @app.route('/api/summary/<int:start>/<int:end>')
    def summary(start, end):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute('SELECT name FROM accounts WHERE account_type = 3')
                res = cur.fetchall()
                income_keys = [d['name'] for d in res]

                cur.execute('SELECT name FROM accounts WHERE account_type = 4')
                res = cur.fetchall()
                expense_keys = [d['name'] for d in res]

                month = start

                income = []
                expense = []

                while month <= end:
                    cur.execute('SELECT * FROM get_month_income(%s)', (month,))
                    res = cur.fetchall()
                    d = {d['name']: d['amount'] for d in res}
                    d['month'] = month
                    income.append(d)

                    cur.execute('SELECT * FROM get_month_expense(%s)',
                                (month,))
                    res = cur.fetchall()
                    d = {d['name']: d['amount'] for d in res}
                    d['month'] = month
                    expense.append(d)

                    month += 1

                    if month % 100 > 12:
                        month = (int(month / 100) + 1) * 100 + 1

        data = {'income_keys': income_keys,
                'income': list(reversed(income)),
                'expense_keys': expense_keys,
                'expense': list(reversed(expense))}

        return jsonify(data)
