package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"time"
)

//go:embed templates/log_template.docx templates/evaluation_template.docx templates/university-logo.png templates/*.woff2 templates/time_extra.xml templates/day_extra.xml
var logAssets embed.FS

var logMarker = regexp.MustCompile(`\{\{[a-zA-Z0-9_]+\}\}`)

func thaiLongDate(value string) string {
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "................................"
	}
	months := []string{"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม"}
	return fmt.Sprintf("%d %s พ.ศ. %d", date.Day(), months[date.Month()-1], date.Year()+543)
}

func logLocation(u User) string {
	if u.Province == "other" {
		return u.Country
	}
	return u.Province
}

func logWorkLines(l DayLog) [3]string {
	var lines [3]string
	lines[0] = l.Work
	if lines[0] == "" {
		lines[0] = kindLabel(l.Kind)
	}
	if l.Blockers != "" {
		lines[1] = "ปัญหา: " + l.Blockers
	}
	if l.Learning != "" {
		lines[2] = "สิ่งที่ได้เรียนรู้: " + l.Learning
	}
	return lines
}

func logPageCounts(days int) (int, int) {
	return max(6, (days+22)/23), max(17, (days+6)/7)
}

func logValues(d *documentData) map[string]string {
	x, u := d.Internship, d.Student
	values := map[string]string{
		"company":         clean(x.Company),
		"start_date_long": thaiLongDate(x.StartDate),
		"end_date_long":   thaiLongDate(x.EndDate),
		"student_name":    clean(u.Name),
		"student_number":  clean(x.StudentNumber),
		"class_group":     clean(x.ClassGroup),
		"major":           clean(x.Major),
		"faculty":         clean(x.Faculty),
		"birth_date_long": thaiLongDate(u.BirthDate),
		"student_age":     studentAge(u.BirthDate, x.StartDate),
		"home_address":    clean(u.HomeAddress),
		"subdistrict":     clean(u.Subdistrict),
		"district":        clean(u.District),
		"province":        clean(logLocation(u)),
		"phone":           clean(u.Phone),
		"skills":          clean(u.Skills),
		"interests":       clean(u.Interests),
	}
	timePages, dayPages := logPageCounts(len(d.Logs))
	for page := 0; page < timePages; page++ {
		key := page
		if page >= 6 {
			key = 23 + page - 6
		}
		values[fmt.Sprintf("signature_%d", key)] = pageSignature(d, page*23, min((page+1)*23, len(d.Logs)))
		for row := 0; row < 23; row++ {
			cells := [5]string{}
			i := page*23 + row
			if i < len(d.Logs) {
				l := d.Logs[i]
				cells = [5]string{thDate(l.Date), l.StartTime, l.EndTime, fmt.Sprintf("%.1f", l.Hours), ""}
				if a, ok := d.Approvals[l.ApprovalID]; ok {
					cells[4] = a.TypedName
				}
			}
			for col, value := range cells {
				values[fmt.Sprintf("cell_%d_%d_%d", key, row, col)] = value
			}
		}
	}
	for page := 0; page < dayPages; page++ {
		first := page * 7
		key := page + 6
		if page >= 17 {
			key = 23 + timePages - 6 + page - 17
		}
		values[fmt.Sprintf("signature_%d", key)] = pageSignature(d, first, min(first+7, len(d.Logs)))
		for day := 0; day < 7; day++ {
			prefix := fmt.Sprintf("day_%d_%d_", key, day)
			i := first + day
			if i >= len(d.Logs) {
				continue
			}
			l := d.Logs[i]
			values[prefix+"date"] = thDate(l.Date)
			if l.Kind == "work" {
				values[prefix+"time"] = l.StartTime + "–" + l.EndTime
			}
			for line, value := range logWorkLines(l) {
				values[fmt.Sprintf("%swork_%d", prefix, line)] = value
			}
		}
	}
	return values
}

func logPhotoAnchor(d *documentData) string {
	if len(d.Photo) == 0 {
		return ""
	}
	return `<w:r><w:drawing><wp:anchor xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" distT="0" distB="0" distL="0" distR="0" simplePos="0" relativeHeight="251659265" behindDoc="0" locked="0" layoutInCell="1" allowOverlap="1"><wp:simplePos x="0" y="0"/><wp:positionH relativeFrom="column"><wp:posOffset>4425950</wp:posOffset></wp:positionH><wp:positionV relativeFrom="paragraph"><wp:posOffset>-662305</wp:posOffset></wp:positionV><wp:extent cx="1106170" cy="1192530"/><wp:wrapNone/><wp:docPr id="999" name="Student photo"/><wp:cNvGraphicFramePr/><a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="999" name="Student photo"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="rId99"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="1106170" cy="1192530"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:anchor></w:drawing></w:r>`
}

func makeLogDOCX(d *documentData) ([]byte, error) {
	template, err := logAssets.ReadFile("templates/log_template.docx")
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, err
	}
	values := logValues(d)
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
		switch file.Name {
		case "word/document.xml":
			xml := string(content)
			xml, err = expandLogTemplate(xml, len(d.Logs))
			if err != nil {
				return nil, err
			}
			xml = strings.Replace(xml, `<w:r><w:t>{{student_photo}}</w:t></w:r>`, logPhotoAnchor(d), 1)
			xml = logMarker.ReplaceAllStringFunc(xml, func(marker string) string {
				return xmlText(values[strings.TrimSuffix(strings.TrimPrefix(marker, "{{"), "}}")])
			})
			content = []byte(xml)
		case "word/_rels/document.xml.rels":
			if len(d.Photo) > 0 {
				ext := "png"
				if d.PhotoType == "image/jpeg" {
					ext = "jpg"
				}
				rel := `<Relationship Id="rId99" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/student-photo.` + ext + `"/>`
				content = []byte(strings.Replace(string(content), "</Relationships>", rel+"</Relationships>", 1))
			}
		case "[Content_Types].xml":
			if len(d.Photo) > 0 && d.PhotoType == "image/jpeg" && !bytes.Contains(content, []byte(`Extension="jpg"`)) {
				content = []byte(strings.Replace(string(content), "</Types>", `<Default Extension="jpg" ContentType="image/jpeg"/></Types>`, 1))
			}
		}
		writer, err := zw.Create(file.Name)
		if err != nil {
			return nil, err
		}
		if _, err = writer.Write(content); err != nil {
			return nil, err
		}
	}
	if len(d.Photo) > 0 {
		ext := "png"
		if d.PhotoType == "image/jpeg" {
			ext = "jpg"
		}
		writer, err := zw.Create("word/media/student-photo." + ext)
		if err != nil {
			return nil, err
		}
		if _, err = writer.Write(d.Photo); err != nil {
			return nil, err
		}
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func expandLogTemplate(xml string, days int) (string, error) {
	timePages, dayPages := logPageCounts(days)
	if timePages > 6 {
		snippet, err := logAssets.ReadFile("templates/time_extra.xml")
		if err != nil {
			return "", err
		}
		marker := strings.Index(xml, "{{day_6_0_date}}")
		if marker < 0 {
			return "", fmt.Errorf("ไม่พบตำแหน่งหน้าแบบบันทึกในเทมเพลต")
		}
		table := max(strings.LastIndex(xml[:marker], "<w:tbl "), strings.LastIndex(xml[:marker], "<w:tbl>"))
		if table < 0 {
			return "", fmt.Errorf("ไม่พบตารางแบบบันทึกในเทมเพลต")
		}
		blank := strings.LastIndex(xml[:table], "<w:p ")
		heading := strings.LastIndex(xml[:blank], "<w:p ")
		if blank < 0 || heading < 0 {
			return "", fmt.Errorf("ไม่พบหัวข้อแบบบันทึกในเทมเพลต")
		}
		var extra strings.Builder
		for page := 6; page < timePages; page++ {
			key := 23 + page - 6
			pageXML := strings.ReplaceAll(string(snippet), "cell_5_", fmt.Sprintf("cell_%d_", key))
			pageXML = strings.ReplaceAll(pageXML, "signature_5", fmt.Sprintf("signature_%d", key))
			extra.WriteString(pageXML)
		}
		xml = xml[:heading] + extra.String() + xml[heading:]
	}
	if dayPages > 17 {
		snippet, err := logAssets.ReadFile("templates/day_extra.xml")
		if err != nil {
			return "", err
		}
		section := strings.LastIndex(xml, "<w:sectPr")
		if section < 0 {
			return "", fmt.Errorf("ไม่พบส่วนท้ายเอกสารในเทมเพลต")
		}
		var extra strings.Builder
		for page := 17; page < dayPages; page++ {
			key := 23 + timePages - 6 + page - 17
			pageXML := strings.ReplaceAll(string(snippet), "day_22_", fmt.Sprintf("day_%d_", key))
			pageXML = strings.ReplaceAll(pageXML, "signature_22", fmt.Sprintf("signature_%d", key))
			extra.WriteString(pageXML)
		}
		xml = xml[:section] + extra.String() + xml[section:]
	}
	return xml, nil
}

func logEsc(value string) string { return html.EscapeString(clean(value)) }

func logFontCSS() string {
	var b strings.Builder
	for _, weight := range []string{"400", "700"} {
		for _, script := range []string{"thai", "latin"} {
			data, err := logAssets.ReadFile("templates/sarabun-" + script + "-" + weight + ".woff2")
			if err != nil {
				continue
			}
			rangeCSS := "U+0E00-0E7F"
			if script == "latin" {
				rangeCSS = "U+0000-00FF"
			}
			fmt.Fprintf(&b, "@font-face{font-family:LogSarabun;src:url(data:font/woff2;base64,%s) format('woff2');font-weight:%s;unicode-range:%s}", base64.StdEncoding.EncodeToString(data), weight, rangeCSS)
		}
	}
	return b.String()
}

func makeLogHTML(d *documentData) string {
	x, u := d.Internship, d.Student
	logo, _ := logAssets.ReadFile("templates/university-logo.png")
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="th"><head><meta charset="utf-8"><style>`)
	b.WriteString(logFontCSS())
	b.WriteString(`
@page{size:A4;margin:0}*{box-sizing:border-box}html{background:#e5e4e2}body{margin:0;color:#111;font-family:LogSarabun,Tahoma,sans-serif;font-size:17px;line-height:1.5}.sheet{position:relative;width:210mm;min-height:297mm;margin:12px auto;padding:17mm 19mm;background:white;box-shadow:0 1px 7px #0002;break-after:page;overflow:hidden}.sheet:last-child{break-after:auto}.page-number{text-align:center;font-size:14px;margin:-8mm 0 8mm}.cover{text-align:center;font-weight:600}.cover-logo{display:block;width:35mm;height:auto;margin:5mm auto 2mm}.cover h1{font-size:23px;line-height:1.25;margin:0 auto;font-weight:700}.cover .company{margin-top:23mm}.cover .student{margin-top:23mm}.cover p{margin:2mm 0}.cover .line{border-bottom:2px dotted #333;max-width:150mm;margin:1mm auto 2mm;padding-bottom:1mm;min-height:8mm}.cover .academic{margin-top:22mm}.profile h2,.table-page h2{text-align:center;font-size:22px;line-height:1.25;margin:0 0 20mm;font-weight:700}.profile .photo-frame{position:absolute;top:46mm;right:23mm;width:32mm;height:36mm;border:1px solid #555;object-fit:cover}.profile .section{margin:4mm 0 7mm}.profile .section strong{display:block;font-size:18px}.profile .section p{margin:1.5mm 0 1.5mm 10mm;line-height:1.65}.profile .fill{border-bottom:1px dotted #777;display:inline-block;min-width:28mm;padding:0 2mm}.table-page h2{margin-bottom:8mm}.doc-table{border-collapse:collapse;width:100%;table-layout:fixed;font-size:14px;line-height:1.2}.doc-table th,.doc-table td{border:1px solid #333;padding:1mm;vertical-align:top;overflow-wrap:anywhere}.doc-table th{text-align:center;font-weight:700}.time-table th:nth-child(1){width:15%}.time-table th:nth-child(2){width:22%}.time-table th:nth-child(3){width:25%}.time-table th:nth-child(4){width:14%}.time-table th:nth-child(5){width:24%}.time-table td{height:7.7mm}.day-table th:nth-child(1){width:20%}.day-table th:nth-child(2){width:21%}.day-table th:nth-child(3){width:59%}.day-table td{height:8mm}.signature-block{text-align:right;margin-top:9mm;margin-right:12mm;line-height:1.55}.signature-block span{display:block}.preview-note{background:#f3f6f1;color:#385b43;border:1px solid #d3d9d3;padding:7px 12px;margin:12px auto;width:210mm;font:14px Tahoma,sans-serif}@media print{html{background:white}.sheet{margin:0;box-shadow:none;min-height:297mm;page-break-after:always}.sheet:last-child{page-break-after:auto}.preview-note{display:none}}
</style></head><body>`)
	b.WriteString(`<section class="sheet cover"><img class="cover-logo" src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(logo) + `" alt="ตรามหาวิทยาลัยราชภัฏนครปฐม"><h1>รายงานการเตรียมฝึกประสบการณ์วิชาชีพ<br>ด้านวิศวกรรมซอฟต์แวร์<br>แบบลงเวลาปฏิบัติงานการเตรียมฝึก<br>ประสบการณ์</h1>`)
	b.WriteString(`<div class="company"><p>สถานที่/หน่วยงาน</p><div class="line">` + logEsc(x.Company) + `</div><p>เริ่มเตรียมฝึกประสบการณ์ วันที่ ` + html.EscapeString(thaiLongDate(x.StartDate)) + `</p><p>สิ้นสุดเตรียมฝึกประสบการณ์ วันที่ ` + html.EscapeString(thaiLongDate(x.EndDate)) + `</p></div>`)
	b.WriteString(`<div class="student"><p>โดย</p><p>นาย/นางสาว ` + logEsc(u.Name) + `</p><p>รหัสนักศึกษา ` + logEsc(x.StudentNumber) + `　หมู่เรียน ` + logEsc(x.ClassGroup) + `</p><p>สาขาวิชา ` + logEsc(x.Major) + `　คณะ ` + logEsc(x.Faculty) + `</p></div><div class="academic"><p>ภาคเรียนที่ 2 ปีการศึกษา 2569</p><p>สาขาวิศวกรรมซอฟต์แวร์ คณะวิทยาศาสตร์และเทคโนโลยี</p><p>มหาวิทยาลัยราชภัฏนครปฐม</p></div></section>`)
	b.WriteString(`<section class="sheet profile"><div class="page-number">3</div><h2>ประวัตินักศึกษาเตรียมฝึกประสบการณ์วิชาชีพด้าน<br>วิศวกรรมซอฟต์แวร์</h2>`)
	if len(d.Photo) > 0 {
		b.WriteString(`<img class="photo-frame" src="data:` + d.PhotoType + `;base64,` + base64.StdEncoding.EncodeToString(d.Photo) + `" alt="รูปนักศึกษา">`)
	} else {
		b.WriteString(`<div class="photo-frame" aria-label="ช่องรูปนักศึกษา"></div>`)
	}
	b.WriteString(`<div class="section" style="margin-top:30mm"><strong>ข้อมูลส่วนตัว</strong><p>ชื่อ (นาย/นางสาว) <span class="fill">` + logEsc(u.Name) + `</span></p><p>เกิดวันที่ <span class="fill">` + html.EscapeString(thaiLongDate(u.BirthDate)) + `</span>　อายุ <span class="fill">` + html.EscapeString(studentAge(u.BirthDate, x.StartDate)) + `</span> ปี</p><p>บ้านเลขที่ ถนน/ซอย <span class="fill">` + logEsc(u.HomeAddress) + `</span>　ตำบล/แขวง <span class="fill">` + logEsc(u.Subdistrict) + `</span></p><p>อำเภอ/เขต <span class="fill">` + logEsc(u.District) + `</span>　จังหวัด/ประเทศ <span class="fill">` + logEsc(logLocation(u)) + `</span>　เบอร์โทร <span class="fill">` + logEsc(u.Phone) + `</span></p></div>`)
	b.WriteString(`<div class="section"><strong>ข้อมูลการศึกษา</strong><p>สาขา <span class="fill">` + logEsc(x.Major) + `</span>　ชั้นปี <span class="fill">66</span>　หมู่เรียน <span class="fill">` + logEsc(x.ClassGroup) + `</span></p><p>คณะ <span class="fill">` + logEsc(x.Faculty) + `</span></p></div><div class="section"><strong>ข้อมูลการทำงาน</strong><p>ความรู้ความสามารถพิเศษ <span class="fill">` + logEsc(u.Skills) + `</span></p><p>ความสนใจพิเศษ <span class="fill">` + logEsc(u.Interests) + `</span></p></div></section>`)
	timePages, dayPages := logPageCounts(len(d.Logs))
	for page := 0; page < timePages; page++ {
		b.WriteString(`<section class="sheet table-page"><div class="page-number">` + fmt.Sprint(page+4) + `</div><h2>แบบลงเวลาปฏิบัติงาน</h2><table class="doc-table time-table"><thead><tr><th>วันที่</th><th>เวลาเริ่มปฏิบัติงาน</th><th>เวลาสิ้นสุดการปฏิบัติงาน</th><th>รวมชั่วโมง</th><th>ลายเซ็น</th></tr></thead><tbody>`)
		for row := 0; row < 23; row++ {
			i := page*23 + row
			cells := [5]string{}
			if i < len(d.Logs) {
				l := d.Logs[i]
				cells = [5]string{thDate(l.Date), l.StartTime, l.EndTime, fmt.Sprintf("%.1f", l.Hours), ""}
				if a, ok := d.Approvals[l.ApprovalID]; ok {
					cells[4] = a.TypedName
				}
			}
			b.WriteString("<tr>")
			for _, cell := range cells {
				b.WriteString("<td>" + html.EscapeString(cell) + "</td>")
			}
			b.WriteString("</tr>")
		}
		b.WriteString(`</tbody></table><div class="signature-block"><span>ลงชื่อ ` + html.EscapeString(pageSignature(d, page*23, min((page+1)*23, len(d.Logs)))) + `</span><span>ผู้ควบคุมเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์</span></div></section>`)
	}
	for page := 0; page < dayPages; page++ {
		first := page * 7
		b.WriteString(`<section class="sheet table-page"><div class="page-number">` + fmt.Sprint(page+4+timePages) + `</div><h2>บันทึกการเตรียมฝึกประสบการณ์วิชาชีพ<br>ด้านวิศวกรรมซอฟต์แวร์</h2><table class="doc-table day-table"><thead><tr><th>วัน/เดือน/ปี</th><th>เวลา</th><th>งานที่ปฏิบัติในแต่ละวัน</th></tr></thead><tbody>`)
		for day := 0; day < 7; day++ {
			i := first + day
			date, clock := "", ""
			lines := [3]string{}
			if i < len(d.Logs) {
				l := d.Logs[i]
				date = thDate(l.Date)
				if l.Kind == "work" {
					clock = l.StartTime + "–" + l.EndTime
				}
				lines = logWorkLines(l)
			}
			for line := 0; line < 3; line++ {
				b.WriteString("<tr>")
				if line == 0 {
					b.WriteString(`<td rowspan="3">` + html.EscapeString(date) + `</td>`)
				}
				if line == 0 {
					b.WriteString("<td>" + html.EscapeString(clock) + "</td>")
				} else {
					b.WriteString("<td></td>")
				}
				b.WriteString("<td>" + html.EscapeString(lines[line]) + "</td></tr>")
			}
		}
		b.WriteString(`</tbody></table><div class="signature-block"><span>ลงชื่อ ` + html.EscapeString(pageSignature(d, first, min(first+7, len(d.Logs)))) + `</span><span>ผู้ควบคุมเตรียมฝึกประสบการณ์วิชาชีพด้านวิศวกรรมซอฟต์แวร์</span></div></section>`)
	}
	b.WriteString(`</body></html>`)
	return b.String()
}
