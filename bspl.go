package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
)

// 残高
type balance struct {
	id          int
	accountType int
	name        string
	balance     int
}

func (d *balance) String() string {
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

		fmt.Println(&d)

		assetSum += d.balance
	}

	fmt.Println()
	fmt.Println("負債:")
	for _, d := range items {
		if d.accountType != acTypeLiability {
			continue
		}

		fmt.Println(&d)

		liabilitySum += d.balance
	}

	fmt.Println()
	fmt.Printf("総資産: %s\n", int2str(assetSum))
	fmt.Printf("総負債: %s\n", int2str(liabilitySum))
	fmt.Printf("純資産: %s\n", int2str(assetSum-liabilitySum))

	return nil
}

const sqlGetBalances = `
SELECT account_id, account_type, name, balance
FROM balance_view
`

func dbGetBalances(db *sql.DB) ([]balance, error) {
	rows, err := db.Query(sqlGetBalances)
	if err != nil {
		return nil, err
	}

	var balances []balance

	for rows.Next() {
		var d balance

		if err := rows.Scan(&d.id, &d.accountType, &d.name, &d.balance); err != nil {
			return nil, err
		}

		balances = append(balances, d)
	}
	rows.Close()

	return balances, nil
}
