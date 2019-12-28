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
	orderNo     int
	parent      struct {
		id   int
		name string
	}
}

func (d *account) String() string {
	return d.name
}

func cmdListAccount(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	src := getAccountsReader(accounts)

	fmt.Print(src)

	return nil
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

func cmdReorderAccount(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	parents, err := dbGetAccountsThatHasChildren(db)
	if err != nil {
		return err
	}

	i, err := selectOrderTarget(parents)
	if err != nil {
		return err
	}

	if i == 0 {
		return nil
	}

	var accounts []account
	var startNo int

	switch i {
	case 1: // 資産
		accounts, err = dbGetAccountsByType(db, acTypeAsset)
	case 2: // 負債
		accounts, err = dbGetAccountsByType(db, acTypeLiability)
	case 3: // 収入
		accounts, err = dbGetAccountsByType(db, acTypeIncome)
	case 4: // 費用
		accounts, err = dbGetAccountsByType(db, acTypeExpense)
	default:
		d := parents[i-5]
		accounts, err = dbGetAccountChildren(db, d.id)
		startNo = d.orderNo + 1
	}

	if err != nil {
		return err
	}

	if len(accounts) < 1 {
		fmt.Fprintln(os.Stderr, "並び替える対象がない")
		return nil
	}

	src := new(bytes.Buffer)
	for i, d := range accounts {
		src.WriteString(fmt.Sprintf("%d %s\n", i, d.name))
	}

	text, cancel, err := scanWithEditor(src.String())
	if err != nil {
		return err
	}

	if cancel {
		return nil
	}

	nwo, err := readOrder(text, len(accounts))
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i := 0; i < len(accounts); i++ {
		nwo[i] += startNo

		if accounts[i].orderNo != nwo[i] {
			accounts[i].orderNo = nwo[i]

			err = dbReorderAccount(tx, &accounts[i])
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func selectOrderTarget(parents []account) (int, error) {
	src := new(bytes.Buffer)

	src.WriteString("1 資産 sisan\n")
	src.WriteString("2 負債 fusai\n")
	src.WriteString("3 収入 syuunyuu\n")
	src.WriteString("4 費用 hiyou\n")

	for i, d := range parents {
		src.WriteString(fmt.Sprintf("%d %s %s\n", 5+i, d.name, d.searchWords))
	}

	dst := new(bytes.Buffer)
	args := []string{
		"--header=並べ替え対象",
	}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	arr := strings.Split(dst.String(), " ")

	i, err := strconv.Atoi(arr[0])
	if err != nil {
		return 0, err
	}

	var name string

	switch i {
	case 1:
		name = "資産"
	case 2:
		name = "負債"
	case 3:
		name = "収入"
	case 4:
		name = "費用"
	default:
		idx := i - 5
		if idx >= 0 && idx < len(parents) {
			name = parents[idx].name
		} else {
			return 0, errors.New("out of index")
		}
	}

	fmt.Printf("並べ替え対象: %s\n", name)

	return i, nil

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
SELECT ac.account_id, ac.account_type, ac.name, ac.search_words, p.account_id, p.name
FROM accounts ac
LEFT JOIN accounts AS p ON ac.parent = p.account_id
ORDER BY ac.account_type, p.order_no, ac.order_no, ac.account_id
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

const sqlGetAccountsByType = `
SELECT account_id, name, order_no
FROM accounts
WHERE account_type = $1 AND
      account_id = parent
ORDER BY order_no, account_id
`

func dbGetAccountsByType(db *sql.DB, t int) ([]account, error) {
	rows, err := db.Query(sqlGetAccountsByType, t)
	if err != nil {
		return nil, err
	}

	var accounts []account

	for rows.Next() {
		var ac account

		if err := rows.Scan(&ac.id, &ac.name, &ac.orderNo); err != nil {
			return nil, err
		}

		accounts = append(accounts, ac)
	}
	rows.Close()

	return accounts, nil
}

const sqlGetAccountsThatHasChildren = `
SELECT p.account_id, p.account_type, p.name, p.search_words, p.order_no
FROM accounts ac
LEFT JOIN accounts AS p ON ac.parent = p.account_id
GROUP BY p.account_id, p.account_type, p.name, p.search_words, p.order_no
HAVING COUNT(*) >= 2
ORDER BY p.account_type, p.order_no, p.account_id
`

// 2人以上の子を持つ勘定科目を取得
func dbGetAccountsThatHasChildren(db *sql.DB) ([]account, error) {
	rows, err := db.Query(sqlGetAccountsThatHasChildren)
	if err != nil {
		return nil, err
	}

	var accounts []account

	for rows.Next() {
		var ac account

		if err := rows.Scan(&ac.id, &ac.accountType, &ac.name, &ac.searchWords, &ac.orderNo); err != nil {
			return nil, err
		}

		accounts = append(accounts, ac)
	}
	rows.Close()

	return accounts, nil
}

const sqlGetAccountChildren = `
SELECT account_id, account_type, name, order_no
FROM accounts ac
WHERE parent = $1 AND account_id <> parent
ORDER BY order_no, account_id
`

func dbGetAccountChildren(db *sql.DB, parentID int) ([]account, error) {
	rows, err := db.Query(sqlGetAccountChildren, parentID)
	if err != nil {
		return nil, err
	}

	var accounts []account

	for rows.Next() {
		var ac account

		if err := rows.Scan(&ac.id, &ac.accountType, &ac.name, &ac.orderNo); err != nil {
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

const sqlReorderAccount = `
UPDATE accounts
SET order_no = $2
WHERE account_id = $1
`

func dbReorderAccount(db dbtx, d *account) error {
	_, err := db.Exec(sqlReorderAccount, d.id, d.orderNo)

	return err
}

func getAccountsReader(accounts []account) io.Reader {
	src := new(bytes.Buffer)

	maxNo := len(accounts) - 1
	noWidth := len(strconv.Itoa(maxNo))

	for i, ac := range accounts {
		src.WriteString(fmt.Sprintf("%*d", noWidth, i))

		nameWidth := getTextWidth(ac.name)
		nw := 16 - nameWidth
		if nw < 0 {
			nw = 0
		}
		src.WriteString(fmt.Sprintf(" %s%*s", ac.name, nw, ""))

		parentWidth := getTextWidth(ac.parent.name)
		pw := 18 - parentWidth
		if pw < 0 {
			pw = 0
		}

		src.WriteString(fmt.Sprintf(" (%s)%*s", ac.parent.name, pw, ""))

		src.WriteString(fmt.Sprintf(" %s\n", ac.searchWords))
	}

	return src
}

func selectAccount(accounts []account, header string) (*account, error) {
	src := getAccountsReader(accounts)
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

	arr := strings.Split(skipSpace(dst.String()), " ")

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
