TYPE_ASSET = 1      # 資産
TYPE_LIABILITY = 2  # 負債
TYPE_INCOME = 3     # 収入
TYPE_EXPENSE = 4    # 費用
TYPE_OTHER = 5      # その他

TYPE_NAMES = ('資産', '負債', '収入', '費用', 'その他')
AVAILABLE_TYPES = (TYPE_ASSET, TYPE_LIABILITY,
                   TYPE_INCOME, TYPE_EXPENSE, TYPE_OTHER)

STR2TYPE = {name: tp for name, tp in zip(TYPE_NAMES, AVAILABLE_TYPES)}
TYPE2STR = {str(tp): name for name, tp in zip(TYPE_NAMES, AVAILABLE_TYPES)}


class Account:
    def __init__(self, data=None, form=None):
        if form:
            self.account_id = form.get('account_id')
            self.account_type = form.get('account_type')
            self.name = form.get('name')

            self.clean()
        elif data:
            self.account_id = data['account_id']
            self.account_type = data['account_type']
            self.name = data['name']
        else:
            self.account_id = None
            self.account_type = None
            self.name = ''

        if self.account_type == TYPE_ASSET:
            self.account_type_name = '資産'
        elif self.account_type == TYPE_LIABILITY:
            self.account_type_name = '負債'
        elif self.account_type == TYPE_INCOME:
            self.account_type_name = '収入'
        elif self.account_type == TYPE_EXPENSE:
            self.account_type_name = '費用'
        else:
            self.account_type_name = 'Unknown'

    def clean(self):
        self.account_id = conv_int(self.account_id, None)
        self.account_type = conv_int(self.account_type, None)

    def validate(self):
        if self.account_type not in AVAILABLE_TYPES:
            self.error_msg = "'type' は {} のいずれかじゃないとダメ: value = {}" \
                             .format(AVAILABLE_TYPES, self.account_type)
            return False

        return True


def conv_int(s, default_value):
    try:
        return int(s)
    except (TypeError, ValueError):
        return default_value
