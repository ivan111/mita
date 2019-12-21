package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"strconv"
	"strings"
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
