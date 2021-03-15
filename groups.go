package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"strconv"
	"strings"
)

type group struct {
	id           int
	name         string
	checkAccount account
	debit        int
	credit       int
	items        []transaction
}

func (d *group) String() string {
	var ok string
	if d.debit == d.credit {
		ok = "o"
	} else {
		ok = "x"
	}

	return fmt.Sprintf("%s %s [%s, %d, %d]", ok, d.name, d.checkAccount.name, d.debit, d.credit)
}

func cmdListGroups(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runListGroups(db)
}

func runListGroups(db *sql.DB) error {
	g, err := dbGetGroups(db)
	if err != nil {
		return err
	}

	src := getGroupsReader(g)

	print(src)

	return nil
}

func cmdAddGroup(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runAddGroup(db)
}

func runAddGroup(db *sql.DB) error {
	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	var d *group

	d, err = scanGroup(accounts)
	if err != nil {
		return err
	}
	if d == nil {
		return nil
	}

	trs, err := selectMultiTransactions(db, d.checkAccount.id)
	if err != nil {
		return err
	}

	d.items = trs
	calcGroupBalance(d)

	ok, err := confirmGroup(accounts, d)
	if err != nil {
		return err
	}

	if ok == false {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	groupId, err := dbAddGroup(tx, d)

	for _, tr := range trs {
		if err = dbAddGroupsDetail(tx, groupId, tr.id); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// グループに所属する取引の貸借バランスを計算する
func calcGroupBalance(gr *group) {
	gr.debit = 0
	gr.credit = 0

	for _, tr := range gr.items {
		if tr.debit.id == gr.checkAccount.id {
			gr.debit += tr.amount
		}

		if tr.credit.id == gr.checkAccount.id {
			gr.credit += tr.amount
		}
	}
}

// グループには１つしか所属できないようにした
const sqlGetMultiTransactions1 = `
SELECT ` + transactionRows + `
FROM transactions_view
WHERE transaction_id NOT IN (SELECT transaction_id FROM groups_detail)
`

const sqlGetMultiTransactions2 = `
ORDER BY date DESC, transaction_id DESC
`

func getMultiTransactions(db *sql.DB, checkAccountId int) ([]transaction, error) {
	sql := sqlGetMultiTransactions1

	if checkAccountId != 0 {
		sql += `AND (debit_id = $1 OR credit_id = $1) `
	} else {
		sql += `AND debit_id <> $1 `
	}

	sql += sqlGetMultiTransactions2

	rows, err := db.Query(sql, checkAccountId)
	if err != nil {
		return nil, err
	}

	return rows2transactions(rows)
}

func selectMultiTransactions(db *sql.DB, checkAccountId int) ([]transaction, error) {
	transactions, err := getMultiTransactions(db, checkAccountId)
	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return nil, errors.New("取引が1件も登録されてない")
	}

	src := getTransactionsReader(transactions, true)
	dst := new(bytes.Buffer)
	args := []string{"--multi"}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	arr, err := readInts(dst.String())

	var res []transaction
	for _, i := range arr {
		d := transactions[i]

		res = append(res, d)
	}

	return res, nil
}

func cmdEditGroup(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	gr, err := selectGroup(db)
	if gr == nil || err != nil {
		return err
	}

	return editGroup(db, gr)
}

func editGroup(db *sql.DB, gr *group) error {
	const q = "n(ame), a(dd), r(emove), q(uit): "

	println()
	println(gr)
	for i, d := range gr.items {
		println(i, &d)
	}

	var a string
	if len(gr.items) == 0 {
		a = "add"
	} else {
		print(q)
		s, err := input()
		if err != nil {
			return err
		}
		a = strings.ToLower(s)
	}

	for a != "q" {
		isUpdateItems := false

		switch a {
		case "n", "name":
			gr.name = scanGroupName()
			err := dbUpdateGroupName(db, gr)
			if err != nil {
				return err
			}
		case "a", "add":
			trs, err := selectMultiTransactions(db, gr.checkAccount.id)
			if err != nil {
				return err
			}

			if trs != nil && len(trs) > 0 {
				for _, tr := range trs {
					if err = dbAddGroupsDetail(db, gr.id, tr.id); err != nil {
						return err
					}
				}

				isUpdateItems = true
			}
		case "r", "remove":
			trs, err := selectGroupItems(db, gr)
			if err != nil {
				return err
			}
			if trs == nil {
				break
			}

			err = dbRemoveGroupItems(db, gr.id, trs)
			if err != nil {
				return err
			}

			isUpdateItems = true
		}

		if isUpdateItems {
			items, err := dbGetGroupItems(db, gr.id)
			if err != nil {
				return err
			}

			gr.items = items
			calcGroupBalance(gr)
		}

		println()
		println(gr)
		for i, d := range gr.items {
			println(i, &d)
		}

		print(q)
		s, err := input()
		if err != nil {
			return err
		}
		a = strings.ToLower(s)
	}

	return nil
}

func selectGroupItems(db *sql.DB, gr *group) ([]transaction, error) {
	if len(gr.items) == 0 {
		return nil, errors.New("取引が1件も登録されてない")
	}

	src := getTransactionsReader(gr.items, true)
	dst := new(bytes.Buffer)
	args := []string{"--multi"}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	arr, err := readInts(dst.String())

	var res []transaction
	for _, i := range arr {
		d := gr.items[i]

		res = append(res, d)
	}

	return res, nil
}

func cmdRemoveGroup(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runRemoveGroup(db)
}

func runRemoveGroup(db *sql.DB) error {
	gr, err := selectGroup(db)

	if gr == nil || err != nil {
		return err
	}

	if confirmYesNo("本当に削除する?") {
		err = dbRemoveGroup(db, gr.id)
	}

	return err
}

const sqlGetGroups = `
SELECT gr.group_id, gr.name, gr.check_account_id, ac.name
FROM groups AS gr
LEFT JOIN accounts AS ac ON gr.check_account_id = ac.account_id
ORDER BY group_id DESC
`

func dbGetGroups(db *sql.DB) ([]group, error) {
	rows, err := db.Query(sqlGetGroups)
	if err != nil {
		return nil, err
	}

	groups, err := rows2groups(rows)
	if err != nil {
		return nil, err
	}

	for i, gr := range groups {
		items, err := dbGetGroupItems(db, gr.id)
		if err != nil {
			return nil, err
		}

		groups[i].items = items
		calcGroupBalance(&groups[i])
	}

	return groups, err
}

const sqlGetGroupBalance = `
SELECT SUM(CASE WHEN tr.debit_id = $2 THEN tr.amount ELSE 0 END),
       SUM(CASE WHEN tr.credit_id = $2 THEN tr.amount ELSE 0 END)
FROM groups_detail AS gd
LEFT JOIN transactions AS tr ON gd.transaction_id = tr.transaction_id
WHERE gd.group_id = $1
GROUP BY gd.group_id
`

func dbGetGroupBalance(db *sql.DB, gr *group) error {
	return db.QueryRow(sqlGetGroupBalance, gr.id, gr.checkAccount.id).Scan(&gr.debit, &gr.credit)
}

const sqlGetGroupItems = `
SELECT ` + transactionRows + `
FROM transactions_view
WHERE transaction_id IN (SELECT transaction_id FROM groups_detail WHERE group_id = $1)
ORDER BY date, transaction_id
`

func dbGetGroupItems(db *sql.DB, id int) ([]transaction, error) {
	rows, err := db.Query(sqlGetGroupItems, id)
	if err != nil {
		return nil, err
	}

	return rows2transactions(rows)
}

func rows2groups(rows *sql.Rows) ([]group, error) {
	var grs []group

	for rows.Next() {
		var gr group

		err := rows.Scan(&gr.id, &gr.name, &gr.checkAccount.id, &gr.checkAccount.name)
		if err != nil {
			return nil, err
		}

		grs = append(grs, gr)
	}
	rows.Close()

	return grs, nil
}

func confirmGroup(accounts []account, gr *group) (bool, error) {
	for {
		println()
		println(gr)

		print("y(es), n(ame), a(ccount), q(uit): ")
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
		case "n", "name":
			gr.name = scanGroupName()
		case "a", "account":
			ac, err := selectAccount(accounts, "対応を確認する勘定科目")
			if err != nil {
				return false, err
			}
			if ac != nil {
				gr.checkAccount = *ac
				calcGroupBalance(gr)
			}
		}
	}
}

const sqlUpdateGroupName = `
UPDATE groups
SET name = $2
WHERE group_id = $1
`

func dbUpdateGroupName(db dbtx, d *group) error {
	_, err := db.Exec(sqlUpdateGroupName, d.id, d.name)

	return err
}

const sqlAddGroup = `
INSERT INTO groups(name, check_account_id)
VALUES($1, $2)
RETURNING group_id
`

func dbAddGroup(db dbtx, d *group) (int, error) {
	var idStr string
	err := db.QueryRow(sqlAddGroup, d.name, d.checkAccount.id).Scan(&idStr)
	if err != nil {
		return 0, err
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}

	return id, err
}

const sqlAddGroupsDetail = `
INSERT INTO groups_detail(group_id, transaction_id)
VALUES($1, $2)
`

func dbAddGroupsDetail(db dbtx, groupId int, trId int) error {
	_, err := db.Exec(sqlAddGroupsDetail, groupId, trId)
	return err
}

const sqlRemoveGroup = `
DELETE FROM groups
WHERE group_id = $1
`

func dbRemoveGroup(db *sql.DB, id int) error {
	_, err := db.Exec(sqlRemoveGroup, id)

	return err
}

const sqlRemoveGroupItems = `
DELETE FROM groups_detail
WHERE group_id = $1 AND transaction_id IN
`

func dbRemoveGroupItems(db *sql.DB, id int, items []transaction) error {
	var ids []string

	for _, item := range items {
		ids = append(ids, strconv.Itoa(item.id))
	}

	_, err := db.Exec(sqlRemoveGroupItems+"("+strings.Join(ids[:], ",")+")", id)

	return err
}

func getGroupsReader(grs []group) io.Reader {
	src := new(bytes.Buffer)

	maxNo := len(grs) - 1
	noWidth := len(strconv.Itoa(maxNo))

	for i, gr := range grs {
		src.WriteString(fmt.Sprintf("%*d %s\n",
			noWidth, i, &gr))
	}

	return src
}

func selectGroup(db *sql.DB) (*group, error) {
	groups, err := dbGetGroups(db)
	if err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		return nil, errors.New("グループが1件も登録されてない")
	}

	src := getGroupsReader(groups)
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

	d := groups[i]

	println("グループ:", &d)

	return &d, nil
}

func scanGroup(accounts []account) (*group, error) {
	var d group

	d.name = scanGroupName()

	checkAccount, err := selectAccount(accounts, "対応を確認する勘定科目")
	if err != nil {
		return nil, err
	}
	if checkAccount == nil {
		return nil, nil
	}
	d.checkAccount = *checkAccount

	return &d, nil
}

func scanGroupName() string {
	return scanText("グループ名", 1, 16)
}
