from flask import render_template, request, redirect, abort
from psycopg2.extras import RealDictCursor

import db
from transaction import Transaction


def use_relations(app):

    # ------------------------------------------------------------------------
    # Create
    # ------------------------------------------------------------------------

    INSERT_RELATION_SQL = '''
    INSERT INTO relations(transaction_id, related_transaction_id)
    VALUES(%s, %s)
    '''

    @app.route('/relations/create/<int:transaction_id>', methods=['POST'])
    def rel_create(transaction_id):
        related_id = request.form['related_transaction_id']

        with db.connect() as conn:
            with conn.cursor() as cur:
                cur.execute(INSERT_RELATION_SQL,
                            (transaction_id, related_id))

        return redirect('/transactions/' + str(transaction_id))


    # ------------------------------------------------------------------------
    # Delete
    # ------------------------------------------------------------------------

    DELETE_RELATION_SQL = '''
    DELETE FROM relations
    WHERE transaction_id = %s AND related_transaction_id = %s
    '''

    @app.route('/relations/delete/<int:transaction_id>/<int:related_id>')
    def rel_delete(transaction_id, related_id):
        with db.connect() as conn:
            with conn.cursor() as cur:
                cur.execute(DELETE_RELATION_SQL, (transaction_id, related_id))

        return redirect('/transactions/' + str(transaction_id))
