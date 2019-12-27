package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
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

func cmdAddTemplate(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

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

	fmt.Println()
	for i, d := range tmpl.items {
		fmt.Println(i, &d)
	}

	var a string
	if len(tmpl.items) == 0 {
		a = "add"
	} else {
		fmt.Print(q)
		stdin.Scan()
		a = strings.ToLower(stdin.Text())
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

		fmt.Println()
		for i, d := range tmpl.items {
			fmt.Println(i, &d)
		}

		if len(tmpl.items) == 0 {
			q = "a(dd), q(uit): "
		}

		fmt.Print(q)
		stdin.Scan()
		a = strings.ToLower(stdin.Text())
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

	tmpl, err := selectTemplate(db)
	if tmpl == nil || err != nil {
		return err
	}

	ok := confirmRemoveTemplate(tmpl)

	if ok {
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
		if err != nil {
			return err
		}

		fmt.Println("削除完了")
	} else {
		fmt.Println("キャンセルした")
	}

	return nil
}

func confirmRemoveTemplate(tmpl *template) bool {
	fmt.Println(tmpl)
	fmt.Print("本当に削除する? (Y/[no]): ")
	stdin.Scan()
	return stdin.Text() == "Y"
}

func cmdUseTemplate(context *cli.Context) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

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
			fmt.Println(&d)
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
		fmt.Fprintln(os.Stderr, "テンプレートの中身がない")
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
		fmt.Fprintln(os.Stderr, err)
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

func confirmTemplate(accounts []account, d *templateDetail) (bool, error) {
	for {
		fmt.Println()
		fmt.Println(d)

		fmt.Print("y(es), l(eft), r(ight), a(mount), n(ote), q(uit): ")
		stdin.Scan()
		a := strings.ToLower(stdin.Text())

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
		fmt.Println()
		for i, tr := range trs {
			fmt.Println(i, &tr)
		}

		fmt.Print(q)
		stdin.Scan()
		a := strings.ToLower(stdin.Text())

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
					fmt.Fprintln(os.Stderr, err)
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

func selectTemplate(db *sql.DB) (*template, error) {
	templates, err := dbGetTemplates(db)
	if err != nil {
		return nil, err
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

	fmt.Printf("テンプレート: %s\n", tmpl.name)

	return tmpl, nil
}

func selectTemplateDetail(items []templateDetail) (*templateDetail, error) {
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
