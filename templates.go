package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"strconv"
	"strings"
)

type template struct {
	id    int
	name  string
	items []templateDetail
}

func (d *template) String() string {
	return d.name
}

type templateDetail struct {
	templateID int
	no         int
	orderNo    int
	debit      account
	credit     account
	amount     int
	note       string
}

func (d *templateDetail) String() string {
	return fmt.Sprintf("%s / %s %s %s", d.debit.name, d.credit.name,
		int2str(d.amount), d.note)
}

func cmdListTemplates(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	count, err := dbCountTemplateItems(db)

	maxNo := count - 1
	noWidth := len(strconv.Itoa(maxNo))

	templates, err := dbGetTemplates(db)
	if err != nil {
		return err
	}

	i := 0

	for _, tmpl := range templates {
		name := tmpl.name

		items, err := dbGetTemplateItems(db, tmpl.id)
		if err != nil {
			return err
		}

		nameWidth := getTextWidth(name)
		nw := 16 - nameWidth
		if nw < 0 {
			nw = 0
		}

		for _, d := range items {
			debitWidth := getTextWidth(d.debit.name)
			dw := 16 - debitWidth
			if dw < 0 {
				dw = 0
			}

			creditWidth := getTextWidth(d.credit.name)
			cw := 16 - creditWidth
			if cw < 0 {
				cw = 0
			}

			printf("%*d %s%*s %s%*s %s%*s %9s %s\n",
				noWidth, i,
				name, nw, "",
				d.debit.name, dw, "",
				d.credit.name, cw, "",
				int2str(d.amount), d.note)

			i++
		}
	}

	return nil
}

func cmdAddTemplate(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runAddTemplate(db)
}

func runAddTemplate(db *sql.DB) error {
	var tmpl template

	tmpl.name = scanTemplateName()

	id, err := dbAddTemplate(db, tmpl.name)
	if err != nil {
		return err
	}

	tmpl.id = id

	return editTemplate(db, &tmpl)
}

func cmdEditTemplate(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tmpl, err := selectTemplate(db)
	if tmpl == nil || err != nil {
		return err
	}

	return editTemplate(db, tmpl)
}

func editTemplate(db *sql.DB, tmpl *template) error {
	const qFmt = "%s, a(dd), r(emove), o(rder), q(uit): "

	q := fmt.Sprintf(qFmt, getRangeString(len(tmpl.items)))

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	maxNo := 1
	maxOrderNo := 1
	for _, d := range tmpl.items {
		if maxNo < d.no {
			maxNo = d.no
		}

		if maxOrderNo < d.orderNo {
			maxOrderNo = d.orderNo
		}
	}

	println()
	for i, d := range tmpl.items {
		println(i, &d)
	}

	var a string
	if len(tmpl.items) == 0 {
		a = "add"
	} else {
		print(q)
		s, err := input()
		if err != nil {
			return err
		}
		a = strings.ToLower(s)
	}

	for a != "q" {
		isUpdateItems := false

		switch a {
		case "a", "add":
			d, err := scanTemplateDetail(accounts)
			if err != nil {
				return err
			}

			if d == nil {
				break
			}

			ok, err := confirmTemplate(accounts, d)
			if err != nil {
				return err
			}

			if ok {
				maxNo++
				maxOrderNo++

				d.templateID = tmpl.id
				d.no = maxNo
				d.orderNo = maxOrderNo

				err = dbAddTemplateItem(db, d)
				if err != nil {
					return err
				}

				isUpdateItems = true
			}
		case "r", "remove":
			d, err := selectTemplateDetail(tmpl.items)
			if err != nil {
				return err
			}
			if d == nil {
				break
			}

			err = dbRemoveTemplateItem(db, d.templateID, d.no)
			if err != nil {
				return err
			}

			isUpdateItems = true
		case "o", "order":
			ok, err := reorderTemplateDetails(db, tmpl)
			if err != nil {
				return err
			}

			if ok {
				isUpdateItems = true
			}
		default:
			no, err := strconv.Atoi(a)
			if err == nil && no >= 0 && no < len(tmpl.items) {
				d := tmpl.items[no]
				ok, err := confirmTemplate(accounts, &d)
				if err != nil {
					return err
				}

				if ok {
					err = dbEditTemplateItem(db, &d)
					if err != nil {
						return err
					}

					isUpdateItems = true
				}
			}
		}

		if isUpdateItems {
			items, err := dbGetTemplateItems(db, tmpl.id)
			if err != nil {
				return err
			}

			q = fmt.Sprintf(qFmt, getRangeString(len(items)))

			tmpl.items = items
		}

		println()
		for i, d := range tmpl.items {
			println(i, &d)
		}

		if len(tmpl.items) == 0 {
			q = "a(dd), q(uit): "
		}

		print(q)
		s, err := input()
		if err != nil {
			return err
		}
		a = strings.ToLower(s)
	}

	return nil
}

func getRangeString(length int) string {
	var s string

	switch length {
	case 0:
		s = ""
	case 1:
		s = "0"
	case 2:
		s = "0, 1"
	default:
		s = "0-" + strconv.Itoa(length-1)
	}

	return s
}

func cmdRemoveTemplate(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runRemoveTemplate(db)
}

func runRemoveTemplate(db *sql.DB) error {
	tmpl, err := selectTemplate(db)
	if tmpl == nil || err != nil {
		return err
	}

	if confirmYesNo("本当に削除する?") {
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		err = dbRemoveTemplateItems(db, tmpl.id)
		if err != nil {
			tx.Rollback()
			return err
		}

		err = dbRemoveTemplate(db, tmpl.id)
		if err != nil {
			tx.Rollback()
			return err
		}

		err = tx.Commit()
	}

	return err
}

func cmdUseTemplate(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	return runUseTemplate(db)
}

func runUseTemplate(db *sql.DB) error {
	tmpl, err := selectTemplate(db)
	if tmpl == nil || err != nil {
		return err
	}

	if len(tmpl.items) == 0 {
		return errors.New("テンプレートに行が登録されてない")
	}

	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	date := scanDate()
	trs := make([]transaction, len(tmpl.items))

	for i, d := range tmpl.items {
		trs[i].debit = d.debit
		trs[i].credit = d.credit
		trs[i].amount = d.amount
		trs[i].note = d.note

		if d.amount == 0 {
			println(&d)
			trs[i].amount = scanAmount()
		}

		trs[i].date = date
	}

	ok, err := confirmUseTemplate(accounts, trs)
	if err != nil {
		return err
	}

	if ok == false {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, tr := range trs {
		if tr.amount != 0 {
			_, err = dbAddTransaction(tx, &tr)
		}

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func reorderTemplateDetails(db *sql.DB, tmpl *template) (bool, error) {
	if len(tmpl.items) == 0 {
		eprintln("テンプレートの中身がない")
		return false, nil
	}

	src := new(bytes.Buffer)

	for i, d := range tmpl.items {
		src.Write([]byte(fmt.Sprintf("%d %v\n", i, &d)))
	}

	text, cancel, err := scanWithEditor(src.String())
	if err != nil {
		return false, err
	}

	if cancel {
		return false, nil
	}

	nwo, err := readOrder(text, len(tmpl.items))
	if err != nil {
		eprintln(err)
		return false, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return false, err
	}

	for i := 0; i < len(tmpl.items); i++ {
		if tmpl.items[i].orderNo != nwo[i] {
			tmpl.items[i].orderNo = nwo[i]

			err = dbReorderTemplateItem(tx, &tmpl.items[i])
			if err != nil {
				tx.Rollback()
				return false, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return false, err
	}

	return true, nil
}

func cmdImportTemplate(context *cli.Context) error {
	return importItems(context.Args().First(), readTemplates)
}

func readTemplates(db *sql.DB, f io.Reader) error {
	accounts, err := dbGetAccounts(db)
	if err != nil {
		return err
	}

	name2id := make(map[string]int)

	for _, d := range accounts {
		name2id[d.name] = d.id
	}

	scanner := bufio.NewScanner(f)

	var keys []string
	name2items := make(map[string][]*templateDetail)

	lineNo := 0

	for scanner.Scan() {
		lineNo++

		line := skipSpace(scanner.Text())

		if line == "" || line[0] == '#' {
			continue
		}

		arr := strings.Split(line, "\t")

		d, err := arr2templateItem(name2id, arr)
		if err != nil {
			return fmt.Errorf("%d:%s", lineNo, err)
		}

		if name2items[arr[0]] == nil {
			keys = append(keys, arr[0])
		}

		name2items[arr[0]] = append(name2items[arr[0]], d)
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, parentName := range keys {
		items := name2items[parentName]

		id, err := dbAddTemplate(tx, parentName)
		if err != nil {
			tx.Rollback()
			return err
		}

		for i, item := range items {
			item.templateID = id
			item.no = i + 1
			item.orderNo = i + 1

			err := dbAddTemplateItem(tx, item)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func arr2templateItem(name2id map[string]int, arr []string) (*templateDetail, error) {
	arrLen := len(arr)
	if arrLen < 3 || arrLen > 5 {
		return nil, fmt.Errorf("項目数が3から5でない")
	}

	var d templateDetail

	d.debit.id = name2id[arr[1]]
	if d.debit.id == 0 {
		return nil, fmt.Errorf("借方:存在しない勘定科目'%s'", arr[1])
	}

	d.credit.id = name2id[arr[2]]
	if d.credit.id == 0 {
		return nil, fmt.Errorf("貸方:存在しない勘定科目'%s'", arr[2])
	}

	if len(arr) >= 4 && arr[3] != "" {
		amount, err := strconv.Atoi(arr[3])
		if err != nil {
			return nil, fmt.Errorf("金額:%s", err)
		}
		d.amount = amount
	}

	if len(arr) >= 5 {
		d.note = arr[4]
	}

	return &d, nil
}

func cmdExportTemplates(context *cli.Context) error {
	return exportItems(context.Args().First(), writeTemplates)
}

func writeTemplates(db *sql.DB, f io.Writer) error {
	b := bufio.NewWriter(f)

	templates, err := dbGetTemplates(db)
	if err != nil {
		return err
	}

	for _, tmpl := range templates {
		name := tmpl.name

		items, err := dbGetTemplateItems(db, tmpl.id)
		if err != nil {
			return err
		}

		for _, d := range items {
			_, err := b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%s\n",
				name, d.debit.name, d.credit.name, d.amount, d.note))
			if err != nil {
				return err
			}
		}
	}

	b.Flush()

	return nil
}

func confirmTemplate(accounts []account, d *templateDetail) (bool, error) {
	for {
		println()
		println(d)

		print("y(es), l(eft), r(ight), a(mount), n(ote), q(uit): ")
		s, err := input()
		if err != nil {
			return false, err
		}
		a := strings.ToLower(s)

		switch a {
		case "q", "quit":
			return false, nil
		case "y", "yes":
			return true, nil
		case "l", "left":
			debit, err := selectAccount(accounts, "借方")
			if err != nil {
				return false, err
			}
			if debit != nil {
				d.debit = *debit
			}
		case "r", "right":
			credit, err := selectAccount(accounts, "貸方")
			if err != nil {
				return false, err
			}
			if credit != nil {
				d.credit = *credit
			}
		case "a", "amount":
			d.amount = scanAmount()
		case "n", "note":
			d.note = scanNote()
		}
	}
}

func confirmUseTemplate(accounts []account, trs []transaction) (bool, error) {
	q := fmt.Sprintf("y(es), d(ate), %s, q(uit): ", getRangeString(len(trs)))

	for {
		println()
		for i, tr := range trs {
			println(i, &tr)
		}

		print(q)
		s, err := input()
		if err != nil {
			return false, err
		}
		a := strings.ToLower(s)

		switch a {
		case "q", "quit":
			return false, nil
		case "y", "yes":
			return true, nil
		case "d", "date":
			date := scanDate()
			for i := 0; i < len(trs); i++ {
				trs[i].date = date
			}
		default:
			no, err := strconv.Atoi(a)
			if err == nil && no >= 0 && no < len(trs) {
				tr := trs[no]
				ok, err := confirmTransaction(accounts, &tr)
				if err != nil {
					eprintln(err)
				}

				if err == nil && ok {
					trs[no] = tr
				}
			}
		}
	}
}

const sqlGetTemplates = `
SELECT template_id, name
FROM templates
ORDER BY template_id
`

/*
テンプレート一覧を取得
template.items は設定されないことに注意
*/
func dbGetTemplates(db *sql.DB) ([]template, error) {
	rows, err := db.Query(sqlGetTemplates)
	if err != nil {
		return nil, err
	}

	var templates []template

	for rows.Next() {
		var d template

		if err := rows.Scan(&d.id, &d.name); err != nil {
			return nil, err
		}

		templates = append(templates, d)
	}
	rows.Close()

	return templates, nil
}

const sqlAddTemplate = `
INSERT INTO templates(name)
VALUES($1)
RETURNING template_id
`

func dbAddTemplate(db dbtx, name string) (int, error) {
	var s string
	err := db.QueryRow(sqlAddTemplate, name).Scan(&s)
	if err != nil {
		return 0, err
	}

	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	return id, err
}

const sqlRemoveTemplate = `
DELETE FROM templates
WHERE template_id = $1
`

func dbRemoveTemplate(db *sql.DB, id int) error {
	_, err := db.Exec(sqlRemoveTemplate, id)

	return err
}

const sqlGetTemplateItems = `
SELECT template_id, no, order_no,
       debit_id, debit_name, debit_search_words,
       credit_id, credit_name, credit_search_words,
	   amount, description
FROM templates_detail_view
WHERE template_id = $1
ORDER BY order_no, no
`

func dbGetTemplateItems(db *sql.DB, id int) ([]templateDetail, error) {
	rows, err := db.Query(sqlGetTemplateItems, id)
	if err != nil {
		return nil, err
	}

	var items []templateDetail

	for rows.Next() {
		var d templateDetail

		if err := rows.Scan(&d.templateID, &d.no, &d.orderNo,
			&d.debit.id, &d.debit.name, &d.debit.searchWords,
			&d.credit.id, &d.credit.name, &d.credit.searchWords,
			&d.amount, &d.note); err != nil {
			return nil, err
		}

		items = append(items, d)
	}
	rows.Close()

	return items, nil
}

const sqlAddTemplateItem = `
INSERT INTO templates_detail(template_id, no, order_no, debit_id, credit_id, amount, description)
VALUES($1, $2, $3, $4, $5, $6, $7)
`

func dbAddTemplateItem(db dbtx, d *templateDetail) error {
	_, err := db.Exec(sqlAddTemplateItem, d.templateID, d.no, d.orderNo,
		d.debit.id, d.credit.id, d.amount, d.note)

	return err
}

// d.templateID と d.no は変更されない前提
func dbEditTemplateItem(db *sql.DB, d *templateDetail) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = dbRemoveTemplateItem(db, d.templateID, d.no)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = dbAddTemplateItem(db, d)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

const sqlRemoveTemplateItem = `
DELETE FROM templates_detail
WHERE template_id = $1 AND no = $2
`

func dbRemoveTemplateItem(db *sql.DB, id int, no int) error {
	_, err := db.Exec(sqlRemoveTemplateItem, id, no)

	return err
}

const sqlRemoveTemplateItems = `
DELETE FROM templates_detail
WHERE template_id = $1
`

func dbRemoveTemplateItems(db *sql.DB, id int) error {
	_, err := db.Exec(sqlRemoveTemplateItems, id)

	return err
}

const sqlReorderTemplateItem = `
UPDATE templates_detail
SET order_no = $3
WHERE template_id = $1 AND no = $2
`

func dbReorderTemplateItem(db dbtx, d *templateDetail) error {
	_, err := db.Exec(sqlReorderTemplateItem, d.templateID, d.no, d.orderNo)

	return err
}

const sqlCountTemplateItems = `
SELECT COUNT(*)
FROM templates_detail
`

func dbCountTemplateItems(db dbtx) (int, error) {
	var countStr string

	err := db.QueryRow(sqlCountTemplateItems).Scan(&countStr)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func selectTemplate(db *sql.DB) (*template, error) {
	templates, err := dbGetTemplates(db)
	if err != nil {
		return nil, err
	}

	if len(templates) == 0 {
		return nil, nil
	}

	src := new(bytes.Buffer)

	for i, d := range templates {
		src.Write([]byte(fmt.Sprintf("%d %v\n", i, &d)))
	}

	dst := new(bytes.Buffer)
	args := []string{}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	arr := strings.Split(dst.String(), " ")

	i, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, err
	}

	tmpl := &templates[i]

	items, err := dbGetTemplateItems(db, tmpl.id)
	if err != nil {
		return nil, err
	}

	tmpl.items = items

	println("テンプレート:", tmpl.name)

	return tmpl, nil
}

func selectTemplateDetail(items []templateDetail) (*templateDetail, error) {
	if len(items) == 0 {
		return nil, nil
	}

	src := new(bytes.Buffer)

	for i, d := range items {
		src.Write([]byte(fmt.Sprintf("%d %v\n", i, &d)))
	}

	dst := new(bytes.Buffer)
	args := []string{}

	cancel, err := fzf(src, dst, os.Stderr, args)
	if cancel {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	arr := strings.Split(dst.String(), " ")

	i, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, err
	}

	return &items[i], nil
}

func scanTemplateDetail(accounts []account) (*templateDetail, error) {
	var d templateDetail

	debit, err := selectAccount(accounts, "借方")
	if err != nil {
		return nil, err
	}
	if debit == nil {
		return nil, nil
	}
	d.debit = *debit

	credit, err := selectAccount(accounts, "貸方")
	if err != nil {
		return nil, err
	}
	if credit == nil {
		return nil, nil
	}
	d.credit = *credit

	return &d, nil
}

func scanTemplateName() string {
	return scanText("名前", 1, 32)
}
