package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

// ---- Report review (student) -----------------------------------------------

type reviewIssue struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Severity string `json:"severity"` // error, warn, tip
	Message  string `json:"message"`
	Source   string `json:"source"` // rule, ai
}

var shortOKKeys = map[string]bool{"company": true, "role": true, "scope": true}

func reportFieldLabels() (map[string]string, []string) {
	labels := map[string]string{}
	var order []string
	for _, ch := range reportChapters {
		for _, f := range ch.Fields {
			labels[f.Key] = f.Label
			order = append(order, f.Key)
		}
	}
	return labels, order
}

// ruleReviewIssues are cheap checks that don't need AI.
func ruleReviewIssues(report map[string]string) []reviewIssue {
	labels, order := reportFieldLabels()
	var out []reviewIssue
	for _, key := range order {
		text := strings.TrimSpace(report[key])
		add := func(sev, msg string) {
			out = append(out, reviewIssue{Key: key, Label: labels[key], Severity: sev, Message: msg, Source: "rule"})
		}
		if text == "" {
			add("error", "ยังไม่ได้เขียนหัวข้อนี้")
			continue
		}
		if n := strings.Count(text, "[เติมข้อมูล"); n > 0 {
			add("error", fmt.Sprintf("ยังมี [เติมข้อมูล] ค้างอยู่ %d จุด", n))
		}
		if n := strings.Count(text, "[ภาพที่"); n > 0 {
			add("warn", fmt.Sprintf("มีตำแหน่งภาพ [ภาพที่ …] %d จุด อย่าลืมแทรกภาพจริงในไฟล์ DOCX แล้วลบวงเล็บออก", n))
		}
		if strings.Contains(text, "**") || strings.Contains(text, "##") || strings.Contains(text, "```") {
			add("warn", "มีสัญลักษณ์ Markdown (** หรือ ##) ค้างอยู่ ควรลบออก")
		}
		if !shortOKKeys[key] && utf8.RuneCountInString(text) < 80 {
			add("warn", "เนื้อหาสั้นมาก ควรขยายความเพิ่ม")
		}
	}
	return out
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// parseAIIssues pulls a JSON array of issues out of a model reply.
func parseAIIssues(reply string, labels map[string]string) []reviewIssue {
	start, end := strings.Index(reply, "["), strings.LastIndex(reply, "]")
	if start < 0 || end <= start {
		return nil
	}
	var raw []struct {
		Key      string `json:"key"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
	}
	if json.Unmarshal([]byte(reply[start:end+1]), &raw) != nil {
		return nil
	}
	var out []reviewIssue
	for _, r := range raw {
		msg := strings.TrimSpace(stripInlineMarkdown(r.Message))
		if msg == "" {
			continue
		}
		label, ok := labels[r.Key]
		key := r.Key
		if !ok {
			key, label = "general", "ภาพรวมทั้งเล่ม"
		}
		sev := r.Severity
		if sev != "error" && sev != "warn" && sev != "tip" {
			sev = "warn"
		}
		out = append(out, reviewIssue{Key: key, Label: label, Severity: sev, Message: truncateRunes(msg, 300), Source: "ai"})
		if len(out) == 20 {
			break
		}
	}
	return out
}

// POST /api/ai/review-report {report?: {key: text}} — checks what is on screen (or the saved report).
func aiReviewReport(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เฉพาะนักศึกษา")
	}
	x, err := internshipFor(u)
	if err != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	var in struct {
		Report map[string]string `json:"report"`
	}
	c.BodyParser(&in)
	labels, order := reportFieldLabels()
	report := map[string]string{}
	for _, key := range order {
		report[key] = x.Report[key]
		if v, ok := in.Report[key]; ok && in.Report != nil {
			report[key] = v
		}
		if len(report[key]) > 20000 {
			return fail(c, 400, "เนื้อหารายงานยาวเกินไป")
		}
	}
	issues := ruleReviewIssues(report)
	resp := fiber.Map{"issues": issues, "ai_used": false, "ai_error": ""}

	var written []string
	for _, key := range order {
		if t := strings.TrimSpace(report[key]); t != "" {
			_, guide := reportGuide(key)
			written = append(written, fmt.Sprintf("[%s] %s\nแนวการเขียน: %s\nเนื้อหา:\n%s", key, labels[key], truncateRunes(guide, 220), truncateRunes(t, 1500)))
		}
	}
	if !aiEnabled() || len(written) == 0 {
		return c.JSON(resp)
	}
	if !aiAllow(u.ID) {
		resp["ai_error"] = "ใช้ AI บ่อยเกินไป ตรวจด้วยกฎพื้นฐานอย่างเดียว"
		return c.JSON(resp)
	}
	digest, _, _ := logsDigest(x.ID, "", "", 8000)
	general, _ := reportGuide("general")
	prompt := fmt.Sprintf(`ตรวจรายงานผลการฝึกงานต่อไปนี้ก่อนนักศึกษาส่งอาจารย์ แล้วตอบเป็น JSON array เท่านั้น ไม่มีข้อความอื่น
รูปแบบ: [{"key":"<รหัสหัวข้อในวงเล็บเหลี่ยม หรือ general>","severity":"error|warn|tip","message":"<ปัญหาและวิธีแก้ ภาษาไทย 1–2 ประโยค>"}]
ตรวจเฉพาะเรื่องเหล่านี้ ไม่เกิน 15 ข้อ เรียงจากสำคัญมากไปน้อย:
- error: ข้อความที่อ้างงาน เครื่องมือ ตัวเลข หรือผลลัพธ์ที่ไม่มีในบันทึกรายวัน หรือขัดแย้งกับบันทึก
- error: ใช้สรรพนามบุรุษที่ 1 (ผม ฉัน เรา หนู) แทนการเขียนแบบบุคคลที่สาม
- warn: ขาดเนื้อหาที่แนวการเขียนของหัวข้อนั้นกำหนด หรือเขียนผิดหัวข้อ
- warn: ภาษาไม่เป็นทางการ คำสะกดผิด ประโยคกำกวม
- tip: ข้อเสนอแนะสั้น ๆ ที่ทำให้รายงานดีขึ้น
ห้ามรายงานเรื่องหัวข้อที่ว่างหรือ [เติมข้อมูล] เพราะระบบตรวจแล้ว ถ้าไม่พบปัญหาให้ตอบ []

แนวการเขียนทั้งเล่ม:
%s

ข้อมูลการฝึกงาน:
%s
บันทึกรายวัน:
%s

รายงาน:
%s`, general, internshipContext(x), dash(digest), strings.Join(written, "\n\n"))
	ctx, cancel := context.WithTimeout(c.Context(), 110*time.Second)
	defer cancel()
	reply, err := aiChat(ctx, prompt)
	if err != nil {
		resp["ai_error"] = "AI ตรวจไม่สำเร็จ: " + err.Error()
		return c.JSON(resp)
	}
	aiIssues := parseAIIssues(reply, labels)
	if aiIssues == nil && !strings.Contains(reply, "[]") {
		resp["ai_error"] = "อ่านผลตรวจจาก AI ไม่ได้ ลองใหม่อีกครั้ง"
		return c.JSON(resp)
	}
	resp["issues"] = append(issues, aiIssues...)
	resp["ai_used"] = true
	return c.JSON(resp)
}

// ---- Approval summary (mentor) ---------------------------------------------

type approvalFlag struct {
	Date     string `json:"date"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func approvalFlags(logs []DayLog, from, to, start, end string, holidays []string, today string) []approvalFlag {
	var flags []approvalFlag
	have := map[string]bool{}
	for _, l := range logs {
		have[l.Date] = true
		if l.Kind != "work" {
			continue
		}
		if l.Late {
			flags = append(flags, approvalFlag{l.Date, "warn", "บันทึกว่ามาสาย"})
		}
		if l.Hours > 10 {
			flags = append(flags, approvalFlag{l.Date, "warn", fmt.Sprintf("ชั่วโมงทำงานสูงผิดปกติ (%.1f ชม.)", l.Hours)})
		} else if l.Hours > 0 && l.Hours < 4 {
			flags = append(flags, approvalFlag{l.Date, "warn", fmt.Sprintf("ชั่วโมงทำงานน้อย (%.1f ชม.)", l.Hours)})
		}
		if utf8.RuneCountInString(strings.TrimSpace(l.Work)) < 20 {
			flags = append(flags, approvalFlag{l.Date, "tip", "รายละเอียดงานสั้นมาก อาจขอให้นักศึกษาเขียนเพิ่ม"})
		}
	}
	skip := map[string]bool{}
	for _, h := range holidays {
		skip[h] = true
	}
	first, last := from, to
	if start != "" && start > first {
		first = start
	}
	if end != "" && end < last {
		last = end
	}
	if today < last {
		last = today
	}
	d, err1 := time.Parse("2006-01-02", first)
	stop, err2 := time.Parse("2006-01-02", last)
	if err1 == nil && err2 == nil {
		for i := 0; !d.After(stop) && i < 400; i++ {
			key := d.Format("2006-01-02")
			if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday && !have[key] && !skip[key] {
				flags = append(flags, approvalFlag{key, "error", "ไม่มีบันทึกในวันทำงานนี้"})
			}
			d = d.AddDate(0, 0, 1)
		}
	}
	return flags
}

// POST /api/ai/approval-summary {from, to}
func aiApprovalSummary(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "mentor" {
		return fail(c, 403, "เฉพาะพี่เลี้ยง")
	}
	x, err := internshipFor(u)
	if err != nil {
		return fail(c, 404, "ไม่พบนักศึกษาที่ดูแล")
	}
	var in struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if c.BodyParser(&in) != nil || !validISODate(in.From) || !validISODate(in.To) || in.From > in.To {
		return fail(c, 400, "ช่วงวันที่ไม่ถูกต้อง")
	}
	rows, err := db.Query(`SELECT id,date,start_time,end_time,break_minutes,kind,late,work,blockers,learning,approval_id FROM day_logs WHERE internship_id=? AND date BETWEEN ? AND ? ORDER BY date`, x.ID, in.From, in.To)
	if err != nil {
		return fail(c, 500, "อ่านบันทึกไม่สำเร็จ")
	}
	var logs []DayLog
	pending := 0
	for rows.Next() {
		var l DayLog
		var approval sql.NullInt64
		if err = rows.Scan(&l.ID, &l.Date, &l.StartTime, &l.EndTime, &l.BreakMinutes, &l.Kind, &l.Late, &l.Work, &l.Blockers, &l.Learning, &approval); err != nil {
			rows.Close()
			return fail(c, 500, "อ่านบันทึกไม่สำเร็จ")
		}
		l.Hours = hours(l)
		l.ApprovalID = approval.Int64
		if !approval.Valid {
			pending++
		}
		logs = append(logs, l)
	}
	rows.Close()
	today := time.Now().In(time.FixedZone("ICT", 7*3600)).Format("2006-01-02")
	resp := fiber.Map{
		"flags":         approvalFlags(logs, in.From, in.To, x.StartDate, x.EndDate, x.HolidayDates, today),
		"log_count":     len(logs),
		"pending_count": pending,
		"summary":       "",
		"ai_used":       false,
		"ai_error":      "",
	}
	if len(logs) == 0 || !aiEnabled() {
		return c.JSON(resp)
	}
	if !aiAllow(u.ID) {
		resp["ai_error"] = "ใช้ AI บ่อยเกินไป กรุณารอสักครู่"
		return c.JSON(resp)
	}
	digest, count, _ := logsDigest(x.ID, in.From, in.To, 9000)
	prompt := fmt.Sprintf(`สรุปบันทึกการฝึกงานของนักศึกษา "%s" ช่วง %s ถึง %s ให้พี่เลี้ยงอ่านก่อนกดรับรอง
ใช้รูปแบบนี้ สั้น กระชับ ไม่เกิน 12 บรรทัด:
ภาพรวม: <1–2 ประโยค>
งานหลัก:
- ...
ปัญหาที่พบ:
- ... (ถ้าไม่มี เขียน "- ไม่มี")
ข้อสังเกตสำหรับพี่เลี้ยง:
- <สิ่งที่ควรถามหรือตรวจเพิ่ม เช่น บันทึกที่คลุมเครือหรือซ้ำกันหลายวัน ถ้าไม่มี เขียน "- ไม่มี">

ข้อมูลการฝึกงาน:
%s
บันทึกรายวัน (%d วัน):
%s`, x.StudentName, in.From, in.To, internshipContext(x), count, digest)
	ctx, cancel := context.WithTimeout(c.Context(), 110*time.Second)
	defer cancel()
	text, err := aiChat(ctx, prompt)
	if err != nil {
		resp["ai_error"] = "AI สรุปไม่สำเร็จ: " + err.Error()
		return c.JSON(resp)
	}
	resp["summary"] = cleanAIText(text, "")
	resp["ai_used"] = true
	return c.JSON(resp)
}
