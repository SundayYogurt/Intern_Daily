package main

import (
	"encoding/base64"
	"fmt"
	"html"
	"sort"
	"strings"
)

type reportField struct{ Label, Key string }
type reportChapter struct {
	Title  string
	Fields []reportField
}

var reportChapters = []reportChapter{
	{"บทที่ 1 บทนำ", []reportField{{"1.1 ความเป็นมาและความสำคัญของปัญหา", "background"}, {"1.2 วัตถุประสงค์ของโครงงาน", "objectives"}, {"1.3 ขอบเขตของโครงงาน", "scope"}, {"1.4 ข้อมูลของบริษัท", "company"}, {"1.5 ประโยชน์ที่คาดว่าจะได้รับ", "benefits"}, {"1.6 ตำแหน่งที่ปฏิบัติ", "role"}}},
	{"บทที่ 2 รายละเอียดโครงงาน", []reportField{{"2.1 วิเคราะห์และออกแบบการทำงานของระบบ", "project"}, {"2.2 วิธีดำเนินงาน", "methods"}}},
	{"บทที่ 3 ผลการดำเนินงาน", []reportField{{"3.1 เครื่องมือที่ใช้", "tools"}, {"3.2 ขั้นตอนการดำเนินงาน", "process"}, {"3.3 ผลการดำเนินงาน", "results"}}},
	{"บทที่ 4 สรุปและข้อเสนอแนะ", []reportField{{"4.1 สรุปผลการดำเนินงาน", "conclusion"}, {"4.2 ประโยชน์ที่ได้รับ", "received_benefits"}, {"4.3 ปัญหาและข้อจำกัด", "constraints"}, {"4.4 ข้อเสนอแนะ", "suggestions"}}},
}

func reportInfo(d *documentData) []string {
	x := d.Internship
	return []string{
		"ชื่อ-นามสกุล: " + clean(d.Student.Name) + "    รหัสนักศึกษา: " + clean(x.StudentNumber),
		"ชื่อสถานประกอบการ: " + clean(x.Company),
		"ที่อยู่: " + clean(x.CompanyAddress),
		"ชื่อพี่เลี้ยง/ผู้บังคับบัญชา: " + clean(x.MentorName),
		"เบอร์โทรติดต่อพี่เลี้ยง/ผู้บังคับบัญชา: " + clean(x.MentorPhone),
		"อีเมลพี่เลี้ยง/ผู้บังคับบัญชา: " + clean(x.MentorEmail),
		"ตำแหน่งงาน/รายละเอียดงานที่ได้รับมอบหมาย: " + clean(x.Position),
		"ระยะเวลาในการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา: " + clean(thDate(x.StartDate)) + " – " + clean(thDate(x.EndDate)),
	}
}

func reportInfoHTML(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return html.EscapeString(line)
	}
	return "<strong>" + html.EscapeString(parts[0]+":") + "</strong>" + html.EscapeString(parts[1])
}

func reportHours(d *documentData) [][]string {
	month := map[string]struct {
		Days  int
		Hours float64
	}{}
	totalDays, totalHours := 0, 0.0
	for _, l := range d.Logs {
		if l.Kind != "work" || len(l.Date) < 7 {
			continue
		}
		key := l.Date[:7]
		item := month[key]
		item.Days++
		item.Hours += l.Hours
		month[key] = item
		totalDays++
		totalHours += l.Hours
	}
	keys := make([]string, 0, len(month))
	for key := range month {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := [][]string{{"เดือน", "วันปฏิบัติงาน", "ชั่วโมง"}}
	for _, key := range keys {
		item := month[key]
		rows = append(rows, []string{key, fmt.Sprint(item.Days), fmt.Sprintf("%.2f", item.Hours)})
	}
	rows = append(rows, []string{"รวม", fmt.Sprint(totalDays), fmt.Sprintf("%.2f", totalHours)})
	return rows
}

func makeReportBlocks(d *documentData) []block {
	x := d.Internship
	b := []block{
		{Kind: "logo"},
		{Kind: "cover-title", Text: "รายงานผลการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา"},
		{Kind: "cover-label", Text: "ผู้จัดทำ"},
		{Kind: "cover-person", Text: clean(d.Student.Name) + "    รหัสนักศึกษา " + clean(x.StudentNumber)},
		{Kind: "cover-footer", Text: "โครงงานเล่มนี้เป็นส่วนหนึ่งของรายวิชาฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา"},
		{Kind: "cover-footer", Text: "รหัสวิชา(7154113)"},
		{Kind: "cover-footer", Text: "สาขาวิชาวิศวกรรมซอฟต์แวร์ คณะวิทยาศาสตร์และเทคโนโลยี"},
		{Kind: "cover-footer", Text: "มหาวิทยาลัยราชภัฏนครปฐม"},
		{Kind: "page"},
		{Kind: "report-center", Text: "ข้อมูลนักศึกษาและสถานประกอบการ"},
	}
	for _, line := range reportInfo(d) {
		b = append(b, block{Kind: "report-info", Text: line})
	}
	b = append(b, block{Kind: "page"}, block{Kind: "report-center", Text: "สารบัญ"})
	for i, chapter := range reportChapters {
		b = append(b, block{Kind: "toc", Text: chapter.Title + "    " + fmt.Sprint(i+1)})
		for _, field := range chapter.Fields {
			b = append(b, block{Kind: "toc-sub", Text: field.Label})
		}
	}
	b = append(b, block{Kind: "toc", Text: "ภาคผนวก"})
	for _, chapter := range reportChapters {
		b = append(b, block{Kind: "page"}, block{Kind: "report-center", Text: chapter.Title})
		for _, field := range chapter.Fields {
			b = append(b, block{Kind: "report-section", Text: field.Label})
			b = append(b, block{Kind: "report-body", Text: clean(x.Report[field.Key])})
		}
	}
	b = append(b, block{Kind: "page"}, block{Kind: "report-center", Text: "ภาคผนวก"}, block{Kind: "page"}, block{Kind: "report-center", Text: "สรุประยะเวลาปฏิบัติงาน"}, block{Kind: "table", Rows: reportHours(d)})
	return b
}

func makeReportHTML(d *documentData) string {
	logo, _ := logAssets.ReadFile("templates/university-logo.png")
	x := d.Internship
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="th"><head><meta charset="utf-8"><style>`)
	b.WriteString(logFontCSS())
	b.WriteString(`@page{size:A4;margin:0}*{box-sizing:border-box}html{background:#e5e4e2}body{margin:0;color:#111;font:17px/1.65 LogSarabun,Tahoma,sans-serif}.sheet{width:210mm;min-height:297mm;margin:12px auto;padding:18mm 29mm;background:#fff;box-shadow:0 1px 7px #0002;break-after:page}.sheet:last-child{break-after:auto}.report-cover{text-align:center}.report-logo{display:block;width:35mm;margin:17mm auto 29mm}.report-cover h1{font-size:19px;font-weight:400;margin:0 0 23mm}.report-cover .author-label{margin-bottom:16mm}.report-cover .author{margin-bottom:33mm}.report-cover .foot{margin:3mm 0}.report-info h2,.toc h2,.chapter h2,.appendix h2{text-align:center;font-size:19px;font-weight:600;margin:0 0 15mm}.report-info h2{margin-top:13mm}.report-info p{margin:0 0 5mm}.page-no{text-align:right;font-size:14px;margin:-8mm 0 9mm}.toc-row{display:flex;justify-content:space-between;gap:8mm;margin:3mm 0}.toc-sub{padding-left:11mm;margin:2mm 0}.chapter h2{line-height:1.35;margin-bottom:13mm}.chapter h3{font-size:17px;margin:8mm 0 2mm;font-weight:600}.chapter p{margin:0 0 2mm}` + reportBodyCSS + `.appendix-title{display:flex;align-items:center;justify-content:center;min-height:210mm}.hours{width:100%;border-collapse:collapse}.hours th,.hours td{border:1px solid #333;padding:2mm;text-align:left}.hours th{background:#eee}@media print{html{background:white}.sheet{margin:0;box-shadow:none;page-break-after:always}.sheet:last-child{page-break-after:auto}}</style></head><body>`)
	b.WriteString(`<section class="sheet report-cover"><img class="report-logo" src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(logo) + `" alt="ตรามหาวิทยาลัยราชภัฏนครปฐม"><h1>รายงานผลการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา</h1><p class="author-label">ผู้จัดทำ</p><p class="author">` + html.EscapeString(clean(d.Student.Name)) + `　รหัสนักศึกษา ` + html.EscapeString(clean(x.StudentNumber)) + `</p><p class="foot">โครงงานเล่มนี้เป็นส่วนหนึ่งของรายวิชาฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา</p><p class="foot">รหัสวิชา(7154113)</p><p class="foot">สาขาวิชาวิศวกรรมซอฟต์แวร์ คณะวิทยาศาสตร์และเทคโนโลยี</p><p class="foot">มหาวิทยาลัยราชภัฏนครปฐม</p></section>`)
	b.WriteString(`<section class="sheet report-info"><h2>ข้อมูลนักศึกษาและสถานประกอบการ</h2>`)
	for _, line := range reportInfo(d) {
		b.WriteString("<p>" + reportInfoHTML(line) + "</p>")
	}
	b.WriteString(`</section><section class="sheet toc"><div class="page-no">ก</div><h2>สารบัญ</h2><div class="toc-row"><strong>หัวข้อ</strong><strong>หน้า</strong></div>`)
	for i, chapter := range reportChapters {
		b.WriteString(`<div class="toc-row"><strong>` + html.EscapeString(chapter.Title) + `</strong><span>` + fmt.Sprint(i+1) + `</span></div>`)
		for _, field := range chapter.Fields {
			b.WriteString(`<div class="toc-sub">` + html.EscapeString(field.Label) + `</div>`)
		}
	}
	b.WriteString(`<div class="toc-row"><strong>ภาคผนวก</strong><span>5</span></div></section>`)
	for i, chapter := range reportChapters {
		b.WriteString(`<section class="sheet chapter"><div class="page-no">` + fmt.Sprint(i+1) + `</div><h2>` + html.EscapeString(chapter.Title) + `</h2>`)
		for _, field := range chapter.Fields {
			b.WriteString("<h3>" + html.EscapeString(field.Label) + "</h3>" + reportBodyHTML(x.Report[field.Key]))
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`<section class="sheet appendix"><div class="page-no">5</div><h2 class="appendix-title">ภาคผนวก</h2></section><section class="sheet appendix"><div class="page-no">6</div><h2>สรุประยะเวลาปฏิบัติงาน</h2><table class="hours"><thead><tr><th>เดือน</th><th>วันปฏิบัติงาน</th><th>ชั่วโมง</th></tr></thead><tbody>`)
	for _, row := range reportHours(d)[1:] {
		b.WriteString("<tr>")
		for _, cell := range row {
			b.WriteString("<td>" + html.EscapeString(cell) + "</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString(`</tbody></table></section></body></html>`)
	return b.String()
}
