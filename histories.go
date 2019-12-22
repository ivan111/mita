package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
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
	rng := ""

	if d.tr.start != 0 {
		rng = fmt.Sprintf("[%s, %s]", month2str(d.tr.start), month2str(d.tr.end))
	}

	operateTime := d.operateTime.Format("2006-01-02 15:04:05")
	date := d.tr.date.Format("2006-01-02")

	return fmt.Sprintf("%s %s %s %s / %s %s %s %s", operateTime,
		d.operation, date, d.tr.debit.name, d.tr.credit.name,
		int2str(d.tr.amount), d.tr.note, rng)
}

const sqlGetHistory1 = `
SELECT operation, operate_time, transaction_id, version, date,
       debit_id, debit, credit_id, credit,
	   amount, description, start_month, end_month
FROM history_view
WHERE transaction_id = $1 AND version = $2
`

func dbGetHistory1(db *sql.DB, transactionID int, version int) (*history, error) {
	var d history

	err := db.QueryRow(sqlGetHistory1, transactionID, version).Scan(
		&d.operation, &d.operateTime,
		&d.tr.id, &d.tr.version, &d.tr.date,
		&d.tr.debit.id, &d.tr.debit.name, &d.tr.credit.id, &d.tr.credit.name,
		&d.tr.amount, &d.tr.note, &d.tr.start, &d.tr.end)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

const sqlGetHistory = `
SELECT operation, operate_time, transaction_id, version, date,
       debit_id, debit, credit_id, credit,
	   amount, description, start_month, end_month
FROM history_view
WHERE transaction_id = $1
ORDER BY operate_time
`

func dbGetHistory(db *sql.DB, transactionID int) ([]history, error) {
	rows, err := db.Query(sqlGetHistory, transactionID)
	if err != nil {
		return nil, err
	}

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

const sqlGetUndoableHistory = `
SELECT operation, operate_time, transaction_id, version, date,
       debit_id, debit, credit_id, credit,
	   amount, description, start_month, end_month
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

func selectUndoableHistory(db *sql.DB) (*history, error) {
	items, err := dbGetUndoableHistory(db)

	src := new(bytes.Buffer)
	for i, d := range items {
		src.Write([]byte(fmt.Sprintf("%d %v\n", i, &d)))
	}

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

	arr := strings.Split(dst.String(), " ")

	i, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, err
	}

	d := items[i]

	fmt.Printf("UNDO取引: %v\n", &d)

	return &d, nil
}
