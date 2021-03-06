package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
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

func cmdListTransactions(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runListTransactions(db, context.Bool("all"), context.Args().First())
}

func runListTransactions(db *sql.DB, isAll bool, monthStr string) error {
	var transactions []transaction
	var err error

	if isAll {
		transactions, err = getTransactions(db, false)
		if err != nil {
			return err
		}
	} else {
		if monthStr == "" {
			monthStr = "-0" // 今月
		}

		ym, err := str2month(monthStr)
		if err != nil {
			return err
		}

		year := ym / 100
		month := ym % 100

		transactions, err = getTransactionsByMonth(db, year, month)
		if err != nil {
			return err
		}
	}

	src := getTransactionsReader(transactions, false)
	print(src)

	return nil
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

	histories, err := dbGetHistory(db, tr.id)
	if err != nil {
		return err
	}

	println()
	println("履歴:")

	for _, d := range histories {
		println(&d)
	}

	return nil
}

func cmdAddTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runAddTransaction(db, context.Args().Slice())
}

func runAddTransaction(db *sql.DB, args []string) error {
	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	var d *transaction

	switch len(args) {
	case 0:
		d, err = scanTransaction(accounts)
		if err != nil {
			return err
		}
		if d == nil {
			return nil
		}

		ok, err := confirmTransaction(accounts, d)
		if err != nil {
			return err
		}

		if ok == false {
			return nil
		}
	case 4, 5, 7:
		name2id := make(map[string]int)

		for _, d := range accounts {
			name2id[d.name] = d.id
		}

		d, err = arr2transaction(name2id, args)
		if err != nil {
			return err
		}
	default:
		return errors.New("Usage: mita transaction add date debit credit amount [description] [startMonth endMonth]")
	}

	_, err = dbAddTransaction(db, d)

	return err
}

func cmdEditTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runEditTransaction(db)
}

func runEditTransaction(db *sql.DB) error {
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

	return runRemoveTransaction(db)
}

func runRemoveTransaction(db *sql.DB) error {
	tr, err := selectTransaction(db)
	if tr == nil || err != nil {
		return err
	}

	if confirmYesNo("本当に削除する?") {
		err = dbRemoveTransaction(db, tr.id)
	}

	return err
}

func cmdUndoTransaction(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runUndoTransaction(db)
}

func runUndoTransaction(db *sql.DB) error {
	d, err := selectUndoableHistory(db)
	if err != nil {
		return err
	}
	if d == nil {
		return nil
	}

	if confirmYesNo("本当にUNDOする? ") {
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
				println(&prev.tr)
			}
		case "INSERT":
			err = dbRemoveTransaction(db, d.tr.id)
		}
	}

	return err
}

func cmdImportTransactions(context *cli.Context) error {
	return importItems(context.Args().First(), readTransactions)
}

func readTransactions(db *sql.DB, f io.Reader) error {
	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	name2id := make(map[string]int)

	for _, d := range accounts {
		name2id[d.name] = d.id
	}

	scanner := bufio.NewScanner(f)

	lineNo := 0

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for scanner.Scan() {
		lineNo++

		line := skipSpace(scanner.Text())

		if line == "" || line[0] == '#' {
			continue
		}

		arr := strings.Split(line, "\t")

		d, err := arr2transaction(name2id, arr)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("%d:%s", lineNo, err)
		}

		_, err = dbAddTransaction(tx, d)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err = scanner.Err(); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func arr2transaction(name2id map[string]int, arr []string) (*transaction, error) {
	arrLen := len(arr)
	if !(arrLen == 4 || arrLen == 5 || arrLen == 7) {
		return nil, fmt.Errorf("項目数が4, 5, 7でない")
	}

	var d transaction

	date, err := str2date(arr[0])
	if err != nil {
		return nil, fmt.Errorf("日付:%s", err)
	}
	d.date = date

	d.debit.id = name2id[arr[1]]
	if d.debit.id == 0 {
		return nil, fmt.Errorf("借方:存在しない勘定科目'%s'", arr[1])
	}

	d.credit.id = name2id[arr[2]]
	if d.credit.id == 0 {
		return nil, fmt.Errorf("貸方:存在しない勘定科目'%s'", arr[2])
	}

	amount, err := strconv.Atoi(arr[3])
	if err != nil {
		return nil, fmt.Errorf("金額:%s", err)
	}
	d.amount = amount

	if len(arr) >= 5 {
		d.note = arr[4]
	}

	if len(arr) >= 7 {
		var start int
		if arr[5] != "0" {
			start, err = str2month(arr[5])
			if err != nil {
				return nil, fmt.Errorf("開始月:%s", err)
			}
		}
		d.start = start

		var end int
		if arr[6] != "0" {
			end, err = str2month(arr[6])
			if err != nil {
				return nil, fmt.Errorf("終了月:%s", err)
			}
		}
		d.end = end

		if (d.start == 0 && d.end != 0) ||
			(d.start != 0 && d.end == 0) {
			return nil, fmt.Errorf("開始月と終了月は両方設定するか、両方設定しないかじゃないとダメ。開始月 = %d, 終了月 = %d", d.start, d.end)
		}
	}

	return &d, nil
}

func cmdExportTransactions(context *cli.Context) error {
	return exportItems(context.Args().First(), writeTransactions)
}

func writeTransactions(db *sql.DB, f io.Writer) error {
	b := bufio.NewWriter(f)

	transactions, err := getTransactions(db, false)
	if err != nil {
		return err
	}

	for _, d := range transactions {
		date := d.date.Format("2006-01-02")

		_, err := b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%d\t%d\n",
			date, d.debit.name, d.credit.name, d.amount, d.note, d.start, d.end))
		if err != nil {
			return err
		}
	}

	b.Flush()

	return nil
}

func confirmTransaction(accounts []account, tr *transaction) (bool, error) {
	for {
		println()
		println(tr)

		print("y(es), d(ate), l(eft), r(ight), a(mount), n(ote), s(tart-end), q(uit): ")
		s, err := input()
		if err != nil {
			return false, err
		}
		a := strings.ToLower(s)

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

func rows2transactions(rows *sql.Rows) ([]transaction, error) {
	var transactions []transaction

	for rows.Next() {
		var tr transaction

		err := rows.Scan(&tr.id, &tr.version, &tr.date,
			&tr.debit.id, &tr.debit.name, &tr.debit.searchWords,
			&tr.credit.id, &tr.credit.name, &tr.credit.searchWords,
			&tr.amount, &tr.note, &tr.start, &tr.end)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, tr)
	}
	rows.Close()

	return transactions, nil
}

const transactionRows = `
transaction_id, version, date, debit_id, debit, debit_search_words, credit_id,
credit, credit_search_words, amount, description, start_month, end_month
`

const sqlGetTransaction = `
SELECT ` + transactionRows + `
FROM transactions_view
WHERE transaction_id = $1
`

func dbGetTransaction(db *sql.DB, id int) (*transaction, error) {
	rows, err := db.Query(sqlGetTransaction, id)
	if err != nil {
		return nil, err
	}

	transactions, err := rows2transactions(rows)
	if err != nil {
		return nil, err
	}

	if len(transactions) != 1 {
		return nil, errors.New("dbGetTransaction: len(transactions) != 1")
	}

	return &transactions[0], nil

}

const sqlGetTransactions = `
SELECT ` + transactionRows + `
FROM transactions_view
`

func getTransactions(db *sql.DB, reverse bool) ([]transaction, error) {
	var sql string
	if reverse {
		sql = sqlGetTransactions + "ORDER BY date DESC, transaction_id DESC"
	} else {
		sql = sqlGetTransactions + "ORDER BY date, transaction_id"
	}

	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}

	return rows2transactions(rows)
}

const sqlGetTransactionsByMonth = `
SELECT ` + transactionRows + `
FROM transactions_view
WHERE EXTRACT(year FROM "date") = $1
      AND EXTRACT(month FROM "date") = $2
ORDER BY date, transaction_id
`

func getTransactionsByMonth(db *sql.DB, year int, month int) ([]transaction, error) {
	rows, err := db.Query(sqlGetTransactionsByMonth, year, month)
	if err != nil {
		return nil, err
	}

	return rows2transactions(rows)
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

func tr2alignedString(d *transaction) string {
	date := d.date.Format("2006-01-02")

	debitWidth := getTextWidth(d.debit.name)
	dw := 16 - debitWidth
	if dw < 0 {
		dw = 0
	}

	creditWidth := getTextWidth(d.credit.name)
	cw := 16 - creditWidth
	if cw < 0 {
		cw = 0
	}

	note := ""
	if d.note != "" {
		note = " " + d.note
	}

	rng := ""

	if d.start != 0 {
		rng = fmt.Sprintf(" [%s, %s]", month2str(d.start), month2str(d.end))
	} else if note != "" {
		rng = "                   "
	}

	return fmt.Sprintf("%s %s%*s %s%*s %9s%s%s", date,
		d.debit.name, dw, "", d.credit.name, cw, "",
		int2str(d.amount), rng, note)
}

func getTransactionsReader(transactions []transaction, showSearchWords bool) io.Reader {
	src := new(bytes.Buffer)

	maxNo := len(transactions) - 1
	noWidth := len(strconv.Itoa(maxNo))

	for i, d := range transactions {
		src.WriteString(fmt.Sprintf("%*d ", noWidth, i))

		src.WriteString(tr2alignedString(&d))

		if showSearchWords {
			src.WriteString(fmt.Sprintf("    %s %s", d.debit.searchWords, d.credit.searchWords))
		}

		src.WriteString("\n")
	}

	return src
}

func selectTransaction(db *sql.DB) (*transaction, error) {
	transactions, err := getTransactions(db, true)
	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return nil, errors.New("取引が1件も登録されてない")
	}

	src := getTransactionsReader(transactions, true)
	dst := new(bytes.Buffer)
	args := []string{}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	arr := strings.Split(skipSpace(dst.String()), " ")

	i, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, err
	}

	d := transactions[i]

	println("取引:", &d)

	return &d, nil
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
	// 期間はあまり設定しないのでコメントアウト
	// tr.start, tr.end = scanRange()

	return &tr, nil
}

func scanDate() time.Time {
	var date time.Time

	for {
		print("日付: ")
		s, err := input()
		if err == nil {
			date, err = str2date(s)
		}

		if err != nil {
			eprintln(err)
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
	print(name + ": ")
	s, err := input()
	if err != nil {
		return 0, err
	}

	return str2month(s)
}

func scanRange() (int, int) {
	var start, end int
	var err error

	for {
		start, err = scanMonth("開始月")

		if err != nil {
			eprintln(err)
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
			eprintln(err)
		} else if end == 0 {
			eprintln("開始月だけ指定はダメ")
		} else if start > end {
			eprintln("開始月 <= 終了月")
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
		v, err := strconv.Atoi(s[1:])
		if err != nil {
			return time.Time{}, err
		}

		date := today.AddDate(0, 0, -v)
		return date, nil
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
		v, err := strconv.Atoi(ss)
		if err != nil {
			return time.Time{}, err
		}

		iArr[i] = v
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

	if len(s) == 0 {
		return 0, nil
	}

	if s[0] == '-' {
		v, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, err
		}

		return subtractMonth(time2month(now), v), nil
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
		v, err := strconv.Atoi(ss)
		if err != nil {
			return 0, err
		}

		iArr[i] = v
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

func subtractMonth(ym int, n int) int {
	year := ym / 100
	month := ym % 100

	year -= n / 12
	month -= n % 12

	if month < 1 {
		month += 12
		year--
	}

	return year*100 + month
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
