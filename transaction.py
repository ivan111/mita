import datetime
import re

RE_DATE = re.compile(r'^\d{4}-\d{2}-\d{2}$')


class Transaction:
    def __init__(self, data=None, form=None):
        if form:
            self.transaction_id = form.get('transaction_id')
            self.date = form.get('date')
            self.debit_id = form.get('debit')
            self.credit_id = form.get('credit')
            self.amount = form.get('amount')
            self.description = form.get('description')
            self.start_month = form.get('start_month')
            self.end_month = form.get('end_month')

            self.clean()
        elif data:
            self.transaction_id = data['transaction_id']
            self.date = data['date']
            self.debit_id = data['debit_id']
            self.debit = data['debit']
            self.credit_id = data['credit_id']
            self.credit = data['credit']
            self.amount = data['amount']
            self.description = data['description']
            self.start_month = data['start_month']
            self.end_month = data['end_month']
        else:
            self.transaction_id = None
            self.date = datetime.datetime.now().strftime('%Y-%m-%d')
            self.debit_id = None
            self.credit_id = None
            self.amount = 0
            self.description = ''
            self.start_month = None
            self.end_month = None

    def clean(self):
        self.transaction_id = conv_int(self.transaction_id, None)
        self.debit_id = conv_int(self.debit_id, None)
        self.credit_id = conv_int(self.credit_id, None)
        self.amount = conv_int(self.amount, 0)
        self.start_month = conv_int(self.start_month, None)
        self.end_month = conv_int(self.end_month, None)

    def validate(self):
        if not RE_DATE.match(self.date):
            self.error_msg = "'date' の書式は '\\d{4}-\\d{2}-\\d{2}'"
            return False

        if self.debit_id is None:
            self.error_msg = "'debit_id' が数値でない"
            return False

        if self.credit_id is None:
            self.error_msg = "'credit_id' が数値でない"
            return False

        if (self.start_month is None and self.end_month is not None) or \
           (self.start_month is not None and self.end_month is None):
            self.error_msg = "'start_month' と 'end_month' は片方だけ設定できない"
            return False

        if self.start_month is not None:
            month = self.start_month % 100
            if month < 1 or 12 < month:
                self.error_msg = "'start_month' の月が 1 から 12 の間でない"
                return False

        if self.end_month is not None:
            month = self.end_month % 100
            if month < 1 or 12 < month:
                self.error_msg = "'end_month' の月が 1 から 12 の間でない"
                return False

        if self.start_month is not None and self.end_month is not None and \
           self.start_month > self.end_month:
            self.error_msg = "'start_month' は 'end_month' よりも前でないといけない"
            return False

        return True


def conv_int(s, default_value):
    try:
        return int(s)
    except (TypeError, ValueError):
        return default_value
