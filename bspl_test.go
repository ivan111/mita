package main

import (
	"bytes"
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

	err = runBS(db)
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

	err = runPL(db, "2019-11")
	if err != nil {
		t.Fatal(err)
	}

	b, err := ioutil.ReadFile(filepath.Join("testdata", "pl_201911.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != buf.String() {
		t.Fatal("string(bytes) != buf.String()")
	}

	buf = new(bytes.Buffer)
	stdout = buf
	stderr = new(bytes.Buffer)

	err = runPL(db, "2019-12")
	if err != nil {
		t.Fatal(err)
	}

	b, err = ioutil.ReadFile(filepath.Join("testdata", "pl_201912.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != buf.String() {
		t.Fatal("string(bytes) != buf.String()")
	}
}
