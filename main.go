package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"unicode/utf8"
)

const version = "0.2.0"

const dbname = "mita"

var stdin = bufio.NewScanner(os.Stdin)

func main() {
	_, err := exec.LookPath("fzf")
	if err != nil {
		fmt.Fprintf(os.Stderr, "実行ファイル'fzf'が見つからない\n")
		return
	}

	app := cli.NewApp()

	app.Name = "mita-cli"
	app.Usage = "家計簿のミタCLI"
	app.Version = version

	app.Commands = []cli.Command{
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "取引を検索",
			Action:  cmdSearchTransaction,
		},
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "取引を追加",
			Action:  cmdAddTransaction,
		},
		{
			Name:    "edit",
			Aliases: []string{"e"},
			Usage:   "取引を編集",
			Action:  cmdEditTransaction,
		},
		{
			Name:    "remove",
			Aliases: []string{"r"},
			Usage:   "取引を削除",
			Action:  cmdRemoveTransaction,
		},
		{
			Name:   "bs",
			Usage:  "資産・負債の一覧",
			Action: cmdBS,
		},
		{
			Name:   "pl",
			Usage:  "月の収入・費用の一覧",
			Action: cmdPL,
		},
		{
			Name:   "undo",
			Usage:  "取引への操作を元に戻す",
			Action: cmdUndoTransaction,
		},
		{
			Name:    "account",
			Aliases: []string{"ac"},
			Usage:   "勘定科目のオプション",
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "勘定科目を追加",
					Action:  cmdAddAccount,
				},
				{
					Name:    "edit",
					Aliases: []string{"e"},
					Usage:   "勘定科目を編集",
					Action:  cmdEditAccount,
				},
				{
					Name:    "remove",
					Aliases: []string{"r"},
					Usage:   "勘定科目を削除",
					Action:  cmdRemoveAccount,
				},
			},
		},
		{
			Name:    "template",
			Aliases: []string{"t"},
			Usage:   "テンプレートのオプション",
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "テンプレートを追加",
					Action:  cmdAddTemplate,
				},
				{
					Name:    "edit",
					Aliases: []string{"e"},
					Usage:   "テンプレートを編集",
					Action:  cmdEditTemplate,
				},
				{
					Name:    "remove",
					Aliases: []string{"r"},
					Usage:   "テンプレートを削除",
					Action:  cmdRemoveTemplate,
				},
				{
					Name:    "use",
					Aliases: []string{"u"},
					Usage:   "テンプレートを使用",
					Action:  cmdUseTemplate,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "エラー:", err)
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
			fmt.Fprintln(os.Stderr, "エラー: fzfの標準入力への書き込みに失敗", err)
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
		// ESCキーを押して選択がキャンセルされた
		return true, nil
	}

	return false, err
}

/* テキストエディタを開いてユーザからテキストを得る
 */
func scanWithEditor(text string) (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}

	filename := f.Name()

	defer os.Remove(filename)

	f.WriteString(text)
	f.Close()

	if err = openFileInEditor(filename); err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func openFileInEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	executable, err := exec.LookPath(editor)
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func scanInt(prompt string, minValue int, maxValue int) int {
	for {
		fmt.Printf("%s: ", prompt)
		stdin.Scan()
		text := stdin.Text()
		v, err := strconv.Atoi(text)

		if v >= minValue && v <= maxValue {
			return v
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, "エラー: 数値を入力してください")
		} else {
			fmt.Fprintf(os.Stderr, "エラー: 値が範囲外 [%d, %d]\n", minValue, maxValue)
		}
	}
}

func scanText(prompt string, minLen int, maxLen int) string {
	for {
		fmt.Printf("%s: ", prompt)
		stdin.Scan()
		text := stdin.Text()
		textLen := utf8.RuneCountInString(text)

		if textLen >= minLen && textLen <= maxLen {
			return text
		}

		fmt.Fprintf(os.Stderr, "エラー: 文字数が範囲外 [%d, %d]\n", minLen, maxLen)
	}
}
