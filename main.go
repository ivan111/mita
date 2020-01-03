package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
	"golang.org/x/text/width"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode/utf8"
)

const appName = "mita"
const version = "0.9.0"

const defaultDBName = "mita"
const defaultPort = 5001

type config struct {
	DB     database `toml:"database"`
	Server server   `toml:"server"`
}

type database struct {
	Name     string `toml:"name"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}

type server struct {
	Port int `toml:"port"`
}

var configData = config{
	database{
		Name:     defaultDBName,
		User:     "",
		Password: "",
	},
	server{
		Port: defaultPort,
	},
}

var testMode = false
var scanner = bufio.NewScanner(os.Stdin)
var stdin io.Reader = os.Stdin
var stdout io.Writer = os.Stdout
var stderr io.Writer = os.Stderr

func main() {
	err := loadOrCreateConfig()
	if err != nil {
		eprintln(err)
		return
	}

	_, err = exec.LookPath("fzf")
	if err != nil {
		eprintln("実行ファイル'fzf'が見つからない")
		return
	}

	app := cli.NewApp()

	app.Name = appName
	app.Usage = "家計簿のミタ"
	app.Version = version

	app.Commands = []cli.Command{
		{
			Name:    "transaction",
			Aliases: []string{"tr"},
			Usage:   "取引のオプション",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "取引を一覧",
					Flags: []cli.Flag{
						cli.BoolFlag{Name: "all, a"},
					},
					Action: cmdListTransactions,
				},
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
					Name:   "import",
					Usage:  "取引のインポート",
					Action: cmdImportTransactions,
				},
				{
					Name:   "export",
					Usage:  "取引のエクスポート",
					Action: cmdExportTransactions,
				},
			},
		},
		{
			Name:    "account",
			Aliases: []string{"ac"},
			Usage:   "勘定科目のオプション",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "勘定科目を一覧",
					Action:  cmdListAccounts,
				},
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
				{
					Name:    "order",
					Aliases: []string{"o"},
					Usage:   "勘定科目の並べ替え",
					Action:  cmdReorderAccount,
				},
				{
					Name:   "import",
					Usage:  "勘定科目のインポート",
					Action: cmdImportAccounts,
				},
				{
					Name:   "export",
					Usage:  "勘定科目のエクスポート",
					Action: cmdExportAccounts,
				},
			},
		},
		{
			Name:    "template",
			Aliases: []string{"te"},
			Usage:   "テンプレートのオプション",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "テンプレートを一覧",
					Action:  cmdListTemplates,
				},
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
				{
					Name:   "import",
					Usage:  "テンプレートのインポート",
					Action: cmdImportTemplate,
				},
				{
					Name:   "export",
					Usage:  "テンプレートのエクスポート",
					Action: cmdExportTemplates,
				},
			},
		},
		{
			Name:  "history",
			Usage: "履歴のオプション",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "履歴を一覧",
					Flags: []cli.Flag{
						cli.BoolFlag{Name: "all, a"},
					},
					Action: cmdListHistory,
				},
			},
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
			Name:  "server",
			Usage: "グラフサイトを表示するHTTPサーバを起動",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port, p",
					Value: configData.Server.Port,
				},
			},
			Action: cmdServer,
		},
		{
			Name:   "undo",
			Usage:  "取引への操作を元に戻す",
			Action: cmdUndoTransaction,
		},
	}

	if err := app.Run(os.Args); err != nil {
		eprintln("エラー:", err)
		os.Exit(1)
	}
}

func loadOrCreateConfig() error {
	configDir := getConfigDir()

	if _, err := os.Stat(configDir); err != nil {
		if err := os.MkdirAll(configDir, 0777); err != nil {
			return err
		}
	}

	filename := filepath.Join(configDir, "config.toml")

	conf := &configData

	if _, err := os.Stat(filename); err == nil {
		if _, err := toml.DecodeFile(filename, &conf); err != nil {
			return err
		}
	} else {
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := toml.NewEncoder(f).Encode(conf); err != nil {
			eprintln(err)
		}
	}

	if conf.DB.Name == "" {
		conf.DB.Name = defaultDBName
	}

	if conf.Server.Port == 0 {
		conf.Server.Port = defaultPort
	}

	return nil
}

func getConfigDir() string {
	home := os.Getenv("HOME")

	var configDir string

	if home == "" && runtime.GOOS == "windows" {
		configDir = filepath.Join(os.Getenv("APPDATA"), appName)
	} else {
		configDir = filepath.Join(home, ".config", appName)
	}

	return configDir
}

type dbtx interface {
	QueryRow(string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
}

const pgDomain = "/var/run/postgresql/"

func connectDB() (*sql.DB, error) {
	name := configData.DB.Name

	if s := os.Getenv("MITA_DB"); s != "" {
		name = s
	}

	if runtime.GOOS != "windows" {
		if _, err := os.Stat(pgDomain); err == nil {
			/* peer認証で接続するために、hostを指定して
			   UNIXドメインで接続してみる */
			db, err := sql.Open("postgres",
				fmt.Sprintf("host=%s dbname=%s sslmode=disable", pgDomain, name))
			if err == nil {
				return db, nil
			}
		}
	}

	var dataSrcName string

	user := configData.DB.User
	password := configData.DB.Password

	if user != "" {
		dataSrcName = fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, name)
	} else {
		dataSrcName = fmt.Sprintf("dbname=%s sslmode=disable", name)
	}

	return sql.Open("postgres", dataSrcName)
}

func fzf(src io.Reader, dst io.Writer, errDst io.Writer, args []string) (bool, error) {
	if testMode {
		s := input()
		if s == "" {
			return true, nil
		}

		dst.Write([]byte(s))
		return false, nil
	}

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
			eprintln("エラー: fzfの標準入力への書き込みに失敗", err)
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
戻り値のboolはcancel
*/
func scanWithEditor(text string) (string, bool, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", false, err
	}

	filename := f.Name()

	defer os.Remove(filename)

	f.WriteString(text)
	f.Close()

	file, err := os.Stat(filename)
	if err != nil {
		return "", false, err
	}

	modTime := file.ModTime()

	if err = openFileInEditor(filename); err != nil {
		return "", false, err
	}

	file, err = os.Stat(filename)
	if err != nil {
		return "", false, err
	}

	if file.ModTime() == modTime {
		return "", true, nil
	}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", false, err
	}

	return string(bytes), false, nil
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

func readOrder(text string, numItems int) ([]int, error) {
	var noArr []int

	for _, line := range strings.Split(text, "\n") {
		arr := strings.Split(line, " ")

		if len(arr) <= 1 {
			continue
		}

		no, err := strconv.Atoi(arr[0])
		if err != nil {
			return nil, errors.New("先頭が数値以外の行がある")
		}

		if no < 0 || no >= numItems {
			return nil, errors.New("数値が範囲外")
		}

		noArr = append(noArr, no)
	}

	if len(noArr) != numItems {
		return nil, errors.New("長さが一致しない")
	}

	testArr := make([]int, numItems)
	copy(testArr, noArr)
	sort.Ints(testArr)
	for i, v := range testArr {
		if i != v {
			return nil, errors.New("同じ数値が存在する")
		}
	}

	nwo := make([]int, numItems)
	for i, no := range noArr {
		nwo[no] = i
	}

	return nwo, nil
}

func confirmYesNo(msg string) bool {
	print(msg, " (y[es]/[no]): ")
	ans := strings.ToLower(input())
	return ans == "y" || ans == "yes"
}

func input() string {
	scanner.Scan()
	return scanner.Text()
}

func scanInt(prompt string, minValue int, maxValue int) int {
	for {
		print(prompt + ": ")
		text := input()
		v, err := strconv.Atoi(text)

		if v >= minValue && v <= maxValue {
			return v
		}

		if err != nil {
			eprintln("エラー: 数値を入力してください")
		} else {
			eprintf("エラー: 値が範囲外 [%d, %d]\n", minValue, maxValue)
		}
	}
}

func scanText(prompt string, minLen int, maxLen int) string {
	for {
		print(prompt + ": ")
		text := input()
		textLen := utf8.RuneCountInString(text)

		if textLen >= minLen && textLen <= maxLen {
			return text
		}

		eprintf("エラー: 文字数が範囲外 [%d, %d]\n", minLen, maxLen)
	}
}

func getTextWidth(s string) int {
	var w int

	for _, ch := range s {
		kind := width.LookupRune(ch).Kind()

		if kind == width.EastAsianWide || kind == width.EastAsianFullwidth {
			w += 2
		} else {
			w++
		}
	}

	return w
}

func skipSpace(s string) string {
	for i, ch := range s {
		if ch != ' ' {
			return s[i:]
		}
	}

	return ""
}

func importItems(filename string, fn func(*sql.DB, io.Reader) error) error {
	var f io.Reader

	if filename == "" {
		f = os.Stdin
	} else {
		_, err := os.Stat(filename)
		if err != nil {
			return errors.New("ファイルが見つからない:" + filename)
		}

		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		f = file
	}

	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return fn(db, f)
}

func exportItems(filename string, fn func(*sql.DB, io.Writer) error) error {
	var f io.Writer

	if filename == "" {
		f = os.Stdout
	} else {
		_, err := os.Stat(filename)
		if err == nil {
			return errors.New("ファイルが既に存在する:" + filename)
		}

		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		f = file
	}

	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return fn(db, f)
}

func eprintln(a ...interface{}) (int, error) {
	return fmt.Fprintln(stderr, a...)
}

func eprintf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(stderr, format, a...)
}

func println(a ...interface{}) (int, error) {
	return fmt.Fprintln(stdout, a...)
}

func printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(stdout, format, a...)
}

func print(a ...interface{}) (int, error) {
	return fmt.Fprint(stdout, a...)
}
