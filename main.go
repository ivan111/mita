package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
)

const version = "0.2.0"

const dbname = "mita"

var stdin = bufio.NewScanner(os.Stdin)

func main() {
	_, err := exec.LookPath("fzf")
	if err != nil {
		fmt.Fprintf(os.Stderr, "not found 'fzf' executable\n")
		return
	}

	app := cli.NewApp()

	app.Name = "mita-cli"
	app.Usage = "cli(command line interface) of Mita's household accounts"
	app.Version = version

	app.Commands = []cli.Command{
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "add a new transaction",
			Action:  cmdAddTransaction,
		},
		{
			Name:    "edit",
			Aliases: []string{"e"},
			Usage:   "edit a transaction",
			Action:  cmdEditTransaction,
		},
		{
			Name:    "remove",
			Aliases: []string{"r"},
			Usage:   "remove a transaction",
			Action:  cmdRemoveTransaction,
		},
		{
			Name:    "template",
			Aliases: []string{"t"},
			Usage:   "options for transactions templates",
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "add a new template",
					Action:  cmdAddTemplate,
				},
				{
					Name:    "edit",
					Aliases: []string{"e"},
					Usage:   "edit a template",
					Action:  cmdEditTemplate,
				},
				{
					Name:    "remove",
					Aliases: []string{"r"},
					Usage:   "remove a template",
					Action:  cmdRemoveTemplate,
				},
				{
					Name:    "use",
					Aliases: []string{"u"},
					Usage:   "use a template",
					Action:  cmdUseTemplate,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
}

type dbtx interface {
	QueryRow(string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
}

const pgDomain = "/var/run/postgresql/"

func connectDB() (*sql.DB, error) {
	name := dbname

	if s := os.Getenv("MITA_DB"); s != "" {
		name = s
	}

	if runtime.GOOS != "windows" {
		if _, err := os.Stat(pgDomain); os.IsNotExist(err) == false {
			/* peer認証で接続するために、hostを指定して
			   UNIXドメインで接続してみる */
			db, err := sql.Open("postgres",
				fmt.Sprintf("host=%s dbname=%s sslmode=disable", pgDomain, name))
			if err == nil {
				return db, nil
			}
		}
	}

	return sql.Open("postgres", fmt.Sprintf("dbname=%s sslmode=disable", name))
}

func fzf(src io.Reader, dst io.Writer, errDst io.Writer, args []string) (bool, error) {
	cmd := exec.Command("fzf", args...)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return false, err
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// stdin
	go func() {
		_, err := io.Copy(stdin, src)

		if e, ok := err.(*os.PathError); ok && e.Err == syscall.EPIPE {
			// ignore EPIPE
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "Error: failed to write to STDIN of fzf", err)
		}

		stdin.Close()
		wg.Done()
	}()

	// stdout
	go func() {
		io.Copy(dst, stdout)
		stdout.Close()
		wg.Done()
	}()

	// stderr
	go func() {
		io.Copy(errDst, stderr)
		stderr.Close()
		wg.Done()
	}()

	wg.Wait()
	err := cmd.Wait()
	if err != nil && err.Error() == "exit status 130" {
		return true, nil
	}

	return false, err
}
