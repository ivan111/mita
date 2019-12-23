package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type transaction struct {
	id      int
	version int
	date    time.Time
	debit   account
	credit  account
	amount  int
	note    string
	start   int
	end     int
}

func (d *transaction) String() string {
	rng := ""

	if d.start != 0 {
		rng = fmt.Sprintf("[%s, %s]", month2str(d.start), month2str(d.end))
	}

	date := d.date.Format("2006-01-02")

	return fmt.Sprintf("%s %s / %s %s %s %s", date, d.debit.name, d.credit.name,
		int2str(d.amount), d.note, rng)
}

func cmdSearchTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tr, err := selectTransaction(db)
	if tr == nil || err != nil {
		return err
	}

	fmt.Println(tr)

	histories, err := dbGetHistory(db, tr.id)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("履歴:")

	for _, d := range histories {
		fmt.Println(&d)
	}

	return nil
}

func cmdAddTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	tr, err := scanTransaction(accounts)
	if err != nil {
		return err
	}
	if tr == nil {
		return nil
	}

	ok, err := confirmTransaction(accounts, tr)
	if err != nil {
		return err
	}

	if ok {
		if _, err = dbAddTransaction(db, tr); err != nil {
			return err
		}
	}

	return nil
}

func cmdEditTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	tr, err := selectTransaction(db)
	if tr == nil || err != nil {
		return err
	}

	ok, err := confirmTransaction(accounts, tr)
	if err != nil {
		return err
	}

	if ok {
		if err := dbEditTransaction(db, tr); err != nil {
			return err
		}
	}

	return nil
}

func cmdRemoveTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tr, err := selectTransaction(db)
	if tr == nil || err != nil {
		return err
	}

	ok := confirmRemoveTransaction(tr)
	if ok {
		err := dbRemoveTransaction(db, tr.id)
		if err != nil {
			return err
		}

		fmt.Println("削除完了")
	} else {
		fmt.Println("キャンセルした")
	}

	return nil
}

func confirmRemoveTransaction(tr *transaction) bool {
	fmt.Println(tr)
	fmt.Print("本当に削除する? (Y/[no]): ")
	stdin.Scan()
	return stdin.Text() == "Y"
}

func cmdUndoTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	d, err := selectUndoableHistory(db)
	if err != nil {
		return err
	}
	if d == nil {
		return nil
	}

	ok := confirmUndoTransaction(d)
	if ok {
		switch d.operation {
		case "DELETE":
			err = dbAddTransactionForUndo(db, d)
		case "UPDATE":
			prev, err := dbGetHistory1(db, d.tr.id, d.tr.version-1)
			if err != nil {
				return err
			}

			err = dbEditTransaction(db, &prev.tr)
			if err == nil {
				fmt.Println(&prev.tr)
			}
		case "INSERT":
			err = dbRemoveTransaction(db, d.tr.id)
		}

		if err != nil {
			return err
		}

		fmt.Println("UNDO完了")
	} else {
		fmt.Println("キャンセルした")
	}

	return nil
}

func confirmUndoTransaction(d *history) bool {
	fmt.Println(d)
	fmt.Print("本当にUNDOする? (Y/[no]): ")
	stdin.Scan()
	return stdin.Text() == "Y"
}

func confirmTransaction(accounts []account, tr *transaction) (bool, error) {
	for {
		fmt.Println()
		fmt.Println(tr)

		fmt.Print("y(es), d(ate), l(eft), r(ight), a(mount), n(ote), s(tart-end), q(uit): ")
		stdin.Scan()
		a := strings.ToLower(stdin.Text())

		switch a {
		case "q", "quit":
			return false, nil
		case "y", "yes":
			return true, nil
		case "d", "date":
			tr.date = scanDate()
		case "l", "left":
			debit, err := selectAccount(accounts, "借方")
			if err != nil {
				return false, err
			}
			if debit != nil {
				tr.debit = *debit
			}
		case "r", "right":
			credit, err := selectAccount(accounts, "貸方")
			if err != nil {
				return false, err
			}
			if credit != nil {
				tr.credit = *credit
			}
		case "a", "amount":
			tr.amount = scanAmount()
		case "n", "note":
			tr.note = scanNote()
		case "s":
			tr.start, tr.end = scanRange()
		}
	}
}

const sqlGetTransaction = `
SELECT transaction_id, version, date, debit_id, debit, debit_search_words, credit_id,
       credit, credit_search_words, amount, description, start_month, end_month
FROM transactions_view
WHERE transaction_id = $1
`

func dbGetTransaction(db *sql.DB, id int) (*transaction, error) {
	var tr transaction
	err := db.QueryRow(sqlGetTransaction, id).Scan(&tr.id, &tr.version, &tr.date,
		&tr.debit.id, &tr.debit.name, &tr.debit.searchWords,
		&tr.credit.id, &tr.credit.name, &tr.credit.searchWords,
		&tr.amount, &tr.note, &tr.start, &tr.end)
	if err != nil {
		return nil, err
	}

	return &tr, nil

}

const sqlGetTransactions = `
SELECT transaction_id, version, date, debit_id, debit, debit_search_words, credit_id,
       credit, credit_search_words, amount, description, start_month, end_month
FROM transactions_view
ORDER BY date DESC, transaction_id DESC
`

func getTransactionsReader(db *sql.DB) (io.Reader, error) {
	rows, err := db.Query(sqlGetTransactions)
	if err != nil {
		return nil, err
	}

	reader := new(bytes.Buffer)

	for rows.Next() {
		var tr transaction

		err = rows.Scan(&tr.id, &tr.version, &tr.date,
			&tr.debit.id, &tr.debit.name, &tr.debit.searchWords,
			&tr.credit.id, &tr.credit.name, &tr.credit.searchWords,
			&tr.amount, &tr.note, &tr.start, &tr.end)
		if err != nil {
			return nil, err
		}

		reader.Write([]byte(fmt.Sprintf("%d %s %s %s %s %s %s %s\n",
			tr.id, tr.date.Format("2006-01-02"), tr.debit.name, tr.credit.name,
			int2str(tr.amount), tr.note, tr.debit.searchWords, tr.credit.searchWords)))
	}
	rows.Close()

	return reader, nil
}

const sqlAddTransaction = `
INSERT INTO transactions(date, debit_id, credit_id, amount, description, start_month, end_month)
VALUES($1, $2, $3, $4, $5, $6, $7)
RETURNING transaction_id
`

func dbAddTransaction(db dbtx, tr *transaction) (string, error) {
	var id string
	err := db.QueryRow(sqlAddTransaction, tr.date, tr.debit.id, tr.credit.id, tr.amount, tr.note, tr.start, tr.end).Scan(&id)

	return id, err
}

const sqlAddTransactionForUndo = `
INSERT INTO transactions(transaction_id, version, date, debit_id, credit_id, amount, description, start_month, end_month)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

func dbAddTransactionForUndo(db dbtx, d *history) error {
	if d.operation != "DELETE" {
		return errors.New("dbAddTransactionForUndoの引数はDELETEの履歴のみ")
	}

	_, err := db.Exec(sqlAddTransactionForUndo, d.tr.id, d.tr.version+1,
		d.tr.date, d.tr.debit.id, d.tr.credit.id, d.tr.amount, d.tr.note, d.tr.start, d.tr.end)

	return err
}

const sqlEditTransaction = `
UPDATE transactions SET
date = $2,
debit_id = $3,
credit_id = $4,
amount = $5,
description = $6,
start_month = $7,
end_month = $8
WHERE transaction_id = $1
`

func dbEditTransaction(db *sql.DB, tr *transaction) error {
	_, err := db.Exec(sqlEditTransaction, tr.id, tr.date, tr.debit.id, tr.credit.id, tr.amount, tr.note, tr.start, tr.end)

	return err
}

const sqlRemoveTransaction = `
DELETE FROM transactions
WHERE transaction_id = $1
`

func dbRemoveTransaction(db *sql.DB, id int) error {
	_, err := db.Exec(sqlRemoveTransaction, id)

	return err
}

func selectTransaction(db *sql.DB) (*transaction, error) {
	src, err := getTransactionsReader(db)
	if err != nil {
		return nil, err
	}

	dst := new(bytes.Buffer)
	args := []string{}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	arr := strings.Split(dst.String(), " ")

	v, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, err
	}

	return dbGetTransaction(db, v)
}

func scanTransaction(accounts []account) (*transaction, error) {
	var tr transaction

	tr.date = scanDate()

	debit, err := selectAccount(accounts, "借方")
	if err != nil {
		return nil, err
	}
	if debit == nil {
		return nil, nil
	}
	tr.debit = *debit

	credit, err := selectAccount(accounts, "貸方")
	if err != nil {
		return nil, err
	}
	if credit == nil {
		return nil, nil
	}
	tr.credit = *credit

	tr.amount = scanAmount()
	tr.note = scanNote()
	tr.start, tr.end = scanRange()

	return &tr, nil
}

func scanDate() time.Time {
	for {
		fmt.Print("日付: ")
		stdin.Scan()
		text := stdin.Text()
		date, err := str2date(text)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			return date
		}
	}
}

func scanAmount() int {
	return scanInt("金額", math.MinInt32, math.MaxInt32)
}

func scanNote() string {
	return scanText("摘要", 0, 64)

}

func scanMonth(name string) (int, error) {
	fmt.Print(name + ": ")
	stdin.Scan()
	text := stdin.Text()

	return str2month(text)
}

func scanRange() (int, int) {
	var start, end int
	var err error

	for {
		start, err = scanMonth("開始月")

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			break
		}
	}

	if start == 0 {
		return 0, 0
	}

	for {
		end, err = scanMonth("終了月")

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if start > end {
			fmt.Fprintln(os.Stderr, "開始月 <= 終了月")
		} else {
			break
		}
	}

	return start, end
}

/*
日付形式文字列を解析して日付型へ変換

以下、d, m, y は数値を表す
空文字 : 今日の日付
-d : は今日から-nした日付。例えば-1は昨日の日付
d : 今月のn日の日付
m/d || m-d : 今年のm月d日の日付
y/m/d || y-m-d : y年m月d日の日付
*/
func str2date(s string) (time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	if len(s) == 0 {
		return today, nil
	}

	if s[0] == '-' {
		if v, err := strconv.Atoi(s[1:]); err != nil {
			return time.Time{}, err
		} else {
			date := today.AddDate(0, 0, -v)
			return date, nil
		}
	}

	if v, err := strconv.Atoi(s); err == nil {
		date := time.Date(today.Year(), today.Month(), v, 0, 0, 0, 0, time.Local)

		if date.Day() != v {
			return time.Time{}, errors.New("不正な日付")
		}

		return date, nil
	}

	arr := strings.Split(s, "-")

	if len(arr) == 1 {
		arr = strings.Split(s, "/")
	}

	if len(arr) != 2 && len(arr) != 3 {
		return time.Time{}, errors.New("不正な日付")
	}

	var iArr = make([]int, 3)

	for i, ss := range arr {
		if v, err := strconv.Atoi(ss); err != nil {
			return time.Time{}, err
		} else {
			iArr[i] = v
		}
	}

	var year, month, day int

	if len(arr) == 2 {
		year = today.Year()
		month = iArr[0]
		day = iArr[1]
	} else {
		year = iArr[0]
		month = iArr[1]
		day = iArr[2]
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)

	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return time.Time{}, errors.New("不正な日付")
	}

	return date, nil
}

func time2month(d time.Time) int {
	return d.Year()*100 + int(d.Month())
}

func month2str(d int) string {
	return fmt.Sprintf("%d-%02d", d/100, d%100)
}

/*
月文字列を解析して数値へ変換

以下、d, m, y は数値を表す
空文字 : 月入力なし
-m : は今月から-mした月。例えば-1は先月
m : 今年のm月
y/m || y-m : y年m月
yyyymm: yyyy年mm月
*/
func str2month(s string) (int, error) {
	now := time.Now()
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	if len(s) == 0 {
		return 0, nil
	}

	if s[0] == '-' {
		if v, err := strconv.Atoi(s[1:]); err != nil {
			return 0, err
		} else {
			date := thisMonth.AddDate(0, -v, 0)
			return time2month(date), nil
		}
	}

	if v, err := strconv.Atoi(s); err == nil {
		var date time.Time

		if v > (now.Year()-100)*100 && v < (now.Year()+100)*100 {
			date = time.Date(v/100, time.Month(v%100), 1, 0, 0, 0, 0, time.Local)
			v = v % 100
		} else {
			date = time.Date(now.Year(), time.Month(v), 1, 0, 0, 0, 0, time.Local)
		}

		if date.Month() != time.Month(v) {
			return 0, errors.New("不正な月")
		}

		return time2month(date), nil
	}

	arr := strings.Split(s, "-")

	if len(arr) == 1 {
		arr = strings.Split(s, "/")
	}

	if len(arr) != 2 {
		return 0, errors.New("不正な月")
	}

	var iArr = make([]int, 3)

	for i, ss := range arr {
		if v, err := strconv.Atoi(ss); err != nil {
			return 0, err
		} else {
			iArr[i] = v
		}
	}

	var year, month int

	year = iArr[0]
	month = iArr[1]

	date := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)

	if date.Year() != year || int(date.Month()) != month {
		return 0, errors.New("不正な月")
	}

	return time2month(date), nil
}

/*
数値をカンマ3桁区切りの文字列に変換
*/
func int2str(n int) string {
	s := strconv.Itoa(n)
	resLen := len(s)

	minus := false
	if s[0] == '-' {
		minus = true
		s = s[1:]
	}

	// カンマの分だけ長さを足す
	resLen += (len(s) - 1) / 3

	res := make([]byte, resLen)

	for cnt, i, k := 0, len(s)-1, len(res)-1; cnt < len(s); cnt, i, k = cnt+1, i-1, k-1 {
		if cnt != 0 && cnt%3 == 0 {
			res[k] = ','
			k--
		}

		res[k] = s[i]
	}

	if minus {
		res[0] = '-'
	}

	return string(res)
}
