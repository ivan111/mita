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

func TestAccountCommands(t *testing.T) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)

	t.Run("TestRunAddAccount", func(t *testing.T) {
		db, err := setup()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("5\n開始残高\nkaisi zandaka\n\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runAddAccount(db, nil); err != nil {
			t.Fatal(err)
		}

		accounts, err := dbGetAccounts(db)
		if err != nil {
			t.Fatal(err)
		}

		if len(accounts) != 1 {
			t.Fatal("len(accounts) != 1:", len(accounts))
		}

		testAccount(t, accounts[0], acTypeEquity, "開始残高", "kaisi zandaka", false)
	})

	t.Run("TestRunAddAccountArgs", func(t *testing.T) {
		db, err := setup()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"資産", "現金", "genkin"}
		if err := runAddAccount(db, args); err != nil {
			t.Fatal(err)
		}

		accounts, err := dbGetAccounts(db)
		if err != nil {
			t.Fatal(err)
		}

		if len(accounts) != 1 {
			t.Fatal("len(accounts) != 1:", len(accounts))
		}

		testAccount(t, accounts[0], acTypeAsset, "現金", "genkin", false)
	})

	t.Run("TestRunListAccounts", func(t *testing.T) {
		db, err := setup()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"費用", "食費", "syokuhi"}
		if err := runAddAccount(db, args); err != nil {
			t.Fatal(err)
		}

		args = []string{"費用", "酒", "sake", "食費"}
		if err := runAddAccount(db, args); err != nil {
			t.Fatal(err)
		}

		buf := new(bytes.Buffer)
		stdout = buf

		if err := runListAccounts(db); err != nil {
			t.Fatal(err)
		}

		want := "0 費用 食費             (食費)               syokuhi\n" +
			"1 費用 酒               (食費)               sake\n"

		if buf.String() != want {
			t.Fatal("buf.String() != want:", buf.String())
		}
	})

	t.Run("TestRunEditAccount", func(t *testing.T) {
		db, err := setup()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"費用", "食費", "syokuhi"}
		if err := runAddAccount(db, args); err != nil {
			t.Fatal(err)
		}

		args = []string{"費用", "酒", "sake", "食費"}
		if err := runAddAccount(db, args); err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("1\nn\n自動車\ns\njidousya\np\n\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runEditAccount(db); err != nil {
			t.Fatal(err)
		}

		accounts, err := dbGetAccounts(db)
		if err != nil {
			t.Fatal(err)
		}

		if len(accounts) != 2 {
			t.Fatal("len(accounts) != 2:", len(accounts))
		}

		testAccount(t, accounts[1], acTypeExpense, "自動車", "jidousya", false)
	})

	t.Run("TestRunRemoveAccount", func(t *testing.T) {
		db, err := setup()
		if db != nil {
			defer db.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		args := []string{"費用", "食費", "syokuhi"}
		if err := runAddAccount(db, args); err != nil {
			t.Fatal(err)
		}

		stdin = bytes.NewBufferString("0\ny\n")
		scanner = bufio.NewScanner(stdin)

		if err := runRemoveAccount(db); err != nil {
			t.Fatal(err)
		}

		accounts, err := dbGetAccounts(db)
		if err != nil {
			t.Fatal(err)
		}

		if len(accounts) != 0 {
			t.Fatal("len(accounts) != 0:", len(accounts))
		}
	})
}

func TestReadAccounts(t *testing.T) {
	db, err := setup()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join("testdata", "accounts1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := readAccounts(db, f); err != nil {
		t.Fatal(err)
	}

	accounts, err := dbGetAccounts(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(accounts) != 30 {
		t.Fatal("len(accounts) != 30:", len(accounts))
	}
}

func TestWriteAccounts(t *testing.T) {
	db, err := setupAccounts()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	wf := new(bytes.Buffer)

	if err := writeAccounts(db, wf); err != nil {
		t.Fatal(err)
	}

	rf, err := os.Open(filepath.Join("testdata", "accounts.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer rf.Close()

	bytes, err := ioutil.ReadAll(rf)
	if wf.String() != string(bytes) {
		t.Fatal("wf.String() != string(bytes)")
	}
}
