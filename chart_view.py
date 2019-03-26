from flask import render_template
import datetime

import db


def use_chart(app):

    OLDEST_YEAR_SQL = '''
    SELECT EXTRACT(YEAR FROM date)::int
    FROM transactions
    ORDER BY date
    LIMIT 1
    '''

    @app.route('/chart/', defaults={'year': datetime.datetime.now().year})
    @app.route('/chart/year/<int:year>')
    def chart(year):
        with db.connect() as conn:
            with conn.cursor() as cur:
                cur.execute(OLDEST_YEAR_SQL)
                oldest_year = cur.fetchone()[0]
                print(oldest_year)

        years = list(range(datetime.datetime.now().year, oldest_year - 1, -1))

        return render_template('chart.html', years=years, year=year)
