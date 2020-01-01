package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"path/filepath"
	"testing"
)

func setup() (*sql.DB, error) {
	testMode = true

	configData.DB.Name = "mita_test"

	db, err := connectDB()
	if err != nil {
		return nil, err
	}

	err = dbClean(db)
	if err != nil {
		return db, err
	}

	return db, nil
}

func setupAccounts() (*sql.DB, error) {
	db, err := setup()
	if err != nil {
		return db, err
	}

	f, err := os.Open(filepath.Join("testdata", "accounts.tsv"))
	if err != nil {
		return db, err
	}
	defer f.Close()

	if err := readAccounts(db, f); err != nil {
		return db, err
	}

	return db, nil
}

func setupAcAndTr() (*sql.DB, error) {
	db, err := setupAccounts()
	if err != nil {
		return db, err
	}

	f, err := os.Open(filepath.Join("testdata", "transactions.tsv"))
	if err != nil {
		return db, err
	}
	defer f.Close()

	if err := readTransactions(db, f); err != nil {
		return db, err
	}

	return db, nil
}

func dbClean(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM transactions")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM transactions_history")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM templates_detail")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM templates")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM templates")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM transactions_month")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM transactions_summary")
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM accounts")

	return err
}

func equalSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

type readOrderTest struct {
	argText     string
	argNumItems int
	res         []int
}

var readOrderTests = []readOrderTest{
	{"", 0, []int{}},
	{"0 item1\n", 1, []int{0}},
	{"0 item0\n1 item1\n", 2, []int{0, 1}},
	{"1 item1\n0 item0\n", 2, []int{1, 0}},
	{"0 item0\n1 item1\n2 item2\n", 3, []int{0, 1, 2}},
	{"1 item1\n2 item2\n0 item0\n", 3, []int{2, 0, 1}},
	{"1 item1\n0 item0\n2 item2\n", 3, []int{1, 0, 2}},
	{"0 item0\n2 item2\n1 item1\n", 3, []int{0, 2, 1}},
}

var readOrderErrTests = []readOrderTest{
	{"", 1, nil},
	{"0 item1\n", 0, nil},
	{"0 item1\n", 2, nil},
	{"0 item0\n2 item2\n", 2, nil},
	{"a item0\n1 item1\n", 2, nil},
	{"0 item0\n0 item0\n", 2, nil},
}

func TestReadOrder(t *testing.T) {
	for _, test := range readOrderTests {
		res, _ := readOrder(test.argText, test.argNumItems)
		if equalSlice(res, test.res) == false {
			t.Errorf(`readOrder, got = %d, want = %d`, res, test.res)
		}
	}

	for i, test := range readOrderErrTests {
		_, err := readOrder(test.argText, test.argNumItems)
		if err == nil {
			t.Errorf(`readOrder, readOrderErrTest[%d]`, i)
		}
	}
}

type getTextWidthTest struct {
	arg string
	res int
}

var getTextWidthTests = []getTextWidthTest{
	{"", 0},
	{" ", 1},
	{"　", 2},
	{"test", 4},
	{"テスト", 6},
	{"testテスト", 10},
	{"aあア亜", 7},
}

func TestGetTextWidth(t *testing.T) {
	for _, test := range getTextWidthTests {
		res := getTextWidth(test.arg)
		if res != test.res {
			t.Errorf(`getTextWidth, got = %d, want = %d`, res, test.res)
		}
	}
}

type skipSpaceTest struct {
	arg string
	res string
}

var skipSpaceTests = []skipSpaceTest{
	{"", ""},
	{"test", "test"},
	{"テスト", "テスト"},
	{" ", ""},
	{" test", "test"},
	{"  ", ""},
	{"  テスト", "テスト"},
}

func TestSkipSpace(t *testing.T) {
	for _, test := range skipSpaceTests {
		res := skipSpace(test.arg)
		if res != test.res {
			t.Errorf(`skipSpace, got = %s, want = %s`, res, test.res)
		}
	}
}
