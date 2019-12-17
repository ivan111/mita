package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const version = "0.1.0"

type account struct {
	id          int
	name        string
	searchWords string
}

type transaction struct {
	id     int
	date   time.Time
	debit  account
	credit account
	amount int
	note   string
	start  int
	end    int
}

func (tr *transaction) String() string {
	rng := ""

	if tr.start != 0 {
		rng = fmt.Sprintf("[%d-%02d, %d-%02d]", tr.start/100, tr.start%100, tr.end/100, tr.end%100)
	}

	date := tr.date.Format("2006-01-02")

	return fmt.Sprintf("%s %s/%s %s %s %s", date, tr.debit.name, tr.credit.name,
		int2str(tr.amount), tr.note, rng)
}

var stdin = bufio.NewScanner(os.Stdin)

func main() {
	app := cli.NewApp()

	app.Name = "mita-cli"
	app.Usage = "cli(command line interface) of Mita's household accounts"
	app.Version = version

	app.Commands = []cli.Command{
		{
			Name:    "add",
			Aliases: []string{"insert"},
			Usage:   "Insert a new transaction",
			Action:  add,
		},
		{
			Name:    "edit",
			Aliases: []string{"update"},
			Usage:   "edit a transaction",
			Action:  edit,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("Error:", err)
	}
}

func connectDB() (*sql.DB, error) {
	return sql.Open("postgres", "dbname=mita sslmode=disable")
}

func add(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := getAccounts(db)
	if err != nil {
		return err
	}

	var tr transaction

	tr.date = scanDate()

	debit, err := selectAccount(accounts, "Debit")
	if err != nil {
		return err
	}
	tr.debit = *debit

	credit, err := selectAccount(accounts, "Credit")
	if err != nil {
		return err
	}
	tr.credit = *credit

	tr.amount = scanAmount()
	tr.note = scanNote()
	tr.start, tr.end = scanRange()

	ok, err := confirmTransaction(accounts, &tr)
	if err != nil {
		return err
	}

	if ok {
		if _, err = insertTransaction(db, &tr); err != nil {
			return err
		}
	}

	return nil
}

func edit(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := getAccounts(db)
	if err != nil {
		return err
	}

	tr, err := selectTransaction(db)
	if err != nil {
		return err
	}

	ok, err := confirmTransaction(accounts, tr)
	if err != nil {
		return err
	}

	if ok {
		if err := updateTransaction(db, tr); err != nil {
			return err
		}
	}

	return nil
}

func confirmTransaction(accounts []account, tr *transaction) (bool, error) {
	const q = "y(es), d(ate), l(eft), r(ight), a(mount), n(ote), s(tart-end), q(uit)"

	fmt.Println()
	fmt.Println(tr)

	fmt.Println(q)
	stdin.Scan()
	a := strings.ToLower(stdin.Text())

	for a != "q" {
		switch a {
		case "y", "yes":
			return true, nil
		case "d", "date":
			tr.date = scanDate()
		case "l", "left":
			debit, err := selectAccount(accounts, "Debit")
			if err != nil {
				return false, err
			}
			tr.debit = *debit
		case "r", "right":
			credit, err := selectAccount(accounts, "Credit")
			if err != nil {
				return false, err
			}
			tr.credit = *credit
		case "a", "amount":
			tr.amount = scanAmount()
		case "n", "note":
			tr.note = scanNote()
		case "s":
			tr.start, tr.end = scanRange()
		}

		fmt.Println()
		fmt.Println(tr)

		fmt.Println(q)
		stdin.Scan()
		a = strings.ToLower(stdin.Text())
	}

	return false, nil
}

const insertTransactionSQL = `
INSERT INTO transactions(date, debit_id, credit_id, amount, description, start_month, end_month)
VALUES($1, $2, $3, $4, $5, $6, $7)
RETURNING transaction_id
`

func insertTransaction(db *sql.DB, tr *transaction) (string, error) {
	var id string
	err := db.QueryRow(insertTransactionSQL, tr.date, tr.debit.id, tr.credit.id, tr.amount, tr.note, tr.start, tr.end).Scan(&id)

	return id, err
}

const updateTransactionSQL = `
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

func updateTransaction(db *sql.DB, tr *transaction) error {
	_, err := db.Exec(updateTransactionSQL, tr.id, tr.date, tr.debit.id, tr.credit.id, tr.amount, tr.note, tr.start, tr.end)

	return err
}

const selectAccountsSQL = `
SELECT account_id, name, search_words
FROM accounts
ORDER BY account_id
`

func getAccounts(db *sql.DB) ([]account, error) {
	rows, err := db.Query(selectAccountsSQL)
	if err != nil {
		return nil, err
	}

	accounts := make([]account, 0)

	for rows.Next() {
		var ac account

		if err := rows.Scan(&ac.id, &ac.name, &ac.searchWords); err != nil {
			return nil, err
		}

		accounts = append(accounts, ac)
	}
	rows.Close()

	return accounts, nil
}

func selectAccount(accounts []account, header string) (*account, error) {
	src := new(bytes.Buffer)
	for _, ac := range accounts {
		src.Write([]byte(fmt.Sprintf("%d %s %s\n", ac.id, ac.name, ac.searchWords)))
	}

	dst := new(bytes.Buffer)
	args := []string{
		"--header=" + header,
	}

	if err := fzf(src, dst, os.Stderr, args); err != nil {
		return nil, err
	}

	arr := strings.Split(dst.String(), " ")

	v, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, err
	}

	return &account{v, arr[1], arr[2]}, nil
}

const selectTransactionsSQL = `
SELECT transaction_id, date, debit_id, debit, debit_search_words, credit_id,
       credit, credit_search_words, amount, description, start_month, end_month
FROM transactions_view
ORDER BY date DESC, transaction_id DESC
`

func getTransactionsReader(db *sql.DB) (io.Reader, error) {
	rows, err := db.Query(selectTransactionsSQL)
	if err != nil {
		return nil, err
	}

	reader := new(bytes.Buffer)

	for rows.Next() {
		var tr transaction

		err = rows.Scan(&tr.id, &tr.date,
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

func selectTransaction(db *sql.DB) (*transaction, error) {
	src, err := getTransactionsReader(db)
	if err != nil {
		return nil, err
	}

	dst := new(bytes.Buffer)
	args := []string{}

	if err = fzf(src, dst, os.Stderr, args); err != nil {
		return nil, err
	}

	arr := strings.Split(dst.String(), " ")

	if v, err := strconv.Atoi(arr[0]); err != nil {
		return nil, err
	} else {
		return getTransaction(db, v)
	}
}

const selectTransactionSQL = `
SELECT transaction_id, date, debit_id, debit, debit_search_words, credit_id,
       credit, credit_search_words, amount, description, start_month, end_month
FROM transactions_view
WHERE transaction_id = $1
`

func getTransaction(db *sql.DB, id int) (*transaction, error) {
	var tr transaction
	err := db.QueryRow(selectTransactionSQL, id).Scan(&tr.id, &tr.date,
		&tr.debit.id, &tr.debit.name, &tr.debit.searchWords,
		&tr.credit.id, &tr.credit.name, &tr.credit.searchWords,
		&tr.amount, &tr.note, &tr.start, &tr.end)
	if err != nil {
		return nil, err
	}

	return &tr, nil

}

func fzf(src io.Reader, dst io.Writer, errDst io.Writer, args []string) error {
	cmd := exec.Command("fzf", args...)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// stdin
	go func() {
		_, err := io.Copy(stdin, src)

		if e, ok := err.(*os.PathError); ok && e.Err == syscall.EPIPE {
			// ignore EPIPE
		} else if err != nil {
			fmt.Println("Error: failed to write to STDIN of fzf", err)
		}

		stdin.Close()
		wg.Done()
	}()

	// stdout
	go func() {
		io.Copy(dst, stdout)
		stdout.Close()
		wg.Done()
	}()

	// stderr
	go func() {
		io.Copy(errDst, stderr)
		stderr.Close()
		wg.Done()
	}()

	wg.Wait()
	return cmd.Wait()
}

func scanDate() time.Time {
	fmt.Print("Date: ")
	stdin.Scan()
	text := stdin.Text()
	date, err := str2date(text)

	for err != nil {
		fmt.Println(err)

		fmt.Print("Date: ")
		stdin.Scan()
		text = stdin.Text()
		date, err = str2date(text)
	}

	return date
}

func scanAmount() int {
	fmt.Print("Amount: ")
	stdin.Scan()
	text := stdin.Text()
	amount, err := strconv.Atoi(text)

	for err != nil {
		fmt.Println("Error:  enter a numerical value")

		fmt.Print("Amount: ")
		stdin.Scan()
		text = stdin.Text()
		amount, err = strconv.Atoi(text)
	}

	return amount
}

func scanNote() string {
	fmt.Print("Note: ")
	stdin.Scan()
	return stdin.Text()

}

func scanMonth(name string) (int, error) {
	fmt.Print(name + ": ")
	stdin.Scan()
	text := stdin.Text()

	return str2month(text)
}

func scanRange() (int, int) {
	start, err := scanMonth("Start month")

	for err != nil {
		fmt.Println(err)

		start, err = scanMonth("Start month")
	}

	if start == 0 {
		return 0, 0
	}

	var end int
	end, err = scanMonth("End month")

	for err != nil || start > end {
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Error: should: start month <= end month")
		}

		end, err = scanMonth("End month")
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
			return time.Time{}, errors.New("invalid date")
		}

		return date, nil
	}

	arr := strings.Split(s, "-")

	if len(arr) == 1 {
		arr = strings.Split(s, "/")
	}

	if len(arr) != 2 && len(arr) != 3 {
		return time.Time{}, errors.New("invalid date")
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
		return time.Time{}, errors.New("invalid date")
	}

	return date, nil
}

func time2month(d time.Time) int {
	return d.Year()*100 + int(d.Month())
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
			return 0, errors.New("invalid month")
		}

		return time2month(date), nil
	}

	arr := strings.Split(s, "-")

	if len(arr) == 1 {
		arr = strings.Split(s, "/")
	}

	if len(arr) != 2 {
		return 0, errors.New("invalid month")
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
		return 0, errors.New("invalid month")
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
