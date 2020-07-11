package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestRunBS(t *testing.T) {
	buf := new(bytes.Buffer)
	stdout = buf
	stderr = new(bytes.Buffer)

	db, err := setupAcAndTr()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	err = runBS(db, "2020-01")
	if err != nil {
		t.Fatal(err)
	}

	bytes, err := ioutil.ReadFile(filepath.Join("testdata", "bs.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != buf.String() {
		t.Fatal("string(bytes) != buf.String()")
	}
}

func TestRunPL(t *testing.T) {
	db, err := setupAcAndTr()
	if db != nil {
		defer db.Close()
	}
	if err != nil {
		t.Fatal(err)
	}

	subTestRunPL(db, t, false, "2019-11")
	subTestRunPL(db, t, false, "2019-12")
	subTestRunPL(db, t, true, "2019-12")
}

func subTestRunPL(db *sql.DB, t *testing.T, isCash bool, month string) {
	buf := new(bytes.Buffer)
	stdout = buf
	stderr = new(bytes.Buffer)

	err := runPL(db, isCash, month)
	if err != nil {
		t.Fatal(err)
	}

	filename := fmt.Sprintf("pl_%s_", month)
	if isCash {
		filename += "cash.txt"
	} else {
		filename += "accrual.txt"
	}

	b, err := ioutil.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != buf.String() {
		t.Fatalf("string(bytes) != buf.String()\n%s\n%s", string(b), buf.String())
	}
}
