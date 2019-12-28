package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type history struct {
	operation   string
	operateTime time.Time
	tr          transaction
}

func (d *history) String() string {
	operateTime := d.operateTime.Format("2006-01-02 15:04:05")

	return fmt.Sprintf("%s %s %v", operateTime, d.operation, &d.tr)
}

func cmdListHistory(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var history []history

	if context.Bool("all") {
		history, err = dbGetAllHistory(db)
		if err != nil {
			return err
		}
	} else {
		monthStr := context.Args().First()

		if monthStr == "" {
			monthStr = "-0" // 今月
		}

		ym, err := str2month(monthStr)
		if err != nil {
			return err
		}

		year := ym / 100
		month := ym % 100

		history, err = dbGetHistoryByMonth(db, year, month)
		if err != nil {
			return err
		}
	}

	src := getHistoryReader(history)
	fmt.Print(src)

	return nil
}

func rows2histories(rows *sql.Rows) ([]history, error) {
	var items []history

	for rows.Next() {
		var d history

		if err := rows.Scan(&d.operation, &d.operateTime,
			&d.tr.id, &d.tr.version, &d.tr.date,
			&d.tr.debit.id, &d.tr.debit.name, &d.tr.credit.id, &d.tr.credit.name,
			&d.tr.amount, &d.tr.note, &d.tr.start, &d.tr.end); err != nil {
			return nil, err
		}

		items = append(items, d)
	}
	rows.Close()

	return items, nil
}

const historyRows = `
operation, operate_time, transaction_id, version, date,
debit_id, debit, credit_id, credit,
amount, description, start_month, end_month
`

const sqlGetAllHistory = `
SELECT ` + historyRows + `
FROM history_view
`

func dbGetAllHistory(db *sql.DB) ([]history, error) {
	rows, err := db.Query(sqlGetAllHistory)
	if err != nil {
		return nil, err
	}

	return rows2histories(rows)
}

const sqlGetHistoryByMonth = `
SELECT ` + historyRows + `
FROM history_view
WHERE EXTRACT(year FROM "operate_time") = $1
      AND EXTRACT(month FROM "operate_time") = $2
`

func dbGetHistoryByMonth(db *sql.DB, year int, month int) ([]history, error) {
	rows, err := db.Query(sqlGetHistoryByMonth, year, month)
	if err != nil {
		return nil, err
	}

	return rows2histories(rows)
}

const sqlGetHistory1 = `
SELECT ` + historyRows + `
FROM history_view
WHERE transaction_id = $1 AND version = $2
`

func dbGetHistory1(db *sql.DB, transactionID int, version int) (*history, error) {
	rows, err := db.Query(sqlGetHistory1, transactionID, version)
	if err != nil {
		return nil, err
	}

	items, err := rows2histories(rows)
	if err != nil {
		return nil, err
	}

	if len(items) != 1 {
		return nil, errors.New("dbGetHistory1: len(items) != 1")
	}

	return &items[0], nil
}

const sqlGetHistory = `
SELECT ` + historyRows + `
FROM history_view
WHERE transaction_id = $1
`

func dbGetHistory(db *sql.DB, transactionID int) ([]history, error) {
	rows, err := db.Query(sqlGetHistory, transactionID)
	if err != nil {
		return nil, err
	}

	return rows2histories(rows)
}

const sqlGetUndoableHistory = `
SELECT ` + historyRows + `
FROM history_view h1
WHERE version > 0 AND version =
(SELECT MAX(h2.version)
FROM transactions_history h2
WHERE h1.transaction_id = h2.transaction_id)
ORDER BY operate_time DESC
`

func dbGetUndoableHistory(db *sql.DB) ([]history, error) {
	rows, err := db.Query(sqlGetUndoableHistory)
	if err != nil {
		return nil, err
	}

	return rows2histories(rows)
}

func history2alignedString(d *history) string {
	operateTime := d.operateTime.Format("2006-01-02 15:04:05")

	return fmt.Sprintf("%s %s %s", operateTime, d.operation,
		tr2alignedString(&d.tr))
}

func getHistoryReader(history []history) io.Reader {
	src := new(bytes.Buffer)

	maxNo := len(history) - 1
	noWidth := len(strconv.Itoa(maxNo))

	for i, d := range history {
		src.WriteString(fmt.Sprintf("%*d ", noWidth, i))
		src.WriteString(history2alignedString(&d))
		src.WriteString("\n")
	}

	return src
}

func selectUndoableHistory(db *sql.DB) (*history, error) {
	items, err := dbGetUndoableHistory(db)

	src := getHistoryReader(items)
	dst := new(bytes.Buffer)
	args := []string{
		"--header=UNDO取引",
	}

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

	d := items[i]

	fmt.Printf("UNDO取引: %v\n", &d)

	return &d, nil
}
