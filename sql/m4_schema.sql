m4_define(TYPE_ASSET, 1)m4_dnl     -- 資産
m4_define(TYPE_LIABILITY, 2)m4_dnl -- 負債
m4_define(TYPE_INCOME, 3)m4_dnl    -- 収入
m4_define(TYPE_EXPENSE, 4)m4_dnl   -- 費用
m4_define(TYPE_OTHER, 5)m4_dnl     -- その他
m4_define(NUM_OF_RECENT_ROWS, 30)m4_dnl  -- 最近の取引ビューの行数

/*
 * 勘定科目テーブル
 */
CREATE TABLE accounts (
    account_id SERIAL,
    account_type integer NOT NULL CHECK(account_type BETWEEN 1 AND 5),
    name varchar(8) NOT NULL,

    PRIMARY KEY (account_id)
);


/*
 * 取引テーブル
 */
CREATE TABLE transactions (
    transaction_id SERIAL,
    version integer NOT NULL DEFAULT 0,  -- 0 から始まって、更新するごとに 1 増える
    date date NOT NULL,  -- 実際に取引があった日
    debit_id integer NOT NULL REFERENCES accounts (account_id),
    credit_id integer NOT NULL REFERENCES accounts (account_id),
    amount integer NOT NULL,
    description varchar (64) NOT NULL,
    start_month integer,  -- 発生主義から見た開始月
    end_month integer,  -- 発生主義から見た終了月

    PRIMARY KEY (transaction_id),
    CHECK ((start_month IS NULL AND end_month IS NULL) OR (start_month <= end_month))
);


/*
 * 履歴テーブル
 *
 * transactions テーブルを変更すると、
 * トリガーにより自動的にこのテーブルに履歴が追加される。
 */
CREATE TABLE transactions_history (
    operation char(1) NOT NULL CHECK(operation in ('I', 'U', 'D')), -- 'I': insert, 'U': update, 'D': delete
    operate_time timestamp NOT NULL,

    -- 以下は transactions テーブルと同じ内容

    transaction_id integer NOT NULL,
    version integer NOT NULL,
    date date NOT NULL,
    debit_id integer NOT NULL REFERENCES accounts (account_id),
    credit_id integer NOT NULL REFERENCES accounts (account_id),
    amount integer NOT NULL,
    description varchar (64) NOT NULL,
    start_month integer,
    end_month integer,

    PRIMARY KEY (operation, transaction_id, version)
);


/*
 * 月ごとの集計を容易にするための作業用テーブル
 *
 * transactions テーブルを変更すると、
 * トリガーによりこのテーブルも自動的に更新される。
 * 直接 transactions_summary テーブルを作ることもできるが、
 * 開発とデバッグを容易にするために、この中間テーブルを作った。
 */
CREATE TABLE transactions_month (
    tm_id SERIAL,

    transaction_id integer NOT NULL,
    account_id integer NOT NULL REFERENCES accounts (account_id),
    month integer NOT NULL,

    accrual_debit_amount integer NOT NULL,
    accrual_credit_amount integer NOT NULL,

    cash_debit_amount integer NOT NULL,
    cash_credit_amount integer NOT NULL,

    PRIMARY KEY (tm_id)
);


/*
 * 月ごとの集計テーブル
 *
 * transactions テーブルが変更されると、
 * トリガーにより transactions_month テーブルが更新され、
 * それがまた別のトリガーを引き起こし、
 * このテーブルも自動的に更新される。
 */
CREATE TABLE transactions_summary (
    account_id integer NOT NULL REFERENCES accounts (account_id),
    month integer NOT NULL,

    -- 発生主義
    accrual_debit_amount integer NOT NULL,  -- 月ごとの借方金額
    accrual_credit_amount integer NOT NULL,  -- 月ごとの貸方金額
    accrual_accum_diff integer NOT NULL,  -- accrual_debit_amount - accrual_credit_amount の累計

    -- 現金主義
    cash_debit_amount integer NOT NULL,  -- 月ごとの借方金額
    cash_credit_amount integer NOT NULL,  -- 月ごとの貸方金額
    cash_accum_diff integer NOT NULL,  -- cash_debit_amount - cash_credit_amount の累計

    PRIMARY KEY (account_id, month)
);


/*
 * 関連テーブル
 */
CREATE TABLE relations (
    transaction_id integer NOT NULL REFERENCES transactions (transaction_id),
    related_transaction_id integer NOT NULL REFERENCES transactions (transaction_id),

    PRIMARY KEY (transaction_id, related_transaction_id)
);


/*
 * 収支ビュー
 */
CREATE OR REPLACE VIEW bp_view AS
SELECT ts.month,
       SUM(CASE WHEN ac.account_type = TYPE_INCOME THEN accrual_credit_amount - accrual_debit_amount ELSE 0 END)
       -
       SUM(CASE WHEN ac.account_type = TYPE_EXPENSE THEN accrual_debit_amount - accrual_credit_amount ELSE 0 END)
       AS balance
FROM transactions_summary AS ts
LEFT JOIN accounts AS ac ON ts.account_id = ac.account_id
GROUP BY ts.month;


/*
 * 日付から月を表す数値を取得
 * (例) '2019-03-03' を引数に与えると 201903 という数値に変換する
 */
CREATE OR REPLACE FUNCTION get_month(a_date date) RETURNS integer IMMUTABLE AS $$
    SELECT CAST ((EXTRACT(YEAR FROM a_date) * 100 + EXTRACT(MONTH FROM a_date)) AS integer);
$$ LANGUAGE SQL;


/*
 * 取引ビュー
 */
CREATE OR REPLACE VIEW transactions_view AS
SELECT tr.transaction_id, tr.date, tr.debit_id, de.name AS debit,
       tr.credit_id, cr.name AS credit, tr.amount,
       tr.description, tr.start_month, tr.end_month
FROM transactions AS tr
LEFT JOIN accounts AS de ON tr.debit_id = de.account_id
LEFT JOIN accounts AS cr ON tr.credit_id = cr.account_id;


/*
 * 残高ビュー
 */
CREATE OR REPLACE VIEW balance_view AS
WITH T AS (
    SELECT ts.*,
    CASE ac.account_type
    WHEN TYPE_ASSET THEN ts.cash_accum_diff
    WHEN TYPE_LIABILITY THEN -ts.cash_accum_diff
    END AS balance,
    ROW_NUMBER() OVER (PARTITION BY ts.account_id ORDER BY ts.month DESC) AS rn
    FROM transactions_summary AS ts
    LEFT JOIN accounts AS ac ON ts.account_id = ac.account_id
    WHERE ts.month <= get_month(current_date)
)
SELECT ac.account_id, ac.account_type, ac.name, ts.balance
FROM T AS ts
LEFT JOIN accounts AS ac ON ts.account_id = ac.account_id
WHERE ts.rn = 1 AND
(ac.account_type = TYPE_ASSET OR ac.account_type = TYPE_LIABILITY)
ORDER BY ts.account_id;


/*
 * 履歴ビュー
 */
CREATE OR REPLACE VIEW history_view AS
SELECT CASE tr.operation
       WHEN 'I' THEN 'INSERT'
       WHEN 'U' THEN 'UPDATE'
       WHEN 'D' THEN 'DELETE'
                ELSE 'UNKNOWN'
       END AS operation,
       tr.operate_time,
       tr.transaction_id, tr.date, tr.debit_id, de.name AS debit,
       tr.credit_id, cr.name AS credit, tr.amount,
       tr.description, tr.start_month, tr.end_month
FROM transactions_history AS tr
LEFT JOIN accounts AS de ON tr.debit_id = de.account_id
LEFT JOIN accounts AS cr ON tr.credit_id = cr.account_id;


CREATE TYPE month_balance AS (name varchar(8), amount integer);

/*
 * 指定月の残高を取得
 */
CREATE OR REPLACE FUNCTION get_month_balance(a_month integer, a_account_type integer) RETURNS SETOF month_balance AS $$
    SELECT ac.name,
    CASE a_account_type
    WHEN TYPE_ASSET THEN coalesce(ts.cash_accum_diff, 0)
    WHEN TYPE_LIABILITY THEN coalesce(-ts.cash_accum_diff, 0)
    WHEN TYPE_INCOME THEN coalesce(ts.accrual_credit_amount, 0) - coalesce(ts.accrual_debit_amount, 0)
    WHEN TYPE_EXPENSE THEN coalesce(ts.accrual_debit_amount, 0) - coalesce(ts.accrual_credit_amount, 0)
    END AS amount
    FROM accounts AS ac
    LEFT JOIN transactions_summary AS ts ON ts.month = a_month AND ac.account_id = ts.account_id
    WHERE ac.account_type = a_account_type
    ORDER BY ac.account_id;
$$ LANGUAGE SQL;

/*
 * 指定月の収入を取得
 */
CREATE OR REPLACE FUNCTION get_month_income(a_month integer) RETURNS SETOF month_balance AS $$
    SELECT * FROM get_month_balance(a_month, TYPE_INCOME);
$$ LANGUAGE SQL;

/*
 * 指定月の費用を取得
 */
CREATE OR REPLACE FUNCTION get_month_expense(a_month integer) RETURNS SETOF month_balance AS $$
    SELECT * FROM get_month_balance(a_month, TYPE_EXPENSE);
$$ LANGUAGE SQL;


/*
 * 取引テーブルのバージョン番号を 1 増やす
 */
CREATE OR REPLACE FUNCTION update_version() RETURNS TRIGGER AS $$
BEGIN
    IF OLD = NEW THEN
        RETURN NULL;
    END IF;

    IF OLD.version = NEW.version THEN
        NEW.version := OLD.version + 1;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_version
BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE PROCEDURE update_version();


/*
 * トリガー：取引テーブルが変更されると履歴テーブルに履歴を追加する
 */
CREATE OR REPLACE FUNCTION update_transactions_history() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        INSERT INTO transactions_history SELECT 'D', now(), OLD.*;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO transactions_history SELECT 'U', now(), NEW.*;
    ELSIF (TG_OP = 'INSERT') THEN
        INSERT INTO transactions_history SELECT 'I', now(), NEW.*;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_transactions_history
AFTER INSERT OR UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE PROCEDURE update_transactions_history();


/*
 * transactions_month テーブルへの行追加の補助関数
 */
CREATE OR REPLACE FUNCTION insert_months(a_transaction_id integer, a_debit_id integer, a_credit_id integer, a_month integer,
    a_accrual_amount integer, a_cash_amount integer) RETURNS void AS $$

    INSERT INTO transactions_month (transaction_id, account_id, month,
    accrual_debit_amount, accrual_credit_amount, cash_debit_amount, cash_credit_amount)
    VALUES (a_transaction_id, a_debit_id, a_month, a_accrual_amount, 0, a_cash_amount, 0);

    INSERT INTO transactions_month (transaction_id, account_id, month,
    accrual_debit_amount, accrual_credit_amount, cash_debit_amount, cash_credit_amount)
    VALUES (a_transaction_id, a_credit_id, a_month, 0, a_accrual_amount, 0, a_cash_amount);
$$ LANGUAGE SQL;


/*
 * １つ前の月を取得する
 * (例) 201901 を引数に与えると 201812 を返す
 */
CREATE OR REPLACE FUNCTION get_prev_month(a_month integer) RETURNS integer AS $$
DECLARE
    v_prev_month integer := a_month;
BEGIN
    v_prev_month := v_prev_month - 1;

    IF (v_prev_month % 100) = 0 THEN
        v_prev_month := (v_prev_month / 100 - 1) * 100 + 12;
    END IF;

    RETURN v_prev_month;
END;
$$ LANGUAGE plpgsql;


/*
 * １つ次の月を取得する
 * (例) 201812 を引数に与えると 201901 を返す
 */
CREATE OR REPLACE FUNCTION get_next_month(a_month integer) RETURNS integer AS $$
DECLARE
    v_next_month integer := a_month;
BEGIN
    v_next_month := v_next_month + 1;

    IF (v_next_month % 100) >= 13 THEN
        v_next_month := (v_next_month / 100 + 1) * 100 + 1;
    END IF;

    RETURN v_next_month;
END;
$$ LANGUAGE plpgsql;


/*
 * 指定した期間の月を生成する
 * (例) 201811, 201902 を引数に与えると [201811, 201812, 201901, 201902] を返す
 */
CREATE OR REPLACE FUNCTION get_months(a_start_month integer, a_end_month integer) RETURNS SETOF integer AS $$
DECLARE
    v_cnt integer := 0;
    v_month integer := a_start_month;
BEGIN
    WHILE v_month <= a_end_month LOOP
        RETURN NEXT v_month;

        v_month := v_month + 1;

        IF (v_month % 100) >= 13 THEN
            v_month := (v_month / 100 + 1) * 100 + 1;
        END IF;

        v_cnt := v_cnt + 1;

        IF v_cnt > 2048 THEN
            RAISE 'exceed the month limit.';
        END IF;
    END LOOP;

    RETURN;
END;
$$ LANGUAGE plpgsql;


/*
 * トリガー：取引を月ごとに分ける
 *
 * 取引に開始月と終了月が指定されている場合は、
 * 発生主義の金額を計算するために、金額を期間内の各月に振り分ける。
 */
CREATE OR REPLACE FUNCTION update_transactions_month() RETURNS TRIGGER AS $$
DECLARE
    v_transaction_id integer;
    v_month integer;
    v_num_months integer;
    v_remain_amount integer;
    v_amount integer;
BEGIN
    IF (TG_OP = 'DELETE') THEN
        v_transaction_id := OLD.transaction_id;
    ELSE
        v_transaction_id := NEW.transaction_id;
    END IF;

    DELETE FROM transactions_month WHERE transaction_id = v_transaction_id;

    IF (TG_OP = 'DELETE') THEN
        RETURN NULL;
    END IF;

    v_month := get_month(NEW.date);

    IF NEW.start_month IS NULL AND NEW.end_month IS NULL THEN
        -- 期間が指定されてなければ、取引日の月に金額を振り分ける
        PERFORM insert_months(v_transaction_id, NEW.debit_id, NEW.credit_id, v_month, NEW.amount, NEW.amount);

        RETURN NULL;
    END IF;

    -- 期間が指定されている場合は、開始月から終了月の間の各月に金額を振り分ける

    PERFORM insert_months(v_transaction_id, NEW.debit_id, NEW.credit_id, v_month, 0, NEW.amount);

    SELECT COUNT(*) INTO v_num_months FROM get_months(NEW.start_month, NEW.end_month);
    v_remain_amount := NEW.amount;  -- まだ振り分けてない金額
    v_amount := ceil(NEW.amount::real / v_num_months);  -- 各月に振り分ける金額

    FOR v_month IN SELECT get_months(NEW.start_month, NEW.end_month) LOOP
        IF v_amount > v_remain_amount THEN
            v_amount := v_remain_amount;
        END IF;

        PERFORM insert_months(v_transaction_id, NEW.debit_id, NEW.credit_id, v_month, v_amount, 0);

        v_remain_amount := v_remain_amount - v_amount;
    END LOOP;

    ASSERT v_remain_amount = 0, 'v_remain_amount <> 0';

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_transactions_month
AFTER INSERT OR UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE PROCEDURE update_transactions_month();


/*
 * トリガー：勘定科目,月ごとに集計する
 */
CREATE OR REPLACE FUNCTION update_transactions_summary() RETURNS TRIGGER AS $$
DECLARE
    v_rec RECORD;
    v_sign integer;
    v_row transactions_summary%ROWTYPE;
    v_month integer;
    v_accrual_diff integer;
    v_cash_diff integer;
BEGIN
    IF (TG_OP = 'UPDATE') THEN
        RAISE 'do not update transactions_month table';
    END IF;

    IF (TG_OP = 'INSERT') THEN
        -- 集計するための行がなければ作る

        SELECT * INTO v_row FROM transactions_summary WHERE account_id = NEW.account_id AND month = NEW.month;

        IF NOT FOUND THEN
            SELECT * INTO v_row FROM transactions_summary WHERE account_id = NEW.account_id AND month < NEW.month ORDER BY month DESC;

            IF FOUND THEN  -- 過去の集計データがある場合
                FOR v_month IN SELECT get_months(get_next_month(v_row.month), NEW.month) LOOP
                    INSERT INTO transactions_summary
                    VALUES (NEW.account_id, v_month, 0, 0, v_row.accrual_accum_diff, 0, 0, v_row.cash_accum_diff);
                END LOOP;
            ELSE
                SELECT * INTO v_row FROM transactions_summary WHERE account_id = NEW.account_id AND month > NEW.month ORDER BY month;

                IF FOUND THEN  -- 未来の集計データがある場合
                    FOR v_month IN SELECT get_months(NEW.month, get_prev_month(v_row.month)) LOOP
                        INSERT INTO transactions_summary
                        VALUES (NEW.account_id, v_month, 0, 0, 0, 0, 0, 0);
                    END LOOP;
                ELSE  -- 過去にも未来にも指定した勘定科目の集計データがない場合
                    INSERT INTO transactions_summary
                    VALUES (NEW.account_id, NEW.month, 0, 0, 0, 0, 0, 0);
                END IF;
            END IF;
        END IF;
    END IF;

    -- これから実際に、集計データを更新する

    IF (TG_OP = 'DELETE') THEN
        v_rec := OLD;
        v_sign := -1;
    ELSE
        v_rec := NEW;
        v_sign := 1;
    END IF;

    UPDATE transactions_summary
    SET accrual_debit_amount  = accrual_debit_amount  + v_rec.accrual_debit_amount * v_sign,
        accrual_credit_amount = accrual_credit_amount + v_rec.accrual_credit_amount * v_sign,
        cash_debit_amount     = cash_debit_amount     + v_rec.cash_debit_amount * v_sign,
        cash_credit_amount    = cash_credit_amount    + v_rec.cash_credit_amount * v_sign
    WHERE account_id = v_rec.account_id AND month = v_rec.month;

    v_accrual_diff := v_rec.accrual_debit_amount - v_rec.accrual_credit_amount;
    v_cash_diff    := v_rec.cash_debit_amount    - v_rec.cash_credit_amount;

    UPDATE transactions_summary
    SET accrual_accum_diff = accrual_accum_diff + v_accrual_diff * v_sign,
        cash_accum_diff    = cash_accum_diff    + v_cash_diff * v_sign
    WHERE account_id = v_rec.account_id AND month >= v_rec.month;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_transactions_summary
AFTER INSERT OR UPDATE OR DELETE ON transactions_month
    FOR EACH ROW EXECUTE PROCEDURE update_transactions_summary();
