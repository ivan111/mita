package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"os"
	"strconv"
	"strings"
)

const (
	acTypeAsset     = iota + 1 //資産
	acTypeLiability            // 負債
	acTypeIncome               // 収入
	acTypeExpense              // 費用
	acTypeOther                // その他
)

type account struct {
	id          int
	accountType int
	name        string
	searchWords string
}

func (d *account) String() string {
	return d.name
}

func cmdAddAccount(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	d, err := scanAccount()
	if err != nil {
		return err
	}

	ok, err := confirmAccount(d, true)
	if err != nil {
		return err
	}

	if ok {
		if _, err = dbAddAccount(db, d); err != nil {
			return err
		}
	}

	return nil
}

func cmdEditAccount(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	d, err := selectAccount(accounts, "勘定科目")
	if d == nil || err != nil {
		return err
	}

	ok, err := confirmAccount(d, false)
	if err != nil {
		return err
	}

	if ok {
		if err := dbEditAccount(db, d); err != nil {
			return err
		}
	}

	return nil
}

func cmdRemoveAccount(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	d, err := selectAccount(accounts, "勘定科目")
	if d == nil || err != nil {
		return err
	}

	ok := confirmRemoveAccount(d)
	if ok {
		// 使用されてる勘定科目の場合はエラーになる
		err := dbRemoveAccount(db, d)
		if err != nil {
			return err
		}

		fmt.Println("削除完了")
	} else {
		fmt.Println("キャンセルした")
	}

	return nil
}

func confirmRemoveAccount(d *account) bool {
	fmt.Println(d)
	fmt.Print("本当に削除する? (Y/[no]): ")
	stdin.Scan()
	return stdin.Text() == "Y"
}

func confirmAccount(d *account, enableType bool) (bool, error) {
	for {
		fmt.Println()
		fmt.Printf("%s %s %s\n", acType2str(d.accountType), d.name, d.searchWords)

		if enableType {
			fmt.Print("y(es), t(ype), n(ame), s(earch words), q(uit): ")
		} else {
			fmt.Print("y(es), n(ame), s(earch words), q(uit): ")
		}
		stdin.Scan()
		a := strings.ToLower(stdin.Text())

		switch a {
		case "q", "quit":
			return false, nil
		case "y", "yes":
			return true, nil
		case "t", "type":
			if enableType {
				d.accountType = scanAccountType()
			}
		case "n", "name":
			d.name = scanAccountName()
		case "s", "search words":
			d.searchWords = scanSearchWords()
		}
	}
}

const sqlGetAccounts = `
SELECT account_id, account_type, name, search_words
FROM accounts
ORDER BY account_type, account_id
`

func dbGetAccounts(db *sql.DB) ([]account, error) {
	rows, err := db.Query(sqlGetAccounts)
	if err != nil {
		return nil, err
	}

	var accounts []account

	for rows.Next() {
		var ac account

		if err := rows.Scan(&ac.id, &ac.accountType, &ac.name, &ac.searchWords); err != nil {
			return nil, err
		}

		accounts = append(accounts, ac)
	}
	rows.Close()

	return accounts, nil
}

const sqlAddAccount = `
INSERT INTO accounts(account_type, name, search_words)
VALUES($1, $2, $3)
RETURNING account_id
`

func dbAddAccount(db dbtx, d *account) (string, error) {
	var id string
	err := db.QueryRow(sqlAddAccount, d.accountType, d.name, d.searchWords).Scan(&id)

	return id, err
}

const sqlEditAccount = `
UPDATE accounts SET
name = $2,
search_words = $3
WHERE account_id = $1
`

func dbEditAccount(db *sql.DB, d *account) error {
	_, err := db.Exec(sqlEditAccount, d.id, d.name, d.searchWords)

	return err
}

const sqlRemoveAccount = `
DELETE FROM accounts
WHERE account_id = $1
`

func dbRemoveAccount(db *sql.DB, d *account) error {
	_, err := db.Exec(sqlRemoveAccount, d.id)

	return err
}

func selectAccount(accounts []account, header string) (*account, error) {
	src := new(bytes.Buffer)
	for i, ac := range accounts {
		src.Write([]byte(fmt.Sprintf("%d %s %s\n", i, ac.name, ac.searchWords)))
	}

	dst := new(bytes.Buffer)
	args := []string{
		"--header=" + header,
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

	d := accounts[i]

	fmt.Printf("%s: %s\n", header, d.name)

	return &d, nil
}

func scanAccount() (*account, error) {
	var d account

	d.accountType = scanAccountType()
	d.name = scanAccountName()
	d.searchWords = scanSearchWords()

	return &d, nil
}

func scanAccountType() int {
	return scanInt("タイプ (1: 資産, 2: 負債, 3: 収入, 4: 費用, 5: その他)", 1, 5)
}

func scanAccountName() string {
	return scanText("勘定科目名", 1, 8)
}

func scanSearchWords() string {
	return scanText("検索ワード", 0, 32)
}

func acType2str(t int) string {
	var s string

	switch t {
	case 1:
		s = "資産"
	case 2:
		s = "負債"
	case 3:
		s = "収入"
	case 4:
		s = "費用"
	case 5:
		s = "その他"
	default:
		s = "不明"
	}

	return s
}
