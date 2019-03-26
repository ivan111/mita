import os

from flask import Flask

from db import use_db
from index import use_index
from tr_view import use_transactions
from chart_view import use_chart
from ac_view import use_accounts
from hl_view import use_history
from api import use_api
from jinja import use_jinja


app = Flask(__name__)

use_db(app)
use_index(app)
use_transactions(app)
use_chart(app)
use_accounts(app)
use_history(app)
use_api(app)
use_jinja(app)
