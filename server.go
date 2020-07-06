package main

//go:generate statik -f

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/ivan111/mita/statik"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
	"github.com/urfave/cli/v2"
	"net/http"
)

func cmdServer(context *cli.Context) error {
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	fs := http.FileServer(statikFS)
	http.Handle("/", fs)
	http.HandleFunc("/api/assets", apiAssetsHandler)
	http.HandleFunc("/api/bp", apiBPHandler)
	http.HandleFunc("/api/pl", apiPLHandler)

	port := context.Int("port")
	printf("Running on http://localhost:%d/ (Press CTRL+C to quit)\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

type apiAssets struct {
	Month   int `json:"month"`
	Balance int `json:"balance"`
}

func apiAssetsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		eprintln(err)
		return
	}
	defer db.Close()

	err = updateTransactionsSummary(db)
	if err != nil {
		eprintln(err)
		return
	}

	data, err := dbGetAssets(db)
	if err != nil {
		eprintln(err)
		return
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		eprintln(err)
	}
}

type apiBP struct {
	Month   int `json:"month"`
	Balance int `json:"balance"`
}

func apiBPHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		eprintln(err)
		return
	}
	defer db.Close()

	data, err := dbGetBP(db)
	if err != nil {
		eprintln(err)
		return
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		eprintln(err)
	}
}

type apiPL struct {
	ExpenseKeys []string         `json:"expense_keys"`
	IncomeKeys  []string         `json:"income_keys"`
	Expense     []map[string]int `json:"expense"`
	Income      []map[string]int `json:"income"`
}

// 今月より前の12カ月分のデータを返す
func apiPLHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		eprintln(err)
		return
	}
	defer db.Close()

	incomeKeys, err := getAccountTypeKeys(db, acTypeIncome)
	if err != nil {
		eprintln(err)
		return
	}

	expenseKeys, err := getAccountTypeKeys(db, acTypeExpense)
	if err != nil {
		eprintln(err)
		return
	}

	income, err := getPLAmountMap(db, incomeKeys)
	if err != nil {
		eprintln(err)
		return
	}

	expense, err := getPLAmountMap(db, expenseKeys)
	if err != nil {
		eprintln(err)
		return
	}

	data := apiPL{
		IncomeKeys:  incomeKeys,
		Income:      income,
		ExpenseKeys: expenseKeys,
		Expense:     expense,
	}

	w.Header().Set("Content-type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		eprintln(err)
	}
}

const sqlGetAssets = `
SELECT month, SUM(balance)
FROM balance_view group by month
`

func dbGetAssets(db *sql.DB) ([]apiAssets, error) {
	var rows *sql.Rows
	var err error

	rows, err = db.Query(sqlGetAssets)

	if err != nil {
		return nil, err
	}

	isZeroStart := true

	var arr []apiAssets

	for rows.Next() {
		var d apiAssets

		if err := rows.Scan(&d.Month, &d.Balance); err != nil {
			return nil, err
		}

		if isZeroStart == false || d.Balance != 0 {
			isZeroStart = false
			arr = append(arr, d)
		}
	}
	rows.Close()

	return arr, nil
}

func getAccountTypeKeys(db *sql.DB, acType int) ([]string, error) {
	accounts, err := dbGetAccountsByType(db, acType)
	if err != nil {
		return nil, err
	}

	var keys []string

	for _, d := range accounts {
		keys = append(keys, d.name)
	}

	return keys, nil
}

func getPLAmountMap(db *sql.DB, keys []string) ([]map[string]int, error) {
	var arr []map[string]int

	month, _ := str2month("-11")

	for i := 0; i < 12; i++ {
		item2amount := make(map[string]int)

		items, err := dbGetGroupedPL(db, false, month)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			item2amount[item.name] = item.balance
		}

		m := make(map[string]int)

		m["month"] = month

		for _, name := range keys {
			m[name] = item2amount[name]
		}

		arr = append(arr, m)

		month = incrementMonth(month)
	}

	// reverse
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}

	return arr, nil
}

const sqlGetBP = `
SELECT month, balance
FROM bp_view
`

func dbGetBP(db *sql.DB) ([]apiBP, error) {
	rows, err := db.Query(sqlGetBP)
	if err != nil {
		return nil, err
	}

	var arr []apiBP

	thisMonth, _ := str2month("-0")
	nextMonth := 0

	for rows.Next() {
		var d apiBP

		if err := rows.Scan(&d.Month, &d.Balance); err != nil {
			return nil, err
		}

		if nextMonth == 0 {
			nextMonth = d.Month
		}

		if nextMonth == d.Month {
			nextMonth = incrementMonth(nextMonth)
		} else {
			d = apiBP{
				Month:   nextMonth,
				Balance: 0,
			}
		}

		if d.Month > thisMonth {
			break
		}

		arr = append(arr, d)
	}
	rows.Close()

	return arr, nil
}

func incrementMonth(ym int) int {
	year := ym / 100
	month := ym % 100

	month++

	if month > 12 {
		year++
		month = 1
	}

	return year*100 + month
}
