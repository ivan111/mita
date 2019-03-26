from flask import render_template, abort
from psycopg2.extras import RealDictCursor

import db
from pagination import Pagination

PER_PAGE = 20


def use_history(app):

    HISTORY_SQL = '''
    SELECT *
    FROM history_view
    ORDER BY operate_time DESC
    OFFSET %s
    LIMIT %s
    '''

    @app.route('/history/', defaults={'page': 1})
    @app.route('/history/page/<int:page>')
    def hl_list(page):
        with db.connect() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute('SELECT COUNT(*) FROM history_view')
                count = cur.fetchone()['count']

                cur.execute(HISTORY_SQL, ((page-1) * PER_PAGE, PER_PAGE))
                history = cur.fetchall()

        if not history and page != 1:
            abort(404)

        pagination = Pagination(page, PER_PAGE, count)

        return render_template('hl_list.html',
                               pagination=pagination,
                               history=history)
