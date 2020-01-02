package main

import (
	"bufio"
	"bytes"
	_ "github.com/lib/pq"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateCommands(t *testing.T) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)

	t.Run("TestRunAddTemplate", func(t *testing.T) {
		db, err := setupAccounts()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("給与\n3\n6\ny\na\n25\n3\na\n15000\ny\nq\n")
		scanner = bufio.NewScanner(stdin)

		if err := runAddTemplate(db); err != nil {
			t.Fatal(err)
		}

		templates, err := dbGetTemplates(db)
		if err != nil {
			t.Fatal(err)
		}

		if len(templates) != 1 {
			t.Fatal("len(templates) != 1:", len(templates))
		}

		d := templates[0]

		if d.name != "給与" {
			t.Fatal(`d.name != "給与":`, d.name)
		}

		items, err := dbGetTemplateItems(db, d.id)
		if err != nil {
			t.Fatal(err)
		}

		if len(items) != 2 {
			t.Fatal("len(items) != 2:", len(items))
		}

		item := items[0]

		if item.debit.name != "未収入金" {
			t.Fatal(`item.debit.name != "未収入金":`, item.debit.name)
		}

		if item.credit.name != "給与" {
			t.Fatal(`item.credit.name != "給与":`, item.credit.name)
		}

		if item.amount != 0 {
			t.Fatal("item.amount != 0:", item.amount)
		}

		// TestRunUseTemplate

		stdin = bytes.NewBufferString("0\n2019-12-10\n100000\ny\n")
		scanner = bufio.NewScanner(stdin)

		err = runUseTemplate(db)
		if err != nil {
			t.Fatal(err)
		}

		transactions, err := getTransactions(db, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(transactions) != 2 {
			t.Fatal("len(transactions) != 2:", len(transactions))
		}

		testTransaction(t, transactions[0], "2019-12-10", "未収入金", "給与", 100000, "", 0, 0)
		testTransaction(t, transactions[1], "2019-12-10", "年金保険料", "未収入金", 15000, "", 0, 0)

		// TestRunRemoveTemplate

		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		err = runRemoveTemplate(db)
		if err != nil {
			t.Fatal(err)
		}

		templates, err = dbGetTemplates(db)
		if err != nil {
			t.Fatal(err)
		}

		if len(templates) != 0 {
			t.Fatal("len(templates) != 0:", len(templates))
		}
	})
}

func TestReadTemplates(t *testing.T) {
	db, err := setupAccounts()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join("testdata", "templates.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := readTemplates(db, f); err != nil {
		t.Fatal(err)
	}

	templates, err := dbGetTemplates(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(templates) != 3 {
		t.Fatal("len(templates) != 3:", len(templates))
	}

	d := templates[0]

	items, err := dbGetTemplateItems(db, d.id)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 5 {
		t.Fatal("len(items) != 5:", len(items))
	}

	d = templates[1]

	items, err = dbGetTemplateItems(db, d.id)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Fatal("len(items) != 2:", len(items))
	}
}

func TestWriteTemplates(t *testing.T) {
	db, err := setupAccounts()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join("testdata", "templates.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := readTemplates(db, f); err != nil {
		t.Fatal(err)
	}

	wf := new(bytes.Buffer)

	if err := writeTemplates(db, wf); err != nil {
		t.Fatal(err)
	}

	rf, err := os.Open(filepath.Join("testdata", "templates.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer rf.Close()

	bytes, err := ioutil.ReadAll(rf)
	if wf.String() != string(bytes) {
		t.Fatal("wf.String() != string(bytes)")
	}
}
