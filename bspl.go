package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
)

type summary struct {
	id          int
	accountType int
	name        string
	balance     int
}

func (d *summary) String() string {
	return fmt.Sprintf("%s %s", d.name, int2str(d.balance))
}

func cmdBS(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	items, err := dbGetBalances(db)
	if err != nil {
		return err
	}

	var assetSum, liabilitySum int

	fmt.Println("資産:")
	for _, d := range items {
		if d.accountType != acTypeAsset {
			continue
		}

		if d.balance != 0 {
			fmt.Println(&d)

			assetSum += d.balance
		}
	}

	fmt.Println()
	fmt.Println("負債:")
	for _, d := range items {
		if d.accountType != acTypeLiability {
			continue
		}

		if d.balance != 0 {
			fmt.Println(&d)

			liabilitySum += d.balance
		}
	}

	fmt.Println()
	fmt.Printf("総資産: %s\n", int2str(assetSum))
	fmt.Printf("総負債: %s\n", int2str(liabilitySum))
	fmt.Printf("純資産: %s\n", int2str(assetSum-liabilitySum))

	return nil
}

func cmdPL(context *cli.Context) error {
	monthStr := context.Args().First()

	if monthStr == "" {
		monthStr = "-0" // 今月
	}

	month, err := str2month(monthStr)
	if err != nil {
		return err
	}

	fmt.Println(month2str(month))
	fmt.Println()

	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	items, err := dbGetGroupedPL(db, month)
	if err != nil {
		return err
	}

	p2d, err := dbGetPL(db, month)
	if err != nil {
		return err
	}

	var incomeSum, expenseSum int

	fmt.Println("収入:")
	for _, d := range items {
		if d.accountType != acTypeIncome {
			continue
		}

		fmt.Println(&d)

		incomeSum += d.balance

		if len(p2d[d.id]) > 1 {
			printSubItems(p2d[d.id])
		}
	}

	fmt.Println()
	fmt.Println("費用:")
	for _, d := range items {
		if d.accountType != acTypeExpense {
			continue
		}

		fmt.Println(&d)

		expenseSum += d.balance

		if len(p2d[d.id]) > 1 {
			printSubItems(p2d[d.id])
		}
	}

	fmt.Println()
	fmt.Printf("総収入: %s\n", int2str(incomeSum))
	fmt.Printf("総費用: %s\n", int2str(expenseSum))
	fmt.Printf("損益: %s\n", int2str(incomeSum-expenseSum))

	return nil
}

func printSubItems(items []summary) {
	for _, d := range items {
		fmt.Printf("    %v\n", &d)
	}
}

const sqlGetBalances = `
SELECT account_id, account_type, name, balance
FROM balance_view
`

func dbGetBalances(db *sql.DB) ([]summary, error) {
	rows, err := db.Query(sqlGetBalances)
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

const sqlGetGroupedPL = `
SELECT account_id, account_type, name, balance
FROM grouped_pl_view
WHERE month = $1
ORDER BY account_type, account_id
`

func dbGetGroupedPL(db *sql.DB, month int) ([]summary, error) {
	rows, err := db.Query(sqlGetGroupedPL, month)
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

const sqlGetPL = `
SELECT account_id, account_type, name, parent, balance
FROM pl_view
WHERE month = $1
`

func dbGetPL(db *sql.DB, month int) (map[int][]summary, error) {
	rows, err := db.Query(sqlGetPL, month)
	if err != nil {
		return nil, err
	}

	p2d := map[int][]summary{}

	for rows.Next() {
		var d summary
		var p int

		if err := rows.Scan(&d.id, &d.accountType, &d.name, &p, &d.balance); err != nil {
			return nil, err
		}

		p2d[p] = append(p2d[p], d)
	}
	rows.Close()

	return p2d, nil
}
