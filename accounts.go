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
	parent      struct {
		id   int
		name string
	}
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

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	d, err := scanAccount(accounts)
	if err != nil {
		return err
	}

	ok, err := confirmAccount(accounts, d, true)
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

	ok, err := confirmAccount(accounts, d, false)
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
		err := dbRemoveAccount(db, d.id)
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

func confirmAccount(accounts []account, d *account, enableType bool) (bool, error) {
	for {
		fmt.Println()
		fmt.Printf("%s %s %s (%s)\n", acType2str(d.accountType), d.name, d.searchWords, d.parent.name)

		if enableType {
			fmt.Print("y(es), t(ype), n(ame), s(earch words), p(arent), q(uit): ")
		} else {
			fmt.Print("y(es), n(ame), s(earch words), p(arent), q(uit): ")
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
		case "p", "parent":
			parent, err := selectAccount(accounts, "親")
			if err != nil {
				return false, err
			}
			if parent != nil {
				d.parent.id = parent.id
				d.parent.name = parent.name
			} else {
				d.parent.id = 0
				d.parent.name = ""
			}
		case "s", "search words":
			d.searchWords = scanSearchWords()
		}
	}
}

const sqlGetAccounts = `
SELECT ac.account_id, ac.account_type, ac.name, ac.search_words, COALESCE(p.account_id, 0), COALESCE(p.name, '')
FROM accounts ac
LEFT JOIN accounts AS p ON ac.parent = p.account_id
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

		if err := rows.Scan(&ac.id, &ac.accountType, &ac.name, &ac.searchWords, &ac.parent.id, &ac.parent.name); err != nil {
			return nil, err
		}

		accounts = append(accounts, ac)
	}
	rows.Close()

	return accounts, nil
}

const sqlAddAccount = `
INSERT INTO accounts(account_type, name, search_words, parent)
VALUES($1, $2, $3, $4)
RETURNING account_id
`

func dbAddAccount(db dbtx, d *account) (string, error) {
	var id string
	err := db.QueryRow(sqlAddAccount, d.accountType, d.name, d.searchWords, d.parent.id).Scan(&id)

	return id, err
}

const sqlEditAccount = `
UPDATE accounts SET
name = $2,
search_words = $3,
parent = $4
WHERE account_id = $1
`

func dbEditAccount(db *sql.DB, d *account) error {
	_, err := db.Exec(sqlEditAccount, d.id, d.name, d.searchWords, d.parent.id)

	return err
}

const sqlRemoveAccount = `
DELETE FROM accounts
WHERE account_id = $1
`

func dbRemoveAccount(db *sql.DB, id int) error {
	_, err := db.Exec(sqlRemoveAccount, id)

	return err
}

func selectAccount(accounts []account, header string) (*account, error) {
	src := new(bytes.Buffer)
	for i, ac := range accounts {
		src.Write([]byte(fmt.Sprintf("%d %s %s (%s)\n", i, ac.name, ac.searchWords, ac.parent.name)))
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

func scanAccount(accounts []account) (*account, error) {
	var d account

	d.accountType = scanAccountType()
	d.name = scanAccountName()
	d.searchWords = scanSearchWords()
	parent, err := selectAccount(accounts, "親")
	if err != nil {
		return nil, err
	}
	if parent != nil {
		d.parent.id = parent.id
		d.parent.name = parent.name
	}

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
