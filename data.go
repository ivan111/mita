package main

import (
	"errors"
	_ "github.com/ivan111/mita/statik"
	"github.com/rakyll/statik/fs"
	"github.com/urfave/cli/v2"
	"io/ioutil"
)

func cmdData(context *cli.Context) error {
	files := []string{
		"schema.sql",
		"accounts.example.tsv",
	}

	filename := context.Args().First()

	if filename == "" {
		for _, f := range files {
			println(f)
		}
		return nil
	}

	f2b := make(map[string]bool)
	for _, f := range files {
		f2b[f] = true
	}

	if !f2b[filename] {
		return errors.New("file does not exist:" + filename)
	}

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	r, err := statikFS.Open("/data/" + filename)
	if err != nil {
		return err
	}
	defer r.Close()

	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	print(string(contents))

	return nil
}
