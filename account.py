TYPE_ASSETS = 1  # 資産
TYPE_LIABILITIES = 2  # 負債
TYPE_INCOME = 3  # 収入
TYPE_EXPENSE = 4  # 費用
TYPE_OTHER = 5  # その他

AVAILABLE_TYPES = (TYPE_ASSETS, TYPE_LIABILITIES, TYPE_INCOME, TYPE_EXPENSE, TYPE_OTHER)

STR2TYPE = {
    '資産': TYPE_ASSETS,
    '負債': TYPE_LIABILITIES,
    '収入': TYPE_INCOME,
    '費用': TYPE_EXPENSE,
    'その他': TYPE_OTHER,
}

TYPE2STR = {
    str(TYPE_ASSETS): '資産',
    str(TYPE_LIABILITIES): '負債',
    str(TYPE_INCOME): '収入',
    str(TYPE_EXPENSE): '費用',
    str(TYPE_OTHER): 'その他',
}


class Account:
    def __init__(self, data=None, form=None):
        if form:
            self.id = form.get('id')
            self.type = form.get('type')
            self.name = form.get('name')

            self.clean()
        elif data:
            self.id = data['id']
            self.type = data['type']
            self.name = data['name']
        else:
            self.id = None
            self.type = None
            self.name = ''

    def clean(self):
        self.id = conv_int(self.id, None)
        self.type = conv_int(self.type, None)

    def validate(self):
        if self.type not in AVAILABLE_TYPES:
            self.error_msg = "'type' は {} のいずれかじゃないとダメ: value = {}".format(AVAILABLE_TYPES, self.type)
            return False

        return True


def conv_int(s, default_value):
    try:
        return int(s)
    except:
        return default_value
