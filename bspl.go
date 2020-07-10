package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
)

type summary struct {
	id              int
	accountType     int
	name            string
	isExtraordinary bool
	balance         int
}

func (d *summary) String() string {
	src := new(bytes.Buffer)

	nameWidth := getTextWidth(d.name)
	nw := 16 - nameWidth
	if nw < 0 {
		nw = 0
	}
	src.WriteString(fmt.Sprintf("%s%*s", d.name, nw, ""))

	src.WriteString(fmt.Sprintf(" %11s", int2str(d.balance)))

	return src.String()
}

func cmdBS(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runBS(db, context.Args().First())
}

func runBS(db *sql.DB, monthStr string) error {
	if monthStr == "" {
		monthStr = "-0" // 今月
	}

	month, err := str2month(monthStr)
	if err != nil {
		return err
	}

	println(month2str(month))
	println()

	err = updateTransactionsSummary(db)
	if err != nil {
		return err
	}

	items, err := dbGetBalances(db, month)
	if err != nil {
		return err
	}

	var assetSum, liabilitySum int

	println("資産:")
	for _, d := range items {
		if d.accountType != acTypeAsset {
			continue
		}

		if d.balance != 0 {
			println(&d)

			assetSum += d.balance
		}
	}

	println()
	println("負債:")
	for _, d := range items {
		if d.accountType != acTypeLiability {
			continue
		}

		if d.balance != 0 {
			println(&d)

			liabilitySum += d.balance
		}
	}

	println()
	printf("総資産: %20s\n", int2str(assetSum))
	printf("総負債: %20s\n", int2str(liabilitySum))
	printf("純資産: %20s\n", int2str(assetSum+liabilitySum))

	return nil
}

func cmdPL(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runPL(db, context.Bool("cash"), context.Args().First())
}

func runPL(db *sql.DB, isCash bool, monthStr string) error {
	if monthStr == "" {
		monthStr = "-0" // 今月
	}

	month, err := str2month(monthStr)
	if err != nil {
		return err
	}

	println(month2str(month))
	println()

	items, err := dbGetGroupedPL(db, isCash, month)
	if err != nil {
		return err
	}

	p2d, err := dbGetPL(db, isCash, month)
	if err != nil {
		return err
	}

	var incomeSum, expenseSum int

	println("収入:")
	for _, d := range items {
		if d.accountType != acTypeIncome {
			continue
		}

		println(&d)

		incomeSum += d.balance

		if len(p2d[d.id]) > 1 || (len(p2d[d.id]) == 1 && p2d[d.id][0].id != d.id) {
			printSubItems(p2d[d.id])
		}
	}

	println()
	println("費用:")
	for _, d := range items {
		if d.accountType != acTypeExpense {
			continue
		}

		println(&d)

		expenseSum += d.balance

		if len(p2d[d.id]) > 1 || (len(p2d[d.id]) == 1 && p2d[d.id][0].id != d.id) {
			printSubItems(p2d[d.id])
		}
	}

	println()
	printf("総収入: %20s\n", int2str(incomeSum))
	printf("総費用: %20s\n", int2str(expenseSum))
	printf("損益  : %20s\n", int2str(incomeSum+expenseSum))

	return nil
}

func printSubItems(items []summary) {
	for _, d := range items {
		printf("        %v\n", &d)
	}
}

func updateTransactionsSummary(db *sql.DB) error {
	_, err := db.Exec("SELECT add_current_transactions_summary()")

	return err
}

const sqlGetBalances = `
SELECT account_id, account_type, name, balance
FROM balance_view
WHERE month = $1
`

func dbGetBalances(db *sql.DB, month int) ([]summary, error) {
	rows, err := db.Query(sqlGetBalances, month)
	if err != nil {
		return nil, err
	}

	var balances []summary

	for rows.Next() {
		var d summary

		if err := rows.Scan(&d.id, &d.accountType, &d.name, &d.balance); err != nil {
			return nil, err
		}

		balances = append(balances, d)
	}
	rows.Close()

	return balances, nil
}

const sqlGetGroupedPLAccrual = `
SELECT account_id, account_type, name, is_extraordinary, accrual_balance
FROM grouped_pl_view
WHERE month = $1
`

const sqlGetGroupedPLCash = `
SELECT account_id, account_type, name, is_extraordinary, cash_balance
FROM grouped_pl_view
WHERE month = $1
`

func dbGetGroupedPL(db *sql.DB, isCash bool, month int) ([]summary, error) {
	var sqlStr string
	if isCash {
		sqlStr = sqlGetGroupedPLCash
	} else {
		sqlStr = sqlGetGroupedPLAccrual
	}

	rows, err := db.Query(sqlStr, month)
	if err != nil {
		return nil, err
	}

	var balances []summary

	for rows.Next() {
		var d summary

		if err := rows.Scan(&d.id, &d.accountType, &d.name, &d.isExtraordinary, &d.balance); err != nil {
			return nil, err
		}

		balances = append(balances, d)
	}
	rows.Close()

	return balances, nil
}

const sqlGetPLAccrual = `
SELECT account_id, account_type, name, parent, is_extraordinary, accrual_balance
FROM pl_view
WHERE month = $1
`

const sqlGetPLCash = `
SELECT account_id, account_type, name, parent, is_extraordinary, cash_balance
FROM pl_view
WHERE month = $1
`

func dbGetPL(db *sql.DB, isCash bool, month int) (map[int][]summary, error) {
	var sqlStr string
	if isCash {
		sqlStr = sqlGetPLCash
	} else {
		sqlStr = sqlGetPLAccrual
	}

	rows, err := db.Query(sqlStr, month)
	if err != nil {
		return nil, err
	}

	p2d := map[int][]summary{}

	for rows.Next() {
		var d summary
		var p int

		if err := rows.Scan(&d.id, &d.accountType, &d.name, &p, &d.isExtraordinary, &d.balance); err != nil {
			return nil, err
		}

		p2d[p] = append(p2d[p], d)
	}
	rows.Close()

	return p2d, nil
}
