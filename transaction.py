import datetime
import re

from dateutil.relativedelta import relativedelta

RE_DATE = re.compile(r'^\d{4}-\d{2}-\d{2}$')
RE_YM = re.compile(r'^\d{4}-\d{2}$')


class Transaction:
    def __init__(self, data=None, form=None):
        if form:
            self.id = form.get('id')
            self.date = form.get('date')
            self.debit_id = form.get('debit')
            self.credit_id = form.get('credit')
            self.amount = form.get('amount')
            self.note = form.get('note')
            self.start = form.get('start')
            self.end = form.get('end')

            self.clean()
        elif data:
            self.id = data['id']
            self.date = data['date']
            self.debit_id = data['debit_id']
            self.debit = data['debit']
            self.credit_id = data['credit_id']
            self.credit = data['credit']
            self.amount = data['amount']
            self.note = data['note']
            self.start = data['start']
            self.end = data['end']
        else:
            self.id = None
            self.date = datetime.datetime.now().strftime('%Y-%m-%d')
            self.debit_id = None
            self.credit_id = None
            self.amount = 0
            self.note = ''
            self.start = None
            self.end = None

    def clean(self):
        self.id = conv_int(self.id, None)
        self.debit_id = conv_int(self.debit_id, None)
        self.credit_id = conv_int(self.credit_id, None)
        self.amount = conv_int(self.amount, 0)

        if self.start == '':
            self.start = None
        elif RE_YM.match(self.start):
            try:
                start = datetime.datetime.strptime(self.start + '-01', '%Y-%m-%d')
                self.start = datetime.date(start.year, start.month, start.day)
            except:
                pass

        if self.end == '':
            self.end = None
        elif RE_YM.match(self.end):
            try:
                end = datetime.datetime.strptime(self.end + '-01', '%Y-%m-%d')
                end = end + relativedelta(months=+1, days=-1)
                self.end = datetime.date(end.year, end.month, end.day)
            except:
                pass

    def validate(self):
        if not RE_DATE.match(self.date):
            self.error_msg = "'date' の書式は '\d{4}-\d{2}-\d{2}'"
            return False

        if self.debit_id is None:
            self.error_msg = "'debit_id' が数値でない"
            return False

        if self.credit_id is None:
            self.error_msg = "'credit_id' が数値でない"
            return False

        if self.start is not None and not isinstance(self.start, datetime.date):
            self.error_msg = "'start' の書式は '\d{4}-\d{2}'"
            return False

        if self.end is not None and not isinstance(self.end, datetime.date):
            self.error_msg = "'end' の書式は '\d{4}-\d{2}'"
            return False

        if (self.start is None and self.end is not None) or \
           (self.start is not None and self.end is None):
            self.error_msg = "'start' と 'end' は片方だけ設定できない"
            return False

        if self.start is not None and self.end is not None and self.start > self.end:
            self.error_msg = "'start' は 'end' よりも前でないといけない"
            return False

        return True


def conv_int(s, default_value):
    try:
        return int(s)
    except:
        return default_value
