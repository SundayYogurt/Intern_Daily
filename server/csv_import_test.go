package main

import (
	"strings"
	"testing"
)

func TestParseLogCSV(t *testing.T) {
	input := "\ufeffวันที่,ประเภท,เวลาเริ่ม,เวลาสิ้นสุด,พัก (นาที),มาสาย,งานที่ทำ,ปัญหาที่พบ,สิ่งที่เรียนรู้\r\n" +
		"2026-10-20,ปฏิบัติงาน,09:00,17:00,60,ใช่,\"เขียน API, ทดสอบ\",,ใช้ CSV\r\n" +
		"2026-10-21,ลาป่วย,,,,ไม่,,พักรักษา,\r\n"
	rows, issues, err := parseLogCSV([]byte(input))
	if err != nil || len(issues) != 0 || len(rows) != 2 {
		t.Fatalf("rows=%d issues=%v err=%v", len(rows), issues, err)
	}
	if rows[0].Log.Kind != "work" || rows[0].Log.Work != "เขียน API, ทดสอบ" || !rows[0].Log.Late || rows[0].Log.BreakMinutes != 60 {
		t.Fatalf("first row: %+v", rows[0].Log)
	}
	if rows[1].Log.Kind != "sick" {
		t.Fatalf("second row: %+v", rows[1].Log)
	}
}

func TestParseLogCSVRejectsInvalidInput(t *testing.T) {
	for _, input := range [][]byte{
		[]byte("date,kind\n2026-10-20,work\n"),
		[]byte("date,kind,work,blokers\n2026-10-20,work,x,y\n"),
		[]byte("date,kind,work\n2026-10-20,work,\"unfinished\n"),
		[]byte{0xff, 0xfe},
		[]byte("date,kind,work\n" + strings.Repeat("2026-10-20,work,x\n", maxImportRows+1)),
	} {
		if _, _, err := parseLogCSV(input); err == nil {
			t.Fatalf("accepted invalid CSV: %q", string(input[:min(len(input), 60)]))
		}
	}
}

func TestValidateImportedLog(t *testing.T) {
	x := &Internship{StartDate: "2026-10-01", EndDate: "2026-10-31"}
	valid := DayLog{Date: "2026-10-20", Kind: "work", StartTime: "09:00", EndTime: "17:00", BreakMinutes: 60, Work: "งานทดสอบ"}
	if got := validateImportedLog(valid, x, "2026-11-13"); got != "" {
		t.Fatal(got)
	}
	for _, change := range []func(*DayLog){
		func(l *DayLog) { l.Date = "2026-11-01" },
		func(l *DayLog) { l.Date = "2026-10-30" },
		func(l *DayLog) { l.Work = "" },
		func(l *DayLog) { l.EndTime = "08:00" },
	} {
		l := valid
		change(&l)
		if got := validateImportedLog(l, x, "2026-10-29"); got == "" {
			t.Fatalf("accepted invalid log: %+v", l)
		}
	}
}
