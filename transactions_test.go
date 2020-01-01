package main

import (
	"bufio"
	"bytes"
	_ "github.com/lib/pq"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTransactionCommands(t *testing.T) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)

	t.Run("TestRunAddTransaction", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("2019-11-03\n11\n0\n3000\nもやし、鶏卵等\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runAddTransaction(db, nil); err != nil {
			t.Fatal(err)
		}

		transactions, err := getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 1 {
			t.Fatal("len(transactions) != 1:", len(transactions))
		}

		testTransaction(t, transactions[0], "2019-11-03", "食費", "現金", 3000, "もやし、鶏卵等", 0, 0)
	})

	t.Run("TestRunAddTransactionArgs", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"2019-12-20", "年金保険料", "A銀行", "379640", "2年前納", "2019-12", "2021-11"}

		if err := runAddTransaction(db, args); err != nil {
			t.Fatal(err)
		}

		transactions, err := getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 1 {
			t.Fatal("len(transactions) != 1:", len(transactions))
		}

		testTransaction(t, transactions[0], "2019-12-20", "年金保険料", "A銀行", 379640, "2年前納", 201912, 202111)
	})

	t.Run("TestRunListTransaction", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"2019-10-10", "娯楽", "現金", "2000"}

		if err := runAddTransaction(db, args); err != nil {
			t.Fatal(err)
		}

		args = []string{"2019-11-09", "食費", "現金", "3000"}

		if err := runAddTransaction(db, args); err != nil {
			t.Fatal(err)
		}

		buf := new(bytes.Buffer)
		stdout = buf

		if err := runListTransactions(db, false, "2019-10"); err != nil {
			t.Fatal(err)
		}

		want := "0 2019-10-10 娯楽             現金                 2,000\n"

		if buf.String() != want {
			t.Fatal("buf.String() != want:", buf.String())
		}

		buf = new(bytes.Buffer)
		stdout = buf

		if err := runListTransactions(db, false, "2019-11"); err != nil {
			t.Fatal(err)
		}

		want = "0 2019-11-09 食費             現金                 3,000\n"

		if buf.String() != want {
			t.Fatal("buf.String() != want:", buf.String())
		}

		buf = new(bytes.Buffer)
		stdout = buf

		if err := runListTransactions(db, true, ""); err != nil {
			t.Fatal(err)
		}

		want = "0 2019-10-10 娯楽             現金                 2,000\n" +
			"1 2019-11-09 食費             現金                 3,000\n"

		if buf.String() != want {
			t.Fatal("buf.String() != want:", buf.String())
		}
	})

	t.Run("TestRunEditTransaction", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"2019-10-10", "娯楽", "現金", "2000"}

		if err := runAddTransaction(db, args); err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("0\nd\n2018-09-02\nl\n24\nr\n1\na\n15000\nn\n任意保険\ns\n2018-10\n2019-09\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runEditTransaction(db); err != nil {
			t.Fatal(err)
		}

		transactions, err := getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 1 {
			t.Fatal("len(transactions) != 1:", len(transactions))
		}

		testTransaction(t, transactions[0], "2018-09-02", "保険", "A銀行", 15000, "任意保険", 201810, 201909)
	})

	t.Run("TestRunRemoveTransaction", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"2019-10-10", "娯楽", "現金", "2000"}

		if err := runAddTransaction(db, args); err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runRemoveTransaction(db); err != nil {
			t.Fatal(err)
		}

		transactions, err := getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 0 {
			t.Fatal("len(transactions) != 0:", len(transactions))
		}
	})

	t.Run("TestRunUndoTransaction", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"2019-10-10", "娯楽", "現金", "2000"}

		if err := runAddTransaction(db, args); err != nil {
			t.Fatal(err)
		}

		// 金額を3,500円に変更
		stdin = bytes.NewBufferString("0\na\n3500\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runEditTransaction(db); err != nil {
			t.Fatal(err)
		}

		// 金額の変更をUNDO
		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runUndoTransaction(db); err != nil {
			t.Fatal(err)
		}

		transactions, err := getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		d := transactions[0]
		if d.amount != 2000 {
			t.Fatal("d.amount != 2000:", d.amount)
		}

		// 取引の削除
		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runRemoveTransaction(db); err != nil {
			t.Fatal(err)
		}

		// 取引の削除をUNDO
		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runUndoTransaction(db); err != nil {
			t.Fatal(err)
		}

		transactions, err = getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 1 {
			t.Fatal("len(transactions) != 1:", len(transactions))
		}

		// 前のUNDOでまた追加されたので、それをUNDO
		// つまり、また削除
		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runUndoTransaction(db); err != nil {
			t.Fatal(err)
		}

		transactions, err = getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 0 {
			t.Fatal("len(transactions) != 0:", len(transactions))
		}
	})
}

func TestReadTransactions(t *testing.T) {
	db, err := setupAccounts()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join("testdata", "transactions1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := readTransactions(db, f); err != nil {
		t.Fatal(err)
	}

	transactions, err := getTransactions(db, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(transactions) != 26 {
		t.Fatal("len(transactions) != 26:", len(transactions))
	}
}

/*
成功例は、TestWriteTransactionsでテスト済み
*/
func TestArr2transaction(t *testing.T) {
	db, err := setupAccounts()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	accounts, err := dbGetAccounts(db)
	if err != nil {
		t.Fatal(err)
	}

	name2id := make(map[string]int)

	for _, d := range accounts {
		name2id[d.name] = d.id
	}

	arr := []string{"2019-11-23", "なし", "現金"}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}

	arr = []string{"2019-11-23", "なし", "現金", "3000"}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}

	arr = []string{"2019-11-23", "食費", "なし", "3000"}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}

	arr = []string{"2019-11-23", "食費", "現金", "1円"}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}

	arr = []string{"2019-11-23", "食費", "現金", "3000", "", ""}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}

	arr = []string{"2019-11-23", "食費", "現金", "3000", "", "", "2019-12"}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}

	arr = []string{"2019-11-23", "食費", "現金", "3000", "", "2019-11", ""}

	_, err = arr2transaction(name2id, arr)
	if err == nil {
		t.Fatal("エラーになるはず")
	}
}

func TestWriteTransactions(t *testing.T) {
	db, err := setupAcAndTr()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	wf := new(bytes.Buffer)

	if err := writeTransactions(db, wf); err != nil {
		t.Fatal(err)
	}

	rf, err := os.Open(filepath.Join("testdata", "transactions.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer rf.Close()

	bytes, err := ioutil.ReadAll(rf)
	if wf.String() != string(bytes) {
		t.Fatal("wf.String() != string(bytes)")
	}
}

func isSameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func TestStr2date(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	if d, err := str2date(""); err != nil || !isSameDate(d, today) {
		t.Fatalf(`str2date(""), got = %s, want = %s`, d, today)
	}

	yesterday := time.Now().AddDate(0, 0, -1)

	if d, err := str2date("-1"); err != nil || !isSameDate(d, yesterday) {
		t.Fatalf(`str2date("-1"), got = %s, want = %s`, d, yesterday)
	}

	day28 := time.Date(now.Year(), now.Month(), 28, 0, 0, 0, 0, time.Local)

	if d, err := str2date("28"); err != nil || !isSameDate(d, day28) {
		t.Fatalf(`str2date("28"), got = %s, want = %s`, d, day28)
	}

	omisoka := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.Local)

	if d, err := str2date("12/31"); err != nil || !isSameDate(d, omisoka) {
		t.Fatalf(`str2date("12/31"), got = %s, want = %s`, d, omisoka)
	}

	if d, err := str2date("12-31"); err != nil || !isSameDate(d, omisoka) {
		t.Fatalf(`str2date("12-31"), got = %s, want = %s`, d, omisoka)
	}

	happyNewYear := time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local)

	if d, err := str2date("2018/1/1"); err != nil || !isSameDate(d, happyNewYear) {
		t.Fatalf(`str2date("2018/1/1"), got = %s, want = %s`, d, happyNewYear)
	}

	if d, err := str2date("2018-01-01"); err != nil || !isSameDate(d, happyNewYear) {
		t.Fatalf(`str2date("2018-01-01"), got = %s, want = %s`, d, happyNewYear)
	}

	if _, err := str2date("1a"); err == nil {
		t.Fatal(`str2date("1a") should be error`)
	}

	if _, err := str2date("0"); err == nil {
		t.Fatal(`str2date("0") should be error`)
	}

	if _, err := str2date("32"); err == nil {
		t.Fatal(`str2date("32") should be error`)
	}
}

func toMonth(t time.Time) int {
	return t.Year()*100 + int(t.Month())
}

func TestStr2month(t *testing.T) {
	now := time.Now()

	if m, err := str2month(""); err != nil || m != 0 {
		t.Fatalf(`str2month(""), got = %d, want = %d`, m, 0)
	}

	year := now.Year()
	month := now.Month() - 1
	if month == 0 {
		month = 12
		year--
	}
	prevMonth := year*100 + int(month)

	if m, err := str2month("-1"); err != nil || m != prevMonth {
		t.Fatalf(`str2month("-1"), got = %d, want = %d`, m, prevMonth)
	}

	month1 := toMonth(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local))

	if m, err := str2month("1"); err != nil || m != month1 {
		t.Fatalf(`str2month("1"), got = %d, want = %d`, m, month1)
	}

	m1 := toMonth(time.Date(2018, 2, 1, 0, 0, 0, 0, time.Local))

	if m, err := str2month("2018/2"); err != nil || m != m1 {
		t.Fatalf(`str2month("2018/2"), got = %d, want = %d`, m, m1)
	}

	if m, err := str2month("2018-2"); err != nil || m != m1 {
		t.Fatalf(`str2month("12-31"), got = %d, want = %d`, m, m1)
	}

	if m, err := str2month("201802"); err != nil || m != m1 {
		t.Fatalf(`str2month("201801"), got = %d, want = %d`, m, m1)
	}

	if _, err := str2month("1a"); err == nil {
		t.Fatal(`str2month("1a") should be error`)
	}

	if _, err := str2month("0"); err == nil {
		t.Fatal(`str2month("0") should be error`)
	}

	if _, err := str2month("13"); err == nil {
		t.Fatal(`str2month("13") should be error`)
	}

	if _, err := str2month("201813"); err == nil {
		t.Fatal(`str2month("201813") should be error`)
	}
}

type subtractMonthTest struct {
	argYM int
	argN  int
	res   int
}

var subtractMonthTests = []subtractMonthTest{
	{202001, 0, 202001},
	{202002, 1, 202001},
	{202001, 1, 201912},
	{202001, 25, 201712},
}

func TestSubtractMonth(t *testing.T) {
	for i, test := range subtractMonthTests {
		res := subtractMonth(test.argYM, test.argN)
		if res != test.res {
			t.Errorf("#%d: got: %#v want: %#v", i, res, test.res)
		}
	}
}

type int2strTest struct {
	arg int
	res string
}

var int2strTests = []int2strTest{
	{0, "0"},
	{1, "1"},
	{12, "12"},
	{123, "123"},
	{1234, "1,234"},
	{12345, "12,345"},
	{123456, "123,456"},
	{1234567, "1,234,567"},
	{12345678, "12,345,678"},
	{123456789, "123,456,789"},
	{-1, "-1"},
	{-12, "-12"},
	{-123, "-123"},
	{-1234, "-1,234"},
	{-12345, "-12,345"},
	{-123456, "-123,456"},
	{-1234567, "-1,234,567"},
	{-12345678, "-12,345,678"},
	{-123456789, "-123,456,789"},
}

func TestInt2Str(t *testing.T) {
	for i, test := range int2strTests {
		res := int2str(test.arg)
		if res != test.res {
			t.Errorf("#%d: got: %#v want: %#v", i, res, test.res)
		}
	}
}
