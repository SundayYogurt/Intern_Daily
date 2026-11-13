package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

const maxImportRows = 500
const maxImportBytes = 2 << 20

type csvLogRow struct {
	Line int    `json:"line"`
	Date string `json:"date"`
	Kind string `json:"kind"`
	Work string `json:"work"`
	Log  DayLog `json:"-"`
}

type csvIssue struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type csvImportPreview struct {
	Count  int         `json:"count"`
	Rows   []csvLogRow `json:"rows"`
	Errors []csvIssue  `json:"errors"`
}

func csvColumn(name string) string {
	name = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))
	switch name {
	case "วันที่":
		return "date"
	case "ประเภท":
		return "kind"
	case "เวลาเริ่ม":
		return "start_time"
	case "เวลาสิ้นสุด":
		return "end_time"
	case "พัก (นาที)", "เวลาพัก":
		return "break_minutes"
	case "มาสาย":
		return "late"
	case "งานที่ปฏิบัติ", "งานที่ทำ":
		return "work"
	case "ปัญหาที่พบ":
		return "blockers"
	case "สิ่งที่เรียนรู้":
		return "learning"
	default:
		return name
	}
}

func csvKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "work", "ปฏิบัติงาน":
		return "work"
	case "sick", "ลาป่วย":
		return "sick"
	case "personal", "ลากิจ":
		return "personal"
	case "holiday", "วันหยุด":
		return "holiday"
	default:
		return strings.TrimSpace(value)
	}
}

func csvBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no", "ไม่", "ไม่ใช่":
		return false, true
	case "1", "true", "yes", "ใช่":
		return true, true
	default:
		return false, false
	}
}

func parseLogCSV(data []byte) ([]csvLogRow, []csvIssue, error) {
	if !utf8.Valid(data) {
		return nil, nil, fmt.Errorf("ไฟล์ CSV ต้องเป็น UTF-8")
	}
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	head, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("อ่านหัวตาราง CSV ไม่ได้")
	}
	columns := map[string]int{}
	allowed := map[string]bool{"date": true, "kind": true, "start_time": true, "end_time": true, "break_minutes": true, "late": true, "work": true, "blockers": true, "learning": true}
	for i, name := range head {
		key := csvColumn(name)
		if !allowed[key] {
			return nil, nil, fmt.Errorf("ไม่รู้จักคอลัมน์ %q", name)
		}
		if _, exists := columns[key]; exists {
			return nil, nil, fmt.Errorf("หัวตาราง %s ซ้ำกัน", key)
		}
		columns[key] = i
	}
	for _, required := range []string{"date", "kind", "work"} {
		if _, ok := columns[required]; !ok {
			return nil, nil, fmt.Errorf("ต้องมีคอลัมน์ %s", required)
		}
	}
	get := func(record []string, key string) string {
		if i, ok := columns[key]; ok && i < len(record) {
			return strings.TrimSpace(record[i])
		}
		return ""
	}
	rows := []csvLogRow{}
	issues := []csvIssue{}
	recordCount := 0
	for {
		record, readErr := r.Read()
		if readErr == io.EOF {
			break
		}
		recordCount++
		if readErr != nil {
			return nil, nil, fmt.Errorf("CSV แถว %d: %v", recordCount+1, readErr)
		}
		line, _ := r.FieldPos(0)
		if recordCount > maxImportRows {
			return nil, nil, fmt.Errorf("นำเข้าได้สูงสุด %d แถวต่อครั้ง", maxImportRows)
		}
		if len(record) != len(head) {
			issues = append(issues, csvIssue{line, "จำนวนคอลัมน์ไม่ตรงกับหัวตาราง"})
			continue
		}
		empty := true
		for _, cell := range record {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		breakMinutes := 0
		if value := get(record, "break_minutes"); value != "" {
			breakMinutes, err = strconv.Atoi(value)
			if err != nil {
				issues = append(issues, csvIssue{line, "พัก (นาที) ต้องเป็นจำนวนเต็ม"})
				continue
			}
		}
		late, valid := csvBool(get(record, "late"))
		if !valid {
			issues = append(issues, csvIssue{line, "มาสายต้องเป็น true/false หรือ 1/0"})
			continue
		}
		log := DayLog{Date: get(record, "date"), Kind: csvKind(get(record, "kind")), StartTime: get(record, "start_time"), EndTime: get(record, "end_time"), BreakMinutes: breakMinutes, Late: late, Work: get(record, "work"), Blockers: get(record, "blockers"), Learning: get(record, "learning")}
		rows = append(rows, csvLogRow{Line: line, Date: log.Date, Kind: log.Kind, Work: log.Work, Log: log})
	}
	if len(rows) == 0 && len(issues) == 0 {
		return nil, nil, fmt.Errorf("ไฟล์ CSV ไม่มีข้อมูล")
	}
	return rows, issues, nil
}

func validateImportedLog(l DayLog, x *Internship, today string) string {
	if !validDate(l.Date) {
		return "วันที่ต้องเป็น YYYY-MM-DD"
	}
	if l.Kind != "work" && l.Kind != "sick" && l.Kind != "personal" && l.Kind != "holiday" {
		return "ประเภทไม่ถูกต้อง"
	}
	if l.Date > today {
		return "ยังบันทึกวันในอนาคตไม่ได้"
	}
	if x.StartDate != "" && l.Date < x.StartDate || x.EndDate != "" && l.Date > x.EndDate {
		return "วันที่อยู่นอกช่วงฝึกงาน"
	}
	if l.Kind == "work" && (!validClock(l.StartTime) || !validClock(l.EndTime) || l.StartTime >= l.EndTime || l.BreakMinutes < 0 || l.BreakMinutes > 600 || hours(l) <= 0 || l.Work == "") {
		return "กรอกเวลาและงานที่ทำให้ครบ"
	}
	if len(l.Work) > 4000 || len(l.Blockers) > 2000 || len(l.Learning) > 2000 {
		return "ข้อความยาวเกินไป"
	}
	return ""
}

func importLogs(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เฉพาะนักศึกษา")
	}
	x, err := internshipFor(u)
	if err != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	f, err := c.FormFile("file")
	if err != nil || f.Size > maxImportBytes {
		return fail(c, 400, "เลือกไฟล์ CSV ขนาดไม่เกิน 2 MB")
	}
	file, err := f.Open()
	if err != nil {
		return fail(c, 400, "เปิดไฟล์ไม่ได้")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
	if err != nil || len(data) > maxImportBytes {
		return fail(c, 400, "ไฟล์ใหญ่เกินไป")
	}
	rows, issues, err := parseLogCSV(data)
	if err != nil {
		return fail(c, 400, err.Error())
	}
	tx, err := db.Begin()
	if err != nil {
		return fail(c, 500, "ฐานข้อมูลไม่พร้อม")
	}
	defer tx.Rollback()
	if err = tx.QueryRow(`SELECT start_date,end_date FROM internships WHERE id=?`, x.ID).Scan(&x.StartDate, &x.EndDate); err != nil {
		return fail(c, 500, "ตรวจช่วงฝึกงานไม่สำเร็จ")
	}
	existing := map[string]bool{}
	query, err := tx.Query(`SELECT date FROM day_logs WHERE internship_id=?`, x.ID)
	if err != nil {
		return fail(c, 500, "ตรวจบันทึกเดิมไม่สำเร็จ")
	}
	for query.Next() {
		var date string
		if err = query.Scan(&date); err != nil {
			query.Close()
			return fail(c, 500, "ตรวจบันทึกเดิมไม่สำเร็จ")
		}
		existing[date] = true
	}
	err = query.Err()
	query.Close()
	if err != nil {
		return fail(c, 500, "ตรวจบันทึกเดิมไม่สำเร็จ")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if message := validateImportedLog(row.Log, x, todayBangkok()); message != "" {
			issues = append(issues, csvIssue{row.Line, message})
		} else if seen[row.Date] {
			issues = append(issues, csvIssue{row.Line, "วันที่ซ้ำในไฟล์ CSV"})
		} else if existing[row.Date] {
			issues = append(issues, csvIssue{row.Line, "มีบันทึกวันที่นี้อยู่แล้ว"})
		}
		seen[row.Date] = true
	}
	preview := csvImportPreview{Count: len(rows), Rows: rows, Errors: issues}
	if c.Query("preview") == "1" {
		return c.JSON(preview)
	}
	if len(issues) > 0 {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("CSV มีข้อมูลที่ต้องแก้ %d จุด กรุณาตรวจไฟล์อีกครั้ง", len(issues)), "errors": issues})
	}
	for _, row := range rows {
		l := row.Log
		if l.Kind != "work" {
			l.StartTime, l.EndTime, l.BreakMinutes, l.Late = "", "", 0, false
		}
		if _, err = tx.Exec(`INSERT INTO day_logs(internship_id,date,start_time,end_time,break_minutes,kind,late,work,blockers,learning) VALUES(?,?,?,?,?,?,?,?,?,?)`, x.ID, l.Date, l.StartTime, l.EndTime, l.BreakMinutes, l.Kind, l.Late, l.Work, l.Blockers, l.Learning); err != nil {
			return fail(c, 409, "วันที่ซ้ำหรือบันทึกข้อมูลไม่สำเร็จ")
		}
	}
	if err = tx.Commit(); err != nil {
		return fail(c, 500, "นำเข้าไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"imported": len(rows)})
}
