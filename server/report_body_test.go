package main

import (
	"strings"
	"testing"
)

func TestParseReportBody(t *testing.T) {
	got := parseReportBody("**ความเป็นมา**\nบริษัททำซอฟต์แวร์\n\n### ซอฟต์แวร์\n- VS Code\nใช้เขียนโค้ด\n* GitLab\n\n1.2.1 เพื่อเพิ่มทักษะ\n1.2.2 เพื่อศึกษา\n2026 เป็นปีที่เริ่มฝึก")
	want := []bodyLine{
		{Kind: "para", Text: "ความเป็นมา"},
		{Kind: "para", Text: "บริษัททำซอฟต์แวร์"},
		{Kind: "sub", Text: "ซอฟต์แวร์"},
		{Kind: "bullet", Text: "VS Code"},
		{Kind: "cont", Marker: "bullet", Text: "ใช้เขียนโค้ด"},
		{Kind: "bullet", Text: "GitLab"},
		{Kind: "number", Marker: "1.2.1", Text: "เพื่อเพิ่มทักษะ"},
		{Kind: "number", Marker: "1.2.2", Text: "เพื่อศึกษา"},
		{Kind: "cont", Marker: "number", Text: "2026 เป็นปีที่เริ่มฝึก"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines: %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %+v want %+v", i, got[i], want[i])
		}
	}
	if l := parseReportBody("1 ข้อแรก"); l[0].Marker != "1." {
		t.Errorf("single-level marker should get a dot: %+v", l[0])
	}
	if l := parseReportBody("  "); len(l) != 1 || l[0].Text != "—" {
		t.Errorf("empty body should be a dash: %+v", l)
	}
}

func TestReportBodyRenderers(t *testing.T) {
	src := "ย่อหน้า <b>\n- รายการ [เติมข้อมูล: จำนวน] ข้อ"
	h := reportBodyHTML(src)
	for _, want := range []string{`<p class="b-para">ย่อหน้า &lt;b&gt;</p>`, `<div class="b-li"><span class="b-m">•</span>`, `<mark class="todo">[เติมข้อมูล: จำนวน]</mark>`} {
		if !strings.Contains(h, want) {
			t.Errorf("html missing %q in %s", want, h)
		}
	}
	d := reportBodyDocx(src)
	for _, want := range []string{`<w:ind w:firstLine="567"/>`, `<w:ind w:left="964" w:hanging="397"/>`, `<w:highlight w:val="yellow"/>`, `ย่อหน้า &lt;b&gt;`} {
		if !strings.Contains(d, want) {
			t.Errorf("docx missing %q in %s", want, d)
		}
	}
	if strings.Count(d, "<w:p>") != 2 {
		t.Errorf("expected 2 paragraphs, got %s", d)
	}
}

func TestCleanAIText(t *testing.T) {
	in := "แน่นอน นี่คือร่างเนื้อหา:\n## 1.1 ความเป็นมาและความสำคัญของปัญหา\n**บริษัท** ทำซอฟต์แวร์\n\n\n\n* ข้อหนึ่ง\n```\ncode\n```"
	got := cleanAIText(in, "1.1 ความเป็นมาและความสำคัญของปัญหา")
	want := "บริษัท ทำซอฟต์แวร์\n\n- ข้อหนึ่ง\n\ncode"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := cleanAIText("- งาน A\n- งาน B", ""); got != "- งาน A\n- งาน B" {
		t.Fatalf("plain list changed: %q", got)
	}
}
