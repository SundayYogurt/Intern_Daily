package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type documentData struct {
	Student    User
	Internship *Internship
	Logs       []DayLog
	Approvals  map[int64]Approval
	Evaluation *Evaluation
	Photo      []byte
	PhotoType  string
}

type block struct {
	Kind string
	Text string
	Rows [][]string
}

var evaluationGroups = []struct {
	Title string
	Items []struct{ Code, Text string }
}{
	{"1. ระเบียบวินัย", []struct{ Code, Text string }{{"1.1", "การแต่งกายสุภาพเรียบร้อย และถูกระเบียบ"}, {"1.2", "การตรงต่อเวลา และมาปฏิบัติงานอย่างสม่ำเสมอ"}, {"1.3", "ปฏิบัติตามกฎระเบียบ ข้อบังคับ ที่กำหนดไว้อย่างเคร่งครัด"}, {"1.4", "เชื่อฟังและปฏิบัติตามคำแนะนำของหัวหน้างาน"}}},
	{"2. พฤติกรรมในการทำงาน", []struct{ Code, Text string }{{"2.1", "ความซื่อตรงต่องาน"}, {"2.2", "ความขยันขันแข็งในการทำงาน"}, {"2.3", "การรู้จักควบคุมอารมณ์"}, {"2.4", "การรู้จักสิทธิ หน้าที่ และความรับผิดชอบ"}, {"2.5", "ความไว้วางใจสามารถมอบหมายงานให้ได้"}, {"2.6", "ความสามารถในการสื่อสาร ได้ถูกต้องชัดเจนกับผู้ร่วมงานและหัวหน้างาน"}}},
	{"3. คุณภาพของงาน", []struct{ Code, Text string }{{"3.1", "ความสามารถปฏิบัติงานที่ได้รับมอบหมาย"}, {"3.2", "ความสามารถในการแก้ปัญหาเฉพาะหน้าในการทำงาน"}, {"3.3", "ปฏิบัติงานถูกต้องตามลักษณะงาน"}, {"3.4", "ผลงานเป็นที่ยอมรับของหน่วยงาน"}}},
	{"4. วิธีการปฏิบัติงาน", []struct{ Code, Text string }{{"4.1", "ปฏิบัติงานด้วยความรอบคอบและคำนึงถึงความปลอดภัย"}, {"4.2", "เข้าใจขั้นตอนและขอบเขตของการปฏิบัติงาน"}, {"4.3", "การเห็นคุณค่าของสิ่งของ เครื่องใช้"}, {"4.4", "มีความคิดริเริ่มสร้างสรรค์และไหวพริบในการทำงาน"}}},
	{"5. มนุษย์สัมพันธ์", []struct{ Code, Text string }{{"5.1", "ความมีน้ำใจ ให้ความร่วมมือ และประสานงานร่วมกับผู้อื่น"}, {"5.2", "ความสามารถปรับตัวเข้ากับสภาพแวดล้อม"}, {"5.3", "ความสุภาพอ่อนน้อมรู้จักกาละเทศะ"}, {"5.4", "ความสามารถในการแสดงความคิดเห็นและรับฟังผู้อื่น"}}},
}

func loadDocumentData(x *Internship) (*documentData, error) {
	d := &documentData{Internship: x, Logs: []DayLog{}, Approvals: map[int64]Approval{}}
	err := db.QueryRow(`SELECT id,email,role,name,phone,position,birth_date,home_address,subdistrict,district,province,country,skills,interests,photo,photo_type FROM users WHERE id=?`, x.StudentID).Scan(&d.Student.ID, &d.Student.Email, &d.Student.Role, &d.Student.Name, &d.Student.Phone, &d.Student.Position, &d.Student.BirthDate, &d.Student.HomeAddress, &d.Student.Subdistrict, &d.Student.District, &d.Student.Province, &d.Student.Country, &d.Student.Skills, &d.Student.Interests, &d.Photo, &d.PhotoType)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id,date,start_time,end_time,break_minutes,kind,late,work,blockers,learning,COALESCE(approval_id,0) FROM day_logs WHERE internship_id=? ORDER BY date`, x.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l DayLog
		if rows.Scan(&l.ID, &l.Date, &l.StartTime, &l.EndTime, &l.BreakMinutes, &l.Kind, &l.Late, &l.Work, &l.Blockers, &l.Learning, &l.ApprovalID) == nil {
			l.Hours = hours(l)
			d.Logs = append(d.Logs, l)
		}
	}
	rows.Close()
	rows, err = db.Query(`SELECT id,from_date,to_date,typed_name,signed_at FROM approvals WHERE internship_id=? ORDER BY from_date`, x.ID)
	if err == nil {
		for rows.Next() {
			var a Approval
			if rows.Scan(&a.ID, &a.From, &a.To, &a.TypedName, &a.SignedAt) == nil {
				d.Approvals[a.ID] = a
			}
		}
		rows.Close()
	}
	var ev Evaluation
	var scoresJSON string
	err = db.QueryRow(`SELECT scores_json,improvement,strengths,suggestions,typed_name,submitted_at FROM evaluations WHERE internship_id=?`, x.ID).Scan(&scoresJSON, &ev.Improvement, &ev.Strengths, &ev.Suggestions, &ev.TypedName, &ev.SubmittedAt)
	if err == nil {
		ev.Scores = map[string]string{}
		json.Unmarshal([]byte(scoresJSON), &ev.Scores)
		d.Evaluation = &ev
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	return d, nil
}

func exportDocument(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เอกสารสำหรับนักศึกษาเท่านั้น")
	}
	x, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	kind, format := c.Params("kind"), c.Params("format")
	if kind != "log" && kind != "evaluation" && kind != "report" {
		return fail(c, 404, "ไม่พบเอกสาร")
	}
	if format != "docx" && format != "pdf" {
		return fail(c, 400, "รองรับ DOCX หรือ PDF")
	}
	d, e := loadDocumentData(x)
	if e != nil {
		return fail(c, 500, "อ่านข้อมูลไม่สำเร็จ")
	}
	if kind == "evaluation" && (d.Evaluation == nil || d.Evaluation.SubmittedAt == "") {
		return fail(c, 409, "พี่เลี้ยงยังไม่ส่งแบบประเมิน")
	}
	var body []byte
	if kind == "log" && format == "docx" {
		body, e = makeLogDOCX(d)
	} else if kind == "log" {
		body, e = makePDFHTML(makeLogHTML(d))
	} else if kind == "evaluation" && format == "docx" {
		body, e = makeEvaluationDOCX(d)
	} else if kind == "evaluation" {
		body, e = makePDFHTML(makeEvaluationHTML(d))
	} else if kind == "report" && format == "docx" {
		body, e = makeDOCX(makeReportBlocks(d), d)
	} else if kind == "report" {
		body, e = makePDFHTML(makeReportHTML(d))
	} else if format == "docx" {
		body, e = makeDOCX(makeBlocks(kind, d), d)
	} else {
		body, e = makePDF(makeBlocks(kind, d), d)
	}
	if e != nil {
		return fail(c, 500, "สร้างเอกสารไม่สำเร็จ: "+e.Error())
	}
	filename := map[string]string{"log": "บันทึกการฝึกงาน", "evaluation": "แบบประเมินการฝึกงาน", "report": "รายงานการฝึกงาน"}[kind] + "." + format
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", urlEscape(filename)))
	if format == "docx" {
		c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	} else {
		c.Set("Content-Type", "application/pdf")
	}
	return c.Send(body)
}

func previewDocument(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เอกสารสำหรับนักศึกษาเท่านั้น")
	}
	x, err := internshipFor(u)
	if err != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	kind := c.Params("kind")
	if kind != "log" && kind != "evaluation" && kind != "report" {
		return fail(c, 404, "ไม่พบเอกสาร")
	}
	d, err := loadDocumentData(x)
	if err != nil {
		return fail(c, 500, "อ่านข้อมูลไม่สำเร็จ")
	}
	draft := kind == "evaluation" && (d.Evaluation == nil || d.Evaluation.SubmittedAt == "")
	if draft {
		return fail(c, 409, "รอพี่เลี้ยงส่งแบบประเมิน")
	}
	if d.Evaluation == nil {
		d.Evaluation = &Evaluation{Scores: map[string]string{}}
	}
	page := makeHTML(makeBlocks(kind, d), d)
	if kind == "log" {
		page = makeLogHTML(d)
	} else if kind == "evaluation" {
		page = makeEvaluationHTML(d)
	} else if kind == "report" {
		page = makeReportHTML(d)
	}
	label := "ตัวอย่างเอกสาร · ข้อมูลปัจจุบัน"
	if draft {
		label = "แบบร่าง · รอพี่เลี้ยงส่งแบบประเมิน"
	}
	page = strings.Replace(page, "<body>", `<body><div style="padding:8px 12px;margin-bottom:12mm;border:1px solid #d3d9d3;background:#f3f6f1;color:#385b43;font:10pt Tahoma,Arial,sans-serif">`+html.EscapeString(label)+`</div>`, 1)
	c.Set("Cache-Control", "no-store")
	c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:")
	c.Type("html", "utf-8")
	return c.SendString(page)
}

func urlEscape(s string) string {
	var b strings.Builder
	for _, v := range []byte(s) {
		if v >= 'A' && v <= 'Z' || v >= 'a' && v <= 'z' || v >= '0' && v <= '9' || v == '-' || v == '_' || v == '.' {
			b.WriteByte(v)
		} else {
			fmt.Fprintf(&b, "%%%02X", v)
		}
	}
	return b.String()
}
func thDate(s string) string {
	t, e := time.Parse("2006-01-02", s)
	if e != nil {
		return s
	}
	return fmt.Sprintf("%02d/%02d/%04d", t.Day(), t.Month(), t.Year()+543)
}
func thDateTime(s string) string {
	t, e := time.Parse(time.RFC3339, s)
	if e != nil {
		return s
	}
	loc, _ := time.LoadLocation("Asia/Bangkok")
	return thDate(t.In(loc).Format("2006-01-02")) + " " + t.In(loc).Format("15:04")
}
func clean(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}
func signedText(a Approval) string {
	return "พิมพ์ชื่อรับรอง: " + a.TypedName + "  วันที่ " + thDateTime(a.SignedAt) + "  ครอบคลุม " + thDate(a.From) + "–" + thDate(a.To)
}
func kindLabel(s string) string {
	return map[string]string{"work": "ปฏิบัติงาน", "sick": "ลาป่วย", "personal": "ลากิจ", "holiday": "วันหยุด"}[s]
}

func planLabel(s string) string {
	if s == "cooperative" {
		return "สหกิจศึกษา"
	}
	return "ฝึกงานทั่วไป"
}

func studentAge(birthDate, startDate string) string {
	birth, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return "—"
	}
	at, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		at = time.Now()
	}
	years := at.Year() - birth.Year()
	if at.Month() < birth.Month() || (at.Month() == birth.Month() && at.Day() < birth.Day()) {
		years--
	}
	if years < 0 || years > 120 {
		return "—"
	}
	return fmt.Sprint(years)
}

func pageSignature(d *documentData, start, end int) string {
	if start >= end {
		return "........................................................"
	}
	name := ""
	for _, l := range d.Logs[start:end] {
		a, ok := d.Approvals[l.ApprovalID]
		if !ok || (name != "" && name != a.TypedName) {
			return "........................................................"
		}
		name = a.TypedName
	}
	if name == "" {
		return "........................................................"
	}
	return name
}

func makeBlocks(kind string, d *documentData) []block {
	x := d.Internship
	add := func(b *[]block, k, t string) { *b = append(*b, block{Kind: k, Text: t}) }
	var b []block
	switch kind {
	case "log":
		add(&b, "title", "รายงานการเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์")
		add(&b, "subtitle", "แบบลงเวลาปฏิบัติงานการเตรียมฝึกประสบการณ์")
		add(&b, "text", "สถานที่/หน่วยงาน: "+clean(x.Company))
		add(&b, "text", "เริ่มเตรียมฝึกประสบการณ์ วันที่ "+thDate(x.StartDate))
		add(&b, "text", "สิ้นสุดเตรียมฝึกประสบการณ์ วันที่ "+thDate(x.EndDate))
		add(&b, "text", "โดย")
		add(&b, "text", "นาย/นางสาว "+clean(d.Student.Name))
		add(&b, "text", "รหัสนักศึกษา "+clean(x.StudentNumber)+"    หมู่เรียน "+clean(x.ClassGroup))
		add(&b, "text", "สาขาวิชา "+clean(x.Major)+"    คณะ "+clean(x.Faculty))
		add(&b, "text", "ภาคเรียนที่ 2 ปีการศึกษา 2569")
		add(&b, "text", "มหาวิทยาลัยราชภัฏนครปฐม")
		add(&b, "page", "")
		add(&b, "heading", "ประวัตินักศึกษาเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์")
		if len(d.Photo) > 0 {
			add(&b, "image", "รูปนักศึกษา")
		}
		add(&b, "subheading", "ข้อมูลส่วนตัว")
		add(&b, "text", "ชื่อ (นาย/นางสาว) "+clean(d.Student.Name))
		add(&b, "text", "เกิดวันที่ "+clean(thDate(d.Student.BirthDate))+"    อายุ "+studentAge(d.Student.BirthDate, x.StartDate)+" ปี")
		add(&b, "text", "บ้านเลขที่ ถนน/ซอย "+clean(d.Student.HomeAddress)+"    ตำบล/แขวง "+clean(d.Student.Subdistrict))
		location := d.Student.Province
		if location == "other" {
			location = d.Student.Country
		}
		add(&b, "text", "อำเภอ/เขต "+clean(d.Student.District)+"    จังหวัด/ประเทศ "+clean(location)+"    เบอร์โทร "+clean(d.Student.Phone))
		add(&b, "subheading", "ข้อมูลการศึกษา")
		add(&b, "text", "สาขา "+clean(x.Major)+"    ชั้นปี/รหัสรุ่น 66    หมู่เรียน "+clean(x.ClassGroup))
		add(&b, "text", "คณะ "+clean(x.Faculty)+"    มหาวิทยาลัยราชภัฏนครปฐม")
		add(&b, "subheading", "ข้อมูลการทำงาน")
		add(&b, "text", "ความรู้ความสามารถพิเศษ "+clean(d.Student.Skills))
		add(&b, "text", "ความสนใจพิเศษ "+clean(d.Student.Interests))

		timePages := (len(d.Logs) + 22) / 23
		if timePages == 0 {
			timePages = 1
		}
		for page := 0; page < timePages; page++ {
			add(&b, "page", "")
			add(&b, "heading", "แบบลงเวลาปฏิบัติงาน")
			rows := [][]string{{"วันที่", "เวลาเริ่มปฏิบัติงาน", "เวลาสิ้นสุดการปฏิบัติงาน", "รวมชั่วโมง", "ลายเซ็น"}}
			for offset := 0; offset < 23; offset++ {
				i := page*23 + offset
				if i >= len(d.Logs) {
					rows = append(rows, []string{"", "", "", "", ""})
					continue
				}
				l := d.Logs[i]
				signature := ""
				if approval, ok := d.Approvals[l.ApprovalID]; ok {
					signature = approval.TypedName
				}
				rows = append(rows, []string{thDate(l.Date), l.StartTime, l.EndTime, fmt.Sprintf("%.1f", l.Hours), signature})
			}
			b = append(b, block{Kind: "table", Rows: rows})
			add(&b, "signature", "ลงชื่อ "+pageSignature(d, page*23, min((page+1)*23, len(d.Logs))))
			add(&b, "text", "ผู้ควบคุมเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์")
		}

		notePages := (len(d.Logs) + 20) / 21
		if notePages == 0 {
			notePages = 1
		}
		for page := 0; page < notePages; page++ {
			add(&b, "page", "")
			add(&b, "heading", "บันทึกการเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์")
			rows := [][]string{{"วัน/เดือน/ปี", "เวลา", "งานที่ปฏิบัติในแต่ละวัน"}}
			for offset := 0; offset < 21; offset++ {
				i := page*21 + offset
				if i >= len(d.Logs) {
					rows = append(rows, []string{"", "", ""})
					continue
				}
				l := d.Logs[i]
				clock := ""
				if l.Kind == "work" {
					clock = l.StartTime + "–" + l.EndTime
				}
				work := l.Work
				if work == "" {
					work = kindLabel(l.Kind)
				}
				rows = append(rows, []string{thDate(l.Date), clock, work})
			}
			b = append(b, block{Kind: "table", Rows: rows})
			add(&b, "signature", "ลงชื่อ "+pageSignature(d, page*21, min((page+1)*21, len(d.Logs))))
			add(&b, "text", "ผู้ควบคุมเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์")
		}
	case "evaluation":
		ev := d.Evaluation
		add(&b, "text", "คณะวิทยาศาสตร์และเทคโนโลยี สาขาวิชาวิศวกรรมซอฟต์แวร์")
		add(&b, "title", "แบบประเมินการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษาด้านวิศวกรรมซอฟต์แวร์")
		late, sick, personal, days := 0, 0, 0, 0
		for _, l := range d.Logs {
			switch l.Kind {
			case "work":
				days++
			case "sick":
				sick++
			case "personal":
				personal++
			}
			if l.Late {
				late++
			}
		}
		rows := [][]string{
			{"หน่วยงาน/บริษัท: " + clean(x.Company)},
			{"ชื่อนักศึกษา: " + clean(d.Student.Name) + "    เลขประจำตัว: " + clean(x.StudentNumber)},
			{"วันเริ่มการฝึกงาน: " + thDate(x.StartDate) + "    สิ้นสุดการฝึกงาน: " + thDate(x.EndDate)},
			{"ผู้ควบคุมงาน: " + clean(x.MentorName) + "    ตำแหน่ง: " + clean(x.MentorPosition) + "    โทร: " + clean(x.MentorPhone)},
			{"แผนก/งานที่ฝึก: " + clean(x.Department) + " / " + clean(x.Position)},
			{fmt.Sprintf("สาย %d ครั้ง    ลาป่วย %d ครั้ง    ลากิจ %d ครั้ง    รวมเวลาฝึกงาน %d วัน", late, sick, personal, days)},
			{"เกณฑ์การให้คะแนน  4 = ดีมาก  3 = ดี  2 = พอใช้  1 = ต้องปรับปรุง  0 = ไม่ผ่าน  * = ไม่สามารถประเมินได้"},
			{"รายการประเมิน", "4", "3", "2", "1", "0", "*"},
		}
		for _, group := range evaluationGroups {
			rows = append(rows, []string{group.Title})
			for _, item := range group.Items {
				row := []string{item.Code + " " + item.Text, "", "", "", "", "", ""}
				for i, value := range []string{"4", "3", "2", "1", "0", "*"} {
					if ev.Scores[item.Code] == value {
						row[i+1] = "✓"
					}
				}
				rows = append(rows, row)
			}
		}
		rows = append(rows,
			[]string{"ข้อบกพร่องที่ควรแก้ไข ปรับปรุง: " + clean(ev.Improvement)},
			[]string{"ความสามารถพิเศษ: " + clean(ev.Strengths)},
			[]string{"ข้อเสนอแนะอื่น ๆ: " + clean(ev.Suggestions)},
			[]string{"ลงชื่อ " + clean(ev.TypedName) + " ผู้ประเมิน    ตำแหน่ง " + clean(x.MentorPosition) + "    วันที่ " + thDateTime(ev.SubmittedAt)},
		)
		b = append(b, block{Kind: "table", Rows: rows})
	case "report":
		add(&b, "title", "รายงานผลการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา")
		add(&b, "subtitle", "ผู้จัดทำ")
		add(&b, "text", clean(d.Student.Name)+"    รหัสนักศึกษา "+clean(x.StudentNumber))
		add(&b, "text", "โครงงานเล่มนี้เป็นส่วนหนึ่งของรายวิชาฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา")
		add(&b, "text", "รหัสวิชา 7154113")
		add(&b, "text", "สาขาวิชาวิศวกรรมซอฟต์แวร์ คณะวิทยาศาสตร์และเทคโนโลยี")
		add(&b, "text", "มหาวิทยาลัยราชภัฏนครปฐม")
		add(&b, "page", "")
		add(&b, "heading", "ข้อมูลนักศึกษาและสถานประกอบการ")
		for _, s := range []string{"ชื่อ-นามสกุล: " + d.Student.Name + "    รหัสนักศึกษา: " + clean(x.StudentNumber), "ชื่อสถานประกอบการ: " + clean(x.Company), "ที่อยู่: " + clean(x.CompanyAddress), "ชื่อพี่เลี้ยง/ผู้บังคับบัญชา: " + clean(x.MentorName), "เบอร์โทรพี่เลี้ยง/ผู้บังคับบัญชา: " + clean(x.MentorPhone), "อีเมลพี่เลี้ยง/ผู้บังคับบัญชา: " + clean(x.MentorEmail), "ตำแหน่งงาน/รายละเอียดงานที่ได้รับมอบหมาย: " + clean(x.Position), "แผนการฝึก: " + planLabel(x.PlanType), "ระยะเวลาฝึกงาน: " + thDate(x.StartDate) + " ถึง " + thDate(x.EndDate)} {
			add(&b, "text", s)
		}
		add(&b, "page", "")
		add(&b, "heading", "สารบัญ")
		for _, s := range []string{"บทที่ 1 บทนำ", "บทที่ 2 รายละเอียดโครงงาน", "บทที่ 3 ผลการดำเนินงาน", "บทที่ 4 สรุปและข้อเสนอแนะ", "ภาคผนวก สรุประยะเวลาปฏิบัติงาน"} {
			add(&b, "text", s)
		}
		report := x.Report
		sections := []struct {
			Heading string
			Items   []struct{ Label, Key string }
		}{
			{"บทที่ 1 บทนำ", []struct{ Label, Key string }{{"1.1 ความเป็นมาและความสำคัญ", "background"}, {"1.2 วัตถุประสงค์", "objectives"}, {"1.3 ขอบเขต", "scope"}, {"1.4 ข้อมูลบริษัท", "company"}, {"1.5 ประโยชน์ที่คาดว่าจะได้รับ", "benefits"}, {"1.6 ตำแหน่งที่ปฏิบัติ", "role"}}},
			{"บทที่ 2 รายละเอียดโครงงาน", []struct{ Label, Key string }{{"2.1 วิเคราะห์และออกแบบการทำงานของระบบ", "project"}, {"2.2 วิธีดำเนินงาน", "methods"}}},
			{"บทที่ 3 ผลการดำเนินงาน", []struct{ Label, Key string }{{"3.1 เครื่องมือที่ใช้", "tools"}, {"3.2 ขั้นตอนการดำเนินงาน", "process"}, {"3.3 ผลการดำเนินงาน", "results"}}},
			{"บทที่ 4 สรุปและข้อเสนอแนะ", []struct{ Label, Key string }{{"4.1 สรุปผลการดำเนินงาน", "conclusion"}, {"4.2 ประโยชน์ที่ได้รับ", "received_benefits"}, {"4.3 ปัญหาและข้อจำกัด", "constraints"}, {"4.4 ข้อเสนอแนะ", "suggestions"}}},
		}
		for _, section := range sections {
			add(&b, "page", "")
			add(&b, "heading", section.Heading)
			for _, item := range section.Items {
				add(&b, "subheading", item.Label)
				add(&b, "text", clean(report[item.Key]))
			}
		}
		add(&b, "page", "")
		add(&b, "heading", "ภาคผนวก สรุประยะเวลาปฏิบัติงาน")
		month := map[string]struct {
			Days  int
			Hours float64
		}{}
		totalDays := 0
		totalHours := 0.0
		for _, l := range d.Logs {
			if l.Kind != "work" {
				continue
			}
			m := l.Date[:7]
			v := month[m]
			v.Days++
			v.Hours += l.Hours
			month[m] = v
			totalDays++
			totalHours += l.Hours
		}
		keys := make([]string, 0, len(month))
		for k := range month {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		rows := [][]string{{"เดือน", "วันปฏิบัติงาน", "ชั่วโมง"}}
		for _, k := range keys {
			v := month[k]
			rows = append(rows, []string{k, fmt.Sprint(v.Days), fmt.Sprintf("%.2f", v.Hours)})
		}
		rows = append(rows, []string{"รวม", fmt.Sprint(totalDays), fmt.Sprintf("%.2f", totalHours)})
		b = append(b, block{Kind: "table", Rows: rows})
	}
	return b
}

func xmlText(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
func para(text, style string) string {
	var b strings.Builder
	b.WriteString(`<w:p>`)
	if style != "" {
		b.WriteString(`<w:pPr><w:pStyle w:val="` + style + `"/></w:pPr>`)
	}
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			b.WriteString(`<w:r><w:br/></w:r>`)
		}
		b.WriteString(`<w:r><w:t xml:space="preserve">` + xmlText(line) + `</w:t></w:r>`)
	}
	b.WriteString(`</w:p>`)
	return b.String()
}
func reportInfoPara(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return para(line, "ReportInfo")
	}
	return `<w:p><w:pPr><w:pStyle w:val="ReportInfo"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">` + xmlText(parts[0]+":") + `</w:t></w:r><w:r><w:t xml:space="preserve">` + xmlText(parts[1]) + `</w:t></w:r></w:p>`
}
func docxTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	columns := 0
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	b.WriteString(`<w:tbl><w:tblPr><w:tblBorders><w:top w:val="single"/><w:left w:val="single"/><w:bottom w:val="single"/><w:right w:val="single"/><w:insideH w:val="single"/><w:insideV w:val="single"/></w:tblBorders></w:tblPr>`)
	for i, row := range rows {
		b.WriteString(`<w:tr>`)
		for _, cell := range row {
			b.WriteString(`<w:tc><w:tcPr>`)
			if len(row) == 1 && columns > 1 {
				b.WriteString(fmt.Sprintf(`<w:gridSpan w:val="%d"/>`, columns))
			}
			b.WriteString(`</w:tcPr>`)
			if i == 0 {
				b.WriteString(para(cell, "TableHeader"))
			} else {
				b.WriteString(para(cell, "TableText"))
			}
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}
func photoDrawing() string {
	return `<w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" distT="0" distB="0" distL="0" distR="0"><wp:extent cx="1371600" cy="1371600"/><wp:docPr id="1" name="Student photo"/><a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="1" name="photo"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="rId1"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="1371600" cy="1371600"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`
}

func reportLogoDrawing() string {
	return `<w:p><w:pPr><w:jc w:val="center"/><w:spacing w:before="500" w:after="1700"/></w:pPr><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" distT="0" distB="0" distL="0" distR="0"><wp:extent cx="1260000" cy="1600000"/><wp:docPr id="2" name="University logo"/><a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="2" name="University logo"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="rId3"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="1260000" cy="1600000"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`
}
func makeDOCX(blocks []block, d *documentData) ([]byte, error) {
	var document strings.Builder
	document.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, b := range blocks {
		switch b.Kind {
		case "table":
			document.WriteString(docxTable(b.Rows))
		case "page":
			document.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
		case "image":
			document.WriteString(para(b.Text, "Caption"))
			if len(d.Photo) > 0 {
				document.WriteString(photoDrawing())
			}
		case "logo":
			document.WriteString(reportLogoDrawing())
		case "report-info":
			document.WriteString(reportInfoPara(b.Text))
		default:
			document.WriteString(para(b.Text, map[string]string{"title": "Title", "subtitle": "Subtitle", "heading": "Heading1", "subheading": "Heading2", "signature": "Signature", "text": "Normal", "cover-title": "CoverTitle", "cover-label": "CoverLabel", "cover-person": "CoverPerson", "cover-footer": "CoverFooter", "report-center": "ReportCenter", "report-info": "ReportInfo", "toc": "TOC", "toc-sub": "TOCSub", "report-section": "ReportSection", "report-body": "ReportBody"}[b.Kind]))
		}
	}
	document.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="1134" w:bottom="1134" w:left="1134"/></w:sectPr></w:body></w:document>`)
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`
	for _, s := range []struct{ ID, Size, Bold, Before, After string }{{"Normal", "24", "0", "0", "120"}, {"Title", "36", "1", "0", "240"}, {"Subtitle", "26", "0", "0", "180"}, {"Heading1", "32", "1", "240", "120"}, {"Heading2", "26", "1", "180", "100"}, {"TableHeader", "20", "1", "0", "0"}, {"TableText", "20", "0", "0", "0"}, {"Signature", "20", "0", "80", "140"}, {"Caption", "20", "0", "0", "80"}, {"CoverTitle", "28", "0", "0", "1100"}, {"CoverLabel", "26", "0", "0", "850"}, {"CoverPerson", "26", "0", "0", "1800"}, {"CoverFooter", "25", "0", "0", "180"}, {"ReportCenter", "28", "1", "0", "520"}, {"ReportInfo", "24", "0", "0", "240"}, {"TOC", "24", "1", "160", "140"}, {"TOCSub", "22", "0", "0", "90"}, {"ReportSection", "24", "1", "260", "100"}, {"ReportBody", "24", "0", "0", "150"}} {
		styles += `<w:style w:type="paragraph" w:styleId="` + s.ID + `"><w:name w:val="` + s.ID + `"/><w:pPr><w:spacing w:before="` + s.Before + `" w:after="` + s.After + `"/>`
		if s.ID == "CoverTitle" || s.ID == "CoverLabel" || s.ID == "CoverPerson" || s.ID == "CoverFooter" || s.ID == "ReportCenter" {
			styles += `<w:jc w:val="center"/>`
		}
		styles += `</w:pPr><w:rPr><w:rFonts w:ascii="Tahoma" w:hAnsi="Tahoma" w:eastAsia="Tahoma"/><w:sz w:val="` + s.Size + `"/>`
		if s.Bold == "1" {
			styles += `<w:b/>`
		}
		styles += `</w:rPr></w:style>`
	}
	styles += `</w:styles>`
	photoExt := "png"
	if d.PhotoType == "image/jpeg" {
		photoExt = "jpg"
	}
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`
	if len(d.Photo) > 0 {
		contentTypes += `<Default Extension="` + photoExt + `" ContentType="` + d.PhotoType + `"/>`
	}
	if len(d.Photo) == 0 || photoExt != "png" {
		contentTypes += `<Default Extension="png" ContentType="image/png"/>`
	}
	contentTypes += `</Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`
	if len(d.Photo) > 0 {
		rels += `<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/photo.` + photoExt + `"/>`
	}
	rels += `<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/university-logo.png"/>`
	rels += `</Relationships>`
	files := map[string][]byte{"[Content_Types].xml": []byte(contentTypes), "_rels/.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`), "word/_rels/document.xml.rels": []byte(rels), "word/document.xml": []byte(document.String()), "word/styles.xml": []byte(styles)}
	logo, _ := logAssets.ReadFile("templates/university-logo.png")
	files["word/media/university-logo.png"] = logo
	if len(d.Photo) > 0 {
		files["word/media/photo."+photoExt] = d.Photo
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		w, e := zw.Create(name)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(files[name]); e != nil {
			return nil, e
		}
	}
	if e := zw.Close(); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}

func makeHTML(blocks []block, d *documentData) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="th"><head><meta charset="utf-8"><style>@page{size:A4;margin:18mm 17mm}body{font-family:Tahoma,Arial,sans-serif;color:#111;font-size:11pt;line-height:1.48}h1{font-size:18pt;text-align:center;margin:0 0 12mm}h2{font-size:15pt;margin:8mm 0 3mm;break-after:avoid}h3{font-size:12pt;margin:5mm 0 2mm;break-after:avoid}p{margin:0 0 3mm;white-space:pre-wrap}table{border-collapse:collapse;width:100%;font-size:9pt;margin:3mm 0 5mm;page-break-inside:auto}tr{break-inside:avoid}th,td{border:1px solid #333;padding:1.5mm;vertical-align:top}th{background:#eee}.page{break-before:page}.subtitle{text-align:center;font-size:13pt;margin-bottom:9mm}.signature{font-size:9pt;color:#333;margin:2mm 0 5mm}.photo{width:35mm;height:35mm;object-fit:cover}</style></head><body>`)
	for _, x := range blocks {
		switch x.Kind {
		case "title":
			b.WriteString("<h1>" + html.EscapeString(x.Text) + "</h1>")
		case "subtitle":
			b.WriteString(`<p class="subtitle">` + html.EscapeString(x.Text) + `</p>`)
		case "heading":
			b.WriteString("<h2>" + html.EscapeString(x.Text) + "</h2>")
		case "subheading":
			b.WriteString("<h3>" + html.EscapeString(x.Text) + "</h3>")
		case "page":
			b.WriteString(`<div class="page"></div>`)
		case "signature":
			b.WriteString(`<p class="signature">` + html.EscapeString(x.Text) + `</p>`)
		case "image":
			if len(d.Photo) > 0 {
				b.WriteString(`<p>` + html.EscapeString(x.Text) + `</p><img class="photo" src="data:` + d.PhotoType + `;base64,` + base64.StdEncoding.EncodeToString(d.Photo) + `">`)
			}
		case "table":
			columns := 0
			for _, row := range x.Rows {
				if len(row) > columns {
					columns = len(row)
				}
			}
			b.WriteString("<table>")
			for i, row := range x.Rows {
				b.WriteString("<tr>")
				for _, cell := range row {
					tag := "td"
					if i == 0 {
						tag = "th"
					}
					span := ""
					if len(row) == 1 && columns > 1 {
						span = fmt.Sprintf(` colspan="%d"`, columns)
					}
					b.WriteString("<" + tag + span + ">" + html.EscapeString(cell) + "</" + tag + ">")
				}
				b.WriteString("</tr>")
			}
			b.WriteString("</table>")
		default:
			b.WriteString("<p>" + html.EscapeString(x.Text) + "</p>")
		}
	}
	b.WriteString("</body></html>")
	return b.String()
}
func findBrowser() string {
	if p := os.Getenv("PDF_BROWSER"); p != "" {
		return p
	}
	for _, p := range []string{`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`, `C:\Program Files\Google\Chrome\Application\chrome.exe`, `/usr/bin/chromium`, `/usr/bin/google-chrome`} {
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	return ""
}
func makePDF(blocks []block, d *documentData) ([]byte, error) {
	return makePDFHTML(makeHTML(blocks, d))
}

func makePDFHTML(documentHTML string) ([]byte, error) {
	browser := findBrowser()
	if browser == "" {
		return nil, fmt.Errorf("ไม่พบ Edge, Chrome หรือ Chromium; ตั้งค่า PDF_BROWSER ได้")
	}
	dir, e := os.MkdirTemp("", "internship-pdf-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	htmlPath := filepath.Join(dir, "document.html")
	pdfPath := filepath.Join(dir, "document.pdf")
	if e = os.WriteFile(htmlPath, []byte(documentHTML), 0600); e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	fileURL := "file:///" + strings.ReplaceAll(htmlPath, "\\", "/")
	args := []string{"--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--no-pdf-header-footer", "--user-data-dir=" + filepath.Join(dir, "profile"), "--print-to-pdf=" + pdfPath}
	if runtime.GOOS == "linux" {
		// Containers usually run as root without a usable sandbox or a large /dev/shm.
		args = append(args, "--no-sandbox", "--disable-dev-shm-usage")
	}
	cmd := exec.CommandContext(ctx, browser, append(args, fileURL)...)
	out, e := cmd.CombinedOutput()
	if e != nil {
		detail := strings.TrimSpace(string(out))
		if len(detail) > 400 {
			detail = detail[len(detail)-400:]
		}
		return nil, fmt.Errorf("แปลง PDF ไม่สำเร็จ (%s): %v %s", browser, e, detail)
	}
	if info, statErr := os.Stat(pdfPath); statErr != nil || info.Size() == 0 {
		return nil, fmt.Errorf("แปลง PDF ไม่สำเร็จ (%s): ไม่มีไฟล์ PDF ออกมา %s", browser, strings.TrimSpace(string(out)))
	}
	return os.ReadFile(pdfPath)
}
