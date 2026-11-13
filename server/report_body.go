package main

import (
	"html"
	"regexp"
	"strings"
)

// Report section text is free-form (typed by students or drafted by AI). These helpers
// turn it into proper paragraphs, bullet items and numbered items for the HTML/PDF
// preview and the DOCX export, instead of one block with hard line breaks.

type bodyLine struct {
	Kind   string // para, sub, bullet, number, cont
	Marker string // numbering such as "1.2.1" for Kind=="number"; parent list kind for Kind=="cont"
	Text   string
}

var (
	bulletRe      = regexp.MustCompile(`^[-*•·–]\s+(.+)$`)
	numberRe      = regexp.MustCompile(`^(\d{1,2}(?:\.\d{1,2})*)[.)]?\s+(.+)$`)
	headingRe     = regexp.MustCompile(`^#{1,6}\s*(.+)$`)
	placeholderRe = regexp.MustCompile(`\[(?:เติมข้อมูล|ภาพที่)[^\]]*\]`)
)

// stripInlineMarkdown removes emphasis/code markers that AI models like to add.
func stripInlineMarkdown(s string) string {
	for _, m := range []string{"**", "__", "`"} {
		s = strings.ReplaceAll(s, m, "")
	}
	return s
}

func parseReportBody(text string) []bodyLine {
	text = strings.ReplaceAll(strings.TrimSpace(text), "\r", "")
	if text == "" || text == "—" {
		return []bodyLine{{Kind: "para", Text: "—"}}
	}
	var out []bodyLine
	listKind := ""
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(stripInlineMarkdown(raw))
		if line == "" {
			listKind = ""
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			out = append(out, bodyLine{Kind: "sub", Text: strings.TrimSpace(m[1])})
			listKind = ""
			continue
		}
		if m := bulletRe.FindStringSubmatch(line); m != nil {
			out = append(out, bodyLine{Kind: "bullet", Text: strings.TrimSpace(m[1])})
			listKind = "bullet"
			continue
		}
		if m := numberRe.FindStringSubmatch(line); m != nil {
			marker := m[1]
			if !strings.Contains(marker, ".") {
				marker += "."
			}
			out = append(out, bodyLine{Kind: "number", Marker: marker, Text: strings.TrimSpace(m[2])})
			listKind = "number"
			continue
		}
		if listKind != "" {
			out = append(out, bodyLine{Kind: "cont", Marker: listKind, Text: line})
			continue
		}
		out = append(out, bodyLine{Kind: "para", Text: line})
	}
	if len(out) == 0 {
		return []bodyLine{{Kind: "para", Text: "—"}}
	}
	return out
}

// highlightHTML escapes text and marks [เติมข้อมูล: …] / [ภาพที่ …] placeholders.
func highlightHTML(s string) string {
	var b strings.Builder
	last := 0
	for _, loc := range placeholderRe.FindAllStringIndex(s, -1) {
		b.WriteString(html.EscapeString(s[last:loc[0]]))
		b.WriteString(`<mark class="todo">` + html.EscapeString(s[loc[0]:loc[1]]) + `</mark>`)
		last = loc[1]
	}
	b.WriteString(html.EscapeString(s[last:]))
	return b.String()
}

func reportBodyHTML(text string) string {
	var b strings.Builder
	for _, l := range parseReportBody(text) {
		switch l.Kind {
		case "sub":
			b.WriteString(`<p class="b-sub">` + highlightHTML(l.Text) + `</p>`)
		case "bullet":
			b.WriteString(`<div class="b-li"><span class="b-m">•</span><span>` + highlightHTML(l.Text) + `</span></div>`)
		case "number":
			b.WriteString(`<div class="b-li b-num"><span class="b-m">` + html.EscapeString(l.Marker) + `</span><span>` + highlightHTML(l.Text) + `</span></div>`)
		case "cont":
			b.WriteString(`<p class="b-cont b-cont-` + l.Marker + `">` + highlightHTML(l.Text) + `</p>`)
		default:
			b.WriteString(`<p class="b-para">` + highlightHTML(l.Text) + `</p>`)
		}
	}
	return b.String()
}

const reportBodyCSS = `.chapter .b-para{text-indent:10mm;text-align:justify;text-justify:inter-character;margin:0 0 2mm}.chapter .b-sub{font-weight:600;margin:3mm 0 1.5mm}.chapter .b-li{display:flex;gap:2.5mm;margin:0 0 1.5mm 10mm}.chapter .b-li .b-m{flex:none;min-width:4mm}.chapter .b-num .b-m{min-width:11mm}.chapter .b-cont{margin:0 0 1.5mm 16.5mm}.chapter .b-cont-number{margin-left:23.5mm}.chapter mark.todo{background:#fff3b0;padding:0 1mm;border-radius:1mm}`

// docxRuns writes text as runs, highlighting placeholders in yellow.
func docxRuns(s string, bold bool) string {
	rpr := ""
	if bold {
		rpr = `<w:rPr><w:b/></w:rPr>`
	}
	var b strings.Builder
	last := 0
	write := func(t string, mark bool) {
		if t == "" {
			return
		}
		pr := rpr
		if mark {
			pr = `<w:rPr>` + map[bool]string{true: `<w:b/>`, false: ``}[bold] + `<w:highlight w:val="yellow"/></w:rPr>`
		}
		b.WriteString(`<w:r>` + pr + `<w:t xml:space="preserve">` + xmlText(t) + `</w:t></w:r>`)
	}
	for _, loc := range placeholderRe.FindAllStringIndex(s, -1) {
		write(s[last:loc[0]], false)
		write(s[loc[0]:loc[1]], true)
		last = loc[1]
	}
	write(s[last:], false)
	return b.String()
}

// reportBodyDocx renders one report section as Word paragraphs (twips: 567 ≈ 1 cm).
func reportBodyDocx(text string) string {
	var b strings.Builder
	p := func(ind, jc, runs string) {
		b.WriteString(`<w:p><w:pPr><w:pStyle w:val="ReportBody"/>` + ind + jc + `</w:pPr>` + runs + `</w:p>`)
	}
	for _, l := range parseReportBody(text) {
		switch l.Kind {
		case "sub":
			p("", "", docxRuns(l.Text, true))
		case "bullet":
			p(`<w:ind w:left="964" w:hanging="397"/>`, "", docxRuns("•  "+l.Text, false))
		case "number":
			p(`<w:ind w:left="1247" w:hanging="680"/>`, "", docxRuns(l.Marker+"  "+l.Text, false))
		case "cont":
			left := "964"
			if l.Marker == "number" {
				left = "1247"
			}
			p(`<w:ind w:left="`+left+`"/>`, "", docxRuns(l.Text, false))
		default:
			p(`<w:ind w:firstLine="567"/>`, `<w:jc w:val="thaiDistribute"/>`, docxRuns(l.Text, false))
		}
	}
	return b.String()
}
