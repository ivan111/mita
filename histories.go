package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"time"
)

type history struct {
	operation   string
	operateTime time.Time
	// 以下は transaction と同じ
	id     int
	date   time.Time
	debit  account
	credit account
	amount int
	note   string
	start  int
	end    int
}

func (d *history) String() string {
	rng := ""

	if d.start != 0 {
		rng = fmt.Sprintf("[%s, %s]", month2str(d.start), month2str(d.end))
	}

	operateTime := d.operateTime.Format("2006-01-02 15:04:05")
	date := d.date.Format("2006-01-02")

	return fmt.Sprintf("%s %s %s %s / %s %s %s %s", operateTime,
		d.operation, date, d.debit.name, d.credit.name,
		int2str(d.amount), d.note, rng)
}

const sqlGetHistory = `
SELECT operation, operate_time, transaction_id, date,
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

		if err := rows.Scan(&d.operation, &d.operateTime, &d.id, &d.date,
			&d.debit.id, &d.debit.name, &d.credit.id, &d.credit.name,
			&d.amount, &d.note, &d.start, &d.end); err != nil {
			return nil, err
		}

		items = append(items, d)
	}
	rows.Close()

	return items, nil
}
