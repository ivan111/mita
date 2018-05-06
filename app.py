from flask import Flask, render_template

from initdb import use_initdb
from importdb import use_importdb
from exportdb import use_exportdb

from tr_view import use_transactions
from ac_view import use_accounts
from balance_view import use_balance
from chart_view import use_chart

app = Flask(__name__)

use_initdb(app)
use_importdb(app)
use_exportdb(app)

use_transactions(app)
use_accounts(app)
use_balance(app)
use_chart(app)


@app.route('/')
def index():
    return render_template('index.html')


if __name__ == '__main__':
    app.run(debug=True)
