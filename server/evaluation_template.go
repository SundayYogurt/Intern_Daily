package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"strings"
)

func evaluationCounts(d *documentData) (late, sick, personal, work int) {
	for _, l := range d.Logs {
		if l.Late {
			late++
		}
		switch l.Kind {
		case "work":
			work++
		case "sick":
			sick++
		case "personal":
			personal++
		}
	}
	return
}

func evaluationValues(d *documentData) map[string]string {
	x, u, ev := d.Internship, d.Student, d.Evaluation
	late, sick, personal, work := evaluationCounts(d)
	values := map[string]string{
		"company":         clean(x.Company),
		"student_name":    clean(u.Name),
		"student_number":  clean(x.StudentNumber),
		"start_date":      clean(thDate(x.StartDate)),
		"end_date":        clean(thDate(x.EndDate)),
		"mentor_name":     clean(x.MentorName),
		"mentor_position": clean(x.MentorPosition),
		"mentor_phone":    clean(x.MentorPhone),
		"department":      clean(x.Department),
		"position":        clean(x.Position),
		"late_count":      fmt.Sprint(late),
		"sick_count":      fmt.Sprint(sick),
		"personal_count":  fmt.Sprint(personal),
		"work_days":       fmt.Sprint(work),
		"improvement":     clean(ev.Improvement),
		"strengths":       clean(ev.Strengths),
		"suggestions":     clean(ev.Suggestions),
		"typed_name":      clean(ev.TypedName),
		"submitted_date":  clean(thDateTime(ev.SubmittedAt)),
	}
	for _, group := range evaluationGroups {
		for _, item := range group.Items {
			for _, score := range []string{"4", "3", "2", "1", "0", "*"} {
				key := "score_" + strings.ReplaceAll(item.Code, ".", "_") + "_" + score
				if score == "*" {
					key = "score_" + strings.ReplaceAll(item.Code, ".", "_") + "_star"
				}
				if ev.Scores[item.Code] == score {
					values[key] = "✓"
				}
			}
		}
	}
	return values
}

func makeEvaluationDOCX(d *documentData) ([]byte, error) {
	template, err := logAssets.ReadFile("templates/evaluation_template.docx")
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, err
	}
	values := evaluationValues(d)
	var output bytes.Buffer
	zw := zip.NewWriter(&output)
	for _, file := range zr.File {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return nil, err
		}
		if file.Name == "word/document.xml" {
			content = []byte(logMarker.ReplaceAllStringFunc(string(content), func(marker string) string {
				return xmlText(values[strings.TrimSuffix(strings.TrimPrefix(marker, "{{"), "}}")])
			}))
		}
		writer, err := zw.Create(file.Name)
		if err != nil {
			return nil, err
		}
		if _, err = writer.Write(content); err != nil {
			return nil, err
		}
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func evaluationRow(b *strings.Builder, item struct{ Code, Text string }, scores map[string]string) {
	b.WriteString(`<tr><td class="criterion">` + html.EscapeString(item.Code+" "+item.Text) + `</td>`)
	for _, score := range []string{"4", "3", "2", "1", "0", "*"} {
		mark := ""
		if scores[item.Code] == score {
			mark = "✓"
		}
		b.WriteString(`<td class="score">` + mark + `</td>`)
	}
	b.WriteString(`</tr>`)
}

func makeEvaluationHTML(d *documentData) string {
	x, u, ev := d.Internship, d.Student, d.Evaluation
	late, sick, personal, work := evaluationCounts(d)
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="th"><head><meta charset="utf-8"><style>`)
	b.WriteString(logFontCSS())
	b.WriteString(`@page{size:A4;margin:0}*{box-sizing:border-box}html{background:#e5e4e2}body{margin:0;font:16px/1.4 LogSarabun,Tahoma,sans-serif;color:#111}.sheet{width:210mm;min-height:297mm;background:white;margin:12px auto;padding:17mm 18mm;box-shadow:0 1px 7px #0002;break-after:page}.sheet:last-child{break-after:auto}.school{text-align:center;margin:0;font-size:16px}.title{text-align:center;font-size:23px;line-height:1.3;margin:2mm 0 11mm}.form{width:100%;border-collapse:collapse;table-layout:fixed;font-size:14px}.form td,.form th{border:1px solid #333;padding:1mm 1.5mm;vertical-align:middle}.form .criterion{width:66%}.form .score{width:5.66%;text-align:center}.form th{font-weight:700}.form .group{font-weight:700;background:#f5f5f5}.form .meta{font-size:14px}.note{min-height:19mm;vertical-align:top!important}.signature{height:28mm;text-align:center}.continuation{text-align:center;font-size:19px;margin:0 0 7mm}@media print{html{background:white}.sheet{margin:0;box-shadow:none;page-break-after:always}.sheet:last-child{page-break-after:auto}}</style></head><body>`)
	b.WriteString(`<section class="sheet"><p class="school">คณะวิทยาศาสตร์และเทคโนโลยี สาขาวิชาวิศวกรรมซอฟต์แวร์</p><h1 class="title">แบบประเมินการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา<br>ด้านวิศวกรรมซอฟต์แวร์</h1><table class="form"><colgroup><col style="width:66%"><col span="6" style="width:5.666%"></colgroup><tbody>`)
	for _, line := range []string{
		"หน่วยงาน/บริษัท " + clean(x.Company),
		"ชื่อนักศึกษา " + clean(u.Name) + "    เลขประจำตัว " + clean(x.StudentNumber),
		"วันเริ่มการฝึกงาน " + clean(thDate(x.StartDate)) + "    สิ้นสุดการฝึกงาน " + clean(thDate(x.EndDate)),
		"ผู้ควบคุมงาน " + clean(x.MentorName) + "    ตำแหน่ง " + clean(x.MentorPosition) + "    โทร " + clean(x.MentorPhone),
		"แผนก/งานที่ฝึก " + clean(x.Department) + " / " + clean(x.Position),
		fmt.Sprintf("สาย %d ครั้ง    ลาป่วย %d ครั้ง    ลากิจ %d ครั้ง    รวมเวลาฝึกงาน %d วัน", late, sick, personal, work),
		"เกณฑ์การให้คะแนน 4 = ดีมาก  3 = ดี  2 = พอใช้  1 = ต้องปรับปรุง  0 = ไม่ผ่าน  * = ไม่สามารถประเมินได้",
	} {
		b.WriteString(`<tr><td class="meta" colspan="7">` + html.EscapeString(line) + `</td></tr>`)
	}
	b.WriteString(`<tr><th class="criterion">รายการประเมิน</th><th class="score">4</th><th class="score">3</th><th class="score">2</th><th class="score">1</th><th class="score">0</th><th class="score">*</th></tr>`)
	for _, group := range evaluationGroups[:3] {
		b.WriteString(`<tr><td class="group" colspan="7">` + html.EscapeString(group.Title) + `</td></tr>`)
		for _, item := range group.Items {
			evaluationRow(&b, item, ev.Scores)
		}
	}
	b.WriteString(`</tbody></table></section><section class="sheet"><p class="continuation">แบบประเมินการฝึกประสบการณ์วิชาชีพ/สหกิจศึกษา (ต่อ)</p><table class="form"><colgroup><col style="width:66%"><col span="6" style="width:5.666%"></colgroup><tbody>`)
	for _, group := range evaluationGroups[3:] {
		b.WriteString(`<tr><td class="group" colspan="7">` + html.EscapeString(group.Title) + `</td></tr>`)
		for _, item := range group.Items {
			evaluationRow(&b, item, ev.Scores)
		}
	}
	for _, row := range []struct{ label, value string }{{"ข้อบกพร่องที่ควรแก้ไข ปรับปรุง", ev.Improvement}, {"ความสามารถพิเศษ", ev.Strengths}, {"ข้อเสนอแนะอื่น ๆ", ev.Suggestions}} {
		b.WriteString(`<tr><td class="note" colspan="7"><strong>` + html.EscapeString(row.label) + `</strong><br>` + html.EscapeString(clean(row.value)) + `</td></tr>`)
	}
	b.WriteString(`<tr><td class="signature" colspan="7">ลงชื่อ ` + html.EscapeString(clean(ev.TypedName)) + ` ผู้ประเมิน<br>(` + html.EscapeString(clean(ev.TypedName)) + `)<br>ตำแหน่ง ` + html.EscapeString(clean(x.MentorPosition)) + `<br>วันที่ ` + html.EscapeString(clean(thDateTime(ev.SubmittedAt))) + `</td></tr></tbody></table></section></body></html>`)
	return b.String()
}
