package main

import (
	"testing"
	"time"
)

func isSameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func TestStr2date(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	if d, err := str2date(""); err != nil || !isSameDate(d, today) {
		t.Fatalf(`str2date(""), got = %s, want = %s`, d, today)
	}

	yesterday := time.Now().AddDate(0, 0, -1)

	if d, err := str2date("-1"); err != nil || !isSameDate(d, yesterday) {
		t.Fatalf(`str2date("-1"), got = %s, want = %s`, d, yesterday)
	}

	day28 := time.Date(now.Year(), now.Month(), 28, 0, 0, 0, 0, time.Local)

	if d, err := str2date("28"); err != nil || !isSameDate(d, day28) {
		t.Fatalf(`str2date("28"), got = %s, want = %s`, d, day28)
	}

	omisoka := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.Local)

	if d, err := str2date("12/31"); err != nil || !isSameDate(d, omisoka) {
		t.Fatalf(`str2date("12/31"), got = %s, want = %s`, d, omisoka)
	}

	if d, err := str2date("12-31"); err != nil || !isSameDate(d, omisoka) {
		t.Fatalf(`str2date("12-31"), got = %s, want = %s`, d, omisoka)
	}

	happyNewYear := time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local)

	if d, err := str2date("2018/1/1"); err != nil || !isSameDate(d, happyNewYear) {
		t.Fatalf(`str2date("2018/1/1"), got = %s, want = %s`, d, happyNewYear)
	}

	if d, err := str2date("2018-01-01"); err != nil || !isSameDate(d, happyNewYear) {
		t.Fatalf(`str2date("2018-01-01"), got = %s, want = %s`, d, happyNewYear)
	}

	if _, err := str2date("1a"); err == nil {
		t.Fatal(`str2date("1a") should be error`)
	}

	if _, err := str2date("0"); err == nil {
		t.Fatal(`str2date("0") should be error`)
	}

	if _, err := str2date("32"); err == nil {
		t.Fatal(`str2date("32") should be error`)
	}
}

func toMonth(t time.Time) int {
	return t.Year()*100 + int(t.Month())
}

func TestStr2month(t *testing.T) {
	now := time.Now()

	if m, err := str2month(""); err != nil || m != 0 {
		t.Fatalf(`str2month(""), got = %d, want = %d`, m, 0)
	}

	year := now.Year()
	month := now.Month() - 1
	if month == 0 {
		month = 12
		year--
	}
	prevMonth := year*100 + int(month)

	if m, err := str2month("-1"); err != nil || m != prevMonth {
		t.Fatalf(`str2month("-1"), got = %d, want = %d`, m, prevMonth)
	}

	month1 := toMonth(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local))

	if m, err := str2month("1"); err != nil || m != month1 {
		t.Fatalf(`str2month("1"), got = %d, want = %d`, m, month1)
	}

	m1 := toMonth(time.Date(2018, 2, 1, 0, 0, 0, 0, time.Local))

	if m, err := str2month("2018/2"); err != nil || m != m1 {
		t.Fatalf(`str2month("2018/2"), got = %d, want = %d`, m, m1)
	}

	if m, err := str2month("2018-2"); err != nil || m != m1 {
		t.Fatalf(`str2month("12-31"), got = %d, want = %d`, m, m1)
	}

	if m, err := str2month("201802"); err != nil || m != m1 {
		t.Fatalf(`str2month("201801"), got = %d, want = %d`, m, m1)
	}

	if _, err := str2month("1a"); err == nil {
		t.Fatal(`str2month("1a") should be error`)
	}

	if _, err := str2month("0"); err == nil {
		t.Fatal(`str2month("0") should be error`)
	}

	if _, err := str2month("13"); err == nil {
		t.Fatal(`str2month("13") should be error`)
	}

	if _, err := str2month("201813"); err == nil {
		t.Fatal(`str2month("201813") should be error`)
	}
}

type subtractMonthTest struct {
	argYM int
	argN  int
	res   int
}

var subtractMonthTests = []subtractMonthTest{
	{202001, 0, 202001},
	{202002, 1, 202001},
	{202001, 1, 201912},
	{202001, 25, 201712},
}

func TestSubtractMonth(t *testing.T) {
	for i, test := range subtractMonthTests {
		res := subtractMonth(test.argYM, test.argN)
		if res != test.res {
			t.Errorf("#%d: got: %#v want: %#v", i, res, test.res)
		}
	}
}

type int2strTest struct {
	arg int
	res string
}

var int2strTests = []int2strTest{
	{0, "0"},
	{1, "1"},
	{12, "12"},
	{123, "123"},
	{1234, "1,234"},
	{12345, "12,345"},
	{123456, "123,456"},
	{1234567, "1,234,567"},
	{12345678, "12,345,678"},
	{123456789, "123,456,789"},
	{-1, "-1"},
	{-12, "-12"},
	{-123, "-123"},
	{-1234, "-1,234"},
	{-12345, "-12,345"},
	{-123456, "-123,456"},
	{-1234567, "-1,234,567"},
	{-12345678, "-12,345,678"},
	{-123456789, "-123,456,789"},
}

func TestInt2Str(t *testing.T) {
	for i, test := range int2strTests {
		res := int2str(test.arg)
		if res != test.res {
			t.Errorf("#%d: got: %#v want: %#v", i, res, test.res)
		}
	}
}
