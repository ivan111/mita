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

func testAccount(t *testing.T, d account, acType int, name string, searchWords string, hasParent bool) {
	t.Helper()

	if d.accountType != acType {
		t.Fatalf("wrong account.accountType, got= %d, want = %d", d.accountType, acType)
	}

	if d.name != name {
		t.Fatalf("wrong accout.name, got= %s, want = %s", d.name, name)
	}

	if d.searchWords != searchWords {
		t.Fatalf("wrong account.searchWords, got= %s, want = %s", d.searchWords, searchWords)
	}

	if hasParent {
		if d.parent.id == d.id {
			t.Fatal("d.parent.id == d.id")
		}
	} else {
		if d.parent.id != d.id {
			t.Fatal("d.parent.id != d.id")
		}
	}
}

func testTransaction(t *testing.T, d transaction, date string, debit string, credit string, amount int, note string, start int, end int) {
	t.Helper()

	if d.date.Format("2006-01-02") != date {
		t.Fatalf("wrong transaction.date, got= %s, want = %s", d.date.Format("2006-01-02"), date)
	}

	if d.debit.name != debit {
		t.Fatalf("wrong transaction.debit.name, got= %s, want = %s", d.debit.name, debit)
	}

	if d.credit.name != credit {
		t.Fatalf("wrong transaction.credit.name, got= %s, want = %s", d.credit.name, credit)
	}

	if d.amount != amount {
		t.Fatalf("wrong transaction.amount, got= %d, want = %d", d.amount, amount)
	}

	if d.note != note {
		t.Fatalf("wrong transaction.note, got= %s, want = %s", d.note, note)
	}

	if d.start != start {
		t.Fatalf("wrong transaction.start, got= %d, want = %d", d.start, start)
	}

	if d.end != end {
		t.Fatalf("wrong transaction.end, got= %d, want = %d", d.end, end)
	}
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
